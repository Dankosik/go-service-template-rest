package natsjs

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

// pullConsumer is deliberately narrower than jetstream.Consumer: acquisition
// code cannot call continuous consumption or FetchBytes by construction.
type pullConsumer interface {
	Fetch(batch int, opts ...jetstream.FetchOpt) (jetstream.MessageBatch, error)
	Info(ctx context.Context) (*jetstream.ConsumerInfo, error)
}

const settlementSchedulingSlack = time.Second

// NewWorker admits one durable consumer on this client and binds it to handler.
// Everything it checks is broker topology an operator owns, so every failure is
// [ErrRejected] and none of them is worth retrying: the process should fail
// startup rather than consume against a stream it cannot settle messages on.
//
// A client owns at most one worker. The claim below is held for the whole
// admission, not just the final registration: admission is several round trips
// long and a second caller must not interleave with them.
func (c *Client) NewWorker(ctx context.Context, cfg WorkerConfig, handler Handler) (*Worker, error) {
	if handler == nil {
		return nil, fmt.Errorf("%w: messaging handler is required", ErrRejected)
	}
	if err := ValidateWorkerConfig(cfg, c.cfg.MaxPayloadBytes); err != nil {
		return nil, err
	}
	if err := c.claimWorkerSlot(); err != nil {
		return nil, err
	}
	defer c.releaseWorkerSlot()

	probeCtx, cancel := context.WithTimeout(ctx, boundedTimeout(ctx))
	defer cancel()
	dlqStream, err := c.admitStreams(probeCtx, cfg)
	if err != nil {
		return nil, err
	}
	consumer, err := c.admitConsumer(probeCtx, cfg)
	if err != nil {
		return nil, err
	}
	w := &Worker{
		client:    c,
		cfg:       cfg,
		consumer:  consumer,
		dlqStream: dlqStream,
		handler:   handler,
		fatal:     make(chan error, 1),
		runDone:   make(chan struct{}),
	}
	// Registering the consumer widens Client.Check to probe it too, so the check
	// below is the first one that covers the whole admitted topology. Failing it
	// un-registers, because the client outlives this call and must not keep
	// probing a consumer no worker owns.
	c.probeMu.Lock()
	c.consumer = consumer
	c.deadLetterStream = dlqStream
	c.maxDeliveryBytes = cfg.MaxDeliveryBytes
	c.probeMu.Unlock()
	if err := c.Check(ctx); err != nil {
		c.probeMu.Lock()
		c.consumer = nil
		c.deadLetterStream = ""
		c.maxDeliveryBytes = 0
		c.probeMu.Unlock()
		return nil, err
	}
	return w, nil
}

// claimWorkerSlot reserves this client's single worker slot. An already-set
// consumer means a worker was admitted earlier; workerClaimed means one is being
// admitted right now.
func (c *Client) claimWorkerSlot() error {
	c.probeMu.Lock()
	defer c.probeMu.Unlock()
	if c.workerClaimed || c.consumer != nil {
		return fmt.Errorf("%w: messaging client already owns a worker", ErrRejected)
	}
	c.workerClaimed = true
	return nil
}

func (c *Client) releaseWorkerSlot() {
	c.probeMu.Lock()
	c.workerClaimed = false
	c.probeMu.Unlock()
}

// admitStreams proves both streams can carry this worker's envelope, and returns
// the dead-letter stream's name. It is two checks rather than one because the
// two bounds differ on purpose: the source must state a maximum this worker is
// willing to receive, while the dead-letter stream must be able to hold the
// larger envelope the transfer adds headers to.
func (c *Client) admitStreams(probeCtx context.Context, cfg WorkerConfig) (string, error) {
	if err := c.admitSourceStream(probeCtx, cfg); err != nil {
		return "", err
	}
	return c.admitDeadLetterStream(probeCtx, cfg)
}

