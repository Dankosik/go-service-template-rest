package httpx

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/problem"
)

func TestPanicClassClassifiesRecoveredValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rec  any
		want string
	}{
		{name: "runtime error", rec: &runtime.TypeAssertionError{}, want: "runtime_error"},
		{name: "error", rec: errors.New("boom"), want: "error"},
		{name: "string", rec: "boom", want: "string"},
		{name: "value", rec: 42, want: "value"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := panicClass(tt.rec); got != tt.want {
				t.Fatalf("panicClass(%v) = %q, want %q", tt.rec, got, tt.want)
			}
		})
	}
}

func TestRecoverLogsPanicClassWithoutRawValue(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	log := newTestServiceLogger(&out)
	const secretValue = "secret-value"
	const secretPath = "/users/private-account-123"

	handler := RequestCorrelation(Recover(log, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(secretValue)
	})))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, secretPath, nil)
	req.Pattern = "GET /users/{id}"
	req.Header.Set(requestIDHeader, "req-panic-123")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusInternalServerError)
	}
	assertProblemCode(t, resp, problem.CodeInternalError)
	if strings.Contains(out.String(), secretValue) {
		t.Fatalf("panic log leaks raw recovered value: %q", out.String())
	}
	if strings.Contains(out.String(), secretPath) || strings.Contains(out.String(), `"path":`) {
		t.Fatalf("panic log leaks raw request path: %q", out.String())
	}
	if !strings.Contains(out.String(), `"route":"GET /users/{id}"`) {
		t.Fatalf("panic log = %q, want bounded route template", out.String())
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
	if !strings.Contains(out.String(), `"stack":"goroutine `) ||
		!strings.Contains(out.String(), "middleware_recover.go") {
		t.Fatalf("panic log = %q, want recovery stack", out.String())
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

// http.ErrAbortHandler must reach net/http unchanged. Recovering it here would
// log a stack trace and attempt a 500 on every deliberate abort — for example
// every one httputil.ReverseProxy raises.
func TestRecoverRepanicsErrAbortHandler(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer
	log := newTestServiceLogger(&logs)
	handler := Recover(log, http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	resp := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/abort", nil)

	func() {
		defer func() {
			rec := recover()
			if rec == nil {
				t.Fatal("Recover() swallowed http.ErrAbortHandler, want it re-panicked")
			}
			if err, ok := rec.(error); !ok || !errors.Is(err, http.ErrAbortHandler) {
				t.Fatalf("re-panicked value = %v, want http.ErrAbortHandler", rec)
			}
		}()
		handler.ServeHTTP(resp, req)
	}()

	if logs.Len() != 0 {
		t.Fatalf("Recover() logged %q, want nothing for a deliberate abort", logs.String())
	}
	if resp.Body.Len() != 0 {
		t.Fatalf("Recover() wrote %q, want no response body", resp.Body.String())
	}
}

// A panic that merely wraps ErrAbortHandler is still a deliberate abort, and a
// plain error panic must keep the recovering behavior.
func TestRecoverDistinguishesAbortFromOrdinaryPanics(t *testing.T) {
	t.Parallel()

	handler := Recover(slog.New(slog.DiscardHandler), http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic(errors.New("ordinary failure"))
	}))

	resp := doRequest(handler, http.MethodGet, "/boom")

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusInternalServerError)
	}
	assertProblemCode(t, resp, problem.CodeInternalError)
}
