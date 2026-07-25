package bootstrap

import (
	"log"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

// pprofWriteTimeout replaces the API write timeout on the diagnostics listener
// when profiling is enabled.
//
// pprof.Profile and pprof.Trace hold the response open for their ?seconds=
// argument, and the default is 30s — longer than any sane http.write_timeout for
// an API. Reusing the API timeout here silently truncates every CPU profile at
// the moment it becomes useful, so the diagnostics listener gets its own.
const pprofWriteTimeout = 65 * time.Second

// newDiagnosticsServer builds the private listener: metrics always, and the
// runtime profile handlers when they are enabled.
//
// Importing net/http/pprof also registers every profile on
// http.DefaultServeMux — that happens in the package's init and cannot be opted
// out of. What keeps that harmless is that this service never serves
// DefaultServeMux: both http.Server values are constructed with an explicit
// Handler, so the only way to reach a profile is the mux built here, gated by
// observability.pprof.enabled. Handlers are still referenced explicitly rather
// than through a blank import so that gate is visible in the code that applies
// it.
func newDiagnosticsServer(cfg config.Config, metrics *telemetry.Metrics, errorLog *log.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /metrics", metrics.Handler())

	writeTimeout := cfg.HTTP.WriteTimeout
	if cfg.Observability.Pprof.Enabled {
		registerPprof(mux)
		writeTimeout = max(writeTimeout, pprofWriteTimeout)
	}

	return &http.Server{
		Handler:           mux,
		ErrorLog:          errorLog,
		ReadHeaderTimeout: cfg.HTTP.ReadHeaderTimeout,
		ReadTimeout:       cfg.HTTP.ReadTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       cfg.HTTP.IdleTimeout,
		MaxHeaderBytes:    cfg.HTTP.MaxHeaderBytes,
	}
}

// registerPprof mounts the runtime profile handlers. Index serves the profile
// list and dispatches every named profile under /debug/pprof/, including heap,
// goroutine, allocs, and block; the other three have their own handlers.
func registerPprof(mux *http.ServeMux) {
	mux.HandleFunc("GET /debug/pprof/", pprof.Index)
	mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("POST /debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
}
