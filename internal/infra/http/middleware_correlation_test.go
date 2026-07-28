package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/health"
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

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health/live", nil)
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
		log := newTestServiceLogger(&out)
		h := mustNewRouter(t, log, Handlers{
			Health: health.New(),
		}, telemetry.New(), RouterConfig{})
		const invalidRequestID = "user@example.com"

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/does-not-exist", nil)
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
		var decoded map[string]any
		if err := json.Unmarshal(resp.Body.Bytes(), &decoded); err != nil {
			t.Fatalf("unmarshal problem: %v", err)
		}
		if got := decoded["request_id"]; got != responseRequestID {
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

		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/health/live", nil)
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
