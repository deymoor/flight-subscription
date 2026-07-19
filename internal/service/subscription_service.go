package service

import (
	"context"
	"errors"
	"fmt"

	"price-subscriptions/internal/domain"
)

var ErrInvalidSubscription = errors.New("invalid subscription")

type SubscriptionRepository interface {
	Create(ctx context.Context, subscription domain.Subscription) (domain.Subscription, error)
}

type SubscriptionService struct {
	subscriptions SubscriptionRepository
}

type CreateSubscriptionInput struct {
	UserID    string
	Direction domain.Direction
	MaxPrice  domain.Money
}

func NewSubscriptionService(subscriptions SubscriptionRepository) *SubscriptionService {
	return &SubscriptionService{
		subscriptions: subscriptions,
	}
}

func (service *SubscriptionService) Create(ctx context.Context, input CreateSubscriptionInput) (domain.Subscription, error) {
	subscription, err := domain.NewSubscription(
		input.UserID,
		input.Direction,
		input.MaxPrice,
	)
	if err != nil {
		return domain.Subscription{}, fmt.Errorf("%w: %w", ErrInvalidSubscription, err)
	}

	return service.subscriptions.Create(ctx, subscription)
}
