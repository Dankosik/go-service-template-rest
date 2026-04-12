package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/api"
	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/app/ping"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

const testRouterMaxBodyBytes int64 = 1 << 20

func mustNewRouter(t *testing.T, log *slog.Logger, h Handlers, metrics *telemetry.Metrics, cfg RouterConfig) http.Handler {
	t.Helper()

	if h.Health == nil {
		h.Health = health.New()
	}
	if h.Ping == nil {
		h.Ping = ping.New()
	}
	if h.ReadinessGate == nil {
		h.ReadinessGate = func(context.Context) error { return nil }
	}
	if metrics == nil {
		metrics = telemetry.New()
	}
	if cfg.ReadinessTimeout <= 0 {
		cfg.ReadinessTimeout = time.Second
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = testRouterMaxBodyBytes
	}

	handler, err := NewRouter(log, h, metrics, cfg)
	if err != nil {
		t.Fatalf("NewRouter() error = %v, want nil", err)
	}
	return handler
}

func TestRouterEndpoints(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := mustNewRouter(t, log, Handlers{
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

func TestOpenAPIRuntimeContractRouterHTTPPolicy(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := mustNewRouter(t, log, Handlers{
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
		assertAllowHeader(t, resp.Header(), "GET, OPTIONS")
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
		assertAllowHeader(t, resp.Header(), "TRACE, OPTIONS")
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
		assertAllowHeader(t, resp.Header(), "GET, OPTIONS")
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
		if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("Access-Control-Allow-Origin = %q, want empty for fail-closed CORS", got)
		}
		assertAllowHeader(t, resp.Header(), "GET, OPTIONS")
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

func TestStrictRequestErrorDetailsAreSanitized(t *testing.T) {
	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	const attackerDetail = `invalid "token": secret-value`

	// Current operations have no request parameters or bodies that trigger generated parse errors.
	handler := RequestCorrelation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logStrictRequestError(log, r, errors.New(attackerDetail))
		writeMalformedRequestProblem(w, r)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.Header.Set(requestIDHeader, "req-123")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
	var problem map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &problem); err != nil {
		t.Fatalf("unmarshal problem: %v", err)
	}
	if got := problem["detail"]; got != malformedRequestProblemDetail {
		t.Fatalf("detail = %v, want %q", got, malformedRequestProblemDetail)
	}
	if strings.Contains(resp.Body.String(), attackerDetail) {
		t.Fatalf("problem body leaks raw parser detail: %q", resp.Body.String())
	}

	logLine := out.String()
	if strings.Contains(logLine, attackerDetail) {
		t.Fatalf("log line leaks raw parser detail: %q", logLine)
	}
	if !strings.Contains(logLine, `"error_class"`) {
		t.Fatalf("log line = %q, want sanitized error_class", logLine)
	}
	if !strings.Contains(logLine, `"request_id":"req-123"`) {
		t.Fatalf("log line = %q, want request_id", logLine)
	}
}

func TestGeneratedChiRequestErrorDetailsAreSanitized(t *testing.T) {
	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	const attackerDetail = `invalid "token": secret-value`

	options := generatedChiServerOptions(log)
	if options.ErrorHandlerFunc == nil {
		t.Fatalf("generatedChiServerOptions() ErrorHandlerFunc = nil")
	}

	handler := RequestCorrelation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		options.ErrorHandlerFunc(w, r, errors.New(attackerDetail))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
	req.Header.Set(requestIDHeader, "req-chi-123")
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
	if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("content type = %q, want prefix %q", got, "application/problem+json")
	}
	var problem map[string]any
	if err := json.Unmarshal(resp.Body.Bytes(), &problem); err != nil {
		t.Fatalf("unmarshal problem: %v", err)
	}
	if got := problem["detail"]; got != malformedRequestProblemDetail {
		t.Fatalf("detail = %v, want %q", got, malformedRequestProblemDetail)
	}
	if strings.Contains(resp.Body.String(), attackerDetail) {
		t.Fatalf("problem body leaks raw parser detail: %q", resp.Body.String())
	}

	logLine := out.String()
	if strings.Contains(logLine, attackerDetail) {
		t.Fatalf("log line leaks raw parser detail: %q", logLine)
	}
	if !strings.Contains(logLine, `"error_class"`) {
		t.Fatalf("log line = %q, want sanitized error_class", logLine)
	}
	if !strings.Contains(logLine, `"request_id":"req-chi-123"`) {
		t.Fatalf("log line = %q, want request_id", logLine)
	}
}

func TestRouterAddsRequestIDHeader(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := mustNewRouter(t, log, Handlers{
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
	h := mustNewRouter(t, log, Handlers{
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
	h := mustNewRouter(t, log, Handlers{
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
	h := mustNewRouter(t, log, Handlers{
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

func TestRecoverLogsPanicClassWithoutRawValue(t *testing.T) {
	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	const secretValue = "secret-value"

	handler := RequestCorrelation(Recover(log, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(secretValue)
	})))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	req.Header.Set(requestIDHeader, "req-panic-123")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusInternalServerError)
	}
	if strings.Contains(out.String(), secretValue) {
		t.Fatalf("panic log leaks raw recovered value: %q", out.String())
	}
	if !strings.Contains(out.String(), `"panic_class":"string"`) {
		t.Fatalf("panic log = %q, want panic_class", out.String())
	}
	if !strings.Contains(out.String(), `"panic_type":"string"`) {
		t.Fatalf("panic log = %q, want panic_type", out.String())
	}
	if !strings.Contains(out.String(), `"request_id":"req-panic-123"`) {
		t.Fatalf("panic log = %q, want request_id", out.String())
	}
}

func TestOpenAPIRuntimeContractAccessLogIncludesRouteLabel(t *testing.T) {
	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	h := mustNewRouter(t, log, Handlers{
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

func TestAccessLogPreservesFirstFinalStatus(t *testing.T) {
	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	handler := AccessLog(log, nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		w.WriteHeader(http.StatusInternalServerError)
	}))

	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
	}

	var event map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &event); err != nil {
		t.Fatalf("unmarshal access log: %v", err)
	}
	if got := int(event["status"].(float64)); got != http.StatusNoContent {
		t.Fatalf("logged status = %d, want %d", got, http.StatusNoContent)
	}
}

func TestAccessLogResponseControllerCanReachWrappedWriter(t *testing.T) {
	var flushErr error
	handler := AccessLog(nil, nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flushErr = http.NewResponseController(w).Flush()
	}))

	req := httptest.NewRequest(http.MethodGet, "/flush", nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if flushErr != nil {
		t.Fatalf("ResponseController.Flush() error = %v, want nil", flushErr)
	}
	if !resp.Flushed {
		t.Fatalf("ResponseRecorder.Flushed = false, want true")
	}
}

func TestOpenAPIRuntimeContractMetricsExposeRouteLabels(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := mustNewRouter(t, log, Handlers{
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

func TestOpenAPIRuntimeContractRootRouterMetricsRouteHasPriorityOverMountedSubrouter(t *testing.T) {
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

func TestStrictMetricsHandlerIsNotRuntimeOwned(t *testing.T) {
	t.Parallel()

	strict := strictHandlers{metrics: telemetry.New()}
	resp, err := strict.Metrics(context.Background(), api.MetricsRequestObject{})
	if err == nil {
		t.Fatal("strict Metrics() error = nil, want non-nil")
	}
	if resp != nil {
		t.Fatalf("strict Metrics() response = %T, want nil", resp)
	}
	if !strings.Contains(err.Error(), "not runtime-owned") {
		t.Fatalf("strict Metrics() error = %q, want runtime ownership detail", err.Error())
	}
}

func TestOpenAPIRuntimeContractManualRootRouteExceptionsAreDocumented(t *testing.T) {
	openAPIRoutes := openAPIOperationRoutes(t)
	manualRoutes := manualRootRoutes(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	manualRouteKeys := make(map[manualRootRouteKey]struct{}, len(manualRoutes))

	for _, route := range manualRoutes {
		manualRouteKeys[route.key] = struct{}{}
		if strings.HasPrefix(route.key.path, "/api/") {
			t.Fatalf("manual route %s %s uses API namespace; add it through OpenAPI instead", route.key.method, route.key.path)
		}

		reason, allowed := documentedManualRootRouteExceptions[route.key]
		if !allowed {
			t.Fatalf("manual route %s %s is missing a documented root exception", route.key.method, route.key.path)
		}
		if reason == "" {
			t.Fatalf("manual route %s %s has an empty documented root exception reason", route.key.method, route.key.path)
		}
	}

	for key, reason := range documentedManualRootRouteExceptions {
		if reason == "" {
			t.Fatalf("documented root exception %s %s has an empty reason", key.method, key.path)
		}
		if _, manual := manualRouteKeys[key]; !manual {
			t.Fatalf("documented root exception %s %s has no matching manual root route", key.method, key.path)
		}
		if _, generated := openAPIRoutes[key]; generated && reason != metricsRootRouteReason {
			t.Fatalf("documented generated-route overlap %s %s reason = %q, want %q", key.method, key.path, reason, metricsRootRouteReason)
		}
	}
}

func TestOpenAPIRuntimeContractRootRouteTreeContainsOnlyGeneratedOrDocumentedRoutes(t *testing.T) {
	openAPIRoutes := openAPIOperationRoutes(t)

	expectedCounts := make(map[manualRootRouteKey]int, len(openAPIRoutes)+len(documentedManualRootRouteExceptions))
	for key := range openAPIRoutes {
		expectedCounts[key]++
	}
	for key := range documentedManualRootRouteExceptions {
		expectedCounts[key]++
	}

	apiSubrouter := chi.NewRouter()
	for key := range openAPIRoutes {
		apiSubrouter.Method(key.method, key.path, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		}))
	}

	rootRouter := newRootRouter(apiSubrouter, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	seenCounts := make(map[manualRootRouteKey]int, len(expectedCounts))
	err := chi.Walk(rootRouter, func(method string, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		key := manualRootRouteKey{method: method, path: route}
		seenCounts[key]++

		if strings.HasPrefix(route, "/api/") {
			if _, generated := openAPIRoutes[key]; !generated {
				t.Fatalf("root route tree contains manual API route %s %s; add it through OpenAPI instead", method, route)
			}
		}
		if _, expected := expectedCounts[key]; !expected {
			t.Fatalf("root route tree contains undocumented manual route %s %s", method, route)
		}

		return nil
	})
	if err != nil {
		t.Fatalf("chi.Walk() error = %v", err)
	}

	for key, want := range expectedCounts {
		if got := seenCounts[key]; got != want {
			t.Fatalf("root route tree route %s %s count = %d, want %d", key.method, key.path, got, want)
		}
	}
}

func openAPIOperationRoutes(t *testing.T) map[manualRootRouteKey]struct{} {
	t.Helper()

	swagger, err := api.GetSwagger()
	if err != nil {
		t.Fatalf("GetSwagger() error = %v", err)
	}

	routes := make(map[manualRootRouteKey]struct{})
	for path, item := range swagger.Paths.Map() {
		if item == nil {
			continue
		}
		for method := range item.Operations() {
			routes[manualRootRouteKey{method: method, path: path}] = struct{}{}
		}
	}
	return routes
}

func TestOpenAPIRuntimeContractRouteTemplateUsedForOTelSpanName(t *testing.T) {
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
	h := mustNewRouter(t, log, Handlers{
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
	wantHTTPRoutes := map[string]string{
		"GET /api/v1/ping": "/api/v1/ping",
		"GET /metrics":     "/metrics",
	}

	spanNames := make([]string, 0, len(spans))
	for _, span := range spans {
		name := span.Name()
		spanNames = append(spanNames, name)
		if _, ok := wantSpanNames[name]; ok {
			wantSpanNames[name] = true
			if gotRoute := spanHTTPRoute(span); gotRoute != wantHTTPRoutes[name] {
				t.Fatalf("span %q http.route = %q, want %q", name, gotRoute, wantHTTPRoutes[name])
			}
		}
	}
	if got, want := len(spans), len(wantSpanNames); got != want {
		t.Fatalf("ended spans len = %d, want %d without nested server spans; got names %v", got, want, spanNames)
	}

	for wantName, found := range wantSpanNames {
		if !found {
			t.Fatalf("span name %q not found; got %v", wantName, spanNames)
		}
	}
}

func spanHTTPRoute(span sdktrace.ReadOnlySpan) string {
	for _, attr := range span.Attributes() {
		if attr.Key == semconv.HTTPRouteKey {
			return attr.Value.AsString()
		}
	}
	return ""
}

func assertAllowHeader(t *testing.T, header http.Header, want string) {
	t.Helper()

	if got := header.Get("Allow"); got != want {
		t.Fatalf("Allow = %q, want %q", got, want)
	}
	if got := header.Values("Allow"); len(got) != 1 || got[0] != want {
		t.Fatalf("Allow header values = %v, want single value %q", got, want)
	}
}
