package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/api"
	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/otel/propagation"
)

func TestOpenAPIRuntimeContractRouterHTTPPolicy(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.DiscardHandler)
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
	}, telemetry.New(), RouterConfig{})

	t.Run("not found uses problem envelope", func(t *testing.T) {
		t.Parallel()

		resp := doRequest(h, http.MethodGet, "/does-not-exist")

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
		}
		assertProblemContentType(t, resp.Header())
		assertProblemCode(t, resp, problemCodeNotFound)
	})

	t.Run("unknown method on missing path returns not found", func(t *testing.T) {
		t.Parallel()

		resp := doRequest(h, "BREW", "/does-not-exist")

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
		}
		assertProblemContentType(t, resp.Header())
		if got := resp.Header().Get("Allow"); got != "" {
			t.Fatalf("Allow = %q, want empty", got)
		}
	})

	t.Run("method not allowed uses problem envelope and allow header", func(t *testing.T) {
		t.Parallel()

		resp := doRequest(h, http.MethodPost, "/health/live")

		if resp.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusMethodNotAllowed)
		}
		assertProblemContentType(t, resp.Header())
		assertProblemCode(t, resp, problemCodeMethodNotAllowed)
		assertAllowHeader(t, resp.Header(), "GET, OPTIONS")
	})

	t.Run("unknown method on existing path returns method not allowed", func(t *testing.T) {
		t.Parallel()

		resp := doRequest(h, "BREW", "/health/live")

		if resp.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusMethodNotAllowed)
		}
		assertProblemContentType(t, resp.Header())
		assertAllowHeader(t, resp.Header(), "GET, OPTIONS")
	})

	t.Run("method not allowed allow header includes trace when route exists", func(t *testing.T) {
		t.Parallel()

		apiSubrouter := chi.NewRouter()
		apiSubrouter.Trace("/trace-only", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		})

		rootRouter := newRootRouter(apiSubrouter, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		resp := doRequest(rootRouter, http.MethodPost, "/trace-only")

		if resp.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusMethodNotAllowed)
		}
		assertAllowHeader(t, resp.Header(), "TRACE, OPTIONS")
	})

	t.Run("options for known path returns no content with allow", func(t *testing.T) {
		t.Parallel()

		resp := doRequest(h, http.MethodOptions, "/health/live")

		if resp.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
		}
		if resp.Body.Len() != 0 {
			t.Fatalf("body length = %d, want 0", resp.Body.Len())
		}
		assertAllowHeader(t, resp.Header(), "GET, OPTIONS")
	})

	t.Run("cors preflight is explicit and fail-closed", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodOptions, "/health/live", nil)
		req.Header.Set("Origin", "https://example.com")
		req.Header.Set("Access-Control-Request-Method", http.MethodGet)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusMethodNotAllowed)
		}
		assertProblemContentType(t, resp.Header())
		if !strings.Contains(resp.Body.String(), "cors preflight is not enabled") {
			t.Fatalf("body = %q, want to contain preflight policy detail", resp.Body.String())
		}
		if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "" {
			t.Fatalf("Access-Control-Allow-Origin = %q, want empty for fail-closed CORS", got)
		}
		assertAllowHeader(t, resp.Header(), "GET, OPTIONS")
	})

	t.Run("options for unknown path returns not found", func(t *testing.T) {
		t.Parallel()

		resp := doRequest(h, http.MethodOptions, "/unknown")

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
		}
		assertProblemContentType(t, resp.Header())
	})
}

