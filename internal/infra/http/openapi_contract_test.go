package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/api"
	"github.com/example/go-service-template-rest/internal/app/health"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/getkin/kin-openapi/routers/gorillamux"
)

func TestOpenAPIRuntimeContractEndpoints(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.DiscardHandler)
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
	}, telemetry.New(), RouterConfig{})

	testCases := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantBody   string
	}{
		{
			name:       "ping",
			method:     http.MethodGet,
			path:       "/api/v1/ping",
			wantStatus: http.StatusOK,
			wantBody:   "pong",
		},
		{
			name:       "health live",
			method:     http.MethodGet,
			path:       "/health/live",
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
		{
			name:       "health ready",
			method:     http.MethodGet,
			path:       "/health/ready",
			wantStatus: http.StatusOK,
			wantBody:   "ok",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resp := doRequest(h, tc.method, tc.path)

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d", resp.Code, tc.wantStatus)
			}
			if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/plain") {
				t.Fatalf("content type = %q, want prefix %q", got, "text/plain")
			}
			if tc.wantBody != "" && resp.Body.String() != tc.wantBody {
				t.Fatalf("body = %q, want %q", resp.Body.String(), tc.wantBody)
			}
		})
	}
}

// TEMPLATE EXAMPLE: delete this test with the synthetic OpenAPI operation,
// or replace its cases with the real operation's boundary contract.
func TestOpenAPIRuntimeContractTemplateExample(t *testing.T) {
	t.Parallel()

	h := mustNewRouter(
		t,
		slog.New(slog.DiscardHandler),
		Handlers{Health: health.New()},
		telemetry.New(),
		RouterConfig{},
	)

	t.Run("accepts validated path query and body", func(t *testing.T) {
		t.Parallel()

		resp := doJSONRequest(h, http.MethodPost, "/api/v1/template-example/demo?copies=2", `{"message":"hello"}`)

		if resp.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %q", resp.Code, http.StatusOK, resp.Body.String())
		}
		var body api.TemplateExampleResponse
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Slug != "demo" {
			t.Fatalf("slug = %q, want %q", body.Slug, "demo")
		}
		if got, want := body.Messages, []string{"hello", "hello"}; !slices.Equal(got, want) {
			t.Fatalf("messages = %v, want %v", got, want)
		}
	})

	for _, tc := range []struct {
		name string
		path string
		body string
	}{
		{
			name: "invalid path",
			path: "/api/v1/template-example/INVALID?copies=1",
			body: `{"message":"hello"}`,
		},
		{
			name: "invalid query",
			path: "/api/v1/template-example/demo?copies=4",
			body: `{"message":"hello"}`,
		},
		{
			name: "invalid body",
			path: "/api/v1/template-example/demo?copies=1",
			body: `{"message":""}`,
		},
		{
			name: "unknown body field",
			path: "/api/v1/template-example/demo?copies=1",
			body: `{"message":"hello","unknown":true}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			resp := doJSONRequest(h, http.MethodPost, tc.path, tc.body)

			if resp.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body = %q", resp.Code, http.StatusBadRequest, resp.Body.String())
			}
			assertProblemContentType(t, resp.Header())
			assertProblemCode(t, resp, problemCodeBadRequest)
		})
	}
}

func TestOpenAPIRuntimeContractReadinessUnavailable(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.DiscardHandler)
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(failingProbe{name: "db", err: errors.New("down")}),
	}, telemetry.New(), RouterConfig{})

	resp := doRequest(h, http.MethodGet, "/health/ready")

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
	}
	if body := resp.Body.String(); body != "not ready" {
		t.Fatalf("body = %q, want %q", body, "not ready")
	}
}

func TestOpenAPIRuntimeContractReadinessUnavailableWhenDraining(t *testing.T) {
	t.Parallel()

	healthSvc := health.New()
	healthSvc.StartDrain()

	log := slog.New(slog.DiscardHandler)
	h := mustNewRouter(t, log, Handlers{
		Health: healthSvc,
	}, telemetry.New(), RouterConfig{})

	resp := doRequest(h, http.MethodGet, "/health/ready")

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
	}
	if body := resp.Body.String(); body != "not ready" {
		t.Fatalf("body = %q, want %q", body, "not ready")
	}
}

func TestOpenAPIRuntimeContractReadinessUnavailableBeforeAdmission(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.DiscardHandler)
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
		ReadinessGate: func(context.Context) error {
			return errors.New("startup admission is not ready")
		},
	}, telemetry.New(), RouterConfig{})

	resp := doRequest(h, http.MethodGet, "/health/ready")

	if resp.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusServiceUnavailable)
	}
	if body := resp.Body.String(); body != "not ready" {
		t.Fatalf("body = %q, want %q", body, "not ready")
	}
}

func TestOpenAPIRuntimeContractWrongHealthcheckPathRejected(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.DiscardHandler)
	h := mustNewRouter(t, log, Handlers{
		Health: health.New(),
	}, telemetry.New(), RouterConfig{})

	// Deployment admission must fail deterministically when an unknown health path is used.
	resp := doRequest(h, http.MethodGet, "/health")

	if resp.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNotFound)
	}
	assertProblemContentType(t, resp.Header())
}

func TestOpenAPIRuntimeContractRequiresRouterDependencies(t *testing.T) {
	t.Parallel()

	log := slog.New(slog.DiscardHandler)

	testCases := []struct {
		name     string
		log      *slog.Logger
		handlers Handlers
		metrics  *telemetry.Metrics
		cfg      RouterConfig
		wantErr  string
	}{
		{
			name:     "missing logger",
			handlers: Handlers{Health: health.New(), ReadinessGate: func(context.Context) error { return nil }},
			metrics:  telemetry.New(),
			cfg:      RouterConfig{MaxBodyBytes: testRouterMaxBodyBytes, ReadinessTimeout: time.Second},
			wantErr:  "logger is required",
		},
		{
			name:    "missing health",
			log:     log,
			metrics: telemetry.New(),
			cfg:     RouterConfig{MaxBodyBytes: testRouterMaxBodyBytes, ReadinessTimeout: time.Second},
			handlers: Handlers{
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "health service is required",
		},
		{
			name:    "missing readiness gate",
			log:     log,
			metrics: telemetry.New(),
			cfg:     RouterConfig{MaxBodyBytes: testRouterMaxBodyBytes, ReadinessTimeout: time.Second},
			handlers: Handlers{
				Health: health.New(),
			},
			wantErr: "readiness gate is required",
		},
		{
			name: "missing metrics",
			log:  log,
			cfg:  RouterConfig{MaxBodyBytes: testRouterMaxBodyBytes, ReadinessTimeout: time.Second},
			handlers: Handlers{
				Health:        health.New(),
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "metrics is required",
		},
		{
			name:    "missing readiness timeout",
			log:     log,
			metrics: telemetry.New(),
			cfg:     RouterConfig{MaxBodyBytes: testRouterMaxBodyBytes},
			handlers: Handlers{
				Health:        health.New(),
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "readiness timeout must be > 0",
		},
		{
			name:    "missing max body bytes",
			log:     log,
			metrics: telemetry.New(),
			cfg:     RouterConfig{ReadinessTimeout: time.Second},
			handlers: Handlers{
				Health:        health.New(),
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "max body bytes must be > 0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			handler, err := NewRouter(tc.log, tc.handlers, tc.metrics, tc.cfg)
			if err == nil {
				t.Fatalf("NewRouter() error = nil, want %q", tc.wantErr)
			}
			if handler != nil {
				t.Fatalf("NewRouter() handler = %T, want nil on error", handler)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("NewRouter() error = %v, want to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestOpenAPIRuntimeContractOperationsDeclareSecurityDecisions(t *testing.T) {
	t.Parallel()

	swagger := mustOpenAPISwagger(t)

	for path, item := range swagger.Paths.Map() {
		if item == nil {
			continue
		}
		for method, operation := range item.Operations() {
			t.Run(method+" "+path, func(t *testing.T) {
				t.Parallel()

				decision, err := operationSecurityDecision(operation)
				if err != nil {
					t.Fatalf("security decision: %v", err)
				}

				switch decision.exposure {
				case securityExposurePublic:
					if operationHasRealSecurity(swagger, operation) {
						t.Fatalf("%s operation declares real security while marked %q", operation.OperationID, decision.exposure)
					}
				case securityExposureProtected:
					if !operationHasRealSecurity(swagger, operation) {
						t.Fatalf("%s operation is protected but has no real OpenAPI security requirement", operation.OperationID)
					}
					for _, status := range []string{"401", "403"} {
						if !operationHasProblemResponse(swagger, operation, status) {
							t.Fatalf("%s operation is protected but lacks %s application/problem+json response", operation.OperationID, status)
						}
					}
				case securityExposureBlocked:
				default:
					t.Fatalf("exposure = %q, want one of %q, %q, %q", decision.exposure, securityExposurePublic, securityExposureProtected, securityExposureBlocked)
				}
			})
		}
	}
}

// TestOpenAPIRuntimeContractResponsesMatchSpec replays one request per
// reachable operation/status pair and validates the recorded response —
// status, Content-Type, and body schema — against the embedded OpenAPI spec.
// Generated Visit* responses are contract-shaped by construction; this closes
// the remaining gap for the hand-written problem writer and text responses.
func TestOpenAPIRuntimeContractResponsesMatchSpec(t *testing.T) {
	t.Parallel()

	spec := mustOpenAPISwagger(t)
	// Mirror the runtime request validator, which disables host matching via
	// Options.DoNotValidateServers: route lookup must not depend on servers.
	spec.Servers = nil
	specRouter, err := gorillamux.NewRouter(spec)
	if err != nil {
		t.Fatalf("gorillamux.NewRouter() error = %v", err)
	}

	log := slog.New(slog.DiscardHandler)
	ready := mustNewRouter(t, log, Handlers{Health: health.New()}, telemetry.New(), RouterConfig{})
	notReady := mustNewRouter(t, log, Handlers{
		Health: health.New(failingProbe{name: "db", err: errors.New("down")}),
	}, telemetry.New(), RouterConfig{})
	tinyBodyLimit := mustNewRouter(t, log, Handlers{Health: health.New()}, telemetry.New(), RouterConfig{MaxBodyBytes: 1})

	testCases := []struct {
		name       string
		handler    http.Handler
		method     string
		target     string
		body       string
		wantStatus int
	}{
		{
			name:       "ping 200",
			handler:    ready,
			method:     http.MethodGet,
			target:     "/api/v1/ping",
			wantStatus: http.StatusOK,
		},
		{
			name:       "health live 200",
			handler:    ready,
			method:     http.MethodGet,
			target:     "/health/live",
			wantStatus: http.StatusOK,
		},
		{
			name:       "health ready 200",
			handler:    ready,
			method:     http.MethodGet,
			target:     "/health/ready",
			wantStatus: http.StatusOK,
		},
		{
			name:       "health ready 503",
			handler:    notReady,
			method:     http.MethodGet,
			target:     "/health/ready",
			wantStatus: http.StatusServiceUnavailable,
		},
		{
			name:       "template example 200",
			handler:    ready,
			method:     http.MethodPost,
			target:     "/api/v1/template-example/demo?copies=2",
			body:       `{"message":"hello"}`,
			wantStatus: http.StatusOK,
		},
		{
			name:       "template example 400 problem",
			handler:    ready,
			method:     http.MethodPost,
			target:     "/api/v1/template-example/demo?copies=4",
			body:       `{"message":"hello"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "template example 413 problem",
			handler:    tinyBodyLimit,
			method:     http.MethodPost,
			target:     "/api/v1/template-example/demo?copies=1",
			body:       `{"message":"hello"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var resp *httptest.ResponseRecorder
			if tc.body == "" {
				resp = doRequest(tc.handler, tc.method, tc.target)
			} else {
				resp = doJSONRequest(tc.handler, tc.method, tc.target, tc.body)
			}
			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body = %q", resp.Code, tc.wantStatus, resp.Body.String())
			}

			req := httptest.NewRequest(tc.method, tc.target, nil)
			route, pathParams, err := specRouter.FindRoute(req)
			if err != nil {
				t.Fatalf("FindRoute(%s %s) error = %v", tc.method, tc.target, err)
			}

			input := &openapi3filter.ResponseValidationInput{
				RequestValidationInput: &openapi3filter.RequestValidationInput{
					Request:    req,
					PathParams: pathParams,
					Route:      route,
				},
				Status:  resp.Code,
				Header:  resp.Header(),
				Options: &openapi3filter.Options{IncludeResponseStatus: true},
			}
			input.SetBodyBytes(resp.Body.Bytes())

			if err := openapi3filter.ValidateResponse(t.Context(), input); err != nil {
				t.Fatalf("response does not match OpenAPI contract: %v", err)
			}
		})
	}
}

const securityDecisionExtension = "x-security-decision"

const (
	securityExposurePublic    = "public"
	securityExposureProtected = "protected"
	securityExposureBlocked   = "blocked"
)

type openAPISecurityDecision struct {
	exposure  string
	rationale string
}

func operationSecurityDecision(operation *openapi3.Operation) (openAPISecurityDecision, error) {
	if operation == nil {
		return openAPISecurityDecision{}, fmt.Errorf("operation is nil")
	}

	raw, ok := operation.Extensions[securityDecisionExtension]
	if !ok {
		return openAPISecurityDecision{}, fmt.Errorf("missing %s", securityDecisionExtension)
	}
	fields, ok := raw.(map[string]any)
	if !ok {
		return openAPISecurityDecision{}, fmt.Errorf("%s must be an object", securityDecisionExtension)
	}

	decision := openAPISecurityDecision{}
	if exposure, ok := fields["exposure"].(string); ok {
		decision.exposure = strings.TrimSpace(exposure)
	}
	if rationale, ok := fields["rationale"].(string); ok {
		decision.rationale = strings.TrimSpace(rationale)
	}
	if decision.exposure == "" {
		return openAPISecurityDecision{}, fmt.Errorf("%s.exposure is required", securityDecisionExtension)
	}
	if decision.rationale == "" {
		return openAPISecurityDecision{}, fmt.Errorf("%s.rationale is required", securityDecisionExtension)
	}
	return decision, nil
}

func operationHasRealSecurity(swagger *openapi3.T, operation *openapi3.Operation) bool {
	if swagger == nil || swagger.Components == nil || operation == nil || operation.Security == nil {
		return false
	}
	for _, requirement := range *operation.Security {
		for name := range requirement {
			if _, ok := swagger.Components.SecuritySchemes[name]; ok {
				return true
			}
		}
	}
	return false
}

func operationHasProblemResponse(swagger *openapi3.T, operation *openapi3.Operation, status string) bool {
	if operation == nil || operation.Responses == nil {
		return false
	}
	response := resolveResponseRef(swagger, operation.Responses.Value(status))
	if response == nil {
		return false
	}
	mediaType := response.Content.Get("application/problem+json")
	if mediaType == nil || mediaType.Schema == nil {
		return false
	}
	return mediaType.Schema.Ref == "#/components/schemas/Problem"
}

func resolveResponseRef(swagger *openapi3.T, responseRef *openapi3.ResponseRef) *openapi3.Response {
	if responseRef == nil {
		return nil
	}
	if responseRef.Value != nil {
		return responseRef.Value
	}
	name, ok := strings.CutPrefix(responseRef.Ref, "#/components/responses/")
	if !ok || swagger == nil || swagger.Components == nil {
		return nil
	}
	componentRef := swagger.Components.Responses[name]
	if componentRef == nil {
		return nil
	}
	return componentRef.Value
}

func mustOpenAPISwagger(t *testing.T) *openapi3.T {
	t.Helper()

	swagger, err := api.GetSpec()
	if err != nil {
		t.Fatalf("GetSpec() error = %v", err)
	}
	return swagger
}

type failingProbe struct {
	name string
	err  error
}

var _ health.Probe = (*failingProbe)(nil)

func (p failingProbe) Name() string {
	return p.name
}

func (p failingProbe) Check(_ context.Context) error {
	return p.err
}
