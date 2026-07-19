package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"price-subscriptions/internal/config"
	"price-subscriptions/internal/http"
	"price-subscriptions/internal/kafkaproducer"
	"price-subscriptions/internal/service"
	"price-subscriptions/internal/storage/postgres"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()

	pool, err := postgres.Open(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		slog.Error("open postgres", slog.Any("error", err))
		os.Exit(1)
	}
	defer pool.Close()

	if err := postgres.RunMigrations(ctx, pool, cfg.MigrationsDir); err != nil {
		slog.Error("run migrations", slog.Any("error", err))
		os.Exit(1)
	}

	subscriptions := postgres.NewSubscriptionRepository(pool)
	subscriptionService := service.NewSubscriptionService(subscriptions)

	notifications := postgres.NewNotificationRepository(pool)
	notificationQueryService := service.NewNotificationQueryService(notifications)

	priceEvents, err := kafkaproducer.New(kafkaproducer.Config{
		Brokers: cfg.KafkaBrokers,
		Topic:   cfg.KafkaTopic,
	})
	if err != nil {
		slog.Error("create kafka producer", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := priceEvents.Close(); err != nil {
			slog.Error("close kafka producer", slog.Any("error", err))
		}
	}()

	mux := http.NewServeMux()
	httpapi.RegisterRoutes(mux, httpapi.Dependencies{
		Subscriptions: subscriptionService,
		PriceEvents:   priceEvents,
		Notifications: notificationQueryService,
	})

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		errs <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown api server", slog.Any("error", err))
			os.Exit(1)
		}
	case err := <-errs:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("run api server", slog.Any("error", err))
			os.Exit(1)
		}
	}
}
