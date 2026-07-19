package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"price-subscriptions/internal/domain"
	"price-subscriptions/internal/service"
)

type fakeListNotificationsService struct {
	views     []service.NotificationView
	err       error
	direction domain.Direction
	limit     int
}

func (f *fakeListNotificationsService) ListByDirection(
	_ context.Context,
	direction domain.Direction,
	limit int,
) ([]service.NotificationView, error) {
	f.direction = direction
	f.limit = limit

	return f.views, f.err
}

func TestNotificationHandlerList(t *testing.T) {
	svc := &fakeListNotificationsService{
		views: []service.NotificationView{
			{
				ID:             7,
				SubscriptionID: 3,
				UserID:         "user-42",
				Direction:      domain.Direction{From: "LED", To: "AER"},
				EventID:        "evt-1",
				ActualPrice:    domain.Money{Currency: "USD", MinorUnits: 9500},
				CreatedAt:      time.Unix(0, 0).UTC(),
			},
		},
	}
	handler := NewNotificationHandler(svc)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/notifications?from=led&to=aer&limit=10", nil)
	handler.List(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var body []notificationResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body) != 1 || body[0].ID != 7 || body[0].ActualPrice.MinorUnits != 9500 {
		t.Fatalf("unexpected body: %+v", body)
	}
	if svc.limit != 10 || svc.direction.From != "led" || svc.direction.To != "aer" {
		t.Fatalf("unexpected passthrough: limit=%d dir=%+v", svc.limit, svc.direction)
	}
}

func TestNotificationHandlerInvalidLimit(t *testing.T) {
	handler := NewNotificationHandler(&fakeListNotificationsService{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/notifications?from=LED&to=AER&limit=abc", nil)
	handler.List(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestNotificationHandlerInvalidDirection(t *testing.T) {
	handler := NewNotificationHandler(&fakeListNotificationsService{err: service.ErrInvalidDirection})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/notifications?from=LED", nil)
	handler.List(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
}

func TestNotificationHandlerInternalError(t *testing.T) {
	handler := NewNotificationHandler(&fakeListNotificationsService{err: errors.New("db down")})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/notifications?from=LED&to=AER", nil)
	handler.List(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
}
