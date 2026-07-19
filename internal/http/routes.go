package httpapi

import "net/http"

type Dependencies struct {
	Subscriptions CreateSubscriptionService
	PriceEvents   PriceEventPublisher
	Notifications ListNotificationsService
}

func RegisterRoutes(mux *http.ServeMux, dependencies Dependencies) {
	subscriptions := NewSubscriptionHandler(dependencies.Subscriptions)
	priceEvents := NewPriceEventHandler(dependencies.PriceEvents)
	notifications := NewNotificationHandler(dependencies.Notifications)

	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("GET /openapi.yaml", handleOpenAPI)
	mux.HandleFunc("GET /swagger", handleSwagger)
	mux.HandleFunc("POST /subscriptions", subscriptions.Create)
	mux.HandleFunc("POST /price-events", priceEvents.Publish)
	mux.HandleFunc("GET /notifications", notifications.List)
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}
