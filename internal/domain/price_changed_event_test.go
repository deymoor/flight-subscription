package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewPriceChangedEvent(t *testing.T) {
	validPrice := Money{Currency: "USD", MinorUnits: 5000}
	occurredAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name       string
		eventID    string
		direction  Direction
		price      Money
		occurredAt time.Time
		wantErr    error
	}{
		{
			name:       "valid",
			eventID:    " evt-1 ",
			direction:  Direction{From: " LED ", To: " SVO "},
			price:      validPrice,
			occurredAt: occurredAt,
		},
		{
			name:       "empty event id",
			eventID:    "   ",
			direction:  Direction{From: "LED", To: "SVO"},
			price:      validPrice,
			occurredAt: occurredAt,
			wantErr:    ErrEmptyEventID,
		},
		{
			name:       "empty direction",
			eventID:    "evt-1",
			direction:  Direction{From: "LED", To: ""},
			price:      validPrice,
			occurredAt: occurredAt,
			wantErr:    ErrEmptyDirection,
		},
		{
			name:       "invalid price",
			eventID:    "evt-1",
			direction:  Direction{From: "LED", To: "SVO"},
			price:      Money{Currency: "US", MinorUnits: 5000},
			occurredAt: occurredAt,
			wantErr:    ErrInvalidCurrency,
		},
		{
			name:       "zero occurred at",
			eventID:    "evt-1",
			direction:  Direction{From: "LED", To: "SVO"},
			price:      validPrice,
			occurredAt: time.Time{},
			wantErr:    ErrZeroOccurredAt,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := NewPriceChangedEvent(tt.eventID, tt.direction, tt.price, tt.occurredAt)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if event.EventID != "evt-1" {
				t.Fatalf("expected trimmed event id, got %q", event.EventID)
			}
			if event.Direction.From != "LED" || event.Direction.To != "SVO" {
				t.Fatalf("expected normalized direction, got %+v", event.Direction)
			}
		})
	}
}
