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

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/app/ping"
	"github.com/example/go-service-template-rest/internal/domain"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func TestOpenAPIRuntimeContractEndpoints(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewRouter(log, Handlers{
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
	h := NewRouter(log, Handlers{
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

func TestOpenAPIRuntimeContractFallbackServices(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewRouter(log, Handlers{}, nil, RouterConfig{})

	t.Run("ping fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
		}
		if body := resp.Body.String(); body != "pong" {
			t.Fatalf("body = %q, want %q", body, "pong")
		}
	})

	t.Run("metrics fallback", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
		}
		if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
			t.Fatalf("content type = %q, want prefix %q", got, "text/plain")
		}
	})
}

type failingProbe struct {
	name string
	err  error
}

var _ domain.ReadinessProbe = (*failingProbe)(nil)

func (p failingProbe) Name() string {
	return p.name
}

func (p failingProbe) Check(_ context.Context) error {
	return p.err
}
