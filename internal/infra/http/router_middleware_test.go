package httpx

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
)

func TestRouterAddsRequestIDHeader(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.DiscardHandler)
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
	}, telemetry.New(), RouterConfig{})

	t.Run("generates request id when header is absent", func(t *testing.T) {
		t.Parallel()

		resp := doRequest(h, http.MethodGet, "/health/live")

		if got := resp.Header().Get(requestIDHeader); got == "" {
			t.Fatalf("%s header is empty", requestIDHeader)
		}
	})

	t.Run("echoes inbound request id", func(t *testing.T) {
		t.Parallel()

		const wantRequestID = "demo-123"

		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		req.Header.Set(requestIDHeader, wantRequestID)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if got := resp.Header().Get(requestIDHeader); got != wantRequestID {
			t.Fatalf("%s = %q, want %q", requestIDHeader, got, wantRequestID)
		}
	})

	t.Run("replaces invalid request id before echoing problem or logging", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer
		log := slog.New(slog.NewJSONHandler(&out, nil))
		h := mustNewRouter(t, log, Handlers{
			Health: health.New(),
		}, telemetry.New(), RouterConfig{})
		const invalidRequestID = "user@example.com"

		req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)
		req.Header.Set(requestIDHeader, invalidRequestID)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
		}
		responseRequestID := resp.Header().Get(requestIDHeader)
		if responseRequestID == "" {
			t.Fatalf("%s header is empty", requestIDHeader)
		}
		if responseRequestID == invalidRequestID {
			t.Fatalf("%s echoed invalid request ID %q", requestIDHeader, invalidRequestID)
		}
		if strings.Contains(resp.Body.String(), invalidRequestID) {
			t.Fatalf("problem body leaked invalid request ID: %q", resp.Body.String())
		}
		var problem map[string]any
		if err := json.Unmarshal(resp.Body.Bytes(), &problem); err != nil {
			t.Fatalf("unmarshal problem: %v", err)
		}
		if got := problem["request_id"]; got != responseRequestID {
			t.Fatalf("problem request_id = %v, want %q", got, responseRequestID)
		}
		logLine := out.String()
		if strings.Contains(logLine, invalidRequestID) {
			t.Fatalf("access log leaked invalid request ID: %q", logLine)
		}
		if !strings.Contains(logLine, `"request_id":"`+responseRequestID+`"`) {
			t.Fatalf("access log = %q, want generated request_id %q", logLine, responseRequestID)
		}
	})

	t.Run("replaces too long request id", func(t *testing.T) {
		t.Parallel()

		tooLongRequestID := strings.Repeat("a", 129)

		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		req.Header.Set(requestIDHeader, tooLongRequestID)
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		got := resp.Header().Get(requestIDHeader)
		if got == "" {
			t.Fatalf("%s header is empty", requestIDHeader)
		}
		if got == tooLongRequestID {
			t.Fatalf("%s echoed too-long request ID", requestIDHeader)
		}
	})
}

func TestRouterAddsSecurityHeaders(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.DiscardHandler)
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
	}, telemetry.New(), RouterConfig{})

	resp := doRequest(h, http.MethodGet, "/health/live")

	if got := resp.Header().Get(contentTypeOptionsHeader); got != "nosniff" {
		t.Fatalf("%s = %q, want %q", contentTypeOptionsHeader, got, "nosniff")
	}
}

func TestRouterRejectsRequestBodyTooLarge(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.DiscardHandler)
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
	}, telemetry.New(), RouterConfig{MaxBodyBytes: 1})

	t.Run("known content length is rejected before reading", func(t *testing.T) {
		t.Parallel()

		req := httptest.NewRequest(http.MethodGet, "/health/live", strings.NewReader("ab"))
		req.ContentLength = 2
		resp := httptest.NewRecorder()

		h.ServeHTTP(resp, req)

		if resp.Code != http.StatusRequestEntityTooLarge {
			t.Fatalf("status = %d, want %d", resp.Code, http.StatusRequestEntityTooLarge)
		}
		assertProblemContentType(t, resp.Header())
		assertProblemCode(t, resp, problemCodeRequestEntityTooLarge)
		if !strings.Contains(resp.Body.String(), "request body exceeds limit") {
			t.Fatalf("body = %q, want %q", resp.Body.String(), "request body exceeds limit")
		}
	})

	t.Run("unknown content length is capped at the body reader", func(t *testing.T) {
		t.Parallel()

		var readErr error
		handler := RequestBodyLimit(1, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			_, readErr = io.ReadAll(r.Body)
		}))

		req := httptest.NewRequest(http.MethodPost, "/any", strings.NewReader(`{"message":"a"}`))
		req.ContentLength = -1
		resp := httptest.NewRecorder()

		handler.ServeHTTP(resp, req)

		var maxBytesErr *http.MaxBytesError
		if !errors.As(readErr, &maxBytesErr) {
			t.Fatalf("read error = %v, want *http.MaxBytesError", readErr)
		}
	})
}

