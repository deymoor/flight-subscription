package kafkaconsumer

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"price-subscriptions/internal/domain"
)

var ErrInvalidPriceChangedEvent = errors.New("invalid price changed event")

type moneyMessage struct {
	Currency   string `json:"currency"`
	MinorUnits int64  `json:"minor_units"`
}

type priceChangedEventMessage struct {
	EventID       string       `json:"event_id"`
	DirectionFrom string       `json:"direction_from"`
	DirectionTo   string       `json:"direction_to"`
	Price         moneyMessage `json:"price"`
	OccurredAt    string       `json:"occurred_at"`
}

func DecodePriceChangedEvent(data []byte) (domain.PriceChangedEvent, error) {
	var message priceChangedEventMessage
	if err := json.Unmarshal(data, &message); err != nil {
		return domain.PriceChangedEvent{}, err
	}

	occurredAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(message.OccurredAt))
	if err != nil {
		return domain.PriceChangedEvent{}, fmt.Errorf("%w: invalid occurred_at: %w", ErrInvalidPriceChangedEvent, err)
	}

	price, err := domain.NewMoney(message.Price.Currency, message.Price.MinorUnits)
	if err != nil {
		return domain.PriceChangedEvent{}, fmt.Errorf("%w: %w", ErrInvalidPriceChangedEvent, err)
	}

	event, err := domain.NewPriceChangedEvent(
		message.EventID,
		domain.Direction{
			From: message.DirectionFrom,
			To:   message.DirectionTo,
		},
		price,
		occurredAt,
	)
	if err != nil {
		return domain.PriceChangedEvent{}, fmt.Errorf("%w: %w", ErrInvalidPriceChangedEvent, err)
	}

	return event, nil
}
