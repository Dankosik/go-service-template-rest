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

type Client struct {
	cfg Config
	nc  *nats.Conn
	js  jetstream.JetStream

	producer *Producer
	signals  *signals

	ready       atomic.Bool
	draining    atomic.Bool
	intentional atomic.Bool
	events      chan connectionEvent
	reconnected chan struct{}
	terminal    chan error
	closed      chan struct{}
	closedOnce  sync.Once

	probeMu  sync.RWMutex
	consumer pullConsumer
}

func Connect(ctx context.Context, cfg Config, role Role, obs Observability) (*Client, error) {
	if err := validateConfig(cfg); err != nil {
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
	sig, err := newSignals(obs, role, c.Ready)
	if err != nil {
		return nil, fmt.Errorf("create messaging telemetry: %w", err)
	}
	c.signals = sig
	options := []nats.Option{
		nats.Name("service-messaging"),
		nats.Timeout(boundedTimeout(ctx)),
		nats.ReconnectWait(time.Second),
		nats.ReconnectJitter(50*time.Millisecond, 50*time.Millisecond),
		// nats.go applies MaxReconnects to each server URL, not to the pool as
		// a whole. Keep the concrete policy here and document its multi-URL
		// exhaustion behavior instead of exposing speculative tuning knobs.
		nats.MaxReconnects(60),
		nats.ReconnectBufSize(-1),
		nats.DisconnectErrHandler(func(_ *nats.Conn, _ error) {
			c.ready.Store(false)
			c.signals.connection(context.WithoutCancel(ctx), "disconnected")
		}),
		nats.ReconnectHandler(func(_ *nats.Conn) {
			c.ready.Store(false)
			c.signals.connection(context.WithoutCancel(ctx), "reconnected")
			select {
			case c.events <- eventReconnect:
			default:
			}
			select {
			case c.reconnected <- struct{}{}:
			default:
			}
		}),
		nats.ErrorHandler(func(_ *nats.Conn, _ *nats.Subscription, _ error) {
			c.signals.connection(context.WithoutCancel(ctx), "async_error")
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			c.ready.Store(false)
			c.signals.connection(context.WithoutCancel(ctx), "closed")
			c.closedOnce.Do(func() { close(c.closed) })
			if !c.intentional.Load() {
				c.signalTerminal(fmt.Errorf("%w: connection closed after reconnect exhaustion", ErrTerminal))
			}
			_ = nc
		}),
	}
	if cfg.CredentialsFile != "" {
		options = append(options, nats.UserCredentials(cfg.CredentialsFile))
	}
	if cfg.RootCAFile != "" {
		options = append(options, nats.RootCAs(cfg.RootCAFile))
	}
	nc, err := nats.Connect(strings.Join(cfg.URLs, ","), options...)
	if err != nil {
		c.signals.close()
		return nil, fmt.Errorf("%w: messaging connection failed", ErrRejected)
	}
	c.nc = nc
	c.js, err = jetstream.New(nc)
	if err != nil {
		c.intentional.Store(true)
		nc.Close()
		c.signals.close()
		return nil, fmt.Errorf("%w: messaging protocol initialization failed", ErrRejected)
	}
	c.producer = newProducer(c, cfg.MaxPendingPublishes, cfg.MaxPayloadBytes)
	if err := c.Check(ctx); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
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
		return fmt.Errorf("messaging connection is not ready")
	}
	probeCtx, cancel := context.WithTimeout(ctx, boundedTimeout(ctx))
	defer cancel()
	if _, err := c.js.Stream(probeCtx, c.cfg.Stream); err != nil {
		c.ready.Store(false)
		return fmt.Errorf("%w: source stream is unavailable", ErrRejected)
	}
	c.probeMu.RLock()
	consumer := c.consumer
	c.probeMu.RUnlock()
	if consumer != nil {
		if _, err := consumer.Info(probeCtx); err != nil {
			c.ready.Store(false)
			return fmt.Errorf("%w: durable consumer is unavailable", ErrRejected)
		}
	}
	if !c.nc.IsConnected() {
		c.ready.Store(false)
		return fmt.Errorf("messaging connection changed during readiness probe")
	}
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
			probeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), operationTimeout)
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
	errCh := make(chan error, 1)
	go func() { errCh <- c.nc.Drain() }()
	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, nats.ErrConnectionClosed) {
			c.Close()
			return fmt.Errorf("%w: messaging connection drain failed", ErrTerminal)
		}
		select {
		case <-c.closed:
		case <-ctx.Done():
			c.Close()
			return fmt.Errorf("wait for messaging connection drain: %w", ctx.Err())
		}
		c.signals.close()
		return nil
	case <-ctx.Done():
		c.Close()
		return fmt.Errorf("drain messaging connection: %w", ctx.Err())
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
	if c.signals != nil {
		c.signals.close()
	}
}

func (c *Client) signalTerminal(err error) {
	c.ready.Store(false)
	select {
	case c.terminal <- err:
	default:
	}
}

func (c *Client) waitForReconnect(ctx context.Context, err error) bool {
	if c == nil || c.nc == nil {
		return false
	}
	if !c.nc.IsReconnecting() && c.ready.Load() &&
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
