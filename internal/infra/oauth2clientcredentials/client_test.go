package oauth2clientcredentials

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestClientSharesAcquisitionWithoutSharingCallerCancellation(t *testing.T) {
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
	t.Cleanup(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := client.Close(closeCtx); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})

	canceledCtx, cancel := context.WithCancel(context.Background())
	canceled := make(chan error, 1)
	go func() {
		_, err := client.resolve(canceledCtx)
		canceled <- err
	}()
	<-entered

	live := make(chan *oauth2.Token, 1)
	go func() {
		token, _ := client.resolve(context.Background())
		live <- token
	}()
	cancel()
	if err := <-canceled; !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled resolve error = %v, want context.Canceled", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("provider calls before release = %d, want 1", got)
	}

	close(release)
	if token := <-live; token == nil || token.AccessToken != "shared" {
		t.Fatalf("live token = %#v, want shared token", token)
	}
	if _, err := client.resolve(context.Background()); err != nil {
		t.Fatalf("cached resolve error = %v", err)
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("provider calls after cache hit = %d, want 1", got)
	}
}

func TestClientCloseCancelsAndJoinsAcquisition(t *testing.T) {
	entered := make(chan struct{})
	closed := make(chan struct{})
	client := newClient(func(ctx context.Context) (*oauth2.Token, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	}, func() { close(closed) })

	resolved := make(chan error, 1)
	go func() {
		_, err := client.resolve(context.Background())
		resolved <- err
	}()
	<-entered

	closeCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.Close(closeCtx); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	select {
	case <-closed:
	default:
		t.Fatal("Close() returned before idle transport close")
	}
	if err := <-resolved; !errors.Is(err, ErrUnavailable) {
		t.Fatalf("resolve after Close error = %v, want ErrUnavailable", err)
	}
}

func validTestToken(value string) *oauth2.Token {
	return &oauth2.Token{AccessToken: value, TokenType: "Bearer", Expiry: time.Now().Add(time.Hour)}
}
