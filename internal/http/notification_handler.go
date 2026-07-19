package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"price-subscriptions/internal/domain"
	"price-subscriptions/internal/service"
)

type ListNotificationsService interface {
	ListByDirection(ctx context.Context, direction domain.Direction, limit int) ([]service.NotificationView, error)
}

type NotificationHandler struct {
	notifications ListNotificationsService
}

type notificationResponse struct {
	ID             int64        `json:"id"`
	SubscriptionID int64        `json:"subscription_id"`
	UserID         string       `json:"user_id"`
	DirectionFrom  string       `json:"direction_from"`
	DirectionTo    string       `json:"direction_to"`
	EventID        string       `json:"event_id"`
	ActualPrice    moneyPayload `json:"actual_price"`
	CreatedAt      string       `json:"created_at"`
}

func NewNotificationHandler(notifications ListNotificationsService) *NotificationHandler {
	return &NotificationHandler{
		notifications: notifications,
	}
}

func (handler *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	direction := domain.Direction{
		From: query.Get("from"),
		To:   query.Get("to"),
	}

	limit := 0
	if raw := query.Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = parsed
	}

	views, err := handler.notifications.ListByDirection(r.Context(), direction, limit)
	if errors.Is(err, service.ErrInvalidDirection) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, newNotificationResponses(views))
}

func newNotificationResponses(views []service.NotificationView) []notificationResponse {
	responses := make([]notificationResponse, 0, len(views))
	for _, view := range views {
		responses = append(responses, notificationResponse{
			ID:             view.ID,
			SubscriptionID: view.SubscriptionID,
			UserID:         view.UserID,
			DirectionFrom:  view.Direction.From,
			DirectionTo:    view.Direction.To,
			EventID:        view.EventID,
			ActualPrice:    newMoneyPayload(view.ActualPrice),
			CreatedAt:      view.CreatedAt.Format(time.RFC3339Nano),
		})
	}

	return responses
}
