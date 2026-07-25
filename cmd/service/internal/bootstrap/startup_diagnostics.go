package bootstrap

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/pprof"
	"runtime"
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
	mux.Handle("GET /debug/buildinfo", buildInfoHandler(cfg))

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

// buildInfo answers "which build is this process running".
//
// Nothing else in the process could. The image build stamps the revision into an
// OCI label, which is unreachable from inside the container and from any
// dashboard, and it builds with -buildvcs=false because .dockerignore keeps .git
// out of the context — so runtime/debug.ReadBuildInfo carries no revision either.
// The version alone does not answer it: a tag or a branch name moves, and during
// a rollout the question is which of two running pods is which.
type buildInfo struct {
	Service   string `json:"service"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	Env       string `json:"env"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// buildInfoHandler serves the build identity on the private listener.
//
// It is not gated behind observability.pprof.enabled. A build identifier is not
// the kind of disclosure a heap dump is, and the moment an operator needs it most
// is exactly the moment profiling is off — which is the shipped default.
//
// The payload is rendered once, so the route does no work per request and has no
// failure path while it is being scraped or curled during an incident.
func buildInfoHandler(cfg config.Config) http.Handler {
	payload, err := json.Marshal(buildInfo{
		Service:   cfg.Observability.OTel.ServiceName,
		Version:   cfg.App.Version,
		Commit:    cfg.App.Commit,
		Env:       cfg.App.Env,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	})
	if err != nil {
		// A struct of strings cannot fail to marshal. Answering honestly still
		// beats panicking on the listener reached for when things are already bad.
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, "build info unavailable", http.StatusInternalServerError)
		})
	}

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write(payload)
	})
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
