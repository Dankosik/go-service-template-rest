package httpx

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/oidcjwt"
	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
)

func TestHTTPIdempotencyKeyContract(t *testing.T) {
	var authorizations atomic.Int64
	var admissions atomic.Int64
	var handlers atomic.Int64
	operation := testIdempotencyOperation()
	operation.Authorize = func(context.Context, *http.Request) (httpidempotency.Scope, bool) {
		authorizations.Add(1)
		return httpidempotency.Scope{Authority: "authority-a", OperationID: "createWidget", APIVersion: "v1"}, true
	}
	operation.Admit = func(context.Context, httpidempotency.Scope) httpidempotency.Decision {
		admissions.Add(1)
		return httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}
	}
	handler := newIdempotencyTestHandler(t, operation, authenticatedTestRequest, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.Add(1)
		if _, ok := IdempotencyAttemptFromContext(r.Context()); !ok {
			t.Fatal("handler did not receive idempotency attempt")
		}
		w.WriteHeader(http.StatusCreated)
	}))

	for _, values := range [][]string{nil, {""}, {"a,b"}, {"\"a\""}, {"a b"}, {"a\tb"}, {"é"}, {"same", "same"}, {strings.Repeat("a", 65)}} {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
		request.Header.Set("Authorization", "Bearer test")
		for _, value := range values {
			request.Header.Add(httpidempotency.Header, value)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("key values %q status = %d, want 400", values, response.Code)
		}
		assertProblemCode(t, response, problem.CodeBadRequest)
		if got := response.Header().Get("Retry-After"); got != "" {
			t.Fatalf("key values %q Retry-After = %q, want empty", values, got)
		}
		if strings.Contains(response.Body.String(), "same") || strings.Contains(response.Body.String(), "aaaa") {
			t.Fatalf("key values %q leaked into Problem", values)
		}
	}
	if admissions.Load() != 0 || handlers.Load() != 0 {
		t.Fatalf("invalid keys reached admission=%d handler=%d", admissions.Load(), handlers.Load())
	}
	if authorizations.Load() != 9 {
		t.Fatalf("authorization calls = %d, want 9 validly authenticated requests", authorizations.Load())
	}

	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
	request.Header.Set("Authorization", "Bearer test")
	request.Header.Set(httpidempotency.Header, strings.Repeat("a", 64))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("boundary key status = %d, want 201", response.Code)
	}
	if handlers.Load() != 1 {
		t.Fatalf("handler calls = %d, want 1", handlers.Load())
	}
}

