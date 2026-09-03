package oauth2clientcredentials

import (
	"context"
	"sync/atomic"
	"time"

	"golang.org/x/oauth2"
)

const (
	defaultAcquisitionTimeout = 5 * time.Second
	defaultEarlyExpiry        = 10 * time.Second
	maxTokenResponseHeaders   = 32 << 10
	maxTokenResponseBody      = 1 << 20
	maxTokenRequestsInFlight  = 1
)

type acquireToken func(context.Context) (*oauth2.Token, error)

type tokenSourceFunc func() (*oauth2.Token, error)

func (f tokenSourceFunc) Token() (*oauth2.Token, error) { return f() }

// Client owns one process-local token source and its acquisition transport.
// Composition code owns Client; feature code receives only HTTP or gRPC clients.
type Client struct {
	source    oauth2.TokenSource
	closeIdle func()
	cancel    context.CancelFunc
	closed    atomic.Bool
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
	client := &Client{closeIdle: closeIdle, cancel: cancel}
	if acquire != nil {
		client.source = oauth2.ReuseTokenSourceWithExpiry(nil, tokenSourceFunc(func() (*oauth2.Token, error) {
			ctx, cancelAcquisition := context.WithTimeout(processCtx, defaultAcquisitionTimeout)
			defer cancelAcquisition()
			token, err := acquire(ctx)
			if err != nil {
				return nil, ErrUnavailable
			}
			return sanitizeToken(token)
		}), defaultEarlyExpiry)
	}
	return client
}

func (c *Client) available() bool {
	return c != nil && c.source != nil && !c.closed.Load()
}

type clientTokenSource struct {
	client *Client
}

func (s clientTokenSource) Token() (*oauth2.Token, error) {
	if !s.client.available() {
		return nil, ErrUnavailable
	}
	token, err := s.client.source.Token()
	if err != nil {
		return nil, ErrUnavailable
	}
	return token, nil
}

// Close retires authentication and releases the token transport's idle connections.
func (c *Client) Close() {
	if c == nil || c.closed.Swap(true) {
		return
	}
	if c.cancel != nil {
		c.cancel()
	}
	if c.closeIdle != nil {
		c.closeIdle()
	}
}
