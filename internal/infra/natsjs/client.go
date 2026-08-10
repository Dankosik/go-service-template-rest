package natsjs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type connectionEvent uint8

const (
	eventReconnect connectionEvent = iota + 1
)

// Client is one connection to the broker and everything that outlives a single
// publish or delivery: the reconnect state machine, the readiness probe, and the
// telemetry both roles share. [Connect] is the only constructor.
//
// One client serves one role. A producer client publishes through
// [Client.Producer]; a worker client additionally admits one [Worker] through
// [Client.NewWorker], and registering that worker widens [Client.Check] to probe
// the durable consumer as well as the stream.
//
// Its state is three flags rather than one because they answer different
// questions: ready is what the probe reports, draining is set once shutdown
// starts and never clears, and intentional separates a close this process asked
// for from reconnect exhaustion — only the latter is terminal.
type Client struct {
	cfg Config
	nc  *nats.Conn
	js  jetstream.JetStream

	producer  *Producer
	telemetry *telemetry

	ready       atomic.Bool
	draining    atomic.Bool
	intentional atomic.Bool
	events      chan connectionEvent
	reconnected chan struct{}
	terminal    chan error
	closed      chan struct{}
	closedOnce  sync.Once

	probeMu          sync.RWMutex
	consumer         pullConsumer
	deadLetterStream string
	maxDeliveryBytes int
	workerClaimed    bool
}

func Connect(ctx context.Context, cfg Config, role Role, obs Observability) (*Client, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: connect context: %w", ErrRejected, err)
	}
	c := &Client{
		cfg:         cfg,
		events:      make(chan connectionEvent, 1),
		reconnected: make(chan struct{}, 1),
		terminal:    make(chan error, 1),
		closed:      make(chan struct{}),
	}
	telemetry, err := newTelemetry(obs, role, c.Ready)
	if err != nil {
		return nil, fmt.Errorf("create messaging telemetry: %w", err)
	}
	c.telemetry = telemetry
	// Every failure from here on releases through Close, which guards each field
	// it touches and so is correct at any point in this construction. One
	// spelling means a resource added to Client is released on all of them.
	nc, err := nats.Connect(strings.Join(cfg.URLs, ","), c.connectOptions(ctx, cfg)...)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("%w: messaging connection failed", ErrRejected)
	}
	c.nc = nc
	c.js, err = jetstream.New(nc)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("%w: messaging protocol initialization failed", ErrRejected)
	}
	c.producer = newProducer(c, cfg.MaxPendingPublishes, cfg.MaxPayloadBytes)
	if err := c.Check(ctx); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// connectOptions is the connection policy and the four asynchronous handlers
// that maintain client state behind it. Only the ClosedHandler can end the
// process.
//
// Every handler records against context.WithoutCancel(ctx): they fire long after
// Connect returns, when the caller's startup context is usually cancelled.
func (c *Client) connectOptions(ctx context.Context, cfg Config) []nats.Option {
	options := []nats.Option{
		nats.Name("service-messaging"),
		nats.Timeout(boundedTimeout(ctx)),
		nats.ReconnectWait(time.Second),
		nats.ReconnectJitter(50*time.Millisecond, 50*time.Millisecond),
		// The nats client library applies MaxReconnects to each server URL, not
		// to the pool as a whole. Keep the concrete policy here and document its
		// multi-URL exhaustion behavior instead of exposing speculative tuning
		// knobs.
		nats.MaxReconnects(60),
		nats.ReconnectBufSize(-1),
		nats.DisconnectErrHandler(func(_ *nats.Conn, _ error) {
			c.ready.Store(false)
			c.telemetry.recordConnection(context.WithoutCancel(ctx), connectionDisconnected)
		}),
		// A reconnect clears readiness rather than restoring it: the connection is
		// back, but nothing has re-probed the stream or the consumer yet. The
		// event sent here is what makes Client.Run do that.
		nats.ReconnectHandler(func(_ *nats.Conn) {
			c.ready.Store(false)
			c.telemetry.recordConnection(context.WithoutCancel(ctx), connectionReconnected)
			select {
			case c.events <- eventReconnect:
			default:
			}
			select {
			case c.reconnected <- struct{}{}:
			default:
			}
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			c.telemetry.recordAsyncError(context.WithoutCancel(ctx), err)
		}),
		// The one handler that can stop the process. A close this process asked
		// for already set intentional; anything else means reconnect exhaustion,
		// which no amount of waiting recovers from.
		nats.ClosedHandler(func(_ *nats.Conn) {
			c.ready.Store(false)
			c.telemetry.recordConnection(context.WithoutCancel(ctx), connectionClosed)
			c.closedOnce.Do(func() { close(c.closed) })
			if !c.intentional.Load() {
				c.signalTerminal(fmt.Errorf("%w: connection closed after reconnect exhaustion", ErrTerminal))
			}
		}),
	}
	if cfg.CredentialsFile != "" {
		options = append(options, nats.UserCredentials(cfg.CredentialsFile))
	}
	if cfg.RootCAFile != "" {
		options = append(options, nats.RootCAs(cfg.RootCAFile))
	}
	return options
}

func (c *Client) Producer() *Producer { return c.producer }

