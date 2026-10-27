package runtimeopts_test

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/waittest"
)

// TestListenDiagnosticsRefusesAnOccupiedAddress covers why the bind happens
// before the serving goroutine starts: an address already in use has to fail
// startup, rather than surfacing later as a diagnostics server that stopped for
// no stated reason.
func TestListenDiagnosticsRefusesAnOccupiedAddress(t *testing.T) {
	t.Parallel()

	occupied, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("occupy a diagnostics address: %v", err)
	}
	defer func() { _ = occupied.Close() }()

	served, err := runtimeopts.ListenDiagnostics(
		t.Context(), occupied.Addr().String(), "probe", func() bool { return true }, telemetry.NewMetrics(), false,
	)
	if err == nil {
		_ = served.Stop(t.Context(), time.Second)
		t.Fatal("ListenDiagnostics(occupied address) error = nil")
	}
	// The component is in the message because an operator matching on it should
	// not have to know which shared helper produced it.
	if !strings.Contains(err.Error(), "listen for probe diagnostics") {
		t.Fatalf("ListenDiagnostics(occupied address) error = %v, want it to name the component", err)
	}
}

// TestDiagnosticsListenerStopsAndJoins holds the two halves of the ownership
// this type exists for: Stopped stays open while the server is serving, and Stop
// both ends the server and joins its goroutine, so a caller that only selects on
// Stopped never has to reason about the serve error itself.
func TestDiagnosticsListenerStopsAndJoins(t *testing.T) {
	t.Parallel()

	served, err := runtimeopts.ListenDiagnostics(
		t.Context(), "127.0.0.1:0", "probe", func() bool { return true }, telemetry.NewMetrics(), false,
	)
	if err != nil {
		t.Fatalf("ListenDiagnostics() error = %v", err)
	}
	select {
	case <-served.Stopped():
		t.Fatal("diagnostics reported itself stopped while it was serving")
	default:
	}

	if err := served.Stop(context.Background(), time.Second); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	select {
	case <-served.Stopped():
	default:
		t.Fatal("Stop() returned before the serving goroutine had been joined")
	}
}

// TestDiagnosticsListenerStopForcesLocalTimeoutAndJoins covers a scrape that
// outlives the stop budget: Stop drops it, joins the server, and reports the
// expired local stage as success because the caller's own budget is intact.
func TestDiagnosticsListenerStopForcesLocalTimeoutAndJoins(t *testing.T) {
	t.Parallel()

	started := make(chan struct{})
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })
	address := waittest.FreeTCPAddr(t, "probe diagnostics")
	served, err := runtimeopts.ListenDiagnostics(t.Context(), address, "probe", func() bool {
		close(started)
		<-release
		return true
	}, telemetry.NewMetrics(), false)
	if err != nil {
		t.Fatalf("ListenDiagnostics() error = %v", err)
	}

	requestDone := make(chan error, 1)
	go func() {
		_, requestErr := diagnosticsStatus(t.Context(), address, "/health/ready")
		requestDone <- requestErr
	}()
	waittest.ReceiveSignal(t, started, 5*time.Second, "the readiness request to reach its handler")

	if err := served.Stop(context.Background(), time.Millisecond); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	select {
	case <-served.Stopped():
	default:
		t.Fatal("Stop() returned before the serving goroutine had been joined")
	}
	if err := waittest.Receive(t, requestDone, 5*time.Second, "the readiness request to end"); err == nil {
		t.Fatal("forced connection close returned no client error")
	}
}

func TestListenDiagnosticsServesPprofOnlyWhenEnabled(t *testing.T) {
	t.Parallel()

	disabled := listenDiagnostics(t, false)
	status, err := diagnosticsStatus(t.Context(), disabled, "/debug/pprof/")
	if err != nil {
		t.Fatalf("GET /debug/pprof/ with pprof disabled: %v", err)
	}
	if status != http.StatusNotFound {
		t.Fatalf("GET /debug/pprof/ with pprof disabled = %d, want 404", status)
	}

	enabled := listenDiagnostics(t, true)
	status, err = diagnosticsStatus(t.Context(), enabled, "/debug/pprof/")
	if err != nil {
		t.Fatalf("GET /debug/pprof/ with pprof enabled: %v", err)
	}
	if status != http.StatusOK {
		t.Fatalf("GET /debug/pprof/ with pprof enabled = %d, want 200", status)
	}
}

// listenDiagnostics serves a ready diagnostics listener for the rest of the
// test and returns its address.
func listenDiagnostics(t *testing.T, pprofEnabled bool) string {
	t.Helper()

	address := waittest.FreeTCPAddr(t, "probe diagnostics")
	served, err := runtimeopts.ListenDiagnostics(
		t.Context(), address, "probe", func() bool { return true }, telemetry.NewMetrics(), pprofEnabled,
	)
	if err != nil {
		t.Fatalf("ListenDiagnostics() error = %v", err)
	}
	t.Cleanup(func() { _ = served.Stop(context.Background(), time.Second) })
	return address
}

func diagnosticsStatus(ctx context.Context, address, path string) (int, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+address+path, http.NoBody)
	if err != nil {
		return 0, fmt.Errorf("build diagnostics request: %w", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("get diagnostics: %w", err)
	}
	_ = response.Body.Close()
	return response.StatusCode, nil
}

// TestRegisterPprofHandlersServesGoroutineLeak proves the repository-owned
// route. The standard library owns the profile algorithm, so the test does not
// create an intentional leak in this shared test process.
func TestRegisterPprofHandlersServesGoroutineLeak(t *testing.T) {
	mux := http.NewServeMux()
	runtimeopts.RegisterPprofHandlers(mux)
	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/debug/pprof/goroutineleak", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("/debug/pprof/goroutineleak status = %d, want %d", response.Code, http.StatusOK)
	}
	if response.Body.Len() == 0 {
		t.Fatal("/debug/pprof/goroutineleak returned an empty body")
	}
}