func TestGeneratedStrictRequestErrorDetailsAreSanitized(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	const attackerDetail = `invalid "token": secret-value`

	options := generatedStrictServerOptions(log)
	if options.RequestErrorHandlerFunc == nil {
		t.Fatalf("generatedStrictServerOptions() RequestErrorHandlerFunc = nil")
	}

	handler := RequestCorrelation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		options.RequestErrorHandlerFunc(w, r, errors.New(attackerDetail))
	}))

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	req.Header.Set(requestIDHeader, "req-123")
	req.Header.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	req = req.WithContext(propagation.TraceContext{}.Extract(req.Context(), propagation.HeaderCarrier(req.Header)))
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
	if !strings.Contains(logLine, `"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736"`) {
		t.Fatalf("log line = %q, want trace_id", logLine)
	}
	if !strings.Contains(logLine, `"span_id":"00f067aa0ba902b7"`) {
		t.Fatalf("log line = %q, want span_id", logLine)
	}
}

func TestGeneratedChiRequestErrorDetailsAreSanitized(t *testing.T) {
	t.Parallel()

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

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	req.Header.Set(requestIDHeader, "req-chi-123")
	req.Header.Set("Traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	req = req.WithContext(propagation.TraceContext{}.Extract(req.Context(), propagation.HeaderCarrier(req.Header)))
	resp := httptest.NewRecorder()

	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusBadRequest)
	}
	assertProblemContentType(t, resp.Header())
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
	if !strings.Contains(logLine, `"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736"`) {
		t.Fatalf("log line = %q, want trace_id", logLine)
	}
	if !strings.Contains(logLine, `"span_id":"00f067aa0ba902b7"`) {
		t.Fatalf("log line = %q, want span_id", logLine)
	}
}

func TestOpenAPIRuntimeContractAccessLogIncludesRouteLabel(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
	}, nil, RouterConfig{})

	const (
		requestID = "demo-123"
		traceID   = "4bf92f3577b34da6a3ce929d0e0e4736"
	)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	req.Header.Set(requestIDHeader, requestID)
	req.Header.Set("Traceparent", "00-"+traceID+"-00f067aa0ba902b7-01")
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
	if got := event["route"]; got != "GET /health/ready" {
		t.Fatalf("route = %v, want %q", got, "GET /health/ready")
	}
}

func TestOpenAPIRuntimeContractMetricsExposeRouteLabels(t *testing.T) {
	log := slog.New(slog.DiscardHandler)
	metrics := telemetry.New()
	telemetrytest.RestoreGlobals(t)
	shutdown, err := telemetry.SetupMetrics(context.Background(), metrics, telemetry.MetricsConfig{
		ServiceName:    "router-test",
		ServiceVersion: "test",
		DeploymentEnv:  "test",
	})
	if err != nil {
		t.Fatalf("SetupMetrics() error = %v", err)
	}
	t.Cleanup(func() {
		if err := shutdown(context.Background()); err != nil {
			t.Fatalf("shutdown metrics: %v", err)
		}
	})

	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
	}, metrics, RouterConfig{})

	liveReq := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	liveReq.Host = "attacker-controlled.example:65535"
	liveResp := httptest.NewRecorder()
	h.ServeHTTP(liveResp, liveReq)

	metricsResp := doRequest(h, http.MethodGet, "/metrics")

	if metricsResp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", metricsResp.Code, http.StatusOK)
	}

	body := metricsResp.Body.String()
	if !strings.Contains(body, "http_server_request_duration_seconds_bucket") {
		t.Fatalf("metrics output does not contain duration histogram buckets")
	}
	if !strings.Contains(body, `http_request_method="GET"`) ||
		!strings.Contains(body, `http_response_status_code="200"`) ||
		!strings.Contains(body, `http_route="/health/live"`) ||
		!strings.Contains(body, `server_address="router-test"`) {
		t.Fatalf("metrics output does not contain expected duration histogram labels for liveness endpoint")
	}
	if strings.Contains(body, "attacker-controlled.example") || strings.Contains(body, `server_port="65535"`) {
		t.Fatal("metrics output contains attacker-controlled authority labels")
	}

	// Scrape once more to include the previous /metrics request in histogram output.
	metricsRespSecond := doRequest(h, http.MethodGet, "/metrics")
	if metricsRespSecond.Code != http.StatusOK {
		t.Fatalf("second metrics status = %d, want %d", metricsRespSecond.Code, http.StatusOK)
	}
	if !strings.Contains(metricsRespSecond.Body.String(), `http_route="/metrics"`) {
		t.Fatalf("metrics output does not contain expected duration histogram labels for metrics endpoint")
	}
}

