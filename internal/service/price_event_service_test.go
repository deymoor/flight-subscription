package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"price-subscriptions/internal/domain"
)

type fakeNotificationRepository struct {
	subscriptionIDs []int64
	createdSet      map[int64]bool
	err             error
	errAtCall       int

	calls   int
	windows [][]int64
}

func (repo *fakeNotificationRepository) CreateForMatchingBatch(
	_ context.Context,
	_ domain.PriceChangedEvent,
	afterID int64,
	limit int,
) (NotificationBatch, error) {
	call := repo.calls
	repo.calls++

	if repo.err != nil && call == repo.errAtCall {
		return NotificationBatch{}, repo.err
	}

	window := make([]int64, 0, limit)
	for _, id := range repo.subscriptionIDs {
		if id <= afterID {
			continue
		}
		if len(window) == limit {
			break
		}
		window = append(window, id)
	}
	repo.windows = append(repo.windows, window)

	batch := NotificationBatch{Scanned: len(window)}
	for _, id := range window {
		batch.LastID = id
		if repo.createdSet == nil || repo.createdSet[id] {
			batch.Created++
		}
	}

	return batch, nil
}

func testEvent() domain.PriceChangedEvent {
	return domain.PriceChangedEvent{
		EventID:    "evt-1",
		Direction:  domain.Direction{From: "LED", To: "SVO"},
		Price:      domain.Money{Currency: "USD", MinorUnits: 5000},
		OccurredAt: time.Now(),
	}
}

func TestHandlePriceChangedNoMatches(t *testing.T) {
	notifications := &fakeNotificationRepository{}
	service := NewPriceEventService(notifications, 10)

	created, err := service.HandlePriceChanged(context.Background(), testEvent())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created != 0 {
		t.Fatalf("expected 0 created, got %d", created)
	}
	if notifications.calls != 1 {
		t.Fatalf("expected a single batch call for empty result, got %d", notifications.calls)
	}
}

func TestHandlePriceChangedCountsOnlyNew(t *testing.T) {
	notifications := &fakeNotificationRepository{
		subscriptionIDs: []int64{1, 2, 3},
		createdSet:      map[int64]bool{1: true, 2: false, 3: true},
	}
	service := NewPriceEventService(notifications, 10)

	created, err := service.HandlePriceChanged(context.Background(), testEvent())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created != 2 {
		t.Fatalf("expected 2 new notifications, got %d", created)
	}
	if notifications.calls != 1 {
		t.Fatalf("expected 1 batch (all fit in window), got %d", notifications.calls)
	}
}

func TestHandlePriceChangedIteratesBatchesViaKeyset(t *testing.T) {
	notifications := &fakeNotificationRepository{
		subscriptionIDs: []int64{1, 2, 3, 4, 5},
	}
	service := NewPriceEventService(notifications, 2)

	created, err := service.HandlePriceChanged(context.Background(), testEvent())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created != 5 {
		t.Fatalf("expected 5 created across batches, got %d", created)
	}
	if notifications.calls != 3 {
		t.Fatalf("expected 3 batch calls, got %d", notifications.calls)
	}
	wantWindows := [][]int64{{1, 2}, {3, 4}, {5}}
	for i, want := range wantWindows {
		got := notifications.windows[i]
		if len(got) != len(want) {
			t.Fatalf("window %d: expected %v, got %v", i, want, got)
		}
		for j := range want {
			if got[j] != want[j] {
				t.Fatalf("window %d: expected %v, got %v", i, want, got)
			}
		}
	}
}

func TestHandlePriceChangedBatchErrorReturnsPartial(t *testing.T) {
	batchErr := errors.New("insert failed")
	notifications := &fakeNotificationRepository{
		subscriptionIDs: []int64{1, 2, 3, 4},
		err:             batchErr,
		errAtCall:       1,
	}
	service := NewPriceEventService(notifications, 2)

	created, err := service.HandlePriceChanged(context.Background(), testEvent())
	if !errors.Is(err, batchErr) {
		t.Fatalf("expected batch error, got %v", err)
	}
	if created != 2 {
		t.Fatalf("expected partial count of 2 before failure, got %d", created)
	}
}

func TestNewPriceEventServiceDefaultsBatchSize(t *testing.T) {
	service := NewPriceEventService(&fakeNotificationRepository{}, 0)
	if service.batchSize != defaultNotificationBatchSize {
		t.Fatalf("expected default batch size %d, got %d", defaultNotificationBatchSize, service.batchSize)
	}
}
