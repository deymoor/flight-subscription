package httpapi

import (
	"encoding/json"
	"net/http"

	"price-subscriptions/internal/domain"
)

const maxRequestBodyBytes = 1 << 20

type errorResponse struct {
	Error string `json:"error"`
}

type moneyPayload struct {
	Currency   string `json:"currency"`
	MinorUnits int64  `json:"minor_units"`
}

func newMoneyPayload(money domain.Money) moneyPayload {
	return moneyPayload{
		Currency:   money.Currency,
		MinorUnits: money.MinorUnits,
	}
}

func (payload moneyPayload) toDomain() (domain.Money, error) {
	return domain.NewMoney(payload.Currency, payload.MinorUnits)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	return decoder.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