// admitSourceStream proves the source states a maximum message size this worker
// is willing to receive. An unbounded stream is rejected too: without a stated
// maximum, nothing keeps a delivery inside the worker's own bound.
func (c *Client) admitSourceStream(probeCtx context.Context, cfg WorkerConfig) error {
	info, err := c.inspectStream(probeCtx, c.cfg.Stream, "source")
	if err != nil {
		return err
	}
	if info.Config.MaxMsgSize <= 0 || int(info.Config.MaxMsgSize) > cfg.MaxDeliveryBytes {
		return fmt.Errorf("%w: source stream max message size is unbounded or exceeds worker delivery bound", ErrRejected)
	}
	return nil
}

// admitDeadLetterStream resolves the stream behind the dead-letter subject and
// proves it can hold the transfer envelope, which is the payload plus the
// Original-* headers the transfer adds. A stream that is also the source is
// rejected: every dead-lettered message would be a delivery back to this worker.
func (c *Client) admitDeadLetterStream(probeCtx context.Context, cfg WorkerConfig) (string, error) {
	name, err := c.js.StreamNameBySubject(probeCtx, cfg.DeadLetterSubject)
	if err != nil && probeCtx.Err() == nil {
		// A newly connected JetStream client can receive one unavailable response
		// while the server finishes attaching its API subscription.
		time.Sleep(100 * time.Millisecond)
		name, err = c.js.StreamNameBySubject(probeCtx, cfg.DeadLetterSubject)
	}
	if err != nil {
		return "", fmt.Errorf("%w: dead-letter stream is unavailable", ErrRejected)
	}
	if name == c.cfg.Stream {
		return "", fmt.Errorf("%w: source and dead-letter streams must differ", ErrRejected)
	}
	if err := c.inspectDeadLetterStream(probeCtx, name); err != nil {
		return "", err
	}
	return name, nil
}

func (c *Client) inspectDeadLetterStream(ctx context.Context, name string) error {
	info, err := c.inspectStream(ctx, name, "dead-letter")
	if err != nil {
		return err
	}
	minimumSize := c.cfg.MaxPayloadBytes + HeaderLimitBytes
	if info.Config.MaxMsgSize > 0 && int(info.Config.MaxMsgSize) < minimumSize {
		return fmt.Errorf("%w: dead-letter stream cannot contain the configured envelope", ErrRejected)
	}
	return nil
}

// inspectStream reads operator-owned topology without mutating it and rejects
// configurations whose accepted messages can disappear before the declared
// recovery window. Numeric capacity stays operator-owned: a finite count or
// byte limit is safe only with discard-new behavior, so exhaustion rejects
// publication instead of evicting older unconsumed work.
func (c *Client) inspectStream(ctx context.Context, name, kind string) (*jetstream.StreamInfo, error) {
	stream, err := c.js.Stream(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("%w: %s stream is unavailable", ErrRejected, kind)
	}
	info, err := stream.Info(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %s stream configuration is unavailable", ErrRejected, kind)
	}
	if err := validateStreamContract(info.Config, c.cfg); err != nil {
		return nil, fmt.Errorf("%s stream: %w", kind, err)
	}
	return info, nil
}

func validateStreamContract(stream jetstream.StreamConfig, cfg Config) error {
	switch {
	case stream.Storage != jetstream.FileStorage:
		return fmt.Errorf("%w: stream must use file storage", ErrRejected)
	case stream.Replicas < cfg.MinStreamReplicas:
		return fmt.Errorf("%w: stream has fewer replicas than configured", ErrRejected)
	case stream.Retention == jetstream.InterestPolicy:
		return fmt.Errorf("%w: interest retention can delete work before a consumer exists", ErrRejected)
	case stream.Discard != jetstream.DiscardNew &&
		(stream.MaxMsgs > 0 || stream.MaxBytes > 0 || stream.MaxMsgsPerSubject > 0):
		return fmt.Errorf("%w: finite stream capacity must reject new messages instead of evicting old ones", ErrRejected)
	case stream.MaxAge > 0 && stream.MaxAge < cfg.MinStreamRetention:
		return fmt.Errorf("%w: stream retention is shorter than configured", ErrRejected)
	case stream.NoAck:
		return fmt.Errorf("%w: stream must acknowledge accepted publications", ErrRejected)
	case stream.Duplicates <= 0:
		return fmt.Errorf("%w: stream duplicate window must be positive", ErrRejected)
	case stream.AllowMsgTTL:
		return fmt.Errorf("%w: per-message TTL can bypass the stream retention contract", ErrRejected)
	case stream.Sealed:
		return fmt.Errorf("%w: sealed stream cannot accept publications", ErrRejected)
	default:
		return nil
	}
}

