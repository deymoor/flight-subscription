package kafkaconsumer

import (
	"context"
	"errors"

	kafkago "github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"
)

type inflight struct {
	message kafkago.Message
	err     error
	done    chan struct{}
}

func (consumer *Consumer) Run(ctx context.Context) error {
	group, ctx := errgroup.WithContext(ctx)

	results := make(chan *inflight, consumer.concurrency)
	slots := make(chan struct{}, consumer.concurrency)

	group.Go(func() error {
		return consumer.dispatchLoop(ctx, group, results, slots)
	})
	group.Go(func() error {
		return consumer.commitLoop(ctx, results)
	})

	return group.Wait()
}

func (consumer *Consumer) dispatchLoop(ctx context.Context, group *errgroup.Group, results chan<- *inflight, slots chan struct{}) error {
	defer close(results)

	for {
		message, err := consumer.reader.FetchMessage(ctx)
		if err != nil {
			if isShutdown(ctx, err) {
				return nil
			}

			return err
		}

		item := &inflight{message: message, done: make(chan struct{})}

		if err := send(ctx, results, item); err != nil {
			return nil
		}
		if err := send(ctx, slots, struct{}{}); err != nil {
			return nil
		}

		group.Go(func() error {
			defer func() {
				<-slots
				close(item.done)
			}()

			item.err = consumer.processMessage(ctx, item.message)

			return nil
		})
	}
}

func (consumer *Consumer) commitLoop(ctx context.Context, results <-chan *inflight) error {
	for item := range results {
		select {
		case <-item.done:
		case <-ctx.Done():
			return nil
		}

		if item.err != nil {
			if isShutdown(ctx, item.err) {
				return nil
			}

			return item.err
		}

		if err := consumer.reader.CommitMessages(ctx, item.message); err != nil {
			if isShutdown(ctx, err) {
				return nil
			}

			return err
		}
	}

	return nil
}

func isShutdown(ctx context.Context, err error) bool {
	return ctx.Err() != nil || errors.Is(err, context.Canceled)
}

func send[T any](ctx context.Context, ch chan<- T, value T) error {
	select {
	case ch <- value:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