func TestOpenAPIRuntimeContractManualRootRouteExceptionsAreDocumented(t *testing.T) {
	t.Parallel()

	openAPIRoutes := openAPIOperationRoutes(t)
	manualRoutes := manualRootRoutes(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	seen := make(map[manualRootRouteKey]struct{}, len(manualRoutes))

	for _, route := range manualRoutes {
		if strings.HasPrefix(route.key.path, "/api/") {
			t.Fatalf("manual route %s %s uses API namespace; add it through OpenAPI instead", route.key.method, route.key.path)
		}
		if route.reason == "" {
			t.Fatalf("manual route %s %s has an empty documented root exception reason", route.key.method, route.key.path)
		}
		if _, generated := openAPIRoutes[route.key]; generated {
			t.Fatalf("manual route %s %s overlaps the generated OpenAPI surface", route.key.method, route.key.path)
		}
		if _, duplicate := seen[route.key]; duplicate {
			t.Fatalf("manual route %s %s is registered more than once", route.key.method, route.key.path)
		}
		seen[route.key] = struct{}{}
	}
}

func TestOpenAPIRuntimeContractRootRouteTreeContainsOnlyGeneratedOrDocumentedRoutes(t *testing.T) {
	t.Parallel()

	openAPIRoutes := openAPIOperationRoutes(t)
	manualRoutes := manualRootRoutes(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	expectedCounts := make(map[manualRootRouteKey]int, len(openAPIRoutes)+len(manualRoutes))
	for key := range openAPIRoutes {
		expectedCounts[key]++
	}
	for _, route := range manualRoutes {
		expectedCounts[route.key]++
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

	swagger, err := api.GetSpec()
	if err != nil {
		t.Fatalf("GetSpec() error = %v", err)
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

//nolint:paralleltest // Installs a process-wide tracer provider for span capture.
func TestOpenAPIRuntimeContractRouteTemplateUsedForOTelSpanName(t *testing.T) {
	recorder := telemetrytest.InstallSpanRecorder(t)

	log := slog.New(slog.DiscardHandler)
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
	}, telemetry.New(), RouterConfig{})

	liveResp := doRequest(h, http.MethodGet, "/health/live")
	if liveResp.Code != http.StatusOK {
		t.Fatalf("liveness status = %d, want %d", liveResp.Code, http.StatusOK)
	}

	metricsResp := doRequest(h, http.MethodGet, "/metrics")
	if metricsResp.Code != http.StatusOK {
		t.Fatalf("metrics status = %d, want %d", metricsResp.Code, http.StatusOK)
	}

	customMethodResp := doRequest(h, "BREW", "/health/live")
	if customMethodResp.Code != http.StatusMethodNotAllowed {
		t.Fatalf("custom method status = %d, want %d", customMethodResp.Code, http.StatusMethodNotAllowed)
	}

	spans := recorder.Ended()
	if len(spans) == 0 {
		t.Fatalf("expected ended spans")
	}

	wantSpanNames := map[string]bool{
		"GET /health/live": false,
		"GET /metrics":     false,
		"HTTP":             false,
	}
	wantHTTPRoutes := map[string]string{
		"GET /health/live": "/health/live",
		"GET /metrics":     "/metrics",
	}

	spanNames := make([]string, 0, len(spans))
	for _, span := range spans {
		name := span.Name()
		spanNames = append(spanNames, name)
		if _, ok := wantSpanNames[name]; ok {
			wantSpanNames[name] = true
			if wantHTTPRoutes[name] == "" {
				continue
			}
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
