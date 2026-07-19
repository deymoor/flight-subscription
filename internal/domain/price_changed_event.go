package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptyEventID   = errors.New("event id is required")
	ErrEmptyDirection = errors.New("direction from and to are required")
	ErrZeroOccurredAt = errors.New("occurred at is required")
)

type PriceChangedEvent struct {
	EventID    string
	Direction  Direction
	Price      Money
	OccurredAt time.Time
}

func NewPriceChangedEvent(eventID string, direction Direction, price Money, occurredAt time.Time) (PriceChangedEvent, error) {
	eventID = strings.TrimSpace(eventID)
	direction = direction.Normalized()

	if eventID == "" {
		return PriceChangedEvent{}, ErrEmptyEventID
	}
	if direction.From == "" || direction.To == "" {
		return PriceChangedEvent{}, ErrEmptyDirection
	}
	if err := price.Validate(); err != nil {
		return PriceChangedEvent{}, err
	}
	if occurredAt.IsZero() {
		return PriceChangedEvent{}, ErrZeroOccurredAt
	}

	return PriceChangedEvent{
		EventID:    eventID,
		Direction:  direction,
		Price:      price,
		OccurredAt: occurredAt,
	}, nil
}
