package buffer

import (
	"LogStream/internal/models"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func drainIngestChan() {
	for {
		select {
		case <-IngestChan:
		default:
			return
		}
	}
}

func TestSubmit_SendsToChannel(t *testing.T) {
	drainIngestChan()

	log := models.Log{
		ID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Service: "svc",
		Level:   "info",
		Message: "hello",
	}

	Submit(log)

	select {
	case got := <-IngestChan:
		if got.ID != log.ID {
			t.Errorf("ID = %v, want %v", got.ID, log.ID)
		}
		if got.Service != log.Service {
			t.Errorf("Service = %q, want %q", got.Service, log.Service)
		}
		if got.Level != log.Level {
			t.Errorf("Level = %q, want %q", got.Level, log.Level)
		}
		if got.Message != log.Message {
			t.Errorf("Message = %q, want %q", got.Message, log.Message)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log on IngestChan")
	}
}

func TestSubmit_MultipleLogs(t *testing.T) {
	drainIngestChan()

	count := 10
	for i := 0; i < count; i++ {
		Submit(models.Log{
			ID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Service: "svc",
			Level:   "info",
			Message: "msg",
		})
	}

	for i := 0; i < count; i++ {
		select {
		case <-IngestChan:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for log %d", i)
		}
	}
}

func TestSubmit_ChannelBuffered(t *testing.T) {
	drainIngestChan()

	for i := 0; i < 1000; i++ {
		Submit(models.Log{
			ID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Service: "svc",
			Level:   "info",
			Message: "msg",
		})
	}

	select {
	case <-IngestChan:
	case <-time.After(100 * time.Millisecond):
		t.Error("expected at least one log in the buffered channel after 1000 submits")
	}
}

func TestSubmit_ConcurrentSafe(t *testing.T) {
	drainIngestChan()

	var wg sync.WaitGroup
	n := 50

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Submit(models.Log{
				ID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Service: "svc",
				Level:   "info",
				Message: "msg",
			})
		}()
	}

	wg.Wait()

	received := 0
	for i := 0; i < n; i++ {
		select {
		case <-IngestChan:
			received++
		case <-time.After(time.Second):
			t.Fatalf("timed out, received %d/%d logs", received, n)
		}
	}
}

func TestSubmit_ChannelCapacity(t *testing.T) {
	drainIngestChan()

	if cap(IngestChan) != 1000 {
		t.Errorf("IngestChan capacity = %d, want 1000", cap(IngestChan))
	}
}

func TestSubmit_PreservesAllFields(t *testing.T) {
	drainIngestChan()

	id := uuid.MustParse("00000000-0000-0000-0000-000000000010")
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	log := models.Log{
		ID:                id,
		Service:           "api-gateway",
		Level:             "error",
		Message:           "connection timeout",
		EventTimestamp:    ts,
		ReceivedTimestamp: ts,
	}

	Submit(log)

	select {
	case got := <-IngestChan:
		if got.ID != id {
			t.Errorf("ID = %v, want %v", got.ID, id)
		}
		if got.Service != "api-gateway" {
			t.Errorf("Service = %q, want %q", got.Service, "api-gateway")
		}
		if got.Level != "error" {
			t.Errorf("Level = %q, want %q", got.Level, "error")
		}
		if got.Message != "connection timeout" {
			t.Errorf("Message = %q, want %q", got.Message, "connection timeout")
		}
		if !got.EventTimestamp.Equal(ts) {
			t.Errorf("EventTimestamp = %v, want %v", got.EventTimestamp, ts)
		}
		if !got.ReceivedTimestamp.Equal(ts) {
			t.Errorf("ReceivedTimestamp = %v, want %v", got.ReceivedTimestamp, ts)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log on IngestChan")
	}
}