func TestHTTPIdempotencyAuthorizationAndAdmissionOrder(t *testing.T) {
	var authorizations atomic.Int64
	var admissions atomic.Int64
	var handlers atomic.Int64
	var authorityA atomic.Int64
	var authorityB atomic.Int64
	operation := testIdempotencyOperation()
	operation.Authorize = func(_ context.Context, r *http.Request) (httpidempotency.Scope, bool) {
		authorizations.Add(1)
		if r.Header.Get("X-Deny") == "true" {
			return httpidempotency.Scope{}, false
		}
		authority := r.Header.Get("X-Authority")
		scope := httpidempotency.Scope{Authority: authority, OperationID: "createWidget", APIVersion: "v1"}
		switch r.Header.Get("X-Scope-Mismatch") {
		case "operation":
			scope.OperationID = "otherOperation"
		case "version":
			scope.APIVersion = "v2"
		}
		return scope, true
	}
	operation.Admit = func(_ context.Context, scope httpidempotency.Scope) httpidempotency.Decision {
		admissions.Add(1)
		switch scope.Authority {
		case "authority-a":
			authorityA.Add(1)
			return httpidempotency.Decision{Outcome: httpidempotency.OutcomeRateLimited}
		case "authority-b":
			authorityB.Add(1)
			return httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}
		default:
			return httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnavailable}
		}
	}
	handler := newIdempotencyTestHandler(t, operation, authenticatedTestRequest, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.Add(1)
		attempt, ok := IdempotencyAttemptFromContext(r.Context())
		if !ok {
			t.Fatal("handler did not receive attempt")
		}
		if attempt.Scope.Authority != "authority-b" || attempt.Key != "same-key" {
			t.Fatalf("attempt = %+v, want authority-b/same-key", attempt)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	headerTooLarge := newIdempotencyTestHandler(t, operation, func(context.Context, *openapi3filter.AuthenticationInput) error {
		return oidcjwt.NewError(oidcjwt.KindOversize)
	}, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("header-too-large request reached endpoint")
	}))
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
	request.Header.Set("Authorization", "Bearer test")
	request.Header.Set(httpidempotency.Header, "bad,key")
	response := httptest.NewRecorder()
	headerTooLarge.ServeHTTP(response, request)
	if response.Code != http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("header-too-large status = %d, want 431", response.Code)
	}
	assertProblemCode(t, response, problem.CodeRequestHeaderFieldsTooLarge)
	if authorizations.Load() != 0 || admissions.Load() != 0 || handlers.Load() != 0 {
		t.Fatal("header-too-large request reached idempotency callbacks")
	}

	unauthenticated := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
	unauthenticated.Header.Set(httpidempotency.Header, "bad,key")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, unauthenticated)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated malformed key status = %d, want 401", response.Code)
	}
	if authorizations.Load() != 0 || admissions.Load() != 0 || handlers.Load() != 0 {
		t.Fatal("unauthenticated request reached idempotency callbacks")
	}

	forbidden := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
	forbidden.Header.Set("Authorization", "Bearer test")
	forbidden.Header.Set("X-Deny", "true")
	forbidden.Header.Set(httpidempotency.Header, "bad,key")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, forbidden)
	if response.Code != http.StatusForbidden {
		t.Fatalf("forbidden malformed key status = %d, want 403", response.Code)
	}
	assertProblemCode(t, response, problem.CodeForbidden)

	forbiddenMissing := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
	forbiddenMissing.Header.Set("Authorization", "Bearer test")
	forbiddenMissing.Header.Set("X-Deny", "true")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, forbiddenMissing)
	if response.Code != http.StatusForbidden {
		t.Fatalf("forbidden missing key status = %d, want 403", response.Code)
	}
	assertProblemCode(t, response, problem.CodeForbidden)
	if admissions.Load() != 0 || handlers.Load() != 0 {
		t.Fatalf("forbidden missing key reached admission=%d handler=%d", admissions.Load(), handlers.Load())
	}

	malformed := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
	malformed.Header.Set("Authorization", "Bearer test")
	malformed.Header.Set("X-Authority", "authority-a")
	malformed.Header.Set(httpidempotency.Header, "bad,key")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, malformed)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("authorized malformed key status = %d, want 400", response.Code)
	}
	if admissions.Load() != 0 {
		t.Fatalf("malformed key reached admission %d times", admissions.Load())
	}

	for _, mismatch := range []string{"operation", "version"} {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
		request.Header.Set("Authorization", "Bearer test")
		request.Header.Set("X-Authority", "authority-a")
		request.Header.Set("X-Scope-Mismatch", mismatch)
		request.Header.Set(httpidempotency.Header, "bad,key")
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusForbidden {
			t.Fatalf("%s scope mismatch status = %d, want 403", mismatch, response.Code)
		}
		assertProblemCode(t, response, problem.CodeForbidden)
	}
	if admissions.Load() != 0 || handlers.Load() != 0 {
		t.Fatalf("scope mismatch reached admission=%d handler=%d", admissions.Load(), handlers.Load())
	}

	for authority, wantStatus := range map[string]int{"authority-a": http.StatusTooManyRequests, "authority-b": http.StatusCreated} {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
		request.Header.Set("Authorization", "Bearer test")
		request.Header.Set("X-Authority", authority)
		request.Header.Set(httpidempotency.Header, "same-key")
		response = httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != wantStatus {
			t.Fatalf("%s status = %d, want %d", authority, response.Code, wantStatus)
		}
	}
	if authorityA.Load() != 1 || authorityB.Load() != 1 || handlers.Load() != 1 {
		t.Fatalf("authority calls A=%d B=%d handler=%d", authorityA.Load(), authorityB.Load(), handlers.Load())
	}

	unavailable := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
	unavailable.Header.Set("Authorization", "Bearer test")
	unavailable.Header.Set("X-Authority", "authority-c")
	unavailable.Header.Set(httpidempotency.Header, "same-key")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, unavailable)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable admission status = %d, want 503", response.Code)
	}
	assertProblemCode(t, response, problem.CodeIdempotencyUnavailable)
	if response.Header().Get("Retry-After") == "" {
		t.Fatal("unavailable admission omitted Retry-After")
	}
	if handlers.Load() != 1 {
		t.Fatalf("unavailable admission reached endpoint %d times, want 1 total", handlers.Load())
	}
}

func TestHTTPIdempotencyReplayRendering(t *testing.T) {
	result := httpidempotency.Result{
		Status:    http.StatusCreated,
		MediaType: "application/json",
		Codec:     "create-widget/v1",
		Headers:   http.Header{"Location": {"/widgets/w_1"}},
		Payload:   []byte(`{"id":"w_1"}`),
	}
	handler := RequestCorrelation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !writeIdempotencyDecision(w, r, testIdempotencyOperation().Contract, httpidempotency.Decision{Outcome: httpidempotency.OutcomeReplay, Result: &result}) {
			t.Fatal("replay continued into execution")
		}
	}))
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
	request.Header.Set(requestIDHeader, "fresh-request")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || response.Body.String() != `{"id":"w_1"}` {
		t.Fatalf("replay = %d %q", response.Code, response.Body.String())
	}
	if response.Header().Get("Location") != "/widgets/w_1" || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("replay headers = %#v", response.Header())
	}
	if response.Header().Get(requestIDHeader) != "fresh-request" || response.Header().Get("Retry-After") != "" || response.Header().Get("Set-Cookie") != "" {
		t.Fatalf("replay transport headers = %#v", response.Header())
	}

	result.Headers.Set("Set-Cookie", "replayed-secret")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusInternalServerError || response.Header().Get("Set-Cookie") != "" {
		t.Fatalf("invalid replay = %d %#v, want sanitized 500 without Set-Cookie", response.Code, response.Header())
	}
}

