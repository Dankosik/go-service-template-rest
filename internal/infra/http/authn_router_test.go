package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/bearerauthn"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	serviceopenapi "github.com/example/go-service-template-rest/internal/openapi"
	"github.com/example/go-service-template-rest/internal/problem"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/example/go-service-template-rest/internal/waittest"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
)

const secureTestContract = `openapi: 3.0.3
info:
  title: secure test
  version: 1.0.0
paths:
  /secure:
    get:
      operationId: secure
      security:
        - bearerAuth: []
      responses:
        "204":
          description: accepted
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
`

func TestHTTPAuthnBoundary(t *testing.T) {
	spec, err := openapi3.NewLoader().LoadFromData([]byte(secureTestContract))
	if err != nil {
		t.Fatalf("load secure test contract: %v", err)
	}
	if err := spec.Validate(t.Context()); err != nil {
		t.Fatalf("validate secure test contract: %v", err)
	}

	tests := []struct {
		name          string
		kind          bearerauthn.Kind
		err           error
		headers       []string
		wantStatus    int
		wantChallenge string
		wantRetry     string
		wantCode      problem.Code
		wantDetail    string
		wantCalls     int64
	}{
		{
			name:       "valid identity reaches handler",
			headers:    []string{"Bearer opaque-token"},
			wantStatus: http.StatusNoContent,
			wantCalls:  1,
		},
		{
			name:          "missing credential",
			kind:          bearerauthn.KindMissing,
			wantStatus:    http.StatusUnauthorized,
			wantChallenge: "Bearer",
			wantCode:      problem.CodeUnauthorized,
			wantDetail:    "credentials are missing",
		},
		{
			name:          "invalid token",
			kind:          bearerauthn.KindInvalid,
			headers:       []string{"Bearer invalid-token"},
			wantStatus:    http.StatusUnauthorized,
			wantChallenge: `Bearer error="invalid_token"`,
			wantCode:      problem.CodeUnauthorized,
			wantDetail:    "credentials are invalid",
		},
		{
			name:          "malformed credential",
			kind:          bearerauthn.KindMalformed,
			headers:       []string{"Basic opaque-token"},
			wantStatus:    http.StatusBadRequest,
			wantChallenge: `Bearer error="invalid_request"`,
			wantCode:      problem.CodeBadRequest,
			wantDetail:    "authentication credential is malformed",
		},
		{
			name:          "duplicate credential",
			kind:          bearerauthn.KindMalformed,
			headers:       []string{"Bearer one", "Bearer two"},
			wantStatus:    http.StatusBadRequest,
			wantChallenge: `Bearer error="invalid_request"`,
			wantCode:      problem.CodeBadRequest,
			wantDetail:    "authentication credential is malformed",
		},
		{
			name:       "oversized token",
			kind:       bearerauthn.KindOversize,
			headers:    []string{"Bearer " + strings.Repeat("x", bearerauthn.MaxTokenBytes+1)},
			wantStatus: http.StatusRequestHeaderFieldsTooLarge,
			wantCode:   problem.CodeRequestHeaderFieldsTooLarge,
			wantDetail: "authentication credential is too large",
		},
		{
			name:       "unavailable trust requests a bounded retry",
			kind:       bearerauthn.KindUnavailable,
			headers:    []string{"Bearer opaque-token"},
			wantStatus: http.StatusServiceUnavailable,
			wantRetry:  "30",
			wantCode:   problem.CodeServiceUnavailable,
			wantDetail: "authentication trust is unavailable",
		},
		{
			name:       "authentication deadline",
			err:        fmt.Errorf("wait for JWKS refresh: %w", context.DeadlineExceeded),
			headers:    []string{"Bearer opaque-token"},
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   problem.CodeGatewayTimeout,
			wantDetail: "authentication verification did not complete",
		},
		{
			name:       "authentication canceled",
			err:        fmt.Errorf("wait for JWKS refresh: %w", context.Canceled),
			headers:    []string{"Bearer opaque-token"},
			wantStatus: http.StatusGatewayTimeout,
			wantCode:   problem.CodeGatewayTimeout,
			wantDetail: "authentication verification did not complete",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var calls atomic.Int64
			resolve := func(
				_ context.Context,
				input *openapi3filter.AuthenticationInput,
			) (reqctx.Principal, error) {
				request := authenticatedRequest(input)
				if request == nil {
					t.Fatal("authentication input has no request")
				}
				request.Header.Del("Authorization")
				if testCase.err != nil {
					return reqctx.Principal{}, testCase.err
				}
				if testCase.kind != 0 {
					return reqctx.Principal{}, fmt.Errorf(
						"poison parser/provider detail: %w",
						bearerauthn.NewError(testCase.kind),
					)
				}
				return reqctx.Principal{Subject: "opaque-subject"}, nil
			}
			reject := RejectRequest(slog.New(slog.DiscardHandler), "Bearer")
			router := chi.NewRouter()
			router.Use(requestValidator(spec, Authenticated(resolve), reject))
			router.Get("/secure", func(w http.ResponseWriter, request *http.Request) {
				calls.Add(1)
				principal, ok := reqctx.PrincipalFromContext(request.Context())
				if !ok || principal.Subject != "opaque-subject" {
					t.Fatalf("handler principal = (%+v, %v), want opaque subject", principal, ok)
				}
				if request.Header.Get("Authorization") != "" {
					t.Fatal("handler-visible Authorization header was retained")
				}
				w.WriteHeader(http.StatusNoContent)
			})

			response := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/secure", nil)
			for _, value := range testCase.headers {
				request.Header.Add("Authorization", value)
			}
			router.ServeHTTP(response, request)

			if response.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d; body = %s", response.Code, testCase.wantStatus, response.Body.String())
			}
			if got := response.Header().Get("WWW-Authenticate"); got != testCase.wantChallenge {
				t.Fatalf("WWW-Authenticate = %q, want %q", got, testCase.wantChallenge)
			}
			if calls.Load() != testCase.wantCalls {
				t.Fatalf("handler calls = %d, want %d", calls.Load(), testCase.wantCalls)
			}
			if strings.Contains(response.Body.String(), "poison") ||
				strings.Contains(response.Body.String(), "parser") ||
				strings.Contains(response.Body.String(), "provider") {
				t.Fatal("response leaked authentication error detail")
			}
			if got := response.Header().Get("Retry-After"); got != testCase.wantRetry {
				t.Fatalf("Retry-After = %q, want %q", got, testCase.wantRetry)
			}
			if testCase.wantCode != "" {
				assertAuthnProblem(t, response, testCase.wantCode, testCase.wantStatus, testCase.wantDetail)
			}
		})
	}
}

