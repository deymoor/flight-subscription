package kafkaconsumer

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"price-subscriptions/internal/domain"
)

type stubHandler struct {
	errs  []error
	calls int
}

func (h *stubHandler) HandlePriceChanged(_ context.Context, _ domain.PriceChangedEvent) (int, error) {
	index := h.calls
	h.calls++
	if index < len(h.errs) {
		return 0, h.errs[index]
	}
	return 1, nil
}

type stubDLQ struct {
	published  int
	lastReason error
	err        error
	closed     bool
}

func (d *stubDLQ) Publish(_ context.Context, _ kafkago.Message, reason error) error {
	d.published++
	d.lastReason = reason
	return d.err
}

func (d *stubDLQ) Close() error {
	d.closed = true
	return nil
}

func newTestConsumer(handler PriceEventHandler, dlq DeadLetterPublisher, maxRetries int) *Consumer {
	return &Consumer{
		handler:        handler,
		dlq:            dlq,
		maxRetries:     maxRetries,
		initialBackoff: time.Nanosecond,
		maxBackoff:     time.Nanosecond,
	}
}

func TestNewConsumerValidation(t *testing.T) {
	handler := &stubHandler{}

	tests := []struct {
		name    string
		config  Config
		handler PriceEventHandler
		wantErr bool
	}{
		{name: "no brokers", config: Config{Topic: "t", GroupID: "g"}, handler: handler, wantErr: true},
		{name: "blank brokers", config: Config{Brokers: []string{"  "}, Topic: "t", GroupID: "g"}, handler: handler, wantErr: true},
		{name: "no topic", config: Config{Brokers: []string{"b"}, GroupID: "g"}, handler: handler, wantErr: true},
		{name: "no group", config: Config{Brokers: []string{"b"}, Topic: "t"}, handler: handler, wantErr: true},
		{name: "nil handler", config: Config{Brokers: []string{"b"}, Topic: "t", GroupID: "g"}, handler: nil, wantErr: true},
		{name: "valid", config: Config{Brokers: []string{"127.0.0.1:1"}, Topic: "t", GroupID: "g"}, handler: handler},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumer, err := New(tt.config, tt.handler)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidConsumerConfig) {
					t.Fatalf("expected ErrInvalidConsumerConfig, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if consumer == nil {
				t.Fatal("expected consumer")
			}
			_ = consumer.Close()
		})
	}
}

