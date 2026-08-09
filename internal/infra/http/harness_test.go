package httpx

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/observability/logctx"
	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

const (
	testRouterMaxBodyBytes   int64 = 1 << 20
	testRouterRequestTimeout       = 5 * time.Second
)

func mustNewRouter(tb testing.TB, log *slog.Logger, h Handlers, metrics *telemetry.Metrics, cfg RouterConfig) http.Handler {
	tb.Helper()

	if h.Health == nil {
		h.Health = health.New()
	}
	if h.ReadinessGate == nil {
		h.ReadinessGate = func(context.Context) error { return nil }
	}
	if h.API == nil {
		h.API = unimplementedAPI{}
	}
	// Readiness is served from cached state, and an unevaluated cache fails
	// closed. Seeding it here mirrors what bootstrap does before it admits
	// traffic; a test that wants "not ready" supplies a failing probe.
	_ = h.Health.Refresh(context.Background(), time.Second, 1)
	if metrics == nil {
		metrics = telemetry.New()
	}
	if cfg.MaxBodyBytes <= 0 {
		cfg.MaxBodyBytes = testRouterMaxBodyBytes
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = testRouterRequestTimeout
	}
	if cfg.OTelServerName == "" {
		cfg.OTelServerName = "router-test"
	}

	handler, err := NewRouter(log, h, metrics, cfg)
	if err != nil {
		tb.Fatalf("NewRouter() error = %v, want nil", err)
	}
	return handler
}

// doRequest executes a bodyless request against h and returns the recorded
// response. Tests that need custom headers, bodies with unusual framing, or
// other request mutation keep building the request explicitly at the call site.
func doRequest(h http.Handler, method, target string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, target, nil)
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	return resp
}

// doJSONRequest executes a request carrying a JSON body against h and returns
// the recorded response.
func doJSONRequest(h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequestWithContext(context.Background(), method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	return resp
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

func assertProblemContentType(t *testing.T, header http.Header) {
	t.Helper()

	got := header.Get("Content-Type")
	gotMediaType, _, err := mime.ParseMediaType(got)
	if err != nil {
		t.Fatalf("Content-Type = %q, want valid problem media type: %v", got, err)
	}
	wantMediaType, _, err := mime.ParseMediaType(problemJSONContentType)
	if err != nil {
		t.Fatalf("parse problem content type %q: %v", problemJSONContentType, err)
	}
	if gotMediaType != wantMediaType {
		t.Fatalf("Content-Type media type = %q, want %q", gotMediaType, wantMediaType)
	}
}

func assertProblemCode(t *testing.T, resp *httptest.ResponseRecorder, wantCode problem.Code) {
	t.Helper()

	var decoded openapi.Problem
	if err := json.Unmarshal(resp.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("unmarshal problem: %v", err)
	}
	if got := decoded.Code; got != string(wantCode) {
		t.Fatalf("problem code = %q, want %q", got, wantCode)
	}
}

// unimplementedAPI satisfies the generated server interface without implementing
// any operation.
//
// Embedding leaves every method nil, which is what a test exercising only the
// platform probes or the fallback policy needs. It exists so adding the first real
// operation to the contract does not break every inherited router test — those
// tests are not about that operation. A test that does exercise one passes a real
// implementation instead.
type unimplementedAPI struct {
	openapi.StrictServerInterface
}

// newTestServiceLogger builds a logger with the same shape a service runs with:
// the correlation decorator over a JSON handler. Tests that assert on log content
// need it, because this package no longer assembles request_id, trace_id, and
// span_id per call site — the decorator publishes them from the context. A test
// that logged through a bare handler would assert that correlation is absent and
// pass whether or not the wiring is right.
func newTestServiceLogger(out io.Writer) *slog.Logger {
	return slog.New(logctx.New(slog.NewJSONHandler(out, nil)))
}
