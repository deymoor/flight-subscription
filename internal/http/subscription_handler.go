package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"price-subscriptions/internal/domain"
	"price-subscriptions/internal/service"
)

type CreateSubscriptionService interface {
	Create(ctx context.Context, input service.CreateSubscriptionInput) (domain.Subscription, error)
}

type SubscriptionHandler struct {
	subscriptions CreateSubscriptionService
}

type createSubscriptionRequest struct {
	UserID        string       `json:"user_id"`
	DirectionFrom string       `json:"direction_from"`
	DirectionTo   string       `json:"direction_to"`
	MaxPrice      moneyPayload `json:"max_price"`
}

type subscriptionResponse struct {
	ID            int64        `json:"id"`
	UserID        string       `json:"user_id"`
	DirectionFrom string       `json:"direction_from"`
	DirectionTo   string       `json:"direction_to"`
	MaxPrice      moneyPayload `json:"max_price"`
	Active        bool         `json:"active"`
	CreatedAt     string       `json:"created_at"`
}

func NewSubscriptionHandler(subscriptions CreateSubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{
		subscriptions: subscriptions,
	}
}

func (handler *SubscriptionHandler) Create(w http.ResponseWriter, r *http.Request) {
	var request createSubscriptionRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json")
		return
	}

	maxPrice, err := request.MaxPrice.toDomain()
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	subscription, err := handler.subscriptions.Create(r.Context(), service.CreateSubscriptionInput{
		UserID: request.UserID,
		Direction: domain.Direction{
			From: request.DirectionFrom,
			To:   request.DirectionTo,
		},
		MaxPrice: maxPrice,
	})
	if errors.Is(err, service.ErrInvalidSubscription) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, newSubscriptionResponse(subscription))
}

func newSubscriptionResponse(subscription domain.Subscription) subscriptionResponse {
	return subscriptionResponse{
		ID:            subscription.ID,
		UserID:        subscription.UserID,
		DirectionFrom: subscription.Direction.From,
		DirectionTo:   subscription.Direction.To,
		MaxPrice:      newMoneyPayload(subscription.MaxPrice),
		Active:        subscription.Active,
		CreatedAt:     subscription.CreatedAt.Format(time.RFC3339Nano),
	}
}