func TestRecoverLogsPanicClassWithoutRawValue(t *testing.T) {
	t.Parallel()

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
	assertProblemCode(t, resp, problemCodeInternalError)
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

func TestRecoverDoesNotWriteProblemAfterCommittedResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantBody   string
	}{
		{
			name: "write header",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
				panic("after status")
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "write body",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte("partial response"))
				panic("after body")
			},
			wantStatus: http.StatusOK,
			wantBody:   "partial response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			handler := Recover(slog.New(slog.DiscardHandler), tt.handler)
			resp := doRequest(handler, http.MethodGet, "/panic")

			if resp.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Code, tt.wantStatus)
			}
			if body := resp.Body.String(); body != tt.wantBody {
				t.Fatalf("body = %q, want %q", body, tt.wantBody)
			}
			if strings.Contains(resp.Body.String(), "request failed") {
				t.Fatalf("body contains appended problem payload: %q", resp.Body.String())
			}
		})
	}
}

func TestRecoverPreservesFlusherInterfaceAndCommit(t *testing.T) {
	t.Parallel()

	var sawFlusher bool
	handler := Recover(slog.New(slog.DiscardHandler), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		sawFlusher = true
		flusher.Flush()
		panic("after flush")
	}))

	resp := doRequest(handler, http.MethodGet, "/panic")

	if !sawFlusher {
		t.Fatal("wrapped ResponseWriter does not implement http.Flusher")
	}
	if !resp.Flushed {
		t.Fatal("ResponseRecorder.Flushed = false, want true")
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if body := resp.Body.String(); body != "" {
		t.Fatalf("body = %q, want empty after committed flush", body)
	}
}

func TestAccessLogPreservesFirstFinalStatus(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&out, nil))
	handler := AccessLog(log, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		w.WriteHeader(http.StatusInternalServerError)
	}))

	resp := doRequest(handler, http.MethodGet, "/status")

	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
	}

	var event map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &event); err != nil {
		t.Fatalf("unmarshal access log: %v", err)
	}
	status, ok := event["status"].(float64)
	if !ok {
		t.Fatalf("logged status type = %T, want float64", event["status"])
	}
	if got := int(status); got != http.StatusNoContent {
		t.Fatalf("logged status = %d, want %d", got, http.StatusNoContent)
	}
}

func TestAccessLogPreservesFlusherInterface(t *testing.T) {
	t.Parallel()

	var directFlusher bool
	var flushErr error
	handler := AccessLog(nil, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		directFlusher = true
		flusher.Flush()
		flushErr = http.NewResponseController(w).Flush()
	}))

	resp := doRequest(handler, http.MethodGet, "/flush")

	if !directFlusher {
		t.Fatal("wrapped ResponseWriter does not implement http.Flusher")
	}
	if flushErr != nil {
		t.Fatalf("ResponseController.Flush() error = %v, want nil", flushErr)
	}
	if !resp.Flushed {
		t.Fatalf("ResponseRecorder.Flushed = false, want true")
	}
}

func TestHTTPMethodNormalizationBoundsNonStandardRouteLabels(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("BREW", "/coffee", nil)
	req.Pattern = "BREW /coffee"
	if got := requestMethodLabel(req); got != "OTHER" {
		t.Fatalf("requestMethodLabel() = %q, want %q", got, "OTHER")
	}
	if got := routePathTemplateForRequest(req); got != "/coffee" {
		t.Fatalf("routePathTemplateForRequest() = %q, want %q", got, "/coffee")
	}
}

func TestOTelServerNameBoundsAuthorityLabels(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		configured string
		want       string
	}{
		{configured: "orders-api", want: "orders-api:0"},
		{configured: " orders-api ", want: "orders-api:0"},
		{want: "service:0"},
	} {
		if got := otelServerName(tt.configured); got != tt.want {
			t.Fatalf("otelServerName(%q) = %q, want %q", tt.configured, got, tt.want)
		}
	}
}
