package redpanda

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidEvent     = errors.New("redpanda event invalid")
	ErrFingerprint      = errors.New("redpanda event fingerprint mismatch")
	ErrProducerIdentity = errors.New("redpanda producer identity invalid")
	ErrPayloadConflict  = errors.New("redpanda payload conflict")
	ErrRetryable        = errors.New("redpanda retryable failure")
)

type Message struct {
	Topic     string
	Partition int32
	Offset    int64
	Key       []byte
	Value     []byte
	Headers   map[string]string
	Attempt   int
}

type OffsetCommitter interface {
	CommitOffset(context.Context, Message) error
}

type MessageConsumer interface {
	FetchMessage(context.Context) (Message, error)
	CommitOffset(context.Context, Message) error
}

type Closeable interface {
	Close() error
}

type EventObserver interface {
	ObserveEvent(topic, eventType, result, reasonClass string)
}

type Producer interface {
	Produce(context.Context, ProduceMessage) error
}

type ProduceMessage struct {
	Topic   string
	Key     []byte
	Value   []byte
	Headers map[string]string
}

type RetryPolicy struct {
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

func (p RetryPolicy) NextDelay(attempt int) time.Duration {
	base := p.BaseDelay
	if base <= 0 {
		base = 50 * time.Millisecond
	}
	maxDelay := p.MaxDelay
	if maxDelay <= 0 {
		maxDelay = time.Second
	}
	if attempt < 0 {
		attempt = 0
	}
	delay := base
	for i := 0; i < attempt; i++ {
		if delay >= maxDelay/2 {
			return maxDelay
		}
		delay *= 2
	}
	if delay > maxDelay {
		return maxDelay
	}
	return delay
}

type RetryAfterError struct {
	After time.Duration
	Err   error
}

func (e RetryAfterError) Error() string {
	if e.Err == nil {
		return "retry after"
	}
	return e.Err.Error()
}

func (e RetryAfterError) Unwrap() error {
	return e.Err
}

func retryAfter(policy RetryPolicy, attempt int, err error) RetryAfterError {
	return RetryAfterError{After: policy.NextDelay(attempt), Err: err}
}