func assertAuthnProblem(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantCode problem.Code,
	wantStatus int,
	wantDetail string,
) {
	t.Helper()
	if got := response.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/problem+json") {
		t.Fatalf("Content-Type = %q, want application/problem+json", got)
	}
	var decoded serviceopenapi.Problem
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("decode authentication problem: %v", err)
	}
	if decoded.Code != string(wantCode) ||
		int(decoded.Status) != wantStatus ||
		decoded.Detail == nil ||
		*decoded.Detail != wantDetail ||
		decoded.Title == "" ||
		decoded.Type == "" {
		t.Fatalf(
			"authentication problem = %+v, want code %q status %d detail %q with stable title/type",
			decoded,
			wantCode,
			wantStatus,
			wantDetail,
		)
	}
}

func TestHTTPAuthnRunsInsideMaxInFlight(t *testing.T) {
	spec, err := openapi3.NewLoader().LoadFromData([]byte(secureTestContract))
	if err != nil {
		t.Fatalf("load secure test contract: %v", err)
	}
	if err := spec.Validate(t.Context()); err != nil {
		t.Fatalf("validate secure test contract: %v", err)
	}

	entered := make(chan struct{}, 1)
	block := make(chan struct{})
	verifier := &blockingVerifier{entered: entered, block: block}
	runtime, err := bearerauthn.New(verifier, nil)
	if err != nil {
		t.Fatalf("bearerauthn.New() error = %v", err)
	}
	reject := RejectRequest(slog.New(slog.DiscardHandler), "Bearer")
	inner := chi.NewRouter()
	inner.Use(requestValidator(spec, Authenticated(runtime.ResolveHTTP), reject))
	inner.Get("/secure", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	router := MaxInFlight(1, ServerLoad{}, inner)

	var first sync.WaitGroup
	first.Go(func() {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/secure", nil)
		request.Header.Set("Authorization", "Bearer token")
		router.ServeHTTP(httptest.NewRecorder(), request)
	})
	waittest.ReceiveSignal(t, entered, 2*time.Second, "first request to enter verifier")

	shed := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/secure", nil)
		request.Header.Set("Authorization", "Bearer token")
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, request)
		shed <- recorder
	}()
	response := waittest.Receive(t, shed, 2*time.Second, "second request to be shed")
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", response.Code, http.StatusServiceUnavailable, response.Body.String())
	}
	assertAuthnProblem(t, response, problem.CodeServiceUnavailable, http.StatusServiceUnavailable, failure.AtCapacityDetail)
	if verifier.calls.Load() != 1 || verifier.peak.Load() != 1 {
		t.Fatalf("verifier calls/peak = %d/%d, want 1/1", verifier.calls.Load(), verifier.peak.Load())
	}
	close(block)
	first.Wait()
}

