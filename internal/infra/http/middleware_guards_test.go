package httpx

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/problem"
)

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
		assertProblemCode(t, resp, problem.CodeRequestEntityTooLarge)
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