func (c *Client) Name() string { return "messaging" }

func (c *Client) Ready() bool {
	return c != nil && c.ready.Load() && !c.draining.Load()
}

func (c *Client) Check(ctx context.Context) error {
	if c == nil || c.nc == nil || !c.nc.IsConnected() || c.draining.Load() {
		if c != nil {
			c.ready.Store(false)
		}
		return fmt.Errorf("%w: connection is not ready", ErrRejected)
	}
	probeCtx, cancel := context.WithTimeout(ctx, boundedTimeout(ctx))
	defer cancel()
	sourceInfo, err := c.inspectStream(probeCtx, c.cfg.Stream, "source")
	if err != nil {
		c.ready.Store(false)
		return err
	}
	c.probeMu.RLock()
	consumer := c.consumer
	deadLetterStream := c.deadLetterStream
	maxDeliveryBytes := c.maxDeliveryBytes
	c.probeMu.RUnlock()
	if maxDeliveryBytes > 0 &&
		(sourceInfo.Config.MaxMsgSize <= 0 || int(sourceInfo.Config.MaxMsgSize) > maxDeliveryBytes) {
		c.ready.Store(false)
		return fmt.Errorf("%w: source stream max message size is unbounded or exceeds worker delivery bound", ErrRejected)
	}
	if deadLetterStream != "" {
		if err := c.inspectDeadLetterStream(probeCtx, deadLetterStream); err != nil {
			c.ready.Store(false)
			return err
		}
	}
	var consumerInfo *jetstream.ConsumerInfo
	if consumer != nil {
		consumerInfo, err = consumer.Info(probeCtx)
		if err != nil {
			c.ready.Store(false)
			return fmt.Errorf("%w: durable consumer is unavailable", ErrRejected)
		}
	}
	if !c.nc.IsConnected() {
		c.ready.Store(false)
		return fmt.Errorf("%w: connection changed during readiness probe", ErrRejected)
	}
	c.telemetry.recordBrokerObservation(time.Now(), sourceInfo, consumerInfo)
	c.ready.Store(true)
	return nil
}

func (c *Client) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("run messaging client: %w", ctx.Err())
		case err := <-c.terminal:
			return err
		case <-c.events:
			probeCtx, cancel := context.WithTimeout(ctx, operationTimeout)
			_ = c.Check(probeCtx)
			cancel()
		}
	}
}

func (c *Client) StopPublish() {
	if c == nil {
		return
	}
	c.draining.Store(true)
	c.ready.Store(false)
	c.producer.stop()
}

func (c *Client) Shutdown(ctx context.Context) error {
	if c == nil || c.nc == nil {
		return nil
	}
	c.StopPublish()
	if err := c.producer.wait(ctx); err != nil {
		c.Close()
		return fmt.Errorf("wait for admitted messaging publishes: %w", err)
	}
	c.intentional.Store(true)
	if err := c.nc.Drain(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
		c.Close()
		return fmt.Errorf("%w: messaging connection drain failed", ErrTerminal)
	}
	select {
	case <-c.closed:
		c.telemetry.close()
		return nil
	case <-ctx.Done():
		c.Close()
		return fmt.Errorf("wait for messaging connection drain: %w", ctx.Err())
	}
}

func (c *Client) Close() {
	if c == nil {
		return
	}
	c.intentional.Store(true)
	c.ready.Store(false)
	c.draining.Store(true)
	if c.producer != nil {
		c.producer.stop()
	}
	if c.nc != nil {
		c.nc.Close()
	}
	if c.telemetry != nil {
		c.telemetry.close()
	}
}

func (c *Client) signalTerminal(err error) {
	c.ready.Store(false)
	select {
	case c.terminal <- err:
	default:
	}
}

// waitForReconnect reports whether err is an outage this client can ride out,
// having waited for the connection to come back if so. A false result is what
// turns a failed pull terminal, so the error set below is the whole difference
// between a blip and a stopped worker.
//
// Those four errors are the ones the nats client library raises for a
// connection that is gone but recoverable; IsReconnecting covers the case
// where the library noticed first.
// Anything else — a permission fault, a deleted stream, a malformed request — is
// a condition waiting cannot fix, so it must not land here.
func (c *Client) waitForReconnect(ctx context.Context, err error) bool {
	if c == nil || c.nc == nil {
		return false
	}
	if !c.nc.IsReconnecting() &&
		!errors.Is(err, nats.ErrDisconnected) &&
		!errors.Is(err, nats.ErrReconnectBufExceeded) &&
		!errors.Is(err, jetstream.ErrConnectionClosed) &&
		!errors.Is(err, jetstream.ErrServerShutdown) {
		return false
	}
	for {
		if c.nc.IsConnected() {
			return true
		}
		select {
		case <-c.reconnected:
			// A previous fast reconnect can leave a notification queued after
			// the connected-state check already succeeded. Recheck in the loop;
			// a stale notification must not turn the next outage terminal.
		case <-c.closed:
			return false
		case <-ctx.Done():
			return false
		}
	}
}

func boundedTimeout(ctx context.Context) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < operationTimeout {
			if remaining <= 0 {
				return time.Nanosecond
			}
			return remaining
		}
	}
	return operationTimeout
}
