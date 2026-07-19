package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"price-subscriptions/internal/config"
	"price-subscriptions/internal/kafkaconsumer"
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

	notifications := postgres.NewNotificationRepository(pool)
	priceEvents := service.NewPriceEventService(notifications, cfg.NotificationBatchSize)

	consumer, err := kafkaconsumer.New(kafkaconsumer.Config{
		Brokers:        cfg.KafkaBrokers,
		Topic:          cfg.KafkaTopic,
		GroupID:        cfg.KafkaGroupID,
		DLQTopic:       cfg.KafkaDLQTopic,
		MaxRetries:     cfg.KafkaMaxRetries,
		InitialBackoff: cfg.KafkaInitialBackoff,
		MaxBackoff:     cfg.KafkaMaxBackoff,
		Concurrency:    cfg.KafkaConcurrency,
	}, priceEvents)
	if err != nil {
		slog.Error("create kafka consumer", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			slog.Error("close kafka consumer", slog.Any("error", err))
		}
	}()

	if err := consumer.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("run kafka consumer", slog.Any("error", err))
		os.Exit(1)
	}
}
