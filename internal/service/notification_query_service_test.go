package service

import (
	"context"
	"errors"
	"testing"

	"price-subscriptions/internal/domain"
)

type stubNotificationReader struct {
	gotDirection domain.Direction
	gotLimit     int
	views        []NotificationView
}

func (s *stubNotificationReader) ListByDirection(
	_ context.Context,
	direction domain.Direction,
	limit int,
) ([]NotificationView, error) {
	s.gotDirection = direction
	s.gotLimit = limit

	return s.views, nil
}

func TestNotificationQueryServiceListByDirection(t *testing.T) {
	reader := &stubNotificationReader{views: []NotificationView{{ID: 1}}}
	svc := NewNotificationQueryService(reader)

	views, err := svc.ListByDirection(context.Background(), domain.Direction{From: " led ", To: " aer "}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(views))
	}
	if reader.gotDirection.From != "led" || reader.gotDirection.To != "aer" {
		t.Fatalf("direction not normalized: %+v", reader.gotDirection)
	}
	if reader.gotLimit != defaultNotificationListLimit {
		t.Fatalf("expected default limit %d, got %d", defaultNotificationListLimit, reader.gotLimit)
	}
}

func TestNotificationQueryServiceClampsLimit(t *testing.T) {
	reader := &stubNotificationReader{}
	svc := NewNotificationQueryService(reader)

	if _, err := svc.ListByDirection(context.Background(), domain.Direction{From: "LED", To: "AER"}, 10_000); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reader.gotLimit != maxNotificationListLimit {
		t.Fatalf("expected clamp to %d, got %d", maxNotificationListLimit, reader.gotLimit)
	}
}

func TestNotificationQueryServiceRejectsEmptyDirection(t *testing.T) {
	svc := NewNotificationQueryService(&stubNotificationReader{})

	_, err := svc.ListByDirection(context.Background(), domain.Direction{From: "LED", To: "  "}, 0)
	if !errors.Is(err, ErrInvalidDirection) {
		t.Fatalf("expected ErrInvalidDirection, got %v", err)
	}
}