type blockingVerifier struct {
	entered  chan struct{}
	block    <-chan struct{}
	calls    atomic.Int64
	inFlight atomic.Int64
	peak     atomic.Int64
}

func (v *blockingVerifier) Verify(ctx context.Context, _ string) (bearerauthn.Result, error) {
	v.calls.Add(1)
	current := v.inFlight.Add(1)
	defer v.inFlight.Add(-1)
	for {
		previous := v.peak.Load()
		if current <= previous || v.peak.CompareAndSwap(previous, current) {
			break
		}
	}
	if v.entered != nil {
		select {
		case v.entered <- struct{}{}:
		default:
		}
	}
	if v.block != nil {
		select {
		case <-v.block:
		case <-ctx.Done():
			return bearerauthn.Result{}, fmt.Errorf("wait for test barrier: %w", ctx.Err())
		}
	}
	return bearerauthn.Result{
		Principal: reqctx.Principal{Subject: "opaque-subject"},
		ExpiresAt: time.Now().Add(time.Hour),
	}, nil
}

func (v *blockingVerifier) Close() {}

func TestOpenAPIRuntimeContract(t *testing.T) {
	t.Run("Authn", func(t *testing.T) {
		spec, err := serviceopenapi.GetSpec()
		if err != nil {
			t.Fatalf("load generated OpenAPI: %v", err)
		}
		if len(spec.Security) != 1 {
			t.Fatalf("top-level security = %#v, want exactly one requirement", spec.Security)
		}
		requirement := spec.Security[0]
		scopes, ok := requirement["bearerAuth"]
		if !ok || len(requirement) != 1 || len(scopes) != 0 {
			t.Fatalf("top-level security requirement = %#v, want bearerAuth only", requirement)
		}
		if spec.Components == nil || spec.Components.SecuritySchemes["bearerAuth"] == nil {
			t.Fatal("generated OpenAPI has no bearerAuth security scheme")
		}

		for _, path := range []string{"/health/live", "/health/ready"} {
			item := spec.Paths.Value(path)
			if item == nil || item.Get == nil || item.Get.Security == nil || len(*item.Get.Security) != 0 {
				t.Fatalf("%s GET security = %#v, want explicit anonymous override", path, item)
			}
		}

		router := mustNewRouter(
			t,
			slog.New(slog.DiscardHandler),
			Handlers{Health: health.New()},
			telemetry.New(),
			RouterConfig{},
		)
		for _, path := range []string{"/health/live", "/health/ready"} {
			response := doRequest(router, http.MethodGet, path)
			if response.Code != http.StatusOK {
				t.Fatalf("%s status = %d, want %d", path, response.Code, http.StatusOK)
			}
		}
	})
}
