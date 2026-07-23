package httpx

import (
	"context"
	"encoding/json"
	"log/slog"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/api"
	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

const testRouterMaxBodyBytes int64 = 1 << 20

func mustNewRouter(tb testing.TB, log *slog.Logger, h Handlers, metrics *telemetry.Metrics, cfg RouterConfig) http.Handler {
	tb.Helper()

	if h.Health == nil {
		h.Health = health.New()
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
	if cfg.OTelServerName == "" {
		cfg.OTelServerName = "router-test"
	}

	handler, err := NewRouter(log, h, metrics, cfg)
	if err != nil {
		tb.Fatalf("NewRouter() error = %v, want nil", err)
	}
	return handler
}

func BenchmarkTemplateExample(b *testing.B) {
	log := slog.New(slog.DiscardHandler)
	h := mustNewRouter(b, log, Handlers{
		Health: health.New(),
	}, telemetry.New(), RouterConfig{})

	tests := []struct {
		name       string
		target     string
		body       string
		wantStatus int
	}{
		{
			name:       "valid-small",
			target:     "/api/v1/template-example/example?copies=1",
			body:       `{"message":"hello"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "valid-maximum",
			target:     "/api/v1/template-example/" + strings.Repeat("s", 32) + "?copies=3",
			body:       `{"message":"` + strings.Repeat("m", 64) + `"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid-body",
			target:     "/api/v1/template-example/example?copies=1",
			body:       `{"message":"hello","unexpected":true}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			if got := benchmarkTemplateExampleRequest(h, tt.target, tt.body); got != tt.wantStatus {
				b.Fatalf("status = %d, want %d", got, tt.wantStatus)
			}

			b.Run("serial", func(b *testing.B) {
				b.ReportAllocs()
				for b.Loop() {
					benchmarkTemplateExampleRequest(h, tt.target, tt.body)
				}
			})
			b.Run("parallel", func(b *testing.B) {
				b.ReportAllocs()
				b.RunParallel(func(pb *testing.PB) {
					for pb.Next() {
						benchmarkTemplateExampleRequest(h, tt.target, tt.body)
					}
				})
			})
		})
	}
}

func benchmarkTemplateExampleRequest(h http.Handler, target, body string) int {
	req := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	h.ServeHTTP(resp, req)
	return resp.Code
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

func assertProblemCode(t *testing.T, resp *httptest.ResponseRecorder, wantCode problemCode) {
	t.Helper()

	var problem api.Problem
	if err := json.Unmarshal(resp.Body.Bytes(), &problem); err != nil {
		t.Fatalf("unmarshal problem: %v", err)
	}
	if got := problem.Code; got != string(wantCode) {
		t.Fatalf("problem code = %q, want %q", got, wantCode)
	}
}

func TestProblemCatalogDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code    problemCode
		status  int
		title   string
		typeURI string
	}{
		{code: problemCodeBadRequest, status: http.StatusBadRequest, title: "bad request", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.1"},
		{code: problemCodeNotFound, status: http.StatusNotFound, title: "not found", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.5"},
		{code: problemCodeMethodNotAllowed, status: http.StatusMethodNotAllowed, title: "method not allowed", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.6"},
		{code: problemCodeRequestEntityTooLarge, status: http.StatusRequestEntityTooLarge, title: "request entity too large", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.5.14"},
		{code: problemCodeInternalError, status: http.StatusInternalServerError, title: "internal server error", typeURI: "https://www.rfc-editor.org/rfc/rfc9110#section-15.6.1"},
	}

	for _, tt := range tests {
		code, definition := problemDefinitionFor(tt.code)
		if code != tt.code {
			t.Fatalf("problemDefinitionFor(%q) code = %q", tt.code, code)
		}
		if definition.status != tt.status || definition.title != tt.title || definition.typeURI != tt.typeURI {
			t.Fatalf("problemDefinitionFor(%q) = %+v, want status=%d title=%q type=%q", tt.code, definition, tt.status, tt.title, tt.typeURI)
		}
	}

	code, definition := problemDefinitionFor("unknown")
	if code != problemCodeInternalError || definition.status != http.StatusInternalServerError {
		t.Fatalf("unknown problem definition = (%q, %+v), want internal error", code, definition)
	}
}
