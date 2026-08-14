package oauth2clientcredentials

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"go.opentelemetry.io/otel/metric"
)

type acquireToken func(context.Context) (accessToken, error)

type acquisitionWave struct {
	done        chan struct{}
	cancel      context.CancelFunc
	startedAt   time.Time
	failedUntil time.Time
	token       accessToken
	err         error
}

type operationToken struct {
	value     string
	expiresAt time.Time
	now       func() time.Time
}

func (t operationToken) authorization() (string, error) {
	if t.value == "" || t.expiresAt.Sub(t.now()) <= earlyExpiryMargin {
		return "", failure(FailureTokenUnusable)
	}
	return t.value, nil
}

// Client owns one process-local credential and its acquisition lifecycle.
type Client struct {
	config     Config
	now        func() time.Time
	acquire    acquireToken
	closeIdle  func()
	telemetry  telemetry
	processCtx context.Context //nolint:containedctx // Client owns this process lifetime and cancels it in Close.
	cancel     context.CancelFunc

	mu          sync.Mutex
	token       accessToken
	failure     error
	failedUntil time.Time
	wave        *acquisitionWave
	retired     bool
	closeDone   chan struct{}
	closeOnce   sync.Once
}

// New builds one idle credential owner without contacting the provider.
func New(cfg Config, meterProvider metric.MeterProvider, log *slog.Logger) (*Client, error) {
	validated, err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}
	tokenClient, err := newTokenHTTPClient(validated, meterProvider)
	if err != nil {
		return nil, err
	}
	provider, err := newProvider(validated, tokenClient, time.Now)
	if err != nil {
		tokenClient.CloseIdleConnections()
		return nil, err
	}
	client, err := newClient(
		validated,
		meterProvider,
		log,
		time.Now,
		provider.acquire,
		tokenClient.CloseIdleConnections,
	)
	if err != nil {
		tokenClient.CloseIdleConnections()
		return nil, err
	}
	return client, nil
}

func newClient(
	cfg Config,
	meterProvider metric.MeterProvider,
	log *slog.Logger,
	now func() time.Time,
	acquire acquireToken,
	closeIdle func(),
) (*Client, error) {
	validated, err := validateConfig(cfg)
	if err != nil || now == nil || acquire == nil {
		return nil, failure(FailureInvalidConfiguration)
	}
	if closeIdle == nil {
		closeIdle = func() {}
	}
	processCtx, cancel := context.WithCancel(context.Background())
	return &Client{
		config:     validated,
		now:        now,
		acquire:    acquire,
		closeIdle:  closeIdle,
		telemetry:  newTelemetry(meterProvider, validated.DependencyName, log),
		processCtx: processCtx,
		cancel:     cancel,
		closeDone:  make(chan struct{}),
	}, nil
}

func (c *Client) resolve(ctx context.Context) (operationToken, error) {
	if err := ctx.Err(); err != nil {
		return operationToken{}, c.resolutionFailure(ctx, callerFailure(err))
	}

	c.mu.Lock()
	if c.retired {
		c.mu.Unlock()
		return operationToken{}, c.resolutionFailure(ctx, failure(FailureProviderUnavailable))
	}
	now := c.now()
	if c.token.value != "" && c.token.expiresAt.Sub(now) > earlyExpiryMargin {
		token := c.operationToken(c.token)
		c.mu.Unlock()
		c.telemetry.recordResolution(ctx, sourceCache, nil)
		return token, nil
	}
	c.token = accessToken{}
	if c.failure != nil && now.Before(c.failedUntil) {
		err := c.failure
		c.mu.Unlock()
		return operationToken{}, c.resolutionFailure(ctx, err)
	}

	wave := c.wave
	if wave == nil {
		providerCtx, cancel := context.WithTimeout(c.processCtx, c.config.AcquisitionTimeout)
		wave = &acquisitionWave{
			done:        make(chan struct{}),
			cancel:      cancel,
			startedAt:   time.Now(),
			failedUntil: now.Add(c.config.AcquisitionTimeout),
		}
		c.failure = nil
		c.failedUntil = time.Time{}
		c.wave = wave
		go c.runAcquisition(providerCtx, wave) //nolint:contextcheck // Provider work is process-owned and must outlive each caller.
	}
	c.mu.Unlock()

	select {
	case <-ctx.Done():
		return operationToken{}, c.resolutionFailure(ctx, callerFailure(ctx.Err()))
	case <-wave.done:
		if err := ctx.Err(); err != nil {
			return operationToken{}, c.resolutionFailure(ctx, callerFailure(err))
		}
		if wave.err != nil {
			return operationToken{}, c.resolutionFailure(ctx, wave.err)
		}
		c.telemetry.recordResolution(ctx, sourceAcquisition, nil)
		return c.operationToken(wave.token), nil
	}
}

func (c *Client) runAcquisition(ctx context.Context, wave *acquisitionWave) {
	token, err := c.acquire(ctx)
	wave.cancel()
	if err == nil && (token.value == "" || token.expiresAt.Sub(c.now()) <= earlyExpiryMargin) {
		token = accessToken{}
		err = failure(FailureUnsupportedResponse)
	}
	c.telemetry.recordProviderAttempt(context.WithoutCancel(ctx), err, time.Since(wave.startedAt))

	c.mu.Lock()
	switch {
	case c.retired:
		token = accessToken{}
		err = failure(FailureProviderUnavailable)
	case err == nil:
		c.token = token
		c.failure = nil
		c.failedUntil = time.Time{}
	default:
		c.token = accessToken{}
		c.failure = err
		c.failedUntil = wave.failedUntil
	}
	wave.token = token
	wave.err = err
	if c.wave == wave {
		c.wave = nil
	}
	retired := c.retired
	close(wave.done)
	c.mu.Unlock()

	if retired {
		c.finishClose()
	}
}

func (c *Client) operationToken(token accessToken) operationToken {
	return operationToken{value: token.value, expiresAt: token.expiresAt, now: c.now}
}

func (c *Client) resolutionFailure(ctx context.Context, err error) error {
	c.telemetry.recordResolution(context.WithoutCancel(ctx), sourceAcquisition, err)
	return err
}

// Close retires admission, joins provider work, and releases the token pool.
func (c *Client) Close(ctx context.Context) error {
	c.mu.Lock()
	first := !c.retired
	if first {
		c.retired = true
		c.token = accessToken{}
		c.failure = nil
		c.failedUntil = time.Time{}
	}
	wave := c.wave
	c.mu.Unlock()

	if first {
		c.cancel()
		if wave != nil {
			wave.cancel()
		} else {
			c.finishClose()
		}
	}

	select {
	case <-c.closeDone:
		return nil
	default:
	}
	select {
	case <-c.closeDone:
		return nil
	case <-ctx.Done():
		select {
		case <-c.closeDone:
			return nil
		default:
			return failure(FailureProviderUnavailable)
		}
	}
}

func (c *Client) finishClose() {
	c.closeOnce.Do(func() {
		c.closeIdle()
		close(c.closeDone)
	})
}
