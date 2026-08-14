package httpclient

// profile:outbound-auth-http:start

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/example/go-service-template-rest/internal/infra/telemetry/telemetrytest"
	"go.opentelemetry.io/otel/trace"
)

func TestAttemptAuthorizationPreservesRetryPolicy(t *testing.T) {
	const bearerCanary = "Bearer attempt-authorization-canary"

	recorder := telemetrytest.InstallSpanRecorder(t)
	var resourceAttempts atomic.Int32
	seenAuthorization := make(chan string, 2)
	client := newRetryTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		seenAuthorization <- request.Header.Get("Authorization")
		if resourceAttempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}), RetryPolicy{MaxAttempts: retryTestMaxAttempts, BaseDelay: retryTestBaseDelay})
	t.Cleanup(client.CloseIdleConnections)

	request, err := http.NewRequestWithContext(t.Context(), http.MethodGet, client.BaseURL(), http.NoBody)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	var authorizationAttempts atomic.Int32
	response, err := client.DoWithAuthorization(request, func(attempt *http.Request) error {
		authorizationAttempts.Add(1)
		if !trace.SpanFromContext(attempt.Context()).IsRecording() {
			return errors.New("attempt authorization ran outside generic HTTP instrumentation")
		}
		attempt.Header.Set("Authorization", bearerCanary)
		return nil
	})
	if err != nil {
		t.Fatalf("DoWithAuthorization() error = %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := authorizationAttempts.Load(); got != 2 {
		t.Fatalf("authorization attempts = %d, want one for each of 2 resource attempts", got)
	}
	if got := resourceAttempts.Load(); got != 2 {
		t.Fatalf("resource attempts = %d, want 2", got)
	}
	close(seenAuthorization)
	for authorization := range seenAuthorization {
		if authorization != bearerCanary {
			t.Fatalf("Authorization = %q, want %q", authorization, bearerCanary)
		}
	}
	if got := request.Header.Get("Authorization"); got != "" {
		t.Fatalf("original request Authorization = %q, want empty", got)
	}
	for _, span := range recorder.Ended() {
		snapshot := fmt.Sprint(span.Name(), span.Status(), span.Attributes(), span.Events())
		if strings.Contains(snapshot, bearerCanary) {
			t.Fatalf("generic HTTP span disclosed bearer: %s", snapshot)
		}
	}

	t.Run("authorizer failure is not retried", func(t *testing.T) {
		var calls atomic.Int32
		client := newRetryTestClient(t, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			calls.Add(1)
		}), RetryPolicy{MaxAttempts: retryTestMaxAttempts, BaseDelay: retryTestBaseDelay})
		t.Cleanup(client.CloseIdleConnections)
		request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, client.BaseURL(), http.NoBody)
		if err != nil {
			t.Fatalf("build request: %v", err)
		}
		authorizationFailure := errors.New("closed authorization failure")
		var authorizations atomic.Int32
		response, err := client.DoWithAuthorization(request, func(*http.Request) error {
			authorizations.Add(1)
			return authorizationFailure
		})
		if response != nil {
			if response.Body != nil {
				_, _ = io.Copy(io.Discard, response.Body)
				_ = response.Body.Close()
			}
			t.Fatalf("response = %#v, want nil", response)
		}
		if !errors.Is(err, authorizationFailure) {
			t.Fatalf("DoWithAuthorization() error = %v, want wrapped authorization failure", err)
		}
		if got := authorizations.Load(); got != 1 {
			t.Fatalf("authorization attempts = %d, want 1", got)
		}
		if got := calls.Load(); got != 0 {
			t.Fatalf("resource attempts = %d, want 0", got)
		}
	})
}

// profile:outbound-auth-http:end
