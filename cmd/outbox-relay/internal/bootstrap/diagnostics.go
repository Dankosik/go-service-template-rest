package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// diagnostics is the relay's liveness, readiness, and metrics endpoint. It owns
// its listener, its serving goroutine, and the join, so the relay lifecycle
// only starts it, watches one channel, and stops it. Keeping the HTTP mechanics
// here is what lets runRelayLifecycle read as relay lifecycle.
type diagnostics struct {
	server *http.Server
	// done closes once Serve has returned and serveErr is written, so serveErr
	// needs no lock: the close is the only publication of it.
	done     chan struct{}
	serveErr error
}

// startDiagnostics binds the address and begins serving. Binding happens before
// the goroutine starts, so an address already in use fails startup instead of
// surfacing later as a diagnostics server that stopped for no stated reason.
func startDiagnostics(
	ctx context.Context,
	addr string,
	ready func() bool,
	metrics *telemetry.Metrics,
) (*diagnostics, error) {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen for outbox diagnostics: %w", err)
	}
	served := &diagnostics{server: newDiagnosticsServer(ready, metrics), done: make(chan struct{})}
	go func() {
		defer close(served.done)
		if err := served.server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			served.serveErr = err
		}
	}()
	return served, nil
}

// watch reports a diagnostics server that returned on its own, which is a
// process-level fault rather than a shutdown: the relay can still publish, but
// nothing can scrape it or read its readiness.
func (d *diagnostics) watch() <-chan struct{} {
	return d.done
}

// stop drains in-flight scrapes until the budget expires, then drops whatever
// is still open, so Serve always returns and the join cannot outlast the
// budget. The returned error carries whatever Serve reported, so a caller that
// selected on watch only needs to say that it fired.
func (d *diagnostics) stop(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, diagnosticsClose)
	defer cancel()
	err := errors.Join(d.server.Shutdown(ctx), d.server.Close())
	select {
	case <-d.done:
		return errors.Join(err, d.serveErr)
	case <-ctx.Done():
		return errors.Join(err, fmt.Errorf("join outbox diagnostics: %w", ctx.Err()))
	}
}

// newDiagnosticsServer builds the handler set. It carries no Addr, because the
// only caller serves it on a listener startDiagnostics already bound, and
// http.Server.Addr is ignored on that path.
func newDiagnosticsServer(ready func() bool, metrics *telemetry.Metrics) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /health/ready", func(writer http.ResponseWriter, _ *http.Request) {
		if !ready() {
			http.Error(writer, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		writer.WriteHeader(http.StatusOK)
	})
	mux.Handle("GET /metrics", metrics.Handler())
	return &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
}
