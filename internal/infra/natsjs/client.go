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

// Client owns one NATS connection. The NATS client owns reconnect and consume
// lifecycles; Client only projects them into readiness and process shutdown.
type Client struct {
	cfg Config
	nc  *nats.Conn
	js  jetstream.JetStream

	producer  *Producer
	telemetry *telemetry

	ready          atomic.Bool
	draining       atomic.Bool
	closeRequested atomic.Bool
	terminal       chan error
	closed         chan struct{}
	closedOnce     sync.Once

	probeMu       sync.RWMutex
	consumer      pullConsumer
	workerClaimed atomic.Bool
}

// Connect validates cfg, dials the broker, and returns a client whose first
// readiness probe passed. ctx bounds the dial and the probe.
func Connect(ctx context.Context, cfg Config, obs Observability) (*Client, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: connect context: %w", ErrRejected, err)
	}
	c := &Client{cfg: cfg, terminal: make(chan error, 1), closed: make(chan struct{})}
	telemetry, err := newTelemetry(obs)
	if err != nil {
		return nil, fmt.Errorf("create messaging telemetry: %w", err)
	}
	c.telemetry = telemetry
	nc, err := nats.Connect(strings.Join(cfg.URLs, ","), c.connectOptions(ctx, cfg)...)
	if err != nil {
		return nil, fmt.Errorf("%w: messaging connection failed", ErrRejected)
	}
	c.nc = nc
	c.js, err = jetstream.New(nc)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("%w: messaging protocol initialization failed", ErrRejected)
	}
	c.producer = newProducer(c)
	if err := c.Check(ctx); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

// connectOptions is the connection policy. ReconnectBufSize(-1) disables the
// client's reconnect buffer, so a publish during a disconnect fails at once
// rather than waiting in memory for a reconnect that may never come.
// Reconnection gives up after MaxReconnects attempts about ReconnectWait
// apart, and the closed handler then reports a terminal fault unless this
// client asked for the close.
func (c *Client) connectOptions(ctx context.Context, cfg Config) []nats.Option {
	options := []nats.Option{
		nats.Name("service-messaging"),
		nats.Timeout(boundedTimeout(ctx)),
		nats.ReconnectWait(time.Second),
		nats.ReconnectJitter(50*time.Millisecond, 50*time.Millisecond),
		nats.MaxReconnects(60),
		nats.ReconnectBufSize(-1),
		nats.DisconnectErrHandler(func(_ *nats.Conn, _ error) {
			c.ready.Store(false)
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			// The cached readiness probe restores readiness after it confirms the
			// stream and consumer on the new connection.
			c.ready.Store(false)
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, err error) {
			c.telemetry.recordAsyncError(context.WithoutCancel(ctx), err)
		}),
		nats.ClosedHandler(func(_ *nats.Conn) {
			c.ready.Store(false)
			c.closedOnce.Do(func() { close(c.closed) })
			if !c.closeRequested.Load() {
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

// Producer returns the client's publisher.
func (c *Client) Producer() *Producer { return c.producer }

// Name identifies the client as a readiness dependency.
func (c *Client) Name() string { return "messaging" }

// Ready reports the last readiness probe's result, and false once draining
// starts. It does no I/O.
func (c *Client) Ready() bool {
	return c != nil && c.ready.Load() && !c.draining.Load()
}

// Check probes the broker and records the result as readiness.
func (c *Client) Check(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("%w: connection is not ready", ErrRejected)
	}
	err := c.probe(ctx)
	c.ready.Store(err == nil)
	return err
}

// probe confirms the connection, the source stream, and — once a worker owns
// one — the durable consumer. Check records its result as readiness.
func (c *Client) probe(ctx context.Context) error {
	if c.nc == nil || !c.nc.IsConnected() || c.draining.Load() {
		return fmt.Errorf("%w: connection is not ready", ErrRejected)
	}
	probeCtx, cancel := context.WithTimeout(ctx, operationTimeout)
	defer cancel()
	stream, err := c.js.Stream(probeCtx, c.cfg.Stream)
	if err != nil {
		return fmt.Errorf("%w: source stream is unavailable", ErrRejected)
	}
	if _, err := stream.Info(probeCtx); err != nil {
		return fmt.Errorf("%w: source stream information is unavailable", ErrRejected)
	}
	c.probeMu.RLock()
	consumer := c.consumer
	c.probeMu.RUnlock()
	if consumer != nil {
		if _, err := consumer.Info(probeCtx); err != nil {
			return fmt.Errorf("%w: durable consumer is unavailable", ErrRejected)
		}
	}
	if !c.nc.IsConnected() {
		return fmt.Errorf("%w: connection changed during readiness probe", ErrRejected)
	}
	return nil
}

// Run blocks until ctx ends or the connection fails terminally, and returns
// which one happened.
func (c *Client) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("run messaging client: %w", ctx.Err())
	case err := <-c.terminal:
		return err
	}
}

// StopPublish refuses new publications with [ErrDraining] and withdraws
// readiness. Publications already in flight continue.
func (c *Client) StopPublish() {
	if c == nil {
		return
	}
	c.draining.Store(true)
	c.ready.Store(false)
}

// Shutdown stops publishing, drains the connection, and waits for it to close
// or for ctx to end, closing it outright then.
func (c *Client) Shutdown(ctx context.Context) error {
	if c == nil || c.nc == nil {
		return nil
	}
	c.StopPublish()
	c.closeRequested.Store(true)
	if err := c.nc.Drain(); err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
		c.Close()
		return fmt.Errorf("%w: messaging connection drain failed", ErrTerminal)
	}
	select {
	case <-c.closed:
		return nil
	case <-ctx.Done():
		c.Close()
		return fmt.Errorf("wait for messaging connection drain: %w", ctx.Err())
	}
}

// Close closes the connection without draining. It is safe to call more than
// once.
func (c *Client) Close() {
	if c == nil {
		return
	}
	c.closeRequested.Store(true)
	c.ready.Store(false)
	c.draining.Store(true)
	if c.nc != nil {
		c.nc.Close()
	}
}

func (c *Client) signalTerminal(err error) {
	c.ready.Store(false)
	select {
	case c.terminal <- err:
	default:
	}
}

// boundedTimeout is operationTimeout, shortened to what remains of ctx's
// deadline. nats.Timeout takes a duration rather than a context, so the
// connection dial needs the bound spelled out.
func boundedTimeout(ctx context.Context) time.Duration {
	if deadline, ok := ctx.Deadline(); ok {
		return max(time.Nanosecond, min(operationTimeout, time.Until(deadline)))
	}
	return operationTimeout
}
