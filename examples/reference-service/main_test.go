package main

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"
)

func TestServeStopsAfterCancellation(t *testing.T) {
	t.Parallel()

	listener, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := serve(ctx, listener, http.NotFoundHandler()); err != nil {
		t.Fatalf("serve() error = %v, want nil", err)
	}
}

func TestServeReturnsListenerFailure(t *testing.T) {
	t.Parallel()

	listener, err := new(net.ListenConfig).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("net.Listen() error = %v", err)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("listener.Close() error = %v", err)
	}

	err = serve(context.Background(), listener, http.NotFoundHandler())
	if err == nil {
		t.Fatal("serve() error = nil, want listener failure")
	}
	if errors.Is(err, http.ErrServerClosed) || !strings.Contains(err.Error(), "serve reference HTTP") {
		t.Fatalf("serve() error = %v, want listener failure context", err)
	}
}
