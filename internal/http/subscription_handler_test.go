package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"price-subscriptions/internal/domain"
	"price-subscriptions/internal/service"
)

type fakeCreateSubscriptionService struct {
	gotInput service.CreateSubscriptionInput
	result   domain.Subscription
	err      error
	called   bool
}

func (f *fakeCreateSubscriptionService) Create(_ context.Context, input service.CreateSubscriptionInput) (domain.Subscription, error) {
	f.called = true
	f.gotInput = input
	return f.result, f.err
}

func doRequest(t *testing.T, handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/subscriptions", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

func TestSubscriptionHandlerCreated(t *testing.T) {
	svc := &fakeCreateSubscriptionService{
		result: domain.Subscription{
			ID:        7,
			UserID:    "user-1",
			Direction: domain.Direction{From: "LED", To: "SVO"},
			MaxPrice:  domain.Money{Currency: "USD", MinorUnits: 10000},
			Active:    true,
			CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		},
	}
	handler := NewSubscriptionHandler(svc)

	body := `{"user_id":"user-1","direction_from":"LED","direction_to":"SVO","max_price":{"currency":"USD","minor_units":10000}}`
	recorder := doRequest(t, handler.Create, body)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	if !svc.called {
		t.Fatal("expected service to be called")
	}
	if svc.gotInput.MaxPrice.Currency != "USD" || svc.gotInput.MaxPrice.MinorUnits != 10000 {
		t.Fatalf("unexpected mapped input: %+v", svc.gotInput)
	}

	var response subscriptionResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.ID != 7 || response.MaxPrice.Currency != "USD" {
		t.Fatalf("unexpected response: %+v", response)
	}
}

func TestSubscriptionHandlerInvalidJSON(t *testing.T) {
	svc := &fakeCreateSubscriptionService{}
	handler := NewSubscriptionHandler(svc)

	recorder := doRequest(t, handler.Create, `{not-json`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if svc.called {
		t.Fatal("service must not be called on invalid json")
	}
}

func TestSubscriptionHandlerUnknownField(t *testing.T) {
	svc := &fakeCreateSubscriptionService{}
	handler := NewSubscriptionHandler(svc)

	recorder := doRequest(t, handler.Create, `{"user_id":"u","surprise":true}`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unknown field, got %d", recorder.Code)
	}
}

func TestSubscriptionHandlerInvalidMoney(t *testing.T) {
	svc := &fakeCreateSubscriptionService{}
	handler := NewSubscriptionHandler(svc)

	body := `{"user_id":"user-1","direction_from":"LED","direction_to":"SVO","max_price":{"currency":"US","minor_units":10000}}`
	recorder := doRequest(t, handler.Create, body)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid money, got %d", recorder.Code)
	}
	if svc.called {
		t.Fatal("service must not be called when money is invalid")
	}
}

func TestSubscriptionHandlerInvalidSubscription(t *testing.T) {
	svc := &fakeCreateSubscriptionService{
		err: fmt.Errorf("%w: %w", service.ErrInvalidSubscription, domain.ErrEmptyUserID),
	}
	handler := NewSubscriptionHandler(svc)

	body := `{"user_id":"","direction_from":"LED","direction_to":"SVO","max_price":{"currency":"USD","minor_units":10000}}`
	recorder := doRequest(t, handler.Create, body)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid subscription, got %d", recorder.Code)
	}
}

func TestSubscriptionHandlerInternalError(t *testing.T) {
	svc := &fakeCreateSubscriptionService{err: errors.New("db down")}
	handler := NewSubscriptionHandler(svc)

	body := `{"user_id":"user-1","direction_from":"LED","direction_to":"SVO","max_price":{"currency":"USD","minor_units":10000}}`
	recorder := doRequest(t, handler.Create, body)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", recorder.Code)
	}
	var response errorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if response.Error != "internal error" {
		t.Fatalf("unexpected error body: %q", response.Error)
	}
}
