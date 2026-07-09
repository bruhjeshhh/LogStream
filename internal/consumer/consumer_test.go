package consumer

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

type mockReader struct {
	fetchFunc      func(ctx context.Context) (kafka.Message, error)
	commitFunc     func(ctx context.Context, msgs ...kafka.Message) error
	fetchCalls     int
	commitCalls    int
	committedMsgs  []kafka.Message
	mu             sync.Mutex
}

func (m *mockReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	m.mu.Lock()
	m.fetchCalls++
	m.mu.Unlock()
	return m.fetchFunc(ctx)
}

func (m *mockReader) CommitMessages(ctx context.Context, msgs ...kafka.Message) error {
	m.mu.Lock()
	m.commitCalls++
	m.committedMsgs = append(m.committedMsgs, msgs...)
	m.mu.Unlock()
	return m.commitFunc(ctx, msgs...)
}

func TestRun_ProcessesAndCommitsMessage(t *testing.T) {
	fetched := kafka.Message{
		Offset: 1,
		Value:  []byte(`{"id": "00000000-0000-0000-0000-000000000001", "service": "svc", "level": "info", "message": "hello"}`),
	}

	callCount := 0
	reader := &mockReader{
		fetchFunc: func(ctx context.Context) (kafka.Message, error) {
			callCount++
			if callCount == 1 {
				return fetched, nil
			}
			return kafka.Message{}, context.Canceled
		},
		commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
			return nil
		},
	}

	worker := NewWorker()
	c := NewConsumer(reader, worker)

	err := c.Run(context.Background())
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	reader.mu.Lock()
	defer reader.mu.Unlock()

	if reader.commitCalls != 1 {
		t.Errorf("commitCalls = %d, want 1", reader.commitCalls)
	}
	if len(reader.committedMsgs) != 1 {
		t.Fatalf("committed messages count = %d, want 1", len(reader.committedMsgs))
	}
	if reader.committedMsgs[0].Offset != 1 {
		t.Errorf("committed offset = %d, want 1", reader.committedMsgs[0].Offset)
	}
}

func TestRun_SkipsCommitOnProcessError(t *testing.T) {
	badMsg := kafka.Message{
		Offset: 5,
		Value:  []byte(`not valid json`),
	}

	callCount := 0
	reader := &mockReader{
		fetchFunc: func(ctx context.Context) (kafka.Message, error) {
			callCount++
			if callCount == 1 {
				return badMsg, nil
			}
			return kafka.Message{}, context.Canceled
		},
		commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
			t.Error("commit should not be called when Process fails")
			return nil
		},
	}

	worker := NewWorker()
	c := NewConsumer(reader, worker)

	err := c.Run(context.Background())
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	reader.mu.Lock()
	defer reader.mu.Unlock()

	if reader.commitCalls != 0 {
		t.Errorf("commitCalls = %d, want 0", reader.commitCalls)
	}
}

func TestRun_ReturnsFetchError(t *testing.T) {
	fetchErr := errors.New("broker unreachable")

	reader := &mockReader{
		fetchFunc: func(ctx context.Context) (kafka.Message, error) {
			return kafka.Message{}, fetchErr
		},
		commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
			return nil
		},
	}

	worker := NewWorker()
	c := NewConsumer(reader, worker)

	err := c.Run(context.Background())
	if err != fetchErr {
		t.Fatalf("expected fetchErr, got %v", err)
	}
}

func TestRun_ReturnsCommitError(t *testing.T) {
	commitErr := errors.New("commit failed")
	fetched := kafka.Message{
		Offset: 10,
		Value:  []byte(`{"id": "00000000-0000-0000-0000-000000000001", "service": "svc", "level": "info", "message": "ok"}`),
	}

	reader := &mockReader{
		fetchFunc: func(ctx context.Context) (kafka.Message, error) {
			return fetched, nil
		},
		commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
			return commitErr
		},
	}

	worker := NewWorker()
	c := NewConsumer(reader, worker)

	err := c.Run(context.Background())
	if err != commitErr {
		t.Fatalf("expected commitErr, got %v", err)
	}
}

func TestRun_ProcessesMultipleMessages(t *testing.T) {
	msgs := []kafka.Message{
		{Offset: 1, Value: []byte(`{"id": "00000000-0000-0000-0000-000000000001", "service": "svc1", "level": "info", "message": "m1"}`)},
		{Offset: 2, Value: []byte(`{"id": "00000000-0000-0000-0000-000000000002", "service": "svc2", "level": "warn", "message": "m2"}`)},
		{Offset: 3, Value: []byte(`{"id": "00000000-0000-0000-0000-000000000003", "service": "svc3", "level": "error", "message": "m3"}`)},
	}

	idx := 0
	reader := &mockReader{
		fetchFunc: func(ctx context.Context) (kafka.Message, error) {
			if idx < len(msgs) {
				m := msgs[idx]
				idx++
				return m, nil
			}
			return kafka.Message{}, context.Canceled
		},
		commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
			return nil
		},
	}

	worker := NewWorker()
	c := NewConsumer(reader, worker)

	err := c.Run(context.Background())
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	reader.mu.Lock()
	defer reader.mu.Unlock()

	if reader.commitCalls != 3 {
		t.Errorf("commitCalls = %d, want 3", reader.commitCalls)
	}
}

