package kafkaproducer

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"price-subscriptions/internal/domain"
	"price-subscriptions/internal/kafkacommon"
)

var ErrInvalidProducerConfig = errors.New("invalid producer config")

type Config struct {
	Brokers []string
	Topic   string
}

type Producer struct {
	writer *kafkago.Writer
}

type moneyMessage struct {
	Currency   string `json:"currency"`
	MinorUnits int64  `json:"minor_units"`
}

type priceChangedEventMessage struct {
	EventID       string       `json:"event_id"`
	DirectionFrom string       `json:"direction_from"`
	DirectionTo   string       `json:"direction_to"`
	Price         moneyMessage `json:"price"`
	OccurredAt    string       `json:"occurred_at"`
}

func New(config Config) (*Producer, error) {
	config.Topic = strings.TrimSpace(config.Topic)
	config.Brokers = kafkacommon.NormalizeBrokers(config.Brokers)

	if len(config.Brokers) == 0 || config.Topic == "" {
		return nil, ErrInvalidProducerConfig
	}

	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(config.Brokers...),
			Topic:        config.Topic,
			Balancer:     &kafkago.Hash{},
			RequiredAcks: kafkago.RequireAll,
		},
	}, nil
}

func (producer *Producer) PublishPriceChanged(ctx context.Context, event domain.PriceChangedEvent) error {
	data, err := json.Marshal(priceChangedEventMessage{
		EventID:       event.EventID,
		DirectionFrom: event.Direction.From,
		DirectionTo:   event.Direction.To,
		Price: moneyMessage{
			Currency:   event.Price.Currency,
			MinorUnits: event.Price.MinorUnits,
		},
		OccurredAt: event.OccurredAt.Format(time.RFC3339Nano),
	})
	if err != nil {
		return err
	}

	return producer.writer.WriteMessages(ctx, kafkago.Message{
		Key:   []byte(event.EventID),
		Value: data,
	})
}

func (producer *Producer) Close() error {
	return producer.writer.Close()
}
