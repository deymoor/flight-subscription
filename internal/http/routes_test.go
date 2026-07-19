package httpapi

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealthz(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	handleHealthz(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
}

func TestRegisterRoutes(t *testing.T) {
	mux := http.NewServeMux()
	RegisterRoutes(mux, Dependencies{
		Subscriptions: &fakeCreateSubscriptionService{},
		PriceEvents:   &fakePriceEventPublisher{},
		Notifications: &fakeListNotificationsService{},
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected /healthz to be registered, got %d", recorder.Code)
	}
}
