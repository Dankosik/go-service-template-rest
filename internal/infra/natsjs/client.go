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

	ready       atomic.Bool
	draining    atomic.Bool
	intentional atomic.Bool
	terminal    chan error
	closed      chan struct{}
	closedOnce  sync.Once

	probeMu       sync.RWMutex
	consumer      pullConsumer
	workerClaimed atomic.Bool
}

func Connect(ctx context.Context, cfg Config, role Role, obs Observability) (*Client, error) {
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("%w: connect context: %w", ErrRejected, err)
	}
	c := &Client{cfg: cfg, terminal: make(chan error, 1), closed: make(chan struct{})}
	telemetry, err := newTelemetry(obs, role, c.Ready)
	if err != nil {
		return nil, fmt.Errorf("create messaging telemetry: %w", err)
	}
	c.telemetry = telemetry
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
	c.producer = newProducer(c, cfg.MaxPayloadBytes)
	if err := c.Check(ctx); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

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
	stream, err := c.js.Stream(probeCtx, c.cfg.Stream)
	if err != nil {
		c.ready.Store(false)
		return fmt.Errorf("%w: source stream is unavailable", ErrRejected)
	}
	if _, err := stream.Info(probeCtx); err != nil {
		c.ready.Store(false)
		return fmt.Errorf("%w: source stream information is unavailable", ErrRejected)
	}
	c.probeMu.RLock()
	consumer := c.consumer
	c.probeMu.RUnlock()
	if consumer != nil {
		_, err = consumer.Info(probeCtx)
		if err != nil {
			c.ready.Store(false)
			return fmt.Errorf("%w: durable consumer is unavailable", ErrRejected)
		}
	}
	if !c.nc.IsConnected() {
		c.ready.Store(false)
		return fmt.Errorf("%w: connection changed during readiness probe", ErrRejected)
	}
	c.ready.Store(true)
	return nil
}

func (c *Client) Run(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("run messaging client: %w", ctx.Err())
	case err := <-c.terminal:
		return err
	}
}

func (c *Client) StopPublish() {
	if c == nil {
		return
	}
	c.draining.Store(true)
	c.ready.Store(false)
}

func (c *Client) Shutdown(ctx context.Context) error {
	if c == nil || c.nc == nil {
		return nil
	}
	c.StopPublish()
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
