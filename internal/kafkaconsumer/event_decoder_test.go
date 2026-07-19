package kafkaconsumer

import (
	"errors"
	"testing"
	"time"
)

func TestDecodePriceChangedEventValid(t *testing.T) {
	data := []byte(`{"event_id":"evt-1","direction_from":"LED","direction_to":"SVO","price":{"currency":"usd","minor_units":5000},"occurred_at":"2026-01-02T03:04:05Z"}`)

	event, err := DecodePriceChangedEvent(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if event.EventID != "evt-1" {
		t.Fatalf("unexpected event id: %q", event.EventID)
	}
	if event.Direction.From != "LED" || event.Direction.To != "SVO" {
		t.Fatalf("unexpected direction: %+v", event.Direction)
	}
	if event.Price.Currency != "USD" || event.Price.MinorUnits != 5000 {
		t.Fatalf("unexpected price: %+v", event.Price)
	}
	if !event.OccurredAt.Equal(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)) {
		t.Fatalf("unexpected occurred at: %v", event.OccurredAt)
	}
}

func TestDecodePriceChangedEventErrors(t *testing.T) {
	tests := []struct {
		name        string
		data        string
		wantInvalid bool
	}{
		{name: "malformed json", data: `{`},
		{name: "bad occurred_at", data: `{"event_id":"e","direction_from":"LED","direction_to":"SVO","price":{"currency":"USD","minor_units":1},"occurred_at":"nope"}`, wantInvalid: true},
		{name: "invalid price", data: `{"event_id":"e","direction_from":"LED","direction_to":"SVO","price":{"currency":"US","minor_units":1},"occurred_at":"2026-01-02T03:04:05Z"}`, wantInvalid: true},
		{name: "empty event id", data: `{"event_id":"","direction_from":"LED","direction_to":"SVO","price":{"currency":"USD","minor_units":1},"occurred_at":"2026-01-02T03:04:05Z"}`, wantInvalid: true},
		{name: "empty direction", data: `{"event_id":"e","direction_from":"","direction_to":"SVO","price":{"currency":"USD","minor_units":1},"occurred_at":"2026-01-02T03:04:05Z"}`, wantInvalid: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := DecodePriceChangedEvent([]byte(tt.data))
			if err == nil {
				t.Fatal("expected an error")
			}
			if tt.wantInvalid && !errors.Is(err, ErrInvalidPriceChangedEvent) {
				t.Fatalf("expected ErrInvalidPriceChangedEvent, got %v", err)
			}
		})
	}
}
