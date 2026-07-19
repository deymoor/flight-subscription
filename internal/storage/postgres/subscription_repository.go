package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"price-subscriptions/internal/domain"
)

type SubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewSubscriptionRepository(pool *pgxpool.Pool) *SubscriptionRepository {
	return &SubscriptionRepository{
		pool: pool,
	}
}

func (repository *SubscriptionRepository) Create(
	ctx context.Context,
	subscription domain.Subscription,
) (domain.Subscription, error) {
	row := repository.pool.QueryRow(
		ctx,
		`INSERT INTO subscriptions (user_id, direction_from, direction_to, currency, max_price_minor, active)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id, user_id, direction_from, direction_to, currency, max_price_minor, active, created_at`,
		subscription.UserID,
		subscription.Direction.From,
		subscription.Direction.To,
		subscription.MaxPrice.Currency,
		subscription.MaxPrice.MinorUnits,
		subscription.Active,
	)

	return scanSubscription(row)
}

func scanSubscription(row pgx.Row) (domain.Subscription, error) {
	var subscription domain.Subscription

	err := row.Scan(
		&subscription.ID,
		&subscription.UserID,
		&subscription.Direction.From,
		&subscription.Direction.To,
		&subscription.MaxPrice.Currency,
		&subscription.MaxPrice.MinorUnits,
		&subscription.Active,
		&subscription.CreatedAt,
	)
	if err != nil {
		return domain.Subscription{}, err
	}

	return subscription, nil
}
