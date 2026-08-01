package consumer

import (
	"LogStream/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Dashboard observation hub. Serves the API contract documented in
// dashboard/README.md: an in-memory DLQ/retry registry, a bounded ring buffer
// of processed logs, a single multiplexed SSE stream, and three REST endpoints.
// The hub is per-process; each consumer instance exposes its own view.

const (
	dlqRetrying = "retrying"
	dlqDead     = "dead"
	dlqFlushed  = "flushed"

	maxDLQEntries = 500
	logRingMax    = 500
)

type DLQRecord struct {
	LogID          string     `json:"log_id"`
	Service        string     `json:"service"`
	Level          string     `json:"level"`
	MessagePreview string     `json:"message_preview"`
	RetryCount     int        `json:"retry_count"`
	MaxAttempts    int        `json:"max_attempts"`
	State          string     `json:"state"`
	NextRetryAt    *time.Time `json:"next_retry_at"`
	LastError      string     `json:"last_error"`
	FirstFailedAt  time.Time  `json:"first_failed_at"`
	LastAttemptAt  time.Time  `json:"last_attempt_at"`
}

type LogEvent struct {
	ID         string    `json:"id"`
	Service    string    `json:"service"`
	Level      string    `json:"level"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
	ReceivedAt time.Time `json:"received_at"`
}

type sseClient struct {
	send chan []byte
}

type obsHub struct {
	mu      sync.Mutex
	dlq     map[string]*DLQRecord
	ring    []LogEvent
	clients map[*sseClient]struct{}
	last    *models.MetricsSnapshot
}

var hub = &obsHub{
	dlq:     make(map[string]*DLQRecord),
	clients: make(map[*sseClient]struct{}),
}

func messagePreview(s string) string {
	if len(s) > 160 {
		return s[:160]
	}
	return s
}

// ---------------------------------------------------------------------------
// Event producers (called from the consumer run loop)
// ---------------------------------------------------------------------------

// recordProcessed appends a successfully indexed log to the ring buffer and
// pushes it to the live log stream.
func (h *obsHub) recordProcessed(e models.Log) {
	ev := LogEvent{
		ID:         e.ID.String(),
		Service:    e.Service,
		Level:      e.Level,
		Message:    e.Message,
		Timestamp:  e.EventTimestamp,
		ReceivedAt: e.ReceivedTimestamp,
	}
	h.mu.Lock()
	h.ring = append(h.ring, ev)
	if n := len(h.ring); n > logRingMax {
		h.ring = h.ring[n-logRingMax:]
	}
	h.mu.Unlock()
	h.broadcast("log", ev)
}

func (h *obsHub) upsertDLQ(e models.Log, state string, retryCount, maxAttempts int, next *time.Time, cause error) {
	now := time.Now().UTC()
	h.mu.Lock()
	rec, ok := h.dlq[e.ID.String()]
	if !ok {
		rec = &DLQRecord{
			LogID:          e.ID.String(),
			Service:        e.Service,
			Level:          e.Level,
			MessagePreview: messagePreview(e.Message),
			MaxAttempts:    maxAttempts,
			FirstFailedAt:  now,
		}
		h.dlq[rec.LogID] = rec
	}
	rec.RetryCount = retryCount
	rec.State = state
	rec.NextRetryAt = next
	rec.LastAttemptAt = now
	if cause != nil {
		rec.LastError = cause.Error()
	}
	h.pruneLocked()
	h.mu.Unlock()
	h.broadcast("dlq", rec)
}

// recordRetry schedules the next backoff attempt for a failing log.
func (h *obsHub) recordRetry(e models.Log, attempt, maxAttempts int, delay time.Duration, cause error) {
	next := time.Now().UTC().Add(delay)
	h.upsertDLQ(e, dlqRetrying, attempt, maxAttempts, &next, cause)
}

// recordDead marks a log whose retry budget is exhausted and which is being
// sent to the DLQ topic.
func (h *obsHub) recordDead(e models.Log, maxAttempts int, cause error) {
	h.upsertDLQ(e, dlqDead, maxAttempts, maxAttempts, nil, cause)
}

// recordRecovered flips a previously failing log to FLUSHED once a later
// attempt succeeds. No-op when the log was never in the registry.
func (h *obsHub) recordRecovered(e models.Log) {
	key := e.ID.String()
	h.mu.Lock()
	rec, ok := h.dlq[key]
	if !ok {
		h.mu.Unlock()
		return
	}
	rec.State = dlqFlushed
	rec.NextRetryAt = nil
	rec.LastAttemptAt = time.Now().UTC()
	h.mu.Unlock()
	h.broadcast("dlq", rec)
}

func (h *obsHub) pruneLocked() {
	if len(h.dlq) <= maxDLQEntries {
		return
	}
	for len(h.dlq) > maxDLQEntries {
		var oldest *DLQRecord
		for _, rec := range h.dlq {
			if rec.State == dlqRetrying {
				continue
			}
			if oldest == nil || rec.LastAttemptAt.Before(oldest.LastAttemptAt) {
				oldest = rec
			}
		}
		if oldest == nil {
			for _, rec := range h.dlq {
				if oldest == nil || rec.LastAttemptAt.Before(oldest.LastAttemptAt) {
					oldest = rec
				}
			}
		}
		if oldest == nil {
			break
		}
		delete(h.dlq, oldest.LogID)
	}
}

// ---------------------------------------------------------------------------
// Readers for the REST handlers
// ---------------------------------------------------------------------------

func (h *obsHub) retryingCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := 0
	for _, rec := range h.dlq {
		if rec.State == dlqRetrying {
			n++
		}
	}
	return n
}

func (h *obsHub) snapshotDLQ() []*DLQRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]*DLQRecord, 0, len(h.dlq))
	for _, rec := range h.dlq {
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastAttemptAt.After(out[j].LastAttemptAt)
	})
	return out
}

func (h *obsHub) snapshotLogs() []LogEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]LogEvent, len(h.ring))
	copy(out, h.ring)
	return out
}

func (h *obsHub) snapshot() models.MetricsSnapshot {
	h.mu.Lock()
	last := h.last
	h.mu.Unlock()
	if last != nil {
		return *last
	}
	return buildSnapshot(models.IngestionMetrics{})
}

func (h *obsHub) setSnapshot(s models.MetricsSnapshot) {
	h.mu.Lock()
	h.last = &s
	h.mu.Unlock()
}

// ---------------------------------------------------------------------------
// SSE hub
// ---------------------------------------------------------------------------

func (h *obsHub) addClient(c *sseClient) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *obsHub) removeClient(c *sseClient) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// broadcast writes a named SSE frame to every subscriber, dropping slow
// clients instead of blocking the pipeline.
func (h *obsHub) broadcast(event string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	frame := []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", event, data))
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		select {
		case c.send <- frame:
		default:
		}
	}
}

// ---------------------------------------------------------------------------
// REST + SSE handlers
// ---------------------------------------------------------------------------

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func DashboardMetricsHandler(w http.ResponseWriter, _ *http.Request) {
	cors(w)
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(hub.snapshot())
}

func DashboardDLQHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	w.Header().Set("Content-Type", "application/json")
	limit := 200
	if s := r.URL.Query().Get("limit"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			limit = n
		}
	}
	entries := hub.snapshotDLQ()
	if len(entries) > limit {
		entries = entries[:limit]
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"entries": entries})
}

func DashboardLogsHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	w.Header().Set("Content-Type", "application/json")
	tail := 200
	if s := r.URL.Query().Get("tail"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 {
			tail = n
		}
	}
	logs := hub.snapshotLogs()
	if len(logs) > tail {
		logs = logs[len(logs)-tail:]
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"logs": logs})
}

// DashboardStreamHandler is the single multiplexed SSE stream. It pushes
// "metrics" frames every second (from the ingestion poller) plus "log" and
// "dlq" frames as events occur.
func DashboardStreamHandler(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	cors(w)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	c := &sseClient{send: make(chan []byte, 128)}
	hub.addClient(c)
	defer hub.removeClient(c)

	if _, err := fmt.Fprintf(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case frame := <-c.send:
			if _, err := w.Write(frame); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// ---------------------------------------------------------------------------
// Pollers
// ---------------------------------------------------------------------------

// ConsumerSnapshot reads the live consumer-side numbers.
func ConsumerSnapshot() models.ConsumerMetrics {
	return models.ConsumerMetrics{
		ProcessedTotal: metrics.processed.Load(),
		FailedTotal:    metrics.failed.Load(),
		InFlight:       metrics.inFlight.Load(),
		LagMessages:    metrics.lag.Load(),
		RetryingTotal:  hub.retryingCount(),
		Elasticsearch:  esHealthSnapshot(),
	}
}

func buildSnapshot(ing models.IngestionMetrics) models.MetricsSnapshot {
	return models.MetricsSnapshot{
		TS:        time.Now().UnixMilli(),
		Ingestion: ing,
		Consumer:  ConsumerSnapshot(),
	}
}

// PollIngestion fetches the ingestion service's metrics and returns them.
// A zero-value result is returned on any failure so the hub keeps streaming.
func PollIngestion(ctx context.Context, baseURL string) models.IngestionMetrics {
	var ing models.IngestionMetrics
	client := &http.Client{Timeout: 1500 * time.Millisecond}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/metrics", nil)
	if err != nil {
		return ing
	}
	res, err := client.Do(req)
	if err != nil {
		return ing
	}
	defer res.Body.Close()
	var wrapper struct {
		Ingestion models.IngestionMetrics `json:"ingestion"`
	}
	if err := json.NewDecoder(res.Body).Decode(&wrapper); err != nil {
		return ing
	}
	return wrapper.Ingestion
}

// RunIngestionPoller merges ingestion metrics into the hub snapshot and emits
// a "metrics" SSE frame once per second.
func RunIngestionPoller(ctx context.Context, ingestionURL string) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ing := PollIngestion(ctx, ingestionURL)
			snap := buildSnapshot(ing)
			hub.setSnapshot(snap)
			hub.broadcast("metrics", snap)
		}
	}
}

// ---------------------------------------------------------------------------
// Elasticsearch health
// ---------------------------------------------------------------------------

type esHealthState struct {
	mu     sync.RWMutex
	status string
	docs   int64
}

var esHealth = esHealthState{status: "unreachable"}

func esHealthSnapshot() models.ElasticsearchHealth {
	esHealth.mu.RLock()
	defer esHealth.mu.RUnlock()
	return models.ElasticsearchHealth{
		Status:    esHealth.status,
		Index:     "logs",
		DocsTotal: esHealth.docs,
	}
}

// RunESHealthPoller refreshes cluster status and doc count every few seconds.
func RunESHealthPoller(ctx context.Context) {
	pollESHealth(ctx)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pollESHealth(ctx)
		}
	}
}

func pollESHealth(ctx context.Context) {
	if es == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	status := "unreachable"
	if res, err := es.Cluster.Health(es.Cluster.Health.WithContext(ctx)); err == nil {
		defer res.Body.Close()
		if !res.IsError() {
			var body struct {
				Status string `json:"status"`
			}
			if json.NewDecoder(res.Body).Decode(&body) == nil && body.Status != "" {
				status = body.Status
			}
		}
	}

	var docs int64
	if res, err := es.Count(es.Count.WithContext(ctx), es.Count.WithIndex("logs")); err == nil {
		defer res.Body.Close()
		if !res.IsError() {
			var body struct {
				Count int64 `json:"count"`
			}
			if json.NewDecoder(res.Body).Decode(&body) == nil {
				docs = body.Count
			}
		}
	}

	esHealth.mu.Lock()
	esHealth.status = status
	esHealth.docs = docs
	esHealth.mu.Unlock()
}
