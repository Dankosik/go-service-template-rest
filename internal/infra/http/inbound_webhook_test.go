// profile:inbound-webhooks-standard:start
package httpx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/inboundwebhook"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/problem"
)

type recordingReceiver struct {
	mu      sync.Mutex
	calls   int
	bodies  [][]byte
	result  inboundwebhook.Result
	err     error
	block   chan struct{}
	started chan struct{}
}

func (r *recordingReceiver) Receive(ctx context.Context, delivery inboundwebhook.Delivery) (inboundwebhook.Result, error) {
	r.mu.Lock()
	r.calls++
	r.bodies = append(r.bodies, append([]byte(nil), delivery.Body...))
	block, started := r.block, r.started
	r.mu.Unlock()
	if started != nil {
		close(started)
	}
	if block != nil {
		select {
		case <-block:
		case <-ctx.Done():
			return inboundwebhook.Result{}, fmt.Errorf("inbound webhook receive: %w", ctx.Err())
		}
	}
	return r.result, r.err
}

func inboundRouter(t *testing.T, receiver inboundwebhook.Receiver, cfg RouterConfig) http.Handler {
	t.Helper()
	return mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Health:         health.New(),
		API:            unimplementedAPI{},
		InboundWebhook: receiver,
	}, telemetry.New(), cfg)
}

func inboundRequest(body string) *http.Request {
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhooks/orders", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Webhook-Id", "msg_123")
	req.Header.Set("Webhook-Timestamp", "1700000000")
	req.Header.Set("Webhook-Signature", "v1,jUcl6cc4RhnPU/D4RhXcoyQYBvOxqIsONY9102iBndo=")
	return req
}

func TestInboundWebhookRawDispatch(t *testing.T) {
	t.Parallel()

	receiver := &recordingReceiver{result: inboundwebhook.Result{Outcome: inboundwebhook.OutcomeAccepted}}
	handler := inboundRouter(t, receiver, RouterConfig{})
	whitespace := `{ "hello" : "world" }`
	invalid := `{`
	for _, body := range []string{whitespace, invalid} {
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, inboundRequest(body))
		if resp.Code != http.StatusNoContent {
			t.Fatalf("status = %d body = %q", resp.Code, resp.Body.String())
		}
	}
	receiver.mu.Lock()
	defer receiver.mu.Unlock()
	if receiver.calls != 2 || !bytes.Equal(receiver.bodies[0], []byte(whitespace)) || !bytes.Equal(receiver.bodies[1], []byte(invalid)) {
		t.Fatalf("calls=%d bodies=%q", receiver.calls, receiver.bodies)
	}

	live := doRequest(handler, http.MethodGet, "/health/live")
	if live.Code != http.StatusOK {
		t.Fatalf("ordinary operation status = %d", live.Code)
	}
}

func TestInboundWebhookRequestValidation(t *testing.T) {
	t.Parallel()

	receiver := &recordingReceiver{result: inboundwebhook.Result{Outcome: inboundwebhook.OutcomeAccepted}}
	handler := inboundRouter(t, receiver, RouterConfig{})
	cases := []struct {
		name    string
		path    string
		headers map[string][]string
		ctype   string
		want    int
	}{
		{name: "missing id", path: "/webhooks/orders", headers: map[string][]string{"Webhook-Timestamp": {"1700000000"}, "Webhook-Signature": {"v1,sig"}}, ctype: "application/json", want: http.StatusBadRequest},
		{name: "duplicate id", path: "/webhooks/orders", headers: map[string][]string{"Webhook-Id": {"a", "b"}, "Webhook-Timestamp": {"1700000000"}, "Webhook-Signature": {"v1,sig"}}, ctype: "application/json", want: http.StatusBadRequest},
		{name: "wrong type", path: "/webhooks/orders", headers: map[string][]string{"Webhook-Id": {"msg_123"}, "Webhook-Timestamp": {"1700000000"}, "Webhook-Signature": {"v1,sig"}}, ctype: "text/plain", want: http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, tc.path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", tc.ctype)
			for key, values := range tc.headers {
				req.Header[http.CanonicalHeaderKey(key)] = values
			}
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)
			if resp.Code != tc.want {
				t.Fatalf("status = %d want %d body=%q", resp.Code, tc.want, resp.Body.String())
			}
			if strings.Contains(resp.Body.String(), "msg_123") || strings.Contains(resp.Body.String(), "v1,sig") {
				t.Fatalf("response leaked submitted value: %s", resp.Body.String())
			}
		})
	}
	receiver.mu.Lock()
	if receiver.calls != 0 {
		t.Fatalf("receiver calls = %d, want none for malformed requests", receiver.calls)
	}
	receiver.mu.Unlock()

	unknown := &recordingReceiver{result: inboundwebhook.Result{Outcome: inboundwebhook.OutcomeUnknownEndpoint}}
	unknownHandler := inboundRouter(t, unknown, RouterConfig{})
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/webhooks/missing", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Webhook-Id", "msg_123")
	req.Header.Set("Webhook-Timestamp", "1700000000")
	req.Header.Set("Webhook-Signature", "v1,sig")
	resp := httptest.NewRecorder()
	unknownHandler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("unknown endpoint status = %d body=%q", resp.Code, resp.Body.String())
	}
	if strings.Contains(resp.Body.String(), "msg_123") || strings.Contains(resp.Body.String(), "missing") {
		t.Fatalf("response leaked submitted value: %s", resp.Body.String())
	}
}