// admitConsumer finds or creates the durable consumer and proves an existing one
// still matches what this worker would have created. A mismatch is rejected
// rather than reconciled: the consumer may be mid-delivery for another
// deployment, and its AckWait and MaxAckPending are what bound this worker's
// settlement.
//
//nolint:ireturn // The narrowed pullConsumer is the point; see its comment above.
func (c *Client) admitConsumer(probeCtx context.Context, cfg WorkerConfig) (pullConsumer, error) {
	desired := desiredConsumerConfig(cfg)
	consumer, err := c.js.Consumer(probeCtx, c.cfg.Stream, cfg.Consumer)
	if errors.Is(err, jetstream.ErrConsumerNotFound) {
		consumer, err = c.js.CreateConsumer(probeCtx, c.cfg.Stream, desired)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: durable consumer admission failed", ErrRejected)
	}
	info, err := consumer.Info(probeCtx)
	if err != nil {
		return nil, fmt.Errorf("%w: durable consumer configuration is unavailable", ErrRejected)
	}
	if !consumerConfigEqual(info.Config, desired) {
		return nil, fmt.Errorf("%w: existing durable consumer configuration is incompatible", ErrRejected)
	}
	return consumer, nil
}

func desiredConsumerConfig(cfg WorkerConfig) jetstream.ConsumerConfig {
	return jetstream.ConsumerConfig{
		Name:          cfg.Consumer,
		Durable:       cfg.Consumer,
		DeliverPolicy: jetstream.DeliverAllPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		// Cover the handler, DLQ publish, source acknowledgement, and a small
		// scheduling margin without exposing another operator setting.
		AckWait:      cfg.HandlerTimeout + 2*operationTimeout + settlementSchedulingSlack,
		MaxDeliver:   -1,
		ReplayPolicy: jetstream.ReplayInstantPolicy,
		// A canceled pull can remain broker-side until its five-second expiry.
		// Two slots let the one local fetch loop recover without treating that
		// bounded stale request as a terminal MaxWaiting failure.
		MaxWaiting:    2,
		MaxAckPending: cfg.MaxConcurrency,
		// The broker's copy of the worker's own two bounds. Both have to be
		// stated: MaxRequestMaxBytes constrains the batch rather than only
		// rejecting a request that asks for more, so leaving it at one message's
		// size would undo the batching MaxRequestBatch just allowed.
		MaxRequestBatch:    cfg.MaxConcurrency,
		MaxRequestExpires:  operationTimeout,
		MaxRequestMaxBytes: cfg.MaxConcurrency * cfg.MaxDeliveryBytes,
		FilterSubject:      cfg.FilterSubject,
	}
}

func consumerConfigEqual(actual, desired jetstream.ConsumerConfig) bool {
	actual.Description = ""
	actual.Metadata = nil
	desired.Description = ""
	desired.Metadata = nil
	actual.BackOff = nilIfEmpty(actual.BackOff)
	desired.BackOff = nilIfEmpty(desired.BackOff)
	actual.FilterSubjects = nilIfEmpty(actual.FilterSubjects)
	desired.FilterSubjects = nilIfEmpty(desired.FilterSubjects)
	actual.PriorityGroups = nilIfEmpty(actual.PriorityGroups)
	desired.PriorityGroups = nilIfEmpty(desired.PriorityGroups)
	return reflect.DeepEqual(actual, desired)
}

func nilIfEmpty[T any](values []T) []T {
	if len(values) == 0 {
		return nil
	}
	return values
}
