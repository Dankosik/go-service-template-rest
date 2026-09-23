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
	client *Client
}

func newProducer(client *Client) *Producer {
	return &Producer{client: client}
}

func (p *Producer) Publish(ctx context.Context, event Event) (PublishResult, error) {
	started := time.Now()
	if err := validateEvent(event, p.client.cfg.MaxPayloadBytes); err != nil {
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
	// The event was validated above; what remains is the encoded envelope bound.
	msg, err := buildNATSMessage(ctx, event, p.client.cfg.MaxPayloadBytes)
	if err != nil {
		setSpanOutcome(span, outcomeRejected)
		p.client.telemetry.recordPublish(ctx, event, outcomeRejected, reasonInvalidMessage, started)
		return PublishResult{}, err
	}
	ack, err := p.client.publishOnce(ctx, msg, event.PublicationID, p.client.cfg.Stream)
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

// publishOnce makes exactly one publish attempt of msg to stream under
// operationTimeout. msgID is the broker's deduplication id, which is what makes
// retrying an ambiguous attempt safe; retries stay with the caller.
func (c *Client) publishOnce(ctx context.Context, msg *nats.Msg, msgID, stream string) (*jetstream.PubAck, error) {
	publishCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	//nolint:wrapcheck // classifyPublishError maps the broker error onto this package's sentinels.
	return c.js.PublishMsg(
		publishCtx,
		msg,
		jetstream.WithMsgID(msgID),
		jetstream.WithExpectStream(stream),
		jetstream.WithRetryAttempts(0),
	)
}

func classifyPublishError(err error) (outcome, reason string, wrapped error) {
	_, apiErr := errors.AsType[*jetstream.APIError](err)
	if apiErr || errors.Is(err, nats.ErrNoResponders) || errors.Is(err, jetstream.ErrNoStreamResponse) ||
		errors.Is(err, jetstream.ErrStreamNotFound) {
		return outcomeRejected, reasonBrokerRejected, fmt.Errorf("%w: broker rejected publish", ErrRejected)
	}
	return outcomeAmbiguous, reasonAckUnavailable, fmt.Errorf("%w: publish acknowledgement unavailable", ErrAmbiguous)
}