func TestNewConsumerAppliesDefaults(t *testing.T) {
	consumer, err := New(Config{
		Brokers:    []string{"127.0.0.1:1"},
		Topic:      "t",
		GroupID:    "g",
		MaxRetries: -1,
	}, &stubHandler{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer consumer.Close()

	if consumer.maxRetries != defaultMaxRetries {
		t.Fatalf("expected default max retries %d, got %d", defaultMaxRetries, consumer.maxRetries)
	}
	if consumer.initialBackoff != defaultInitialBackoff {
		t.Fatalf("expected default initial backoff, got %v", consumer.initialBackoff)
	}
	if consumer.maxBackoff != defaultMaxBackoff {
		t.Fatalf("expected default max backoff, got %v", consumer.maxBackoff)
	}
	if consumer.dlq != nil {
		t.Fatal("expected no dlq when DLQTopic is empty")
	}
}

func TestNewConsumerClampsMaxBackoff(t *testing.T) {
	consumer, err := New(Config{
		Brokers:        []string{"127.0.0.1:1"},
		Topic:          "t",
		GroupID:        "g",
		InitialBackoff: 10 * time.Second,
		MaxBackoff:     time.Second,
	}, &stubHandler{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer consumer.Close()

	if consumer.maxBackoff != consumer.initialBackoff {
		t.Fatalf("expected maxBackoff clamped to initialBackoff, got %v", consumer.maxBackoff)
	}
}

func TestProcessSuccessFirstAttempt(t *testing.T) {
	handler := &stubHandler{}
	consumer := newTestConsumer(handler, &stubDLQ{}, 3)

	err := consumer.process(context.Background(), kafkago.Message{}, domain.PriceChangedEvent{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handler.calls != 1 {
		t.Fatalf("expected 1 handler call, got %d", handler.calls)
	}
}

func TestProcessRetriesThenSucceeds(t *testing.T) {
	handler := &stubHandler{errs: []error{errors.New("transient"), errors.New("transient")}}
	dlq := &stubDLQ{}
	consumer := newTestConsumer(handler, dlq, 3)

	err := consumer.process(context.Background(), kafkago.Message{}, domain.PriceChangedEvent{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handler.calls != 3 {
		t.Fatalf("expected 3 handler calls, got %d", handler.calls)
	}
	if dlq.published != 0 {
		t.Fatal("expected no dead-letter on eventual success")
	}
}

func TestProcessRetriesExhaustedDeadLetters(t *testing.T) {
	handlerErr := errors.New("permanent")
	handler := &stubHandler{errs: []error{handlerErr, handlerErr, handlerErr}}
	dlq := &stubDLQ{}
	consumer := newTestConsumer(handler, dlq, 2)

	err := consumer.process(context.Background(), kafkago.Message{}, domain.PriceChangedEvent{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handler.calls != 3 {
		t.Fatalf("expected 3 attempts (1 + 2 retries), got %d", handler.calls)
	}
	if dlq.published != 1 {
		t.Fatalf("expected 1 dead-letter, got %d", dlq.published)
	}
	if !errors.Is(dlq.lastReason, handlerErr) {
		t.Fatalf("expected dead-letter reason to be handler error, got %v", dlq.lastReason)
	}
}

func TestProcessDeadLetterWithoutDLQReturnsError(t *testing.T) {
	handlerErr := errors.New("permanent")
	handler := &stubHandler{errs: []error{handlerErr}}
	consumer := newTestConsumer(handler, nil, 0)

	err := consumer.process(context.Background(), kafkago.Message{}, domain.PriceChangedEvent{})
	if !errors.Is(err, handlerErr) {
		t.Fatalf("expected handler error when no dlq configured, got %v", err)
	}
}

func TestProcessStopsOnCancelledContext(t *testing.T) {
	handler := &stubHandler{errs: []error{errors.New("transient")}}
	consumer := newTestConsumer(handler, &stubDLQ{}, 5)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := consumer.process(ctx, kafkago.Message{}, domain.PriceChangedEvent{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestBackoffFor(t *testing.T) {
	consumer := &Consumer{
		initialBackoff: 100 * time.Millisecond,
		maxBackoff:     time.Second,
	}

	tests := []struct {
		attempt int
		want    time.Duration
	}{
		{attempt: 0, want: 100 * time.Millisecond},
		{attempt: 1, want: 200 * time.Millisecond},
		{attempt: 2, want: 400 * time.Millisecond},
		{attempt: 3, want: 800 * time.Millisecond},
		{attempt: 4, want: time.Second},
		{attempt: 10, want: time.Second},
		{attempt: 40, want: time.Second},
	}

	for _, tt := range tests {
		if got := consumer.backoffFor(tt.attempt); got != tt.want {
			t.Fatalf("attempt %d: expected %v, got %v", tt.attempt, tt.want, got)
		}
	}
}

func TestSleepReturnsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := sleep(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestSleepCompletes(t *testing.T) {
	if err := sleep(context.Background(), time.Millisecond); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakeReader struct {
	messages []kafkago.Message
	index    int

	mu        sync.Mutex
	committed []int64
}

func (r *fakeReader) FetchMessage(ctx context.Context) (kafkago.Message, error) {
	if r.index < len(r.messages) {
		message := r.messages[r.index]
		r.index++
		return message, nil
	}

	<-ctx.Done()
	return kafkago.Message{}, ctx.Err()
}

func (r *fakeReader) CommitMessages(_ context.Context, messages ...kafkago.Message) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, message := range messages {
		r.committed = append(r.committed, message.Offset)
	}
	return nil
}

func (r *fakeReader) Close() error { return nil }

func (r *fakeReader) offsets() []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]int64, len(r.committed))
	copy(out, r.committed)
	return out
}

type countingHandler struct {
	active     int32
	maxActive  int32
	totalCalls int32
}

func (h *countingHandler) HandlePriceChanged(_ context.Context, _ domain.PriceChangedEvent) (int, error) {
	atomic.AddInt32(&h.totalCalls, 1)
	current := atomic.AddInt32(&h.active, 1)
	for {
		max := atomic.LoadInt32(&h.maxActive)
		if current <= max || atomic.CompareAndSwapInt32(&h.maxActive, max, current) {
			break
		}
	}
	time.Sleep(5 * time.Millisecond)
	atomic.AddInt32(&h.active, -1)
	return 1, nil
}

func validEventJSON() []byte {
	return []byte(`{"event_id":"evt","direction_from":"LED","direction_to":"SVO",` +
		`"price":{"currency":"USD","minor_units":5000},"occurred_at":"2026-07-17T10:00:00Z"}`)
}

func TestRunPipelinedCommitsInFetchOrder(t *testing.T) {
	const total = 50
	messages := make([]kafkago.Message, 0, total)
	for i := 0; i < total; i++ {
		messages = append(messages, kafkago.Message{Offset: int64(i), Value: validEventJSON()})
	}

	reader := &fakeReader{messages: messages}
	handler := &countingHandler{}
	consumer := &Consumer{
		reader:         reader,
		handler:        handler,
		maxRetries:     0,
		initialBackoff: time.Nanosecond,
		maxBackoff:     time.Nanosecond,
		concurrency:    8,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- consumer.Run(ctx)
	}()

	deadline := time.After(2 * time.Second)
	for {
		if len(reader.offsets()) == total {
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatalf("timeout: committed %d of %d", len(reader.offsets()), total)
		case <-time.After(time.Millisecond):
		}
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}

	offsets := reader.offsets()
	if len(offsets) != total {
		t.Fatalf("expected %d commits, got %d", total, len(offsets))
	}
	for i, offset := range offsets {
		if offset != int64(i) {
			t.Fatalf("offsets not committed in order at %d: got %d", i, offset)
		}
	}
	if handler.maxActive < 2 {
		t.Fatalf("expected concurrent processing, max active was %d", handler.maxActive)
	}
}

func TestRunConcurrencyOneCommitsAllMessages(t *testing.T) {
	messages := []kafkago.Message{
		{Offset: 0, Value: validEventJSON()},
		{Offset: 1, Value: validEventJSON()},
		{Offset: 2, Value: validEventJSON()},
	}
	reader := &fakeReader{messages: messages}
	consumer := &Consumer{
		reader:         reader,
		handler:        &stubHandler{},
		maxRetries:     0,
		initialBackoff: time.Nanosecond,
		maxBackoff:     time.Nanosecond,
		concurrency:    1,
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()

	deadline := time.After(2 * time.Second)
	for {
		if len(reader.offsets()) == len(messages) {
			break
		}
		select {
		case <-deadline:
			cancel()
			t.Fatalf("timeout: committed %d of %d", len(reader.offsets()), len(messages))
		case <-time.After(time.Millisecond):
		}
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if got := reader.offsets(); len(got) != len(messages) {
		t.Fatalf("expected %d commits, got %d", len(messages), len(got))
	}
}
