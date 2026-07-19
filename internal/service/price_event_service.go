package service

import (
	"context"

	"price-subscriptions/internal/domain"
)

const defaultNotificationBatchSize = 1000

type NotificationBatch struct {
	LastID  int64
	Scanned int
	Created int
}

type MatchingNotificationRepository interface {
	CreateForMatchingBatch(
		ctx context.Context,
		event domain.PriceChangedEvent,
		afterID int64,
		limit int,
	) (NotificationBatch, error)
}

type PriceEventService struct {
	notifications MatchingNotificationRepository
	batchSize     int
}

func NewPriceEventService(
	notifications MatchingNotificationRepository,
	batchSize int,
) *PriceEventService {
	if batchSize <= 0 {
		batchSize = defaultNotificationBatchSize
	}

	return &PriceEventService{
		notifications: notifications,
		batchSize:     batchSize,
	}
}

func (service *PriceEventService) HandlePriceChanged(ctx context.Context, event domain.PriceChangedEvent) (int, error) {
	total := 0
	afterID := int64(0)

	for {
		batch, err := service.notifications.CreateForMatchingBatch(ctx, event, afterID, service.batchSize)
		if err != nil {
			return total, err
		}

		total += batch.Created

		if batch.Scanned < service.batchSize {
			return total, nil
		}

		afterID = batch.LastID
	}
}
