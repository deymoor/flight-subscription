package service

import (
	"context"
	"errors"
	"testing"

	"price-subscriptions/internal/domain"
)

type fakeSubscriptionRepository struct {
	called       bool
	gotInput     domain.Subscription
	returnResult domain.Subscription
	returnErr    error
}

func (repo *fakeSubscriptionRepository) Create(_ context.Context, subscription domain.Subscription) (domain.Subscription, error) {
	repo.called = true
	repo.gotInput = subscription
	if repo.returnErr != nil {
		return domain.Subscription{}, repo.returnErr
	}
	return repo.returnResult, nil
}

func validInput() CreateSubscriptionInput {
	return CreateSubscriptionInput{
		UserID:    "user-1",
		Direction: domain.Direction{From: "LED", To: "SVO"},
		MaxPrice:  domain.Money{Currency: "USD", MinorUnits: 10000},
	}
}

func TestSubscriptionServiceCreateSuccess(t *testing.T) {
	repo := &fakeSubscriptionRepository{
		returnResult: domain.Subscription{ID: 42, UserID: "user-1"},
	}
	service := NewSubscriptionService(repo)

	got, err := service.Create(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.called {
		t.Fatal("expected repository Create to be called")
	}
	if got.ID != 42 {
		t.Fatalf("expected persisted subscription id 42, got %d", got.ID)
	}
	if !repo.gotInput.Active {
		t.Fatal("expected domain subscription to be active before persisting")
	}
}

func TestSubscriptionServiceCreateInvalidInput(t *testing.T) {
	repo := &fakeSubscriptionRepository{}
	service := NewSubscriptionService(repo)

	input := validInput()
	input.UserID = "   "

	_, err := service.Create(context.Background(), input)
	if !errors.Is(err, ErrInvalidSubscription) {
		t.Fatalf("expected ErrInvalidSubscription, got %v", err)
	}
	if !errors.Is(err, domain.ErrEmptyUserID) {
		t.Fatalf("expected wrapped domain error, got %v", err)
	}
	if repo.called {
		t.Fatal("expected repository not to be called on invalid input")
	}
}

func TestSubscriptionServiceCreateRepositoryError(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &fakeSubscriptionRepository{returnErr: repoErr}
	service := NewSubscriptionService(repo)

	_, err := service.Create(context.Background(), validInput())
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repository error, got %v", err)
	}
	if errors.Is(err, ErrInvalidSubscription) {
		t.Fatal("repository error must not be wrapped as invalid subscription")
	}
}
