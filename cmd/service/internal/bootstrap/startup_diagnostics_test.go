package bootstrap

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/config"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func diagnosticsConfig(pprofEnabled bool) config.Config {
	return config.Config{
		HTTP: config.HTTPConfig{
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       5 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			MaxHeaderBytes:    16 << 10,
		},
		Observability: config.ObservabilityConfig{
			Metrics: config.MetricsConfig{Addr: "127.0.0.1:0"},
			Pprof:   config.PprofConfig{Enabled: pprofEnabled},
		},
	}
}

func TestDiagnosticsServerAlwaysServesMetrics(t *testing.T) {
	t.Parallel()

	srv := newDiagnosticsServer(diagnosticsConfig(false), telemetry.New(), nil)

	resp := serveDiagnostics(srv, http.MethodGet, "/metrics")
	if resp.Code != http.StatusOK {
		t.Fatalf("/metrics status = %d, want %d", resp.Code, http.StatusOK)
	}
}

// TestDiagnosticsServerWithdrawsPprofWhenDisabled is the default posture: the
// profile handlers disclose heap contents and process arguments, and the
// diagnostics listener binds every interface so a scraper can reach it.
func TestDiagnosticsServerWithdrawsPprofWhenDisabled(t *testing.T) {
	t.Parallel()

	srv := newDiagnosticsServer(diagnosticsConfig(false), telemetry.New(), nil)

	for _, path := range []string{"/debug/pprof/", "/debug/pprof/heap", "/debug/pprof/cmdline"} {
		resp := serveDiagnostics(srv, http.MethodGet, path)
		if resp.Code != http.StatusNotFound {
			t.Fatalf("%s status = %d, want %d when pprof is disabled", path, resp.Code, http.StatusNotFound)
		}
	}
}

func TestDiagnosticsServerServesPprofWhenEnabled(t *testing.T) {
	t.Parallel()

	srv := newDiagnosticsServer(diagnosticsConfig(true), telemetry.New(), nil)

	// heap is routed by pprof.Index rather than by its own pattern, so it proves
	// the prefix mount and not just one explicit handler.
	for _, path := range []string{"/debug/pprof/", "/debug/pprof/heap", "/debug/pprof/goroutine"} {
		resp := serveDiagnostics(srv, http.MethodGet, path)
		if resp.Code != http.StatusOK {
			t.Fatalf("%s status = %d, want %d when pprof is enabled", path, resp.Code, http.StatusOK)
		}
		if resp.Body.Len() == 0 {
			t.Fatalf("%s returned an empty body", path)
		}
	}
}

// TestDiagnosticsServerWidensWriteTimeoutForProfiles keeps a 30s CPU profile from
// being truncated by an API-shaped write deadline, which would break profiling at
// exactly the moment it is wanted.
func TestDiagnosticsServerWidensWriteTimeoutForProfiles(t *testing.T) {
	t.Parallel()

	cfg := diagnosticsConfig(true)
	srv := newDiagnosticsServer(cfg, telemetry.New(), nil)

	if srv.WriteTimeout <= cfg.HTTP.WriteTimeout {
		t.Fatalf("WriteTimeout = %s, want more than the API write timeout %s", srv.WriteTimeout, cfg.HTTP.WriteTimeout)
	}
	if srv.WriteTimeout < pprofWriteTimeout {
		t.Fatalf("WriteTimeout = %s, want at least %s", srv.WriteTimeout, pprofWriteTimeout)
	}
}

// TestDiagnosticsServerKeepsAPIWriteTimeoutWithoutPprof keeps the widening scoped
// to the reason for it.
func TestDiagnosticsServerKeepsAPIWriteTimeoutWithoutPprof(t *testing.T) {
	t.Parallel()

	cfg := diagnosticsConfig(false)
	srv := newDiagnosticsServer(cfg, telemetry.New(), nil)

	if srv.WriteTimeout != cfg.HTTP.WriteTimeout {
		t.Fatalf("WriteTimeout = %s, want the API write timeout %s", srv.WriteTimeout, cfg.HTTP.WriteTimeout)
	}
}

// TestDiagnosticsServerNeverServesDefaultServeMux is the guard that actually
// matters.
//
// Importing net/http/pprof registers every profile on http.DefaultServeMux from
// the package's init, and nothing can prevent that. So the protection is not
// "pprof is not on DefaultServeMux" — it demonstrably is, and this test asserts
// so, to keep the reasoning honest. The protection is that this service never
// serves that mux: an http.Server with a nil Handler falls back to it, so both
// servers must always carry an explicit one.
func TestDiagnosticsServerNeverServesDefaultServeMux(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/debug/pprof/", nil)
	resp := httptest.NewRecorder()
	http.DefaultServeMux.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("DefaultServeMux /debug/pprof/ status = %d; the import is expected to register there", resp.Code)
	}

	for _, pprofEnabled := range []bool{false, true} {
		srv := newDiagnosticsServer(diagnosticsConfig(pprofEnabled), telemetry.New(), nil)
		if srv.Handler == nil {
			t.Fatalf("pprof enabled = %t: diagnostics Handler is nil, which makes net/http fall back to DefaultServeMux", pprofEnabled)
		}
		if srv.Handler == http.Handler(http.DefaultServeMux) {
			t.Fatalf("pprof enabled = %t: diagnostics server serves DefaultServeMux", pprofEnabled)
		}
	}
}

func serveDiagnostics(srv *http.Server, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	resp := httptest.NewRecorder()
	srv.Handler.ServeHTTP(resp, req)
	return resp
}
