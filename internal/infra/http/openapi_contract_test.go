package httpx

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/app/ping"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func TestOpenAPIRuntimeContractEndpoints(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{})

	testCases := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "ping",
			method:     http.MethodGet,
			path:       "/api/v1/ping",
			wantStatus: http.StatusOK,
			wantBody:   "pong",
		},
		{
			name:       "health live",
			method:     http.MethodGet,
			path:       "/health/live",
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name:       "health ready",
			method:     http.MethodGet,
			path:       "/health/ready",
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name:       "metrics",
			method:     http.MethodGet,
			path:       "/metrics",
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			resp := httptest.NewRecorder()

			h.ServeHTTP(resp, req)

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Code, tc.wantStatus)
			}
			if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
				t.Fatalf("content type = %q, want prefix %q", got, "text/plain")
			}
			if tc.wantBody != "" && resp.Body.String() != tc.wantBody {
				t.Fatalf("body = %q, want %q", resp.Body.String(), tc.wantBody)
			}
		})
	}
}

func TestOpenAPIRuntimeContractReadinessUnavailable(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(failingProbe{name: "db", err: errors.New("down")}),
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
	}
	if body := resp.Body.String(); body != "not ready" {
		t.Fatalf("body = %q, want %q", body, "not ready")
	}
}

func TestOpenAPIRuntimeContractReadinessUnavailableWhenDraining(t *testing.T) {
	healthSvc := health.New()
	healthSvc.StartDrain()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := mustNewRouter(t, log, Handlers{
		Health: healthSvc,
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
	}
	if body := resp.Body.String(); body != "not ready" {
		t.Fatalf("body = %q, want %q", body, "not ready")
	}
}

func TestOpenAPIRuntimeContractReadinessUnavailableBeforeAdmission(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
		ReadinessGate: func(context.Context) error {
			return errors.New("startup admission is not ready")
		},
	}, telemetry.New(), RouterConfig{})

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
	}
	if body := resp.Body.String(); body != "not ready" {
		t.Fatalf("body = %q, want %q", body, "not ready")
	}
}

func TestOpenAPIRuntimeContractWrongHealthcheckPathRejected(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{})

	// Deployment admission must fail deterministically when an unknown health path is used.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
	}
	if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("content type = %q, want prefix %q", got, "application/problem+json")
	}
}

func TestOpenAPIRuntimeContractRequiresRouterDependencies(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	testCases := []struct {
		name     string
		log      *slog.Logger
		handlers Handlers
		metrics  *telemetry.Metrics
		cfg      RouterConfig
		wantErr  string
	}{
		{
			name:     "missing logger",
			handlers: Handlers{Health: health.New(), Ping: ping.New(), ReadinessGate: func(context.Context) error { return nil }},
			metrics:  telemetry.New(),
			cfg:      RouterConfig{ReadinessTimeout: time.Second},
			wantErr:  "logger is required",
		},
		{
			name:    "missing health",
			log:     log,
			metrics: telemetry.New(),
			cfg:     RouterConfig{ReadinessTimeout: time.Second},
			handlers: Handlers{
				Ping:          ping.New(),
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "health service is required",
		},
		{
			name:    "missing ping",
			log:     log,
			metrics: telemetry.New(),
			cfg:     RouterConfig{ReadinessTimeout: time.Second},
			handlers: Handlers{
				Health:        health.New(),
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "ping service is required",
		},
		{
			name:    "missing readiness gate",
			log:     log,
			metrics: telemetry.New(),
			cfg:     RouterConfig{ReadinessTimeout: time.Second},
			handlers: Handlers{
				Health: health.New(),
				Ping:   ping.New(),
			},
			wantErr: "readiness gate is required",
		},
		{
			name: "missing metrics",
			log:  log,
			cfg:  RouterConfig{ReadinessTimeout: time.Second},
			handlers: Handlers{
				Health:        health.New(),
				Ping:          ping.New(),
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "metrics is required",
		},
		{
			name:    "missing readiness timeout",
			log:     log,
			metrics: telemetry.New(),
			handlers: Handlers{
				Health:        health.New(),
				Ping:          ping.New(),
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "readiness timeout must be > 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler, err := NewRouter(tc.log, tc.handlers, tc.metrics, tc.cfg)
			if err == nil {
				t.Fatalf("NewRouter() error = nil, want %q", tc.wantErr)
			}
			if handler != nil {
				t.Fatalf("NewRouter() handler = %T, want nil on error", handler)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("NewRouter() error = %v, want to contain %q", err, tc.wantErr)
			}
		})
	}
}

type failingProbe struct {
	name string
	err  error
}

var _ health.Probe = (*failingProbe)(nil)

func (p failingProbe) Name() string {
	return p.name
}

func (p failingProbe) Check(_ context.Context) error {
	return p.err
}
