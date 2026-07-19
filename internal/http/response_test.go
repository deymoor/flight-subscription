package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"price-subscriptions/internal/domain"
)

func TestMoneyPayloadToDomain(t *testing.T) {
	payload := moneyPayload{Currency: "usd", MinorUnits: 100}
	money, err := payload.toDomain()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if money.Currency != "USD" || money.MinorUnits != 100 {
		t.Fatalf("unexpected money: %+v", money)
	}

	if _, err := (moneyPayload{Currency: "US", MinorUnits: 100}).toDomain(); err == nil {
		t.Fatal("expected error for invalid currency")
	}
}

func TestNewMoneyPayload(t *testing.T) {
	payload := newMoneyPayload(domain.Money{Currency: "EUR", MinorUnits: 250})
	if payload.Currency != "EUR" || payload.MinorUnits != 250 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestWriteJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeJSON(recorder, http.StatusTeapot, map[string]string{"hello": "world"})

	if recorder.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: %q", got)
	}

	var decoded map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if decoded["hello"] != "world" {
		t.Fatalf("unexpected body: %v", decoded)
	}
}

func TestWriteError(t *testing.T) {
	recorder := httptest.NewRecorder()
	writeError(recorder, http.StatusBadRequest, "boom")

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", recorder.Code)
	}
	var response errorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if response.Error != "boom" {
		t.Fatalf("unexpected error message: %q", response.Error)
	}
}
