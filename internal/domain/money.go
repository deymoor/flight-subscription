package domain

import (
	"errors"
	"strings"
)

var (
	ErrEmptyCurrency    = errors.New("currency is required")
	ErrInvalidCurrency  = errors.New("currency must be a 3-letter ISO 4217 code")
	ErrNonPositiveMoney = errors.New("amount must be positive")
	ErrCurrencyMismatch = errors.New("cannot compare amounts in different currencies")
)

type Money struct {
	Currency   string
	MinorUnits int64
}

func NewMoney(currency string, minorUnits int64) (Money, error) {
	money := Money{
		Currency:   normalizeCurrency(currency),
		MinorUnits: minorUnits,
	}
	if err := money.Validate(); err != nil {
		return Money{}, err
	}

	return money, nil
}

func (money Money) Validate() error {
	if money.Currency == "" {
		return ErrEmptyCurrency
	}
	if !isValidCurrencyCode(money.Currency) {
		return ErrInvalidCurrency
	}
	if money.MinorUnits <= 0 {
		return ErrNonPositiveMoney
	}

	return nil
}

func (money Money) SameCurrency(other Money) bool {
	return money.Currency == other.Currency
}

func (money Money) GreaterOrEqual(other Money) (bool, error) {
	if !money.SameCurrency(other) {
		return false, ErrCurrencyMismatch
	}

	return money.MinorUnits >= other.MinorUnits, nil
}

func normalizeCurrency(currency string) string {
	return strings.ToUpper(strings.TrimSpace(currency))
}

func isValidCurrencyCode(currency string) bool {
	if len(currency) != 3 {
		return false
	}
	for _, symbol := range currency {
		if symbol < 'A' || symbol > 'Z' {
			return false
		}
	}

	return true
}
