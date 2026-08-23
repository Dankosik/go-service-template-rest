package oauth2clientcredentials

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"golang.org/x/oauth2"
)

func TestClientUsesLibraryCacheForConcurrentCallers(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	client := newClient(func(context.Context) (*oauth2.Token, error) {
		if calls.Add(1) == 1 {
			close(entered)
		}
		<-release
		return validTestToken("shared"), nil
	}, nil)
	t.Cleanup(client.Close)

	source := clientTokenSource{client: client}
	results := make(chan *oauth2.Token, 2)
	for range 2 {
		go func() {
			token, _ := source.Token()
			results <- token
		}()
	}
	<-entered
	close(release)
	for range 2 {
		if token := <-results; token == nil || token.AccessToken != "shared" {
			t.Fatalf("Token() = %#v, want shared token", token)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("provider calls = %d, want 1", got)
	}
}

func TestClientAcquisitionTimeoutIsDeterministic(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		client := newClient(func(ctx context.Context) (*oauth2.Token, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}, nil)
		defer client.Close()

		started := time.Now()
		_, err := (clientTokenSource{client: client}).Token()
		if !errors.Is(err, ErrUnavailable) {
			t.Fatalf("Token() error = %v, want ErrUnavailable", err)
		}
		if elapsed := time.Since(started); elapsed != defaultAcquisitionTimeout {
			t.Fatalf("acquisition elapsed = %v, want %v", elapsed, defaultAcquisitionTimeout)
		}
	})
}

func TestClientCloseRetiresCachedTokenAndClosesOnce(t *testing.T) {
	new(Client).Close()

	var closes atomic.Int32
	client := newClient(func(context.Context) (*oauth2.Token, error) {
		return validTestToken("cached"), nil
	}, func() { closes.Add(1) })
	source := clientTokenSource{client: client}
	if _, err := source.Token(); err != nil {
		t.Fatalf("Token() error = %v", err)
	}

	client.Close()
	client.Close()
	if _, err := source.Token(); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Token() after Close error = %v, want ErrUnavailable", err)
	}
	if got := closes.Load(); got != 1 {
		t.Fatalf("idle closes = %d, want 1", got)
	}
}

func validTestToken(value string) *oauth2.Token {
	return &oauth2.Token{AccessToken: value, TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
}
