package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"price-subscriptions/internal/domain"
)

type fakePriceEventPublisher struct {
	gotEvent domain.PriceChangedEvent
	err      error
	called   bool
}

func (f *fakePriceEventPublisher) PublishPriceChanged(_ context.Context, event domain.PriceChangedEvent) error {
	f.called = true
	f.gotEvent = event
	return f.err
}

func doPublishRequest(t *testing.T, handler http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/price-events", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	return recorder
}

func TestPriceEventHandlerAccepted(t *testing.T) {
	publisher := &fakePriceEventPublisher{}
	handler := NewPriceEventHandler(publisher)

	body := `{"event_id":"evt-1","direction_from":"LED","direction_to":"SVO","price":{"currency":"USD","minor_units":5000},"occurred_at":"2026-01-02T03:04:05Z"}`
	recorder := doPublishRequest(t, handler.Publish, body)

	if recorder.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d (%s)", recorder.Code, recorder.Body.String())
	}
	if !publisher.called {
		t.Fatal("expected publisher to be called")
	}
	if publisher.gotEvent.EventID != "evt-1" || publisher.gotEvent.Price.MinorUnits != 5000 {
		t.Fatalf("unexpected published event: %+v", publisher.gotEvent)
	}
}

func TestPriceEventHandlerInvalidJSON(t *testing.T) {
	publisher := &fakePriceEventPublisher{}
	handler := NewPriceEventHandler(publisher)

	recorder := doPublishRequest(t, handler.Publish, `{`)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	if publisher.called {
		t.Fatal("publisher must not be called on invalid json")
	}
}

func TestPriceEventHandlerInvalidOccurredAt(t *testing.T) {
	publisher := &fakePriceEventPublisher{}
	handler := NewPriceEventHandler(publisher)

	body := `{"event_id":"evt-1","direction_from":"LED","direction_to":"SVO","price":{"currency":"USD","minor_units":5000},"occurred_at":"not-a-time"}`
	recorder := doPublishRequest(t, handler.Publish, body)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad occurred_at, got %d", recorder.Code)
	}
	if publisher.called {
		t.Fatal("publisher must not be called when event is invalid")
	}
}

func TestPriceEventHandlerInvalidPrice(t *testing.T) {
	publisher := &fakePriceEventPublisher{}
	handler := NewPriceEventHandler(publisher)

	body := `{"event_id":"evt-1","direction_from":"LED","direction_to":"SVO","price":{"currency":"USD","minor_units":0},"occurred_at":"2026-01-02T03:04:05Z"}`
	recorder := doPublishRequest(t, handler.Publish, body)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid price, got %d", recorder.Code)
	}
}

func TestPriceEventHandlerPublishFailure(t *testing.T) {
	publisher := &fakePriceEventPublisher{err: errors.New("broker unavailable")}
	handler := NewPriceEventHandler(publisher)

	body := `{"event_id":"evt-1","direction_from":"LED","direction_to":"SVO","price":{"currency":"USD","minor_units":5000},"occurred_at":"2026-01-02T03:04:05Z"}`
	recorder := doPublishRequest(t, handler.Publish, body)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d", recorder.Code)
	}
}
