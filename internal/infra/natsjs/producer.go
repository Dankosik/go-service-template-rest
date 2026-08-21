package natsjs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Producer publishes synchronously and returns only after a JetStream acknowledgement.
// Connection drain, request cancellation, and broker flow control are owned by
// the NATS client; callers bound concurrency at their existing HTTP or River owner.
type Producer struct {
	client          *Client
	maxPayloadBytes int
}

func newProducer(client *Client, maxPayloadBytes int) *Producer {
	return &Producer{client: client, maxPayloadBytes: maxPayloadBytes}
}

func (p *Producer) Publish(ctx context.Context, event Event) (PublishResult, error) {
	started := time.Now()
	if err := validateEvent(event, p.maxPayloadBytes); err != nil {
		p.client.telemetry.recordPublish(ctx, event, outcomeRejected, reasonInvalidMessage, started)
		return PublishResult{}, err
	}
	if err := ctx.Err(); err != nil {
		p.client.telemetry.recordPublish(ctx, event, outcomeRejected, reasonContextDone, started)
		return PublishResult{}, fmt.Errorf("%w: publish context before dispatch: %w", ErrRejected, err)
	}
	if p.client.draining.Load() {
		p.client.telemetry.recordPublish(ctx, event, outcomeRejected, reasonDraining, started)
		return PublishResult{}, fmt.Errorf("%w: %w", ErrRejected, ErrDraining)
	}

	ctx, span := p.client.telemetry.tracer.Start(ctx, publishSpanName(event.Subject), publishSpanOptions(event)...)
	defer span.End()
	msg, err := buildNATSMessage(ctx, event, p.maxPayloadBytes)
	if err != nil {
		setSpanOutcome(span, outcomeRejected)
		p.client.telemetry.recordPublish(ctx, event, outcomeRejected, reasonInvalidMessage, started)
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
		setSpanOutcome(span, outcome)
		p.client.telemetry.recordPublish(ctx, event, outcome, reason, started)
		return PublishResult{}, wrapped
	}
	result := PublishResult{Stream: ack.Stream, Sequence: ack.Sequence, Duplicate: ack.Duplicate}
	setSpanOutcome(span, outcomeAccepted)
	p.client.telemetry.recordPublish(ctx, event, outcomeAccepted, reasonNone, started)
	return result, nil
}

func classifyPublishError(err error) (outcome, reason string, wrapped error) {
	if errors.Is(err, nats.ErrNoResponders) || errors.Is(err, jetstream.ErrNoStreamResponse) || errors.Is(err, jetstream.ErrStreamNotFound) {
		return outcomeRejected, reasonBrokerRejected, fmt.Errorf("%w: broker rejected publish", ErrRejected)
	}
	if _, ok := errors.AsType[*jetstream.APIError](err); ok {
		return outcomeRejected, reasonBrokerRejected, fmt.Errorf("%w: broker rejected publish", ErrRejected)
	}
	return outcomeAmbiguous, reasonAckUnavailable, fmt.Errorf("%w: publish acknowledgement unavailable", ErrAmbiguous)
}
