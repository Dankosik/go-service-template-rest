package runtimeopts

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// diagnosticsReadHeaderTimeout bounds how long a scraper may take to send its
// request line and headers. It is the only timeout this listener sets: a scrape
// is one small request, and a write bound would truncate a slow collector rather
// than protect anything.
const diagnosticsReadHeaderTimeout = 5 * time.Second

// DiagnosticsServer builds the private listener a background binary serves:
// process liveness, readiness, and the metrics scrape.
//
// It carries no Addr, because both callers serve it on a listener they bound
// themselves and http.Server.Addr is ignored on that path. Binding separately is
// what makes an address already in use a startup failure rather than a
// diagnostics server that stopped later for no stated reason.
//
// ready is the whole readiness verdict, so a binary with more than one condition
// combines them at its own call site. That keeps this handler from having to
// know which conditions exist, and keeps this file from naming binaries a build
// profile may have removed.
//
// cmd/service does not use this. Its diagnostics listener additionally serves
// build identity and the gated runtime profiles, and takes its timeouts from
// configuration; see newDiagnosticsServer in cmd/service/internal/bootstrap.
func DiagnosticsServer(ready func() bool, metrics *telemetry.Metrics) *http.Server {
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
	return &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: diagnosticsReadHeaderTimeout,
	}
}

// DiagnosticsListener is a bound and serving [DiagnosticsServer]. It owns its
// listener, its serving goroutine, and the join, so a composition root only
// starts it, watches one channel, and stops it — which is what lets a lifecycle
// function read as lifecycle instead of as HTTP mechanics.
//
// component prefixes its two failures, for the reason [InstallTelemetry] takes
// one: the binaries already publish their own names, and an operator matching on
// "listen for worker diagnostics" should not have to know which shared helper
// produced it.
type DiagnosticsListener struct {
	server    *http.Server
	component string
	// done closes once Serve has returned and serveErr is written, so serveErr
	// needs no lock: the close is the only publication of it.
	done     chan struct{}
	serveErr error
}

// ListenDiagnostics binds the address and begins serving. Binding happens before
// the goroutine starts, so an address already in use fails startup instead of
// surfacing later as a diagnostics server that stopped for no stated reason.
func ListenDiagnostics(
	ctx context.Context,
	addr string,
	component string,
	ready func() bool,
	metrics *telemetry.Metrics,
) (*DiagnosticsListener, error) {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen for %s diagnostics: %w", component, err)
	}
	served := &DiagnosticsListener{
		server:    DiagnosticsServer(ready, metrics),
		component: component,
		done:      make(chan struct{}),
	}
	go func() {
		defer close(served.done)
		if err := served.server.Serve(listener); !errors.Is(err, http.ErrServerClosed) {
			served.serveErr = err
		}
	}()
	return served, nil
}

// Stopped reports a diagnostics server that returned on its own, which is a
// process-level fault rather than a shutdown: the binary can still do its work,
// but nothing can scrape it or read its readiness.
func (d *DiagnosticsListener) Stopped() <-chan struct{} {
	return d.done
}

// Stop drains in-flight scrapes until limit expires, then drops whatever is
// still open, so Serve always returns and the join cannot outlast the budget.
// A local stage timeout is a degraded success once Close has forced the listener
// down; a spent parent budget and Close or Serve failures remain errors.
//
// parent is the caller's own teardown context, and its deadline still binds:
// limit is this stage's ceiling, not a fresh budget.
func (d *DiagnosticsListener) Stop(parent context.Context, limit time.Duration) error {
	ctx, cancel := context.WithTimeout(parent, limit)
	defer cancel()
	shutdownErr := d.server.Shutdown(ctx)
	closeErr := d.server.Close()
	<-d.done
	if parent.Err() == nil && errors.Is(shutdownErr, context.DeadlineExceeded) {
		shutdownErr = nil
	}
	return errors.Join(shutdownErr, closeErr, d.serveErr)
}