func TestRun_MixedValidAndInvalidMessages(t *testing.T) {
	msgs := []kafka.Message{
		{Offset: 1, Value: []byte(`{"id": "00000000-0000-0000-0000-000000000001", "service": "svc", "level": "info", "message": "good"}`)},
		{Offset: 2, Value: []byte(`bad json`)},
		{Offset: 3, Value: []byte(`{"id": "00000000-0000-0000-0000-000000000002", "service": "svc", "level": "info", "message": "also good"}`)},
	}

	idx := 0
	reader := &mockReader{
		fetchFunc: func(ctx context.Context) (kafka.Message, error) {
			if idx < len(msgs) {
				m := msgs[idx]
				idx++
				return m, nil
			}
			return kafka.Message{}, context.Canceled
		},
		commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
			return nil
		},
	}

	worker := NewWorker()
	c := NewConsumer(reader, worker)

	err := c.Run(context.Background())
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	reader.mu.Lock()
	defer reader.mu.Unlock()

	if reader.commitCalls != 2 {
		t.Errorf("commitCalls = %d, want 2 (should skip bad message)", reader.commitCalls)
	}
}

func TestRun_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	reader := &mockReader{
		fetchFunc: func(ctx context.Context) (kafka.Message, error) {
			return kafka.Message{}, ctx.Err()
		},
		commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
			return nil
		},
	}

	worker := NewWorker()
	c := NewConsumer(reader, worker)

	err := c.Run(ctx)
	if err != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRun_ContextCancellationDuringProcessing(t *testing.T) {
	goodMsg := kafka.Message{
		Offset: 1,
		Value:  []byte(`{"id": "00000000-0000-0000-0000-000000000001", "service": "svc", "level": "info", "message": "ok"}`),
	}

	callCount := 0
	reader := &mockReader{
		fetchFunc: func(ctx context.Context) (kafka.Message, error) {
			callCount++
			if callCount == 1 {
				return goodMsg, nil
			}
			return kafka.Message{}, context.DeadlineExceeded
		},
		commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
			return nil
		},
	}

	worker := NewWorker()
	c := NewConsumer(reader, worker)

	err := c.Run(context.Background())
	if err != context.DeadlineExceeded {
		t.Fatalf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestNewConsumer_ReturnsNonNil(t *testing.T) {
	reader := &mockReader{
		fetchFunc: func(ctx context.Context) (kafka.Message, error) {
			return kafka.Message{}, context.Canceled
		},
		commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
			return nil
		},
	}
	worker := NewWorker()
	c := NewConsumer(reader, worker)
	if c == nil {
		t.Fatal("NewConsumer returned nil")
	}
}

func TestRun_FetchAndCommitCalledInOrder(t *testing.T) {
	var order []string
	msg := kafka.Message{
		Offset: 1,
		Value:  []byte(`{"id": "00000000-0000-0000-0000-000000000001"}`),
	}

	callCount := 0
	reader := &mockReader{
		fetchFunc: func(ctx context.Context) (kafka.Message, error) {
			order = append(order, "fetch")
			callCount++
			if callCount == 1 {
				return msg, nil
			}
			return kafka.Message{}, context.Canceled
		},
		commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
			order = append(order, "commit")
			return nil
		},
	}

	worker := NewWorker()
	c := NewConsumer(reader, worker)

	_ = c.Run(context.Background())

	expected := []string{"fetch", "commit", "fetch"}
	if len(order) != len(expected) {
		t.Fatalf("order length = %d, want %d: %v", len(order), len(expected), order)
	}
	for i, v := range order {
		if v != expected[i] {
			t.Errorf("order[%d] = %q, want %q", i, v, expected[i])
		}
	}
}

func TestRun_DoesNotBlockOnSlowProcessing(t *testing.T) {
	worker := NewWorker()

	processed := 0
	reader := &mockReader{
		fetchFunc: func(ctx context.Context) (kafka.Message, error) {
			processed++
			if processed <= 2 {
				return kafka.Message{
					Offset: int64(processed),
					Value:  []byte(`{"id": "00000000-0000-0000-0000-000000000001"}`),
				}, nil
			}
			return kafka.Message{}, context.Canceled
		},
		commitFunc: func(ctx context.Context, msgs ...kafka.Message) error {
			return nil
		},
	}

	c := NewConsumer(reader, worker)

	done := make(chan error, 1)
	go func() {
		done <- c.Run(context.Background())
	}()

	select {
	case err := <-done:
		if err != context.Canceled {
			t.Fatalf("expected context.Canceled, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not exit within timeout")
	}
}
