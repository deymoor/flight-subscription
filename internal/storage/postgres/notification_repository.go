package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"price-subscriptions/internal/domain"
	"price-subscriptions/internal/service"
)

type NotificationRepository struct {
	pool *pgxpool.Pool
}

func NewNotificationRepository(pool *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{
		pool: pool,
	}
}

// CreateForMatchingBatch за один запрос отбирает окно подходящих подписок
// (keyset-пагинация по id) и вставляет для них уведомления идемпотентно.
//
// Возвращает курсор (LastID), число просмотренных подписок в окне (Scanned)
// и число реально созданных уведомлений (Created). Когда Scanned < limit —
// окно последнее, вызывающий код останавливает цикл.
func (repository *NotificationRepository) CreateForMatchingBatch(
	ctx context.Context,
	event domain.PriceChangedEvent,
	afterID int64,
	limit int,
) (service.NotificationBatch, error) {
	row := repository.pool.QueryRow(
		ctx,
		`WITH matched AS (
			SELECT s.id
			FROM subscriptions s
			WHERE s.active
			  AND s.id > $5
			  AND s.direction_from = $2
			  AND s.direction_to = $3
			  AND s.currency = $4
			  AND s.max_price_minor >= $6
			ORDER BY s.id
			LIMIT $7
		),
		inserted AS (
			INSERT INTO notifications (subscription_id, event_id, currency, actual_price_minor)
			SELECT m.id, $1, $4, $6
			FROM matched m
			ON CONFLICT (subscription_id, event_id) DO NOTHING
			RETURNING 1
		)
		SELECT
			COALESCE((SELECT MAX(id) FROM matched), $5) AS last_id,
			(SELECT COUNT(*) FROM matched) AS scanned,
			(SELECT COUNT(*) FROM inserted) AS created`,
		event.EventID,
		event.Direction.From,
		event.Direction.To,
		event.Price.Currency,
		afterID,
		event.Price.MinorUnits,
		limit,
	)

	var batch service.NotificationBatch
	if err := row.Scan(&batch.LastID, &batch.Scanned, &batch.Created); err != nil {
		return service.NotificationBatch{}, err
	}

	return batch, nil
}

func (repository *NotificationRepository) ListByDirection(
	ctx context.Context,
	direction domain.Direction,
	limit int,
) ([]service.NotificationView, error) {
	rows, err := repository.pool.Query(
		ctx,
		`SELECT n.id, n.subscription_id, s.user_id,
		        s.direction_from, s.direction_to,
		        n.event_id, n.currency, n.actual_price_minor, n.created_at
		 FROM notifications n
		 JOIN subscriptions s ON s.id = n.subscription_id
		 WHERE s.direction_from = $1 AND s.direction_to = $2
		 ORDER BY n.id DESC
		 LIMIT $3`,
		direction.From,
		direction.To,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	views := make([]service.NotificationView, 0, limit)
	for rows.Next() {
		var view service.NotificationView

		err := rows.Scan(
			&view.ID,
			&view.SubscriptionID,
			&view.UserID,
			&view.Direction.From,
			&view.Direction.To,
			&view.EventID,
			&view.ActualPrice.Currency,
			&view.ActualPrice.MinorUnits,
			&view.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		views = append(views, view)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return views, nil
}
