package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/problem"
)

func TestRequestTimeoutCancelsHandlerContext(t *testing.T) {
	t.Parallel()

	observed := make(chan error, 1)
	handler := RequestTimeout(10*time.Millisecond, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
		observed <- r.Context().Err()
	}))

	resp := doRequest(handler, http.MethodGet, "/slow")

	select {
	case err := <-observed:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("handler context error = %v, want %v", err, context.DeadlineExceeded)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler context was never canceled")
	}
	if resp.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusGatewayTimeout)
	}
	assertProblemContentType(t, resp.Header())
	assertProblemCode(t, resp, problem.CodeGatewayTimeout)
}

func TestRequestTimeoutLeavesCommittedResponseAlone(t *testing.T) {
	t.Parallel()

	handler := RequestTimeout(10*time.Millisecond, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A handler that starts streaming owns the response. Expiry afterwards
		// must not append a second status or a problem body to it.
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("partial")); err != nil {
			t.Errorf("Write() error = %v", err)
		}
		<-r.Context().Done()
	}))

	resp := doRequest(handler, http.MethodGet, "/streaming")

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}
	if body := resp.Body.String(); body != "partial" {
		t.Fatalf("body = %q, want %q", body, "partial")
	}
}

func TestRequestTimeoutPassesFastHandlerThrough(t *testing.T) {
	t.Parallel()

	handler := RequestTimeout(time.Minute, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	resp := doRequest(handler, http.MethodGet, "/fast")

	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
	}
	if body := resp.Body.String(); body != "" {
		t.Fatalf("body = %q, want empty", body)
	}
}

// A canceled client must not be reported as a budget expiry: the request
// context is already Canceled, not DeadlineExceeded, and writing a 504 onto a
// connection the client abandoned only adds noise.
func TestRequestTimeoutIgnoresClientCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	handler := RequestTimeout(time.Minute, http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		cancel()
		<-r.Context().Done()
	}))

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/canceled", nil).WithContext(ctx)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	if resp.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", resp.Body.String())
	}
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want the recorder default %d", resp.Code, http.StatusOK)
	}
}

func TestRequestTimeoutDisabledWhenNotPositive(t *testing.T) {
	t.Parallel()

	handler := RequestTimeout(0, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := r.Context().Deadline(); ok {
			t.Error("request context has a deadline, want none when the budget is disabled")
		}
		w.WriteHeader(http.StatusTeapot)
	}))

	resp := doRequest(handler, http.MethodGet, "/unbounded")

	if resp.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusTeapot)
	}
}

// The budget must reach handlers through the assembled router, not only when
// the middleware is applied directly. The readiness gate is the one hook that
// observes the request context from inside a generated operation, so it is what
// proves the deadline arrived, and that the request budget is what set it.
func TestRouterAppliesRequestTimeoutToHandlers(t *testing.T) {
	t.Parallel()

	const requestBudget = 250 * time.Millisecond

	deadlines := make(chan time.Time, 1)
	handler := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		ReadinessGate: func(ctx context.Context) error {
			deadline, _ := ctx.Deadline()
			deadlines <- deadline
			return nil
		},
	}, nil, RouterConfig{
		HardenConfig: HardenConfig{RequestTimeout: requestBudget},
	})

	before := time.Now()
	resp := doRequest(handler, http.MethodGet, "/health/ready")
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusOK)
	}

	select {
	case deadline := <-deadlines:
		if deadline.IsZero() {
			t.Fatal("handler context has no deadline, want the request budget")
		}
		// The deadline is set a moment after `before`, so it lands just past the
		// budget. Anything near requestBudget proves the request budget is what
		// installed it rather than some unrelated timeout.
		if budget := deadline.Sub(before); budget > 2*requestBudget {
			t.Fatalf("handler deadline is %s away, want the %s request budget to bind", budget, requestBudget)
		}
	default:
		t.Fatal("readiness gate was never reached")
	}
}

// A generated strict-server operation returns its expired context rather than
// writing a response, so the generated wrapper is what commits one. That path
// must report a spent budget as 504, or every slow dependency hides inside the
// service's 5xx error rate.
func TestGeneratedResponseErrorHandlerMapsExpiredBudget(t *testing.T) {
	t.Parallel()

	options := generatedStrictServerOptions(
		slog.New(slog.DiscardHandler),
		handleGeneratedRequestError(slog.New(slog.DiscardHandler), defaultAuthenticateChallenge),
		nil,
	)

	for _, tc := range []struct {
		name       string
		err        error
		wantStatus int
		wantCode   problem.Code
	}{
		{
			name:       "expired budget",
			err:        context.DeadlineExceeded,
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   problem.CodeGatewayTimeout,
		},
		{
			name:       "wrapped expired budget",
			err:        fmt.Errorf("load order: %w", context.DeadlineExceeded),
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   problem.CodeGatewayTimeout,
		},
		{
			name:       "ordinary failure",
			err:        errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   problem.CodeInternalError,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resp := httptest.NewRecorder()
			options.ResponseErrorHandlerFunc(resp, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil), tc.err)

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Code, tc.wantStatus)
			}
			assertProblemContentType(t, resp.Header())
			assertProblemCode(t, resp, tc.wantCode)
		})
	}
}
