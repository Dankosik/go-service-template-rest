package runtimeopts_test

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/cmd/internal/runtimeopts"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
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
		t.Context(), occupied.Addr().String(), "probe", func() bool { return true }, telemetry.New(), false,
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
		t.Context(), "127.0.0.1:0", "probe", func() bool { return true }, telemetry.New(), false,
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

func TestDiagnosticsServerGatesPprof(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name         string
		pprofEnabled bool
		wantStatus   int
	}{
		{name: "disabled", wantStatus: http.StatusNotFound},
		{name: "enabled", pprofEnabled: true, wantStatus: http.StatusOK},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			server := runtimeopts.DiagnosticsServer(func() bool { return true }, telemetry.New(), test.pprofEnabled)
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/debug/pprof/", nil)
			response := httptest.NewRecorder()
			server.Handler.ServeHTTP(response, request)
			if response.Code != test.wantStatus {
				t.Fatalf("/debug/pprof/ status = %d, want %d", response.Code, test.wantStatus)
			}
		})
	}
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
