package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"price-subscriptions/internal/domain"
)

var ErrInvalidDirection = errors.New("invalid direction")

const (
	defaultNotificationListLimit = 50
	maxNotificationListLimit     = 200
)

type NotificationView struct {
	ID             int64
	SubscriptionID int64
	UserID         string
	Direction      domain.Direction
	EventID        string
	ActualPrice    domain.Money
	CreatedAt      time.Time
}

type NotificationReader interface {
	ListByDirection(ctx context.Context, direction domain.Direction, limit int) ([]NotificationView, error)
}

type NotificationQueryService struct {
	notifications NotificationReader
}

func NewNotificationQueryService(notifications NotificationReader) *NotificationQueryService {
	return &NotificationQueryService{
		notifications: notifications,
	}
}

func (service *NotificationQueryService) ListByDirection(
	ctx context.Context,
	direction domain.Direction,
	limit int,
) ([]NotificationView, error) {
	direction = direction.Normalized()
	if direction.From == "" || direction.To == "" {
		return nil, fmt.Errorf("%w: %w", ErrInvalidDirection, domain.ErrEmptyDirection)
	}

	switch {
	case limit <= 0:
		limit = defaultNotificationListLimit
	case limit > maxNotificationListLimit:
		limit = maxNotificationListLimit
	}

	return service.notifications.ListByDirection(ctx, direction, limit)
}
