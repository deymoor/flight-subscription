package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr      string
	DatabaseURL   string
	MigrationsDir string
	DBMaxConns    int32
	KafkaBrokers  []string
	KafkaTopic    string
	KafkaGroupID  string

	KafkaDLQTopic       string
	KafkaMaxRetries     int
	KafkaInitialBackoff time.Duration
	KafkaMaxBackoff     time.Duration

	KafkaConcurrency      int
	NotificationBatchSize int
}

func Load() Config {
	return Config{
		HTTPAddr:      env("HTTP_ADDR", ":8080"),
		DatabaseURL:   env("DATABASE_URL", "postgres://app:app@localhost:5432/price_subscriptions?sslmode=disable"),
		MigrationsDir: env("MIGRATIONS_DIR", "internal/storage/postgres/migrations"),
		DBMaxConns:    int32(envInt("DB_MAX_CONNS", 25)),
		KafkaBrokers:  split(env("KAFKA_BROKERS", "localhost:9092")),
		KafkaTopic:    env("KAFKA_TOPIC", "price.changed"),
		KafkaGroupID:  env("KAFKA_GROUP_ID", "price-subscriptions-consumer"),

		KafkaDLQTopic:       env("KAFKA_DLQ_TOPIC", "price.changed.dlq"),
		KafkaMaxRetries:     envInt("KAFKA_MAX_RETRIES", 5),
		KafkaInitialBackoff: envDuration("KAFKA_INITIAL_BACKOFF", 500*time.Millisecond),
		KafkaMaxBackoff:     envDuration("KAFKA_MAX_BACKOFF", 30*time.Second),

		KafkaConcurrency:      envInt("KAFKA_CONCURRENCY", 8),
		NotificationBatchSize: envInt("NOTIFICATION_BATCH_SIZE", 1000),
	}
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func split(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}
