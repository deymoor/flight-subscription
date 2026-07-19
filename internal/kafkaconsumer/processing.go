package kafkaconsumer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"price-subscriptions/internal/domain"
)

func (consumer *Consumer) processMessage(ctx context.Context, message kafkago.Message) error {
	event, err := DecodePriceChangedEvent(message.Value)
	if err != nil {
		slog.Warn("dead-letter malformed price event",
			slog.Any("error", err),
			slog.Int("partition", message.Partition),
			slog.Int64("offset", message.Offset),
		)

		return consumer.deadLetter(ctx, message, err)
	}

	return consumer.process(ctx, message, event)
}

func (consumer *Consumer) process(ctx context.Context, message kafkago.Message, event domain.PriceChangedEvent) error {
	initalAttempt := 1
	for attempt := 0; ; attempt++ {
		_, err := consumer.handler.HandlePriceChanged(ctx, event)
		totalAttempts := attempt + initalAttempt

		if err == nil {
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		if attempt >= consumer.maxRetries {
			slog.Error("transient handler error, retries exhausted, dead-lettering",
				slog.Any("error", err),
				slog.String("event_id", event.EventID),
				slog.Int("attempts", totalAttempts),
			)

			return consumer.deadLetter(ctx, message, err)
		}

		backoff := consumer.backoffFor(attempt)
		slog.Warn("transient handler error, retrying",
			slog.Any("error", err),
			slog.String("event_id", event.EventID),
			slog.Int("attempt", totalAttempts),
			slog.Int("max_attempts", consumer.maxRetries+1),
			slog.Duration("backoff", backoff),
		)

		if err := sleep(ctx, backoff); err != nil {
			return err
		}
	}
}

func (consumer *Consumer) deadLetter(ctx context.Context, message kafkago.Message, reason error) error {
	if consumer.dlq == nil {
		return fmt.Errorf("no dead-letter topic configured: %w", reason)
	}

	if err := consumer.dlq.Publish(ctx, message, reason); err != nil {
		return fmt.Errorf("publish to dead-letter topic: %w", err)
	}

	return nil
}

func (consumer *Consumer) backoffFor(attempt int) time.Duration {
	if attempt > 32 {
		return consumer.maxBackoff
	}

	backoff := consumer.initialBackoff * time.Duration(1<<attempt)
	if backoff <= 0 || backoff > consumer.maxBackoff {
		return consumer.maxBackoff
	}

	return backoff
}

func sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
