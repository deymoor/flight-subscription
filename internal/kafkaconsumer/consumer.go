package kafkaconsumer

import (
	"context"
	"errors"
	"strings"
	"time"

	kafkago "github.com/segmentio/kafka-go"

	"price-subscriptions/internal/domain"
	"price-subscriptions/internal/kafkacommon"
)

var ErrInvalidConsumerConfig = errors.New("invalid consumer config")

const (
	defaultMaxRetries     = 5
	defaultInitialBackoff = 500 * time.Millisecond
	defaultMaxBackoff     = 30 * time.Second
	defaultConcurrency    = 1
)

type PriceEventHandler interface {
	HandlePriceChanged(ctx context.Context, event domain.PriceChangedEvent) (int, error)
}

type messageReader interface {
	FetchMessage(ctx context.Context) (kafkago.Message, error)
	CommitMessages(ctx context.Context, messages ...kafkago.Message) error
	Close() error
}

type Config struct {
	Brokers        []string
	Topic          string
	GroupID        string
	DLQTopic       string
	MaxRetries     int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Concurrency    int
}

type Consumer struct {
	reader  messageReader
	handler PriceEventHandler
	dlq     DeadLetterPublisher

	maxRetries     int
	initialBackoff time.Duration
	maxBackoff     time.Duration
	concurrency    int
}

func New(config Config, handler PriceEventHandler) (*Consumer, error) {
	config.Topic = strings.TrimSpace(config.Topic)
	config.GroupID = strings.TrimSpace(config.GroupID)
	config.DLQTopic = strings.TrimSpace(config.DLQTopic)
	config.Brokers = kafkacommon.NormalizeBrokers(config.Brokers)

	if len(config.Brokers) == 0 || config.Topic == "" || config.GroupID == "" || handler == nil {
		return nil, ErrInvalidConsumerConfig
	}

	if config.MaxRetries < 0 {
		config.MaxRetries = defaultMaxRetries
	}
	if config.InitialBackoff <= 0 {
		config.InitialBackoff = defaultInitialBackoff
	}
	if config.MaxBackoff <= 0 {
		config.MaxBackoff = defaultMaxBackoff
	}
	if config.MaxBackoff < config.InitialBackoff {
		config.MaxBackoff = config.InitialBackoff
	}
	if config.Concurrency < 1 {
		config.Concurrency = defaultConcurrency
	}

	var dlq DeadLetterPublisher
	if config.DLQTopic != "" {
		dlq = newKafkaDeadLetterPublisher(config.Brokers, config.DLQTopic)
	}

	return &Consumer{
		reader: kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:  config.Brokers,
			Topic:    config.Topic,
			GroupID:  config.GroupID,
			MinBytes: 1,
			MaxBytes: 10e6,
		}),
		handler:        handler,
		dlq:            dlq,
		maxRetries:     config.MaxRetries,
		initialBackoff: config.InitialBackoff,
		maxBackoff:     config.MaxBackoff,
		concurrency:    config.Concurrency,
	}, nil
}

func (consumer *Consumer) Close() error {
	err := consumer.reader.Close()

	if consumer.dlq != nil {
		if dlqErr := consumer.dlq.Close(); dlqErr != nil && err == nil {
			err = dlqErr
		}
	}

	return err
}
