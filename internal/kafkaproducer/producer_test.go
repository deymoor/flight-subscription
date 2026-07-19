package kafkaproducer

import (
	"errors"
	"testing"
)

func TestNewProducerValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{name: "no brokers", config: Config{Topic: "t"}, wantErr: true},
		{name: "blank brokers", config: Config{Brokers: []string{"   "}, Topic: "t"}, wantErr: true},
		{name: "no topic", config: Config{Brokers: []string{"b"}}, wantErr: true},
		{name: "blank topic", config: Config{Brokers: []string{"b"}, Topic: "  "}, wantErr: true},
		{name: "valid", config: Config{Brokers: []string{"127.0.0.1:1"}, Topic: "t"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			producer, err := New(tt.config)
			if tt.wantErr {
				if !errors.Is(err, ErrInvalidProducerConfig) {
					t.Fatalf("expected ErrInvalidProducerConfig, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if producer == nil {
				t.Fatal("expected producer")
			}
			_ = producer.Close()
		})
	}
}
