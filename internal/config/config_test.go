package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	for _, key := range []string{
		"HTTP_ADDR", "DATABASE_URL", "MIGRATIONS_DIR", "KAFKA_BROKERS",
		"KAFKA_TOPIC", "KAFKA_GROUP_ID", "KAFKA_DLQ_TOPIC",
		"KAFKA_MAX_RETRIES", "KAFKA_INITIAL_BACKOFF", "KAFKA_MAX_BACKOFF",
	} {
		t.Setenv(key, "")
	}

	config := Load()

	if config.HTTPAddr != ":8080" {
		t.Fatalf("unexpected HTTPAddr: %q", config.HTTPAddr)
	}
	if config.KafkaTopic != "price.changed" {
		t.Fatalf("unexpected KafkaTopic: %q", config.KafkaTopic)
	}
	if config.KafkaMaxRetries != 5 {
		t.Fatalf("unexpected KafkaMaxRetries: %d", config.KafkaMaxRetries)
	}
	if config.KafkaInitialBackoff != 500*time.Millisecond {
		t.Fatalf("unexpected KafkaInitialBackoff: %v", config.KafkaInitialBackoff)
	}
	if !reflect.DeepEqual(config.KafkaBrokers, []string{"localhost:9092"}) {
		t.Fatalf("unexpected KafkaBrokers: %v", config.KafkaBrokers)
	}
}

func TestLoadOverrides(t *testing.T) {
	t.Setenv("HTTP_ADDR", ":9090")
	t.Setenv("KAFKA_BROKERS", " a:1 , b:2 , ")
	t.Setenv("KAFKA_MAX_RETRIES", "9")
	t.Setenv("KAFKA_INITIAL_BACKOFF", "2s")

	config := Load()

	if config.HTTPAddr != ":9090" {
		t.Fatalf("unexpected HTTPAddr: %q", config.HTTPAddr)
	}
	if !reflect.DeepEqual(config.KafkaBrokers, []string{"a:1", "b:2"}) {
		t.Fatalf("unexpected KafkaBrokers: %v", config.KafkaBrokers)
	}
	if config.KafkaMaxRetries != 9 {
		t.Fatalf("unexpected KafkaMaxRetries: %d", config.KafkaMaxRetries)
	}
	if config.KafkaInitialBackoff != 2*time.Second {
		t.Fatalf("unexpected KafkaInitialBackoff: %v", config.KafkaInitialBackoff)
	}
}

func TestLoadInvalidValuesFallBack(t *testing.T) {
	t.Setenv("KAFKA_MAX_RETRIES", "not-a-number")
	t.Setenv("KAFKA_INITIAL_BACKOFF", "not-a-duration")

	config := Load()

	if config.KafkaMaxRetries != 5 {
		t.Fatalf("expected fallback retries 5, got %d", config.KafkaMaxRetries)
	}
	if config.KafkaInitialBackoff != 500*time.Millisecond {
		t.Fatalf("expected fallback backoff, got %v", config.KafkaInitialBackoff)
	}
}