func TestHTTPIdempotencyProblemAndRedaction(t *testing.T) {
	contract := testIdempotencyOperation().Contract
	for _, testCase := range []struct {
		name       string
		decision   httpidempotency.Decision
		wantStatus int
		wantCode   problem.Code
		wantRetry  bool
	}{
		{"mismatch", httpidempotency.Decision{Outcome: httpidempotency.OutcomeMismatch}, http.StatusUnprocessableEntity, problem.CodeIdempotencyKeyMismatch, false},
		{"in progress", httpidempotency.Decision{Outcome: httpidempotency.OutcomeInProgress}, http.StatusConflict, problem.CodeIdempotencyInProgress, true},
		{"expired", httpidempotency.Decision{Outcome: httpidempotency.OutcomeExpired}, http.StatusConflict, problem.CodeIdempotencyKeyExpired, false},
		{"rate limited", httpidempotency.Decision{Outcome: httpidempotency.OutcomeRateLimited}, http.StatusTooManyRequests, problem.CodeTooManyRequests, true},
		{"unavailable", httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnavailable}, http.StatusServiceUnavailable, problem.CodeIdempotencyUnavailable, true},
		{"unknown", httpidempotency.Decision{Outcome: httpidempotency.OutcomeUnknown}, http.StatusServiceUnavailable, problem.CodeIdempotencyOutcomeUnknown, true},
		{"too large", httpidempotency.Decision{Outcome: httpidempotency.OutcomeResultTooLarge}, http.StatusInternalServerError, problem.CodeIdempotencyResultTooLarge, false},
		{"integrity", httpidempotency.Decision{Outcome: httpidempotency.OutcomeIntegrityConflict}, http.StatusInternalServerError, problem.CodeInternalError, false},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			handler := RequestCorrelation(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				writeIdempotencyDecision(w, r, contract, testCase.decision)
			}))
			request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
			request.Header.Set(requestIDHeader, "fresh-correlation")
			request.Header.Set(httpidempotency.Header, "sentinel-key-7-authority-9-fingerprint-2-sql-4")
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d", response.Code, testCase.wantStatus)
			}
			assertProblemCode(t, response, testCase.wantCode)
			if (response.Header().Get("Retry-After") != "") != testCase.wantRetry {
				t.Fatalf("Retry-After = %q, want present=%t", response.Header().Get("Retry-After"), testCase.wantRetry)
			}
			for _, sentinel := range []string{"sentinel-key-7", "authority-9", "fingerprint-2", "sql-4"} {
				if strings.Contains(response.Body.String(), sentinel) || strings.Contains(response.Header().Get("Retry-After"), sentinel) {
					t.Fatalf("response leaked %q", sentinel)
				}
			}
			if response.Header().Get(requestIDHeader) != "fresh-correlation" {
				t.Fatalf("request ID = %q, want fresh-correlation", response.Header().Get(requestIDHeader))
			}
		})
	}
}

func newIdempotencyTestHandler(
	tb testing.TB,
	operation IdempotencyOperation,
	authenticate openapi3filter.AuthenticationFunc,
	terminal http.Handler,
) http.Handler {
	tb.Helper()
	spec := idempotencyTestSpec(tb)
	envelope, err := newIdempotencyEnvelope(spec, []IdempotencyOperation{operation})
	if err != nil {
		tb.Fatalf("newIdempotencyEnvelope() error = %v", err)
	}
	router := chi.NewRouter()
	router.Use(captureIdempotencyKey)
	router.Use(envelope.prepareValidation)
	router.Use(requestValidator(spec, authenticate, RejectRequest(slog.New(slog.DiscardHandler), "Bearer")))
	router.Use(envelope.enforce)
	router.Post("/widgets", terminal.ServeHTTP)
	return RequestCorrelation(router)
}

func authenticatedTestRequest(_ context.Context, input *openapi3filter.AuthenticationInput) error {
	if input == nil || input.RequestValidationInput == nil || input.RequestValidationInput.Request == nil || input.RequestValidationInput.Request.Header.Get("Authorization") == "" {
		return errors.New("credentials are missing")
	}
	return nil
}
