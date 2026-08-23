package httpx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/example/go-service-template-rest/internal/openapi"
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
			cfg:      RouterConfig{MaxBodyBytes: testRouterMaxBodyBytes},
			wantErr:  "logger is required",
		},
		{
			name:    "missing health",
			log:     log,
			metrics: telemetry.New(),
			cfg: RouterConfig{
				MaxBodyBytes:   testRouterMaxBodyBytes,
				RequestTimeout: time.Second,
			},
			handlers: Handlers{ReadinessGate: func(context.Context) error { return nil }},
			wantErr:  "health service is required",
		},
		{
			name:    "missing readiness gate",
			log:     log,
			metrics: telemetry.New(),
			cfg: RouterConfig{
				MaxBodyBytes:   testRouterMaxBodyBytes,
				RequestTimeout: time.Second,
			},
			handlers: Handlers{Health: health.New()},
			wantErr:  "readiness gate is required",
		},
		{
			name: "missing metrics",
			log:  log,
			cfg:  RouterConfig{MaxBodyBytes: testRouterMaxBodyBytes},
			handlers: Handlers{
				Health:        health.New(),
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "metrics is required",
		},
		{
			name:    "negative max in flight",
			log:     log,
			metrics: telemetry.New(),
			cfg: RouterConfig{
				MaxBodyBytes:   testRouterMaxBodyBytes,
				RequestTimeout: time.Second,
				MaxInFlight:    -1,
			},
			handlers: Handlers{
				Health:        health.New(),
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "max in flight must be >= 0",
		},
		{
			name:    "missing max body bytes",
			log:     log,
			metrics: telemetry.New(),
			cfg:     RouterConfig{RequestTimeout: time.Second},
			handlers: Handlers{
				Health:        health.New(),
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "max body bytes must be > 0",
		},
		{
			name:    "missing request timeout",
			log:     log,
			metrics: telemetry.New(),
			cfg:     RouterConfig{MaxBodyBytes: testRouterMaxBodyBytes},
			handlers: Handlers{
				Health:        health.New(),
				ReadinessGate: func(context.Context) error { return nil },
			},
			wantErr: "request timeout must be > 0",
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
					if !operationIsPublic(swagger, operation) {
						t.Fatalf("%s operation inherits or declares security while marked %q", operation.OperationID, decision.exposure)
					}
				case securityExposureProtected:
					if !operationUsesBearerSecurityOnly(swagger, operation) {
						t.Fatalf("%s operation is protected but every OpenAPI security alternative is not the wired bearer scheme without scopes", operation.OperationID)
					}
					for _, status := range []string{"400", "401", "403", "431", "503", "504"} {
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

func TestOpenAPIBearerSecurityAlternativesFailClosed(t *testing.T) {
	t.Parallel()

	bearer := &openapi3.SecuritySchemeRef{Value: &openapi3.SecurityScheme{Type: "http", Scheme: "bearer"}}
	apiKey := &openapi3.SecuritySchemeRef{Value: &openapi3.SecurityScheme{Type: "apiKey", In: "header", Name: "X-API-Key"}}
	swagger := &openapi3.T{
		Components: &openapi3.Components{SecuritySchemes: openapi3.SecuritySchemes{
			"bearerAuth": bearer,
			"apiKeyAuth": apiKey,
		}},
		Security: openapi3.SecurityRequirements{{"bearerAuth": {}}},
	}
	security := func(requirements ...openapi3.SecurityRequirement) *openapi3.SecurityRequirements {
		value := openapi3.SecurityRequirements(requirements)
		return &value
	}

	tests := []struct {
		name          string
		operation     *openapi3.Operation
		wantPublic    bool
		wantProtected bool
	}{
		{name: "inherited bearer", operation: &openapi3.Operation{}, wantProtected: true},
		{
			name:          "explicit bearer",
			operation:     &openapi3.Operation{Security: security(openapi3.SecurityRequirement{"bearerAuth": {}})},
			wantProtected: true,
		},
		{name: "explicit public", operation: &openapi3.Operation{Security: security()}, wantPublic: true},
		{
			name: "anonymous alternative",
			operation: &openapi3.Operation{Security: security(
				openapi3.SecurityRequirement{"bearerAuth": {}},
				openapi3.SecurityRequirement{},
			)},
		},
		{
			name:      "bearer scopes are not authorized",
			operation: &openapi3.Operation{Security: security(openapi3.SecurityRequirement{"bearerAuth": {"admin"}})},
		},
		{
			name:      "unknown scheme",
			operation: &openapi3.Operation{Security: security(openapi3.SecurityRequirement{"missingAuth": {}})},
		},
		{
			name:      "unsupported alternative",
			operation: &openapi3.Operation{Security: security(openapi3.SecurityRequirement{"apiKeyAuth": {}})},
		},
		{
			name: "unsupported AND requirement",
			operation: &openapi3.Operation{Security: security(openapi3.SecurityRequirement{
				"bearerAuth": {},
				"apiKeyAuth": {},
			})},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			if got := operationIsPublic(swagger, testCase.operation); got != testCase.wantPublic {
				t.Errorf("operationIsPublic() = %v, want %v", got, testCase.wantPublic)
			}
			if got := operationUsesBearerSecurityOnly(swagger, testCase.operation); got != testCase.wantProtected {
				t.Errorf("operationUsesBearerSecurityOnly() = %v, want %v", got, testCase.wantProtected)
			}
		})
	}
}

// TestOpenAPIRuntimeContractResponsesMatchSpec validates recorded responses —
// status, Content-Type, and body schema — against the embedded OpenAPI spec.
// Generated Visit* responses are contract-shaped by construction; this closes
// the remaining gap for the hand-written problem writer and text responses.
//
// The case table is maintained by hand and does NOT enumerate the spec: a new
// operation or status adds no coverage here and nothing fails to say so. When
// adding an operation, add its reachable status cases below, or the
// hand-written response paths for that operation stay unvalidated.
// TestOpenAPIRuntimeContractOperationsDeclareSecurityDecisions is the test in
// this file that does iterate the spec automatically.
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
			name:       "health live 413 problem",
			handler:    tinyBodyLimit,
			method:     http.MethodGet,
			target:     "/health/live",
			body:       "ab",
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		// profile:inbound-webhooks-standard:start
		{
			name:       "inbound webhook 400 missing headers",
			handler:    ready,
			method:     http.MethodPost,
			target:     "/webhooks/orders",
			body:       "{}",
			wantStatus: http.StatusBadRequest,
		},
		// profile:inbound-webhooks-standard:end
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

			req := httptest.NewRequestWithContext(context.Background(), tc.method, tc.target, nil)
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
		return openAPISecurityDecision{}, errors.New("operation is nil")
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

func operationSecurityRequirements(swagger *openapi3.T, operation *openapi3.Operation) openapi3.SecurityRequirements {
	if operation == nil {
		return nil
	}
	if operation.Security != nil {
		return *operation.Security
	}
	if swagger == nil {
		return nil
	}
	return swagger.Security
}

func operationIsPublic(swagger *openapi3.T, operation *openapi3.Operation) bool {
	return len(operationSecurityRequirements(swagger, operation)) == 0
}

func operationUsesBearerSecurityOnly(swagger *openapi3.T, operation *openapi3.Operation) bool {
	if swagger == nil || swagger.Components == nil {
		return false
	}
	requirements := operationSecurityRequirements(swagger, operation)
	if len(requirements) == 0 {
		return false
	}
	for _, requirement := range requirements {
		if len(requirement) != 1 {
			return false
		}
		for name, scopes := range requirement {
			declaration, ok := swagger.Components.SecuritySchemes[name]
			if !ok || declaration == nil || declaration.Value == nil || len(scopes) != 0 {
				return false
			}
			scheme := declaration.Value
			if !strings.EqualFold(scheme.Type, "http") || !strings.EqualFold(scheme.Scheme, "bearer") {
				return false
			}
		}
	}
	return true
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

	swagger, err := openapi.GetSpec()
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
