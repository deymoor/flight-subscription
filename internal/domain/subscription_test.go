package domain

import (
	"errors"
	"testing"
)

func TestNewSubscription(t *testing.T) {
	validPrice := Money{Currency: "USD", MinorUnits: 10000}

	tests := []struct {
		name      string
		userID    string
		direction Direction
		maxPrice  Money
		wantErr   error
	}{
		{
			name:      "valid",
			userID:    " user-1 ",
			direction: Direction{From: " LED ", To: " SVO "},
			maxPrice:  validPrice,
		},
		{
			name:      "empty user id",
			userID:    "   ",
			direction: Direction{From: "LED", To: "SVO"},
			maxPrice:  validPrice,
			wantErr:   ErrEmptyUserID,
		},
		{
			name:      "empty direction from",
			userID:    "user-1",
			direction: Direction{From: "  ", To: "SVO"},
			maxPrice:  validPrice,
			wantErr:   ErrEmptyDirection,
		},
		{
			name:      "empty direction to",
			userID:    "user-1",
			direction: Direction{From: "LED", To: ""},
			maxPrice:  validPrice,
			wantErr:   ErrEmptyDirection,
		},
		{
			name:      "invalid price",
			userID:    "user-1",
			direction: Direction{From: "LED", To: "SVO"},
			maxPrice:  Money{Currency: "USD", MinorUnits: 0},
			wantErr:   ErrNonPositiveMoney,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subscription, err := NewSubscription(tt.userID, tt.direction, tt.maxPrice)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if subscription.UserID != "user-1" {
				t.Fatalf("expected trimmed user id, got %q", subscription.UserID)
			}
			if subscription.Direction.From != "LED" || subscription.Direction.To != "SVO" {
				t.Fatalf("expected normalized direction, got %+v", subscription.Direction)
			}
			if !subscription.Active {
				t.Fatal("expected new subscription to be active")
			}
		})
	}
}
