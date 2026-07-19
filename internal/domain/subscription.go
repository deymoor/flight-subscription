package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEmptyUserID = errors.New("user id is required")
)

type Subscription struct {
	ID        int64
	UserID    string
	Direction Direction
	MaxPrice  Money
	Active    bool
	CreatedAt time.Time
}

func NewSubscription(userID string, direction Direction, maxPrice Money) (Subscription, error) {
	userID = strings.TrimSpace(userID)
	direction = direction.Normalized()

	if userID == "" {
		return Subscription{}, ErrEmptyUserID
	}
	if direction.From == "" || direction.To == "" {
		return Subscription{}, ErrEmptyDirection
	}
	if err := maxPrice.Validate(); err != nil {
		return Subscription{}, err
	}

	return Subscription{
		UserID:    userID,
		Direction: direction,
		MaxPrice:  maxPrice,
		Active:    true,
	}, nil
}
