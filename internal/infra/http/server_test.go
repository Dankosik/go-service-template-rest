package httpx

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func TestNewServerAppliesConfig(t *testing.T) {
	t.Parallel()

	cfg := Config{
		Addr:              "127.0.0.1:0",
		ReadHeaderTimeout: 1500 * time.Millisecond,
		ReadTimeout:       2 * time.Second,
		WriteTimeout:      3 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    32 << 10,
	}

	srv := New(cfg, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	if srv == nil {
		t.Fatal("New() returned nil")
	}
	if srv.srv == nil {
		t.Fatal("server.srv is nil")
	}
	if srv.srv.Addr != cfg.Addr {
		t.Fatalf("Addr = %q, want %q", srv.srv.Addr, cfg.Addr)
	}
	if srv.srv.ReadHeaderTimeout != cfg.ReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %s, want %s", srv.srv.ReadHeaderTimeout, cfg.ReadHeaderTimeout)
	}
	if srv.srv.ReadTimeout != cfg.ReadTimeout {
		t.Fatalf("ReadTimeout = %s, want %s", srv.srv.ReadTimeout, cfg.ReadTimeout)
	}
	if srv.srv.WriteTimeout != cfg.WriteTimeout {
		t.Fatalf("WriteTimeout = %s, want %s", srv.srv.WriteTimeout, cfg.WriteTimeout)
	}
	if srv.srv.IdleTimeout != cfg.IdleTimeout {
		t.Fatalf("IdleTimeout = %s, want %s", srv.srv.IdleTimeout, cfg.IdleTimeout)
	}
	if srv.srv.MaxHeaderBytes != cfg.MaxHeaderBytes {
		t.Fatalf("MaxHeaderBytes = %d, want %d", srv.srv.MaxHeaderBytes, cfg.MaxHeaderBytes)
	}
}

func TestServerRunAndShutdown(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	srv := New(Config{Addr: addr}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- srv.Run()
	}()

	deadline := time.Now().Add(2 * time.Second)
	for {
		req, err := http.NewRequest(http.MethodGet, "http://"+addr, nil)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("server did not start in time: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}

	select {
	case err := <-runErrCh:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run() did not return after Shutdown()")
	}
}

func TestServerRunInvalidAddress(t *testing.T) {
	t.Parallel()

	srv := New(Config{Addr: "127.0.0.1:not-a-port"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	if err := srv.Run(); err == nil {
		t.Fatal("Run() error = nil, want non-nil")
	}
}

func TestServerShutdownBeforeRun(t *testing.T) {
	t.Parallel()

	srv := New(Config{Addr: "127.0.0.1:0"}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "ok")
	}))
	if err := srv.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown() error = %v, want nil", err)
	}
}

func TestServerServeNilListener(t *testing.T) {
	t.Parallel()

	srv := New(Config{Addr: "127.0.0.1:0"}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	err := srv.Serve(nil)
	if err == nil {
		t.Fatal("Serve(nil) error = nil, want non-nil")
	}
	if !errors.Is(err, ErrNilListener) {
		t.Fatalf("Serve(nil) error = %v, want ErrNilListener", err)
	}
}
