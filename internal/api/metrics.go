package api

import (
	"LogStream/internal/buffer"
	"encoding/json"
	"math"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

const latencyWindow = 1000

// ingMetrics holds ingestion-side counters and a rolling latency histogram.
// Counters are monotonic; the consumer hub (and the dashboard) compute
// per-second rates from their deltas.
type ingMetrics struct {
	requests  atomic.Uint64
	accepted  atomic.Uint64
	rejected  atomic.Uint64
	latMu     sync.Mutex
	latencies []float64 // latency in milliseconds
}

var ing ingMetrics

func recordRequest(accepted, rejected int, dur time.Duration) {
	ing.requests.Add(1)
	ing.accepted.Add(uint64(accepted))
	ing.rejected.Add(uint64(rejected))

	ms := float64(dur.Microseconds()) / 1000.0
	ing.latMu.Lock()
	ing.latencies = append(ing.latencies, ms)
	if n := len(ing.latencies); n > latencyWindow {
		ing.latencies = ing.latencies[n-latencyWindow:]
	}
	ing.latMu.Unlock()
}

func percentile(p float64) float64 {
	ing.latMu.Lock()
	samples := append([]float64(nil), ing.latencies...)
	ing.latMu.Unlock()
	if len(samples) == 0 {
		return 0
	}
	sort.Float64s(samples)
	idx := int(p / 100 * float64(len(samples)-1))
	return samples[idx]
}

// MetricsHandler serves the ingestion side of the dashboard contract.
// The consumer hub polls this once per second and merges it into its snapshot.
func MetricsHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ts": time.Now().UnixMilli(),
		"ingestion": map[string]any{
			"requests_total":  ing.requests.Load(),
			"accepted_total":  ing.accepted.Load(),
			"rejected_total":  ing.rejected.Load(),
			"buffer_fill":     len(buffer.IngestChan),
			"buffer_capacity": cap(buffer.IngestChan),
			"p50_latency_ms":  roundMs(percentile(50)),
			"p99_latency_ms":  roundMs(percentile(99)),
		},
	})
}

func roundMs(v float64) float64 { return math.Round(v*100) / 100 }
