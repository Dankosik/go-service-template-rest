package httpx

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/app/ping"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func TestRouterEndpoints(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewRouter(log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{})

	t.Run("ping", func(t *testing.T) {
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

	t.Run("live", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
		}
	})

	t.Run("ready", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
		}
	})

	t.Run("metrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
		}
	})
}

func TestRouterAddsRequestIDHeader(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewRouter(log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{})

	t.Run("generates request id when header is absent", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if got := resp.Header().Get(requestIDHeader); got == "" {
			t.Fatalf("%s header is empty", requestIDHeader)
		}
	})

	t.Run("echoes inbound request id", func(t *testing.T) {
		const wantRequestID = "demo-123"

		req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
		req.Header.Set(requestIDHeader, wantRequestID)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if got := resp.Header().Get(requestIDHeader); got != wantRequestID {
			t.Fatalf("%s = %q, want %q", requestIDHeader, got, wantRequestID)
		}
	})
}

func TestRouterAddsSecurityHeaders(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewRouter(log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	if got := resp.Header().Get(contentTypeOptionsHeader); got != "nosniff" {
		t.Fatalf("%s = %q, want %q", contentTypeOptionsHeader, got, "nosniff")
	}
}

func TestRouterRejectsConflictingRequestFraming(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewRouter(log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.Header.Set("Transfer-Encoding", "chunked")
	req.Header.Set("Content-Length", "1")
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
	if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("content type = %q, want prefix %q", got, "application/problem+json")
	}
	if !strings.Contains(resp.Body.String(), "invalid request framing") {
		t.Fatalf("body = %q, want %q", resp.Body.String(), "invalid request framing")
	}
	if got := resp.Header().Get(contentTypeOptionsHeader); got != "nosniff" {
		t.Fatalf("%s = %q, want %q", contentTypeOptionsHeader, got, "nosniff")
	}
	if got := resp.Header().Get(requestIDHeader); got == "" {
		t.Fatalf("%s header is empty", requestIDHeader)
	}
}

func TestRouterRejectsRequestBodyTooLarge(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewRouter(log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{MaxBodyBytes: 1})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", strings.NewReader("ab"))
	req.ContentLength = 2
	resp := httptest.NewRecorder()

	h.ServeHTTP(resp, req)

	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusRequestEntityTooLarge)
	}
	if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("content type = %q, want prefix %q", got, "application/problem+json")
	}
	if !strings.Contains(resp.Body.String(), "request body exceeds limit") {
		t.Fatalf("body = %q, want %q", resp.Body.String(), "request body exceeds limit")
	}
}

func TestAccessLogIncludesCorrelationFields(t *testing.T) {
	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	h := NewRouter(log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
	}, nil, RouterConfig{})

	const (
		requestID = "demo-123"
		traceID   = "4bf92f3577b34da6a3ce929d0e0e4736"
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.Header.Set(requestIDHeader, requestID)
	req.Header.Set("traceparent", "00-"+traceID+"-00f067aa0ba902b7-01")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) == 0 || lines[0] == "" {
		t.Fatalf("expected access log line")
	}

	var event map[string]any
	if err := json.Unmarshal([]byte(lines[len(lines)-1]), &event); err != nil {
		t.Fatalf("unmarshal access log: %v", err)
	}

	if got := event["request_id"]; got != requestID {
		t.Fatalf("request_id = %v, want %q", got, requestID)
	}
	if got := event["trace_id"]; got != traceID {
		t.Fatalf("trace_id = %v, want %q", got, traceID)
	}
	if got, ok := event["span_id"].(string); !ok || got == "" {
		t.Fatalf("span_id = %v, want non-empty string", event["span_id"])
	}
}

func TestMetricsExposeDurationHistogram(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewRouter(log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{})

	pingReq := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	pingResp := httptest.NewRecorder()
	h.ServeHTTP(pingResp, pingReq)

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsResp := httptest.NewRecorder()
	h.ServeHTTP(metricsResp, metricsReq)

	if metricsResp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", metricsResp.Code, http.StatusOK)
	}

	body := metricsResp.Body.String()
	if !strings.Contains(body, "http_request_duration_seconds_bucket") {
		t.Fatalf("metrics output does not contain duration histogram buckets")
	}
	if !strings.Contains(body, `method="GET",route="GET /api/v1/ping",status_code="200"`) {
		t.Fatalf("metrics output does not contain expected duration histogram labels for ping endpoint")
	}
}
