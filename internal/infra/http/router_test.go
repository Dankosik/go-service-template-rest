package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Dankosik/search-service/internal/app/health"
	"github.com/Dankosik/search-service/internal/app/ping"
	"github.com/Dankosik/search-service/internal/infra/telemetry"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
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

func TestRouterHTTPPolicy(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewRouter(log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{})

	t.Run("not found uses problem envelope", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
		}
		if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
			t.Fatalf("content type = %q, want prefix %q", got, "application/problem+json")
		}
		var problem map[string]any
		if err := json.Unmarshal(resp.Body.Bytes(), &problem); err != nil {
			t.Fatalf("unmarshal problem: %v", err)
		}
		if got := problem["title"]; got != "not found" {
			t.Fatalf("title = %v, want %q", got, "not found")
		}
		if got := int(problem["status"].(float64)); got != http.StatusNotFound {
			t.Fatalf("problem status = %d, want %d", got, http.StatusNotFound)
		}
	})

	t.Run("method not allowed uses problem envelope and allow header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/ping", nil)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusMethodNotAllowed)
		}
		if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
			t.Fatalf("content type = %q, want prefix %q", got, "application/problem+json")
		}
		allowMethods := resp.Header().Values("Allow")
		if !containsString(allowMethods, http.MethodGet) {
			t.Fatalf("allow header = %v, want to contain %q", allowMethods, http.MethodGet)
		}
	})

	t.Run("method not allowed allow header includes trace when route exists", func(t *testing.T) {
		apiSubrouter := chi.NewRouter()
		apiSubrouter.Trace("/trace-only", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		rootRouter := newRootRouter(apiSubrouter, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodPost, "/trace-only", nil)
		resp := httptest.NewRecorder()

		rootRouter.ServeHTTP(resp, req)

		if resp.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusMethodNotAllowed)
		}
		allowMethods := resp.Header().Values("Allow")
		if !containsString(allowMethods, http.MethodTrace) {
			t.Fatalf("allow header = %v, want to contain %q", allowMethods, http.MethodTrace)
		}
	})

	t.Run("options for known path returns no content with allow", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/ping", nil)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
		}
		if resp.Body.Len() != 0 {
			t.Fatalf("body length = %d, want 0", resp.Body.Len())
		}
		allowMethods := resp.Header().Values("Allow")
		if !containsString(allowMethods, http.MethodGet) {
			t.Fatalf("allow header = %v, want to contain %q", allowMethods, http.MethodGet)
		}
		if !containsString(allowMethods, http.MethodOptions) {
			t.Fatalf("allow header = %v, want to contain %q", allowMethods, http.MethodOptions)
		}
	})

	t.Run("cors preflight is explicit and fail-closed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/api/v1/ping", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusMethodNotAllowed)
		}
		if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
			t.Fatalf("content type = %q, want prefix %q", got, "application/problem+json")
		}
		if !strings.Contains(resp.Body.String(), "cors preflight is not enabled") {
			t.Fatalf("body = %q, want to contain preflight policy detail", resp.Body.String())
		}
		allowMethods := resp.Header().Values("Allow")
		if !containsString(allowMethods, http.MethodGet) {
			t.Fatalf("allow header = %v, want to contain %q", allowMethods, http.MethodGet)
		}
		if !containsString(allowMethods, http.MethodOptions) {
			t.Fatalf("allow header = %v, want to contain %q", allowMethods, http.MethodOptions)
		}
	})

	t.Run("options for unknown path returns not found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/unknown", nil)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
		}
		if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
			t.Fatalf("content type = %q, want prefix %q", got, "application/problem+json")
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
	if got := event["route"]; got != "GET /api/v1/ping" {
		t.Fatalf("route = %v, want %q", got, "GET /api/v1/ping")
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

	// Scrape once more to include the previous /metrics request in histogram output.
	metricsReqSecond := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRespSecond := httptest.NewRecorder()
	h.ServeHTTP(metricsRespSecond, metricsReqSecond)
	if metricsRespSecond.Code != http.StatusOK {
		t.Fatalf("second metrics status = %d, want %d", metricsRespSecond.Code, http.StatusOK)
	}
	if !strings.Contains(metricsRespSecond.Body.String(), `method="GET",route="GET /metrics",status_code="200"`) {
		t.Fatalf("metrics output does not contain expected duration histogram labels for metrics endpoint")
	}
}

func TestRootRouterMetricsRouteHasPriorityOverMountedSubrouter(t *testing.T) {
	apiSubrouter := chi.NewRouter()
	apiSubrouter.Get("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("mounted"))
	})
	apiSubrouter.Get("/api/v1/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("subrouter-ping"))
	})

	metricsHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("root-metrics"))
	})

	rootRouter := newRootRouter(apiSubrouter, metricsHandler)

	t.Run("metrics uses root handler", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
		resp := httptest.NewRecorder()

		rootRouter.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
		}
		if body := resp.Body.String(); body != "root-metrics" {
			t.Fatalf("body = %q, want %q", body, "root-metrics")
		}
	})

	t.Run("non-conflicting routes still served by mounted subrouter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
		resp := httptest.NewRecorder()

		rootRouter.ServeHTTP(resp, req)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
		}
		if body := resp.Body.String(); body != "subrouter-ping" {
			t.Fatalf("body = %q, want %q", body, "subrouter-ping")
		}
	})
}

func TestRouteTemplateUsedForOTelSpanName(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(recorder),
	)
	previousTracerProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(previousTracerProvider)
		otel.SetTextMapPropagator(previousPropagator)
		_ = tp.Shutdown(context.Background())
	})

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewRouter(log, Handlers{
		Health: health.New(),
		Ping:   ping.New(),
	}, telemetry.New(), RouterConfig{})

	pingReq := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	pingResp := httptest.NewRecorder()
	h.ServeHTTP(pingResp, pingReq)
	if pingResp.Code != http.StatusOK {
		t.Fatalf("ping status = %d, want %d", pingResp.Code, http.StatusOK)
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsResp := httptest.NewRecorder()
	h.ServeHTTP(metricsResp, metricsReq)
	if metricsResp.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, want %d", metricsResp.Code, http.StatusOK)
	}

	spans := recorder.Ended()
	if len(spans) == 0 {
		t.Fatalf("expected ended spans")
	}

	wantSpanNames := map[string]bool{
		"GET /api/v1/ping": false,
		"GET /metrics":     false,
	}

	spanNames := make([]string, 0, len(spans))
	for _, span := range spans {
		name := span.Name()
		spanNames = append(spanNames, name)
		if _, ok := wantSpanNames[name]; ok {
			wantSpanNames[name] = true
		}
	}

	for wantName, found := range wantSpanNames {
		if !found {
			t.Fatalf("span name %q not found; got %v", wantName, spanNames)
		}
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