func TestInboundWebhookAdmissionBeforeDurableWork(t *testing.T) {
	t.Parallel()

	receiver := &recordingReceiver{result: inboundwebhook.Result{Outcome: inboundwebhook.OutcomeAccepted}}
	limit := inboundRouter(t, receiver, RouterConfig{MaxBodyBytes: 8})
	exact := httptest.NewRecorder()
	limit.ServeHTTP(exact, inboundRequest("12345678"))
	if exact.Code != http.StatusNoContent {
		t.Fatalf("exact limit status = %d", exact.Code)
	}
	over := httptest.NewRecorder()
	limit.ServeHTTP(over, inboundRequest("123456789"))
	if over.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("over limit status = %d", over.Code)
	}

	blocked := &recordingReceiver{
		result:  inboundwebhook.Result{Outcome: inboundwebhook.OutcomeAccepted},
		block:   make(chan struct{}),
		started: make(chan struct{}),
	}
	shed := inboundRouter(t, blocked, RouterConfig{MaxInFlight: 1})
	go func() {
		rec := httptest.NewRecorder()
		shed.ServeHTTP(rec, inboundRequest(`{}`))
	}()
	<-blocked.started
	second := httptest.NewRecorder()
	shed.ServeHTTP(second, inboundRequest(`{}`))
	close(blocked.block)
	if second.Code != http.StatusServiceUnavailable || second.Header().Get("Retry-After") == "" {
		t.Fatalf("shed status = %d retry=%q", second.Code, second.Header().Get("Retry-After"))
	}

	rateReceiver := &recordingReceiver{result: inboundwebhook.Result{Outcome: inboundwebhook.OutcomeAccepted}}
	limited := inboundRouter(t, rateReceiver, RouterConfig{
		RateLimit:    rejectAllLimiter{},
		RateLimitKey: func(*http.Request) string { return "caller" },
	})
	rate := httptest.NewRecorder()
	limited.ServeHTTP(rate, inboundRequest(`{}`))
	if rate.Code != http.StatusTooManyRequests || rate.Header().Get("Retry-After") == "" {
		t.Fatalf("rate status = %d retry=%q", rate.Code, rate.Header().Get("Retry-After"))
	}
	rateReceiver.mu.Lock()
	defer rateReceiver.mu.Unlock()
	if rateReceiver.calls != 0 {
		t.Fatalf("rate-limited request reached receiver: %d", rateReceiver.calls)
	}
}

type rejectAllLimiter struct{}

func (rejectAllLimiter) Allow(context.Context, string) (bool, time.Duration) {
	return false, time.Second
}

func TestInboundWebhookResponseContract(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		result     inboundwebhook.Result
		err        error
		wantStatus int
		wantRetry  bool
	}{
		{name: "accepted", result: inboundwebhook.Result{Outcome: inboundwebhook.OutcomeAccepted}, wantStatus: http.StatusNoContent},
		{name: "duplicate", result: inboundwebhook.Result{Outcome: inboundwebhook.OutcomeDuplicate}, wantStatus: http.StatusNoContent},
		{name: "rejected", result: inboundwebhook.Result{Outcome: inboundwebhook.OutcomeRejected}, wantStatus: http.StatusBadRequest},
		{name: "unknown", result: inboundwebhook.Result{Outcome: inboundwebhook.OutcomeUnknownEndpoint}, wantStatus: http.StatusNotFound},
		{name: "conflict", result: inboundwebhook.Result{Outcome: inboundwebhook.OutcomeConflict}, wantStatus: http.StatusConflict},
		{name: "unavailable", result: inboundwebhook.Result{Outcome: inboundwebhook.OutcomeUnavailable}, wantStatus: http.StatusServiceUnavailable, wantRetry: true},
		{name: "unavailable err", err: inboundwebhook.ErrUnavailable, wantStatus: http.StatusServiceUnavailable, wantRetry: true},
		{name: "unexpected", err: errors.New("sql canary"), wantStatus: http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler := inboundRouter(t, &recordingReceiver{result: tc.result, err: tc.err}, RouterConfig{})
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, inboundRequest(`{}`))
			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d want %d body=%q", resp.Code, tc.wantStatus, resp.Body.String())
			}
			if tc.wantStatus == http.StatusNoContent && resp.Body.Len() != 0 {
				t.Fatalf("204 body = %q", resp.Body.String())
			}
			if tc.wantRetry && resp.Header().Get("Retry-After") == "" {
				t.Fatal("missing Retry-After")
			}
			if strings.Contains(resp.Body.String(), "sql canary") {
				t.Fatalf("response leaked canary: %s", resp.Body.String())
			}
			if tc.wantStatus >= 400 {
				assertProblemCode(t, resp, map[int]problem.Code{
					400: problem.CodeBadRequest,
					404: problem.CodeNotFound,
					409: problem.CodeConflict,
					500: problem.CodeInternalError,
					503: problem.CodeServiceUnavailable,
				}[tc.wantStatus])
			}
		})
	}

	t.Run("timeout", func(t *testing.T) {
		blocked := &recordingReceiver{block: make(chan struct{}), started: make(chan struct{})}
		handler := inboundRouter(t, blocked, RouterConfig{RequestTimeout: 20 * time.Millisecond})
		resp := httptest.NewRecorder()
		done := make(chan struct{})
		go func() {
			handler.ServeHTTP(resp, inboundRequest(`{}`))
			close(done)
		}()
		<-blocked.started
		<-done
		close(blocked.block)
		if resp.Code != http.StatusGatewayTimeout {
			t.Fatalf("status = %d want 504 body=%q", resp.Code, resp.Body.String())
		}
	})
	_ = io.Discard
}

// profile:inbound-webhooks-standard:end
