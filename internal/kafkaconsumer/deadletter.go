package kafkaconsumer

import (
	"context"
	"strconv"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type DeadLetterPublisher interface {
	Publish(ctx context.Context, message kafkago.Message, reason error) error
	Close() error
}

type kafkaDeadLetterPublisher struct {
	writer *kafkago.Writer
}

func newKafkaDeadLetterPublisher(brokers []string, topic string) *kafkaDeadLetterPublisher {
	return &kafkaDeadLetterPublisher{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafkago.Hash{},
			RequiredAcks: kafkago.RequireAll,
		},
	}
}

func (publisher *kafkaDeadLetterPublisher) Publish(ctx context.Context, message kafkago.Message, reason error) error {
	dlqHeaders := []kafkago.Header{
		{Key: "x-dlq-reason", Value: []byte(reason.Error())},
		{Key: "x-dlq-source-topic", Value: []byte(message.Topic)},
		{Key: "x-dlq-source-partition", Value: []byte(strconv.Itoa(message.Partition))},
		{Key: "x-dlq-source-offset", Value: []byte(strconv.FormatInt(message.Offset, 10))},
		{Key: "x-dlq-failed-at", Value: []byte(time.Now().UTC().Format(time.RFC3339Nano))},
	}

	headers := make([]kafkago.Header, 0, len(message.Headers)+len(dlqHeaders))
	headers = append(headers, message.Headers...)
	headers = append(headers, dlqHeaders...)

	return publisher.writer.WriteMessages(ctx, kafkago.Message{
		Key:     message.Key,
		Value:   message.Value,
		Headers: headers,
	})
}

func (publisher *kafkaDeadLetterPublisher) Close() error {
	return publisher.writer.Close()
}
