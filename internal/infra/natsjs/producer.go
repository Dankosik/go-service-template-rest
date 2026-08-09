package natsjs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

// Producer publishes events on an owning [Client] under a fixed admission
// bound. It never queues: a publish beyond MaxPendingPublishes is rejected with
// [ErrCapacity] rather than made to wait, so a broker that stops acknowledging
// shows up as back pressure on the caller instead of as unbounded memory here.
//
// tokens is the bound and idle is the drain signal; they count the same
// publishes for different readers, which is why stop and wait can let admitted
// work finish while refusing new work.
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
		p.client.telemetry.recordPublish(ctx, event, outcomeRejected, reasonInvalidMessage, started)
		return PublishResult{}, err
	}
	if err := ctx.Err(); err != nil {
		p.client.telemetry.recordPublish(ctx, event, outcomeRejected, reasonContextDone, started)
		return PublishResult{}, fmt.Errorf("%w: publish context before dispatch: %w", ErrRejected, err)
	}
	if err := p.begin(); err != nil {
		reason := reasonCapacity
		if errors.Is(err, ErrDraining) {
			reason = reasonDraining
		}
		p.client.telemetry.recordPublish(ctx, event, outcomeRejected, reason, started)
		return PublishResult{}, err
	}
	defer p.end()

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
	// idle is replaced rather than reset, because a channel cannot be reopened
	// once end closed it. Going from zero in-flight to one is the only edge
	// that needs a fresh one, and wait captures whichever channel was current
	// when it was called — so a drain that started while the producer was idle
	// returns on the already-closed channel instead of waiting for publishes
	// admitted after it.
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

// classifyPublishError decides whether the broker refused a publish or merely
// failed to answer. Only a refusal it can prove is [ErrRejected]; everything
// else is [ErrAmbiguous], because an unanswered publish may still have been
// stored.
func classifyPublishError(err error) (outcome, reason string, wrapped error) {
	if errors.Is(err, nats.ErrNoResponders) || errors.Is(err, jetstream.ErrNoStreamResponse) || errors.Is(err, jetstream.ErrStreamNotFound) {
		return outcomeRejected, reasonBrokerRejected, fmt.Errorf("%w: broker rejected publish", ErrRejected)
	}
	if _, ok := errors.AsType[*jetstream.APIError](err); ok {
		return outcomeRejected, reasonBrokerRejected, fmt.Errorf("%w: broker rejected publish", ErrRejected)
	}
	return outcomeAmbiguous, reasonAckUnavailable, fmt.Errorf("%w: publish acknowledgement unavailable", ErrAmbiguous)
}
