package natsjs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type Producer struct {
	client          *Client
	maxPayloadBytes int
	tokens          chan struct{}

	mu       sync.Mutex
	stopped  bool
	inFlight int
	idle     chan struct{}
}

func newProducer(client *Client, capacity, maxPayloadBytes int) *Producer {
	idle := make(chan struct{})
	close(idle)
	return &Producer{
		client:          client,
		maxPayloadBytes: maxPayloadBytes,
		tokens:          make(chan struct{}, capacity),
		idle:            idle,
	}
}

func (p *Producer) Publish(ctx context.Context, event Event) (PublishResult, error) {
	started := time.Now()
	if err := validateEvent(event, p.maxPayloadBytes); err != nil {
		p.client.signals.publish(ctx, event, "rejected", "invalid_message", started)
		return PublishResult{}, err
	}
	if err := ctx.Err(); err != nil {
		p.client.signals.publish(ctx, event, "rejected", "context_done_before_dispatch", started)
		return PublishResult{}, fmt.Errorf("%w: publish context before dispatch: %w", ErrRejected, err)
	}
	if err := p.begin(); err != nil {
		reason := "capacity"
		if errors.Is(err, ErrDraining) {
			reason = "draining"
		}
		p.client.signals.publish(ctx, event, "rejected", reason, started)
		return PublishResult{}, err
	}
	defer p.end()

	ctx, span := p.client.signals.tracer.Start(ctx, "messaging publish",
		trace.WithSpanKind(trace.SpanKindProducer),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.operation.type", "publish"),
			attribute.String("messaging.destination.name", event.Subject),
			attribute.String("messaging.message.id", event.MessageID),
		),
	)
	defer span.End()
	msg, err := buildNATSMessage(ctx, event, p.maxPayloadBytes)
	if err != nil {
		span.SetAttributes(attribute.String("messaging.operation.outcome", "rejected"))
		p.client.signals.publish(ctx, event, "rejected", "invalid_message", started)
		return PublishResult{}, err
	}
	publishCtx, cancel := context.WithTimeout(ctx, boundedTimeout(ctx))
	defer cancel()
	ack, err := p.client.js.PublishMsg(
		publishCtx,
		msg,
		jetstream.WithMsgID(event.PublicationID),
		jetstream.WithExpectStream(p.client.cfg.Stream),
		jetstream.WithRetryAttempts(0),
	)
	if err != nil {
		outcome, reason, wrapped := classifyPublishError(err)
		span.SetAttributes(attribute.String("messaging.operation.outcome", outcome))
		p.client.signals.publish(ctx, event, outcome, reason, started)
		return PublishResult{}, wrapped
	}
	result := PublishResult{Stream: ack.Stream, Sequence: ack.Sequence, Duplicate: ack.Duplicate}
	span.SetAttributes(attribute.String("messaging.operation.outcome", "accepted"))
	p.client.signals.publish(ctx, event, "accepted", "none", started)
	return result, nil
}

func (p *Producer) begin() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stopped {
		return fmt.Errorf("%w: %w", ErrRejected, ErrDraining)
	}
	select {
	case p.tokens <- struct{}{}:
	default:
		return fmt.Errorf("%w: %w", ErrRejected, ErrCapacity)
	}
	if p.inFlight == 0 {
		p.idle = make(chan struct{})
	}
	p.inFlight++
	return nil
}

func (p *Producer) end() {
	<-p.tokens
	p.mu.Lock()
	p.inFlight--
	if p.inFlight == 0 {
		close(p.idle)
	}
	p.mu.Unlock()
}

func (p *Producer) stop() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.stopped = true
	p.mu.Unlock()
}

func (p *Producer) wait(ctx context.Context) error {
	if p == nil {
		return nil
	}
	p.mu.Lock()
	idle := p.idle
	p.mu.Unlock()
	select {
	case <-idle:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("wait for admitted messaging publishes: %w", ctx.Err())
	}
}

func classifyPublishError(err error) (string, string, error) {
	if errors.Is(err, nats.ErrNoResponders) || errors.Is(err, jetstream.ErrNoStreamResponse) || errors.Is(err, jetstream.ErrStreamNotFound) {
		return "rejected", "broker_rejected", fmt.Errorf("%w: broker rejected publish", ErrRejected)
	}
	if _, ok := errors.AsType[*jetstream.APIError](err); ok {
		return "rejected", "broker_rejected", fmt.Errorf("%w: broker rejected publish", ErrRejected)
	}
	return "ambiguous", "ack_unavailable", fmt.Errorf("%w: publish acknowledgement unavailable", ErrAmbiguous)
}
