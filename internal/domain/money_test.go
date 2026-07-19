package domain

import (
	"errors"
	"testing"
)

func TestNewMoney(t *testing.T) {
	tests := []struct {
		name         string
		currency     string
		minorUnits   int64
		wantErr      error
		wantCurrency string
	}{
		{name: "valid", currency: "usd", minorUnits: 100, wantCurrency: "USD"},
		{name: "trims and upcases", currency: "  eur ", minorUnits: 1, wantCurrency: "EUR"},
		{name: "empty currency", currency: "   ", minorUnits: 100, wantErr: ErrEmptyCurrency},
		{name: "too short", currency: "US", minorUnits: 100, wantErr: ErrInvalidCurrency},
		{name: "too long", currency: "USDD", minorUnits: 100, wantErr: ErrInvalidCurrency},
		{name: "digits", currency: "US1", minorUnits: 100, wantErr: ErrInvalidCurrency},
		{name: "zero amount", currency: "USD", minorUnits: 0, wantErr: ErrNonPositiveMoney},
		{name: "negative amount", currency: "USD", minorUnits: -5, wantErr: ErrNonPositiveMoney},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			money, err := NewMoney(tt.currency, tt.minorUnits)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				if money != (Money{}) {
					t.Fatalf("expected zero money on error, got %+v", money)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if money.Currency != tt.wantCurrency {
				t.Fatalf("expected currency %q, got %q", tt.wantCurrency, money.Currency)
			}
			if money.MinorUnits != tt.minorUnits {
				t.Fatalf("expected minor units %d, got %d", tt.minorUnits, money.MinorUnits)
			}
		})
	}
}

func TestMoneySameCurrency(t *testing.T) {
	usd := Money{Currency: "USD", MinorUnits: 100}
	if !usd.SameCurrency(Money{Currency: "USD", MinorUnits: 999}) {
		t.Fatal("expected same currency to be true")
	}
	if usd.SameCurrency(Money{Currency: "EUR", MinorUnits: 100}) {
		t.Fatal("expected same currency to be false")
	}
}

func TestMoneyGreaterOrEqual(t *testing.T) {
	tests := []struct {
		name    string
		left    Money
		right   Money
		want    bool
		wantErr error
	}{
		{name: "greater", left: Money{"USD", 200}, right: Money{"USD", 100}, want: true},
		{name: "equal", left: Money{"USD", 100}, right: Money{"USD", 100}, want: true},
		{name: "less", left: Money{"USD", 50}, right: Money{"USD", 100}, want: false},
		{name: "mismatch", left: Money{"USD", 50}, right: Money{"EUR", 100}, wantErr: ErrCurrencyMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.left.GreaterOrEqual(tt.right)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
