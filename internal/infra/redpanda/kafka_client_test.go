package redpanda

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestKafkaClientConstructorsValidateRuntimeConfiguration(t *testing.T) {
	t.Parallel()

	if got := cleanBrokers([]string{" redpanda:9092 ", "", " localhost:19092 "}); strings.Join(got, ",") != "redpanda:9092,localhost:19092" {
		t.Fatalf("cleanBrokers() = %v, want trimmed non-empty brokers", got)
	}
	if _, err := NewKafkaConsumer(ClientConfig{}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("NewKafkaConsumer(empty) error = %v, want invalid event", err)
	}
	consumer, err := NewKafkaConsumer(ClientConfig{
		Brokers:       []string{" redpanda:9092 "},
		Topic:         " billing.microlease.terminal.v1 ",
		ConsumerGroup: " billing-worker ",
	})
	if err != nil {
		t.Fatalf("NewKafkaConsumer(valid) error = %v", err)
	}
	if err := consumer.Close(); err != nil {
		t.Fatalf("consumer Close() error = %v", err)
	}

	if _, err := NewKafkaProducer(nil); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("NewKafkaProducer(empty) error = %v, want invalid event", err)
	}
	producer, err := NewKafkaProducer([]string{" redpanda:9092 "})
	if err != nil {
		t.Fatalf("NewKafkaProducer(valid) error = %v", err)
	}
	if err := producer.Close(); err != nil {
		t.Fatalf("producer Close() error = %v", err)
	}
}

func TestKafkaClientNilGuardsAndBrokerProbeFailuresAreSafe(t *testing.T) {
	t.Parallel()

	var consumer *KafkaConsumer
	if _, err := consumer.FetchMessage(context.Background()); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("nil FetchMessage error = %v, want invalid event", err)
	}
	if err := consumer.CommitOffset(context.Background(), Message{}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("nil CommitOffset error = %v, want invalid event", err)
	}
	if err := consumer.Close(); err != nil {
		t.Fatalf("nil consumer Close() error = %v", err)
	}
	var producer *KafkaProducer
	if err := producer.Produce(context.Background(), ProduceMessage{}); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("nil Produce error = %v, want invalid event", err)
	}
	if err := producer.Close(); err != nil {
		t.Fatalf("nil producer Close() error = %v", err)
	}

	probe := NewBrokerProbe(nil, 0)
	if probe.Name() != "redpanda" {
		t.Fatalf("probe name = %q, want redpanda", probe.Name())
	}
	if err := probe.Check(context.Background()); !errors.Is(err, ErrInvalidEvent) {
		t.Fatalf("empty broker probe error = %v, want invalid event", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()
	if err := NewBrokerProbe([]string{"127.0.0.1:1"}, time.Millisecond).Check(ctx); !errors.Is(err, ErrRetryable) {
		t.Fatalf("unreachable broker probe error = %v, want retryable", err)
	}
}

func TestRetryAfterErrorPreservesRetryCause(t *testing.T) {
	t.Parallel()

	cause := errors.New("temporary broker failure")
	wrapped := retryAfter(RetryPolicy{BaseDelay: time.Millisecond, MaxDelay: time.Second}, 2, cause)
	if wrapped.After != 4*time.Millisecond {
		t.Fatalf("retry delay = %s, want exponential delay", wrapped.After)
	}
	if !errors.Is(wrapped, cause) || wrapped.Error() != cause.Error() {
		t.Fatalf("wrapped retry error = %v, want cause preservation", wrapped)
	}
	if got := (RetryAfterError{}).Error(); got != "retry after" {
		t.Fatalf("empty retry error = %q, want fallback message", got)
	}
}
