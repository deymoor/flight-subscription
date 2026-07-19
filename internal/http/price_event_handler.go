package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"price-subscriptions/internal/domain"
)

type PriceEventPublisher interface {
	PublishPriceChanged(ctx context.Context, event domain.PriceChangedEvent) error
}

type PriceEventHandler struct {
	publisher PriceEventPublisher
}

type publishPriceEventRequest struct {
	EventID       string       `json:"event_id"`
	DirectionFrom string       `json:"direction_from"`
	DirectionTo   string       `json:"direction_to"`
	Price         moneyPayload `json:"price"`
	OccurredAt    string       `json:"occurred_at"`
}

type priceEventResponse struct {
	EventID       string       `json:"event_id"`
	DirectionFrom string       `json:"direction_from"`
	DirectionTo   string       `json:"direction_to"`
	Price         moneyPayload `json:"price"`
	OccurredAt    string       `json:"occurred_at"`
}

func NewPriceEventHandler(publisher PriceEventPublisher) *PriceEventHandler {
	return &PriceEventHandler{
		publisher: publisher,
	}
}

func (handler *PriceEventHandler) Publish(w http.ResponseWriter, r *http.Request) {
	var request publishPriceEventRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	event, err := newPriceChangedEvent(request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := handler.publisher.PublishPriceChanged(r.Context(), event); err != nil {
		writeError(w, http.StatusBadGateway, "kafka publish failed")
		return
	}

	writeJSON(w, http.StatusAccepted, newPriceEventResponse(event))
}

func newPriceChangedEvent(request publishPriceEventRequest) (domain.PriceChangedEvent, error) {
	occurredAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(request.OccurredAt))
	if err != nil {
		return domain.PriceChangedEvent{}, fmt.Errorf("invalid occurred_at: %w", err)
	}

	price, err := request.Price.toDomain()
	if err != nil {
		return domain.PriceChangedEvent{}, err
	}

	return domain.NewPriceChangedEvent(
		request.EventID,
		domain.Direction{
			From: request.DirectionFrom,
			To:   request.DirectionTo,
		},
		price,
		occurredAt,
	)
}

func newPriceEventResponse(event domain.PriceChangedEvent) priceEventResponse {
	return priceEventResponse{
		EventID:       event.EventID,
		DirectionFrom: event.Direction.From,
		DirectionTo:   event.Direction.To,
		Price:         newMoneyPayload(event.Price),
		OccurredAt:    event.OccurredAt.Format(time.RFC3339Nano),
	}
}
