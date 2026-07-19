package domain

import "time"

type Notification struct {
	ID             int64
	SubscriptionID int64
	EventID        string
	ActualPrice    Money
	CreatedAt      time.Time
}
