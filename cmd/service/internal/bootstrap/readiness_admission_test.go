package bootstrap

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/app/ping"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
)

type failingProbe struct {
	name string
	err  error
}

func (p failingProbe) Name() string {
	return p.name
}

func (p failingProbe) Check(context.Context) error {
	return p.err
}

func TestAdmissionSuccessRecordedOnReadiness(t *testing.T) {
	recorder, metrics, _ := newBufferedDeployTelemetryRecorder("test")
	log := slog.New(slog.NewJSONHandler(io.Discard, nil))

	var once sync.Once
	handler := httpx.NewRouter(
		log,
		httpx.Handlers{
			Health: health.New(),
			Ping:   ping.New(),
			OnReadySuccess: func(ctx context.Context) error {
				once.Do(func() {
					recorder.RecordAdmission(ctx, "success", "ready", "readiness")
				})
				return nil
			},
		},
		metrics,
		httpx.RouterConfig{},
	)

	liveReq := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	liveResp := httptest.NewRecorder()
	handler.ServeHTTP(liveResp, liveReq)
	if liveResp.Code != http.StatusOK {
		t.Fatalf("GET /health/live status = %d, want %d", liveResp.Code, http.StatusOK)
	}

	metricsText := collectServiceMetricsText(t, metrics)
	if strings.Contains(metricsText, `deploy_health_admission_total`) {
		t.Fatalf("metrics unexpectedly contain admission success before readiness:\n%s", metricsText)
	}

	readyReq := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	readyResp := httptest.NewRecorder()
	handler.ServeHTTP(readyResp, readyReq)
	if readyResp.Code != http.StatusOK {
		t.Fatalf("GET /health/ready status = %d, want %d", readyResp.Code, http.StatusOK)
	}

	metricsText = collectServiceMetricsText(t, metrics)
	if !strings.Contains(metricsText, `deploy_health_admission_total{environment="test",reason_class="ready",result="success"} 1`) {
		t.Fatalf("metrics do not contain readiness-driven admission success:\n%s", metricsText)
	}

	secondReadyResp := httptest.NewRecorder()
	handler.ServeHTTP(secondReadyResp, readyReq)
	metricsText = collectServiceMetricsText(t, metrics)
	if strings.Contains(metricsText, `deploy_health_admission_total{environment="test",reason_class="ready",result="success"} 2`) {
		t.Fatalf("metrics contain duplicate readiness admissions:\n%s", metricsText)
	}
}

func TestAdmissionSuccessNotRecordedWhenReadinessFails(t *testing.T) {
	t.Run("before ready hook failure", func(t *testing.T) {
		recorder, metrics, _ := newBufferedDeployTelemetryRecorder("test")
		log := slog.New(slog.NewJSONHandler(io.Discard, nil))
		var onReadyCalled atomic.Bool

		handler := httpx.NewRouter(
			log,
			httpx.Handlers{
				Health: health.New(),
				Ping:   ping.New(),
				BeforeReady: func(context.Context) error {
					return context.Canceled
				},
				OnReadySuccess: func(ctx context.Context) error {
					onReadyCalled.Store(true)
					recorder.RecordAdmission(ctx, "success", "ready", "readiness")
					return nil
				},
			},
			metrics,
			httpx.RouterConfig{},
		)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf("GET /health/ready status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
		}
		if onReadyCalled.Load() {
			t.Fatal("OnReadySuccess called on before-ready failure")
		}
		metricsText := collectServiceMetricsText(t, metrics)
		if strings.Contains(metricsText, `deploy_health_admission_total{environment="test",reason_class="ready",result="success"} 1`) {
			t.Fatalf("metrics unexpectedly contain readiness admission success:\n%s", metricsText)
		}
	})

	t.Run("health readiness probe failure", func(t *testing.T) {
		recorder, metrics, _ := newBufferedDeployTelemetryRecorder("test")
		log := slog.New(slog.NewJSONHandler(io.Discard, nil))
		var onReadyCalled atomic.Bool

		handler := httpx.NewRouter(
			log,
			httpx.Handlers{
				Health: health.New(failingProbe{name: "db", err: errors.New("down")}),
				Ping:   ping.New(),
				OnReadySuccess: func(ctx context.Context) error {
					onReadyCalled.Store(true)
					recorder.RecordAdmission(ctx, "success", "ready", "readiness")
					return nil
				},
			},
			metrics,
			httpx.RouterConfig{},
		)

		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)

		if resp.Code != http.StatusServiceUnavailable {
			t.Fatalf("GET /health/ready status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
		}
		if onReadyCalled.Load() {
			t.Fatal("OnReadySuccess called on health probe failure")
		}
		metricsText := collectServiceMetricsText(t, metrics)
		if strings.Contains(metricsText, `deploy_health_admission_total{environment="test",reason_class="ready",result="success"} 1`) {
			t.Fatalf("metrics unexpectedly contain readiness admission success:\n%s", metricsText)
		}
	})
}
