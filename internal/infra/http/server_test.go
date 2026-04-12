package httpx

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
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

func TestNewServerUsesNotFoundHandlerWhenHandlerNil(t *testing.T) {
	t.Parallel()

	srv := New(Config{}, nil)
	if srv == nil || srv.srv == nil || srv.srv.Handler == nil {
		t.Fatal("New(Config{}, nil) did not install a handler")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	srv.srv.Handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
	}
}

func TestServerServeAndShutdown(t *testing.T) {
	t.Parallel()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()

	srv := New(Config{Addr: addr}, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- srv.Serve(listener)
	}()

	client := http.Client{Timeout: time.Second}
	req, err := http.NewRequest(http.MethodGet, "http://"+addr, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("client.Do() error = %v, want nil", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
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

func TestServerUninitializedUseReturnsInspectableError(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		srv  *Server
	}{
		{name: "nil receiver"},
		{name: "zero value", srv: &Server{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.srv.Run(); !errors.Is(err, ErrUninitializedServer) {
				t.Fatalf("Run() error = %v, want ErrUninitializedServer", err)
			}
			if err := tc.srv.Serve(nil); !errors.Is(err, ErrUninitializedServer) {
				t.Fatalf("Serve(nil) error = %v, want ErrUninitializedServer", err)
			}
			if err := tc.srv.Shutdown(context.Background()); !errors.Is(err, ErrUninitializedServer) {
				t.Fatalf("Shutdown() error = %v, want ErrUninitializedServer", err)
			}
		})
	}
}
