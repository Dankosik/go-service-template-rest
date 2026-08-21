package oauth2clientcredentials

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/oauth2"
)

type acquireToken func(context.Context) (*oauth2.Token, error)

type acquisitionWave struct {
	done   chan struct{}
	cancel context.CancelFunc
	token  *oauth2.Token
	err    error
}

// Client owns one process-local cache and one cancelable provider acquisition.
// Composition code owns Client; feature code receives only HTTP or gRPC clients.
type Client struct {
	acquire    acquireToken
	closeIdle  func()
	processCtx context.Context //nolint:containedctx // Client owns and cancels this lifetime in Close.
	cancel     context.CancelFunc

	mu        sync.Mutex
	token     *oauth2.Token
	wave      *acquisitionWave
	retired   bool
	closeDone chan struct{}
	closeOnce sync.Once
}

// New validates cfg and builds an idle client factory without provider I/O.
func New(cfg Config) (*Client, error) {
	validated, err := validateConfig(cfg)
	if err != nil {
		return nil, err
	}
	bounded, err := newTokenHTTPClient(validated)
	if err != nil {
		return nil, err
	}
	return newClient(newAcquirer(validated, bounded), bounded.CloseIdleConnections), nil
}

func newClient(acquire acquireToken, closeIdle func()) *Client {
	if closeIdle == nil {
		closeIdle = func() {}
	}
	processCtx, cancel := context.WithCancel(context.Background())
	return &Client{
		acquire:    acquire,
		closeIdle:  closeIdle,
		processCtx: processCtx,
		cancel:     cancel,
		closeDone:  make(chan struct{}),
	}
}

func (c *Client) resolve(ctx context.Context) (*oauth2.Token, error) {
	if c == nil || ctx == nil || c.acquire == nil {
		return nil, ErrInvalidConfiguration
	}
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("resolve outbound authentication: %w", err)
	}

	c.mu.Lock()
	if c.retired {
		c.mu.Unlock()
		return nil, ErrUnavailable
	}
	if c.token.Valid() {
		token := c.token
		c.mu.Unlock()
		return token, nil
	}
	c.token = nil
	wave := c.wave
	if wave == nil {
		providerCtx, cancel := context.WithTimeout(c.processCtx, defaultAcquisitionTimeout)
		wave = &acquisitionWave{done: make(chan struct{}), cancel: cancel}
		c.wave = wave
		go c.runAcquisition(providerCtx, wave) //nolint:contextcheck // The process owns shared provider work.
	}
	c.mu.Unlock()

	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("resolve outbound authentication: %w", ctx.Err())
	case <-wave.done:
		if err := ctx.Err(); err != nil {
			return nil, fmt.Errorf("resolve outbound authentication: %w", err)
		}
		return wave.token, wave.err
	}
}

func (c *Client) runAcquisition(ctx context.Context, wave *acquisitionWave) {
	token, err := c.acquire(ctx)
	wave.cancel()
	if err == nil {
		token, err = sanitizeToken(token)
	}

	c.mu.Lock()
	if c.retired {
		token = nil
		err = ErrUnavailable
	} else if err == nil {
		c.token = token
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

func (c *Client) available() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.acquire != nil && !c.retired
}

// Close rejects new acquisitions, cancels and joins active provider work, and
// releases the token transport's idle connection.
func (c *Client) Close(ctx context.Context) error {
	if c == nil || ctx == nil {
		return ErrInvalidConfiguration
	}
	c.mu.Lock()
	first := !c.retired
	if first {
		c.retired = true
		c.token = nil
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
	case <-ctx.Done():
		return fmt.Errorf("close outbound authentication: %w", ctx.Err())
	}
}

func (c *Client) finishClose() {
	c.closeOnce.Do(func() {
		c.closeIdle()
		close(c.closeDone)
	})
}
