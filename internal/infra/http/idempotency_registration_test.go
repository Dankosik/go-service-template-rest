package httpx

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/example/go-service-template-rest/internal/health"
	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/infra/telemetry"
	"github.com/getkin/kin-openapi/openapi3"
)

const idempotencyTestOpenAPI = `openapi: 3.0.3
info:
  title: idempotency test
  version: 1.0.0
servers:
  - url: /
security:
  - bearerAuth: []
paths:
  /widgets:
    post:
      operationId: createWidget
      parameters:
        - in: header
          name: Idempotency-Key
          required: true
          schema:
            type: string
            maxLength: 64
      responses:
        "200": {description: ok}
        "201": {description: created}
        "400": {description: bad request}
        "401": {description: unauthorized}
        "403": {description: forbidden}
        "409": {description: conflict}
        "422": {description: unprocessable}
        "429": {description: limited}
        "500": {description: internal}
        "503": {description: unavailable}
        "504": {description: timeout}
components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
`

func TestHTTPIdempotencyRegistrationContract(t *testing.T) {
	spec := idempotencyTestSpec(t)
	operation := testIdempotencyOperation()
	registry, err := newIdempotencyRegistry(spec, []IdempotencyOperation{operation})
	if err != nil {
		t.Fatalf("newIdempotencyRegistry() error = %v", err)
	}
	if len(registry.operations) != 1 {
		t.Fatalf("registered operations = %d, want 1", len(registry.operations))
	}

	t.Run("unregistered health route is inert", func(t *testing.T) {
		handler := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{Health: health.New()}, telemetry.New(), RouterConfig{})
		request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/health/live", nil)
		request.Header.Set(httpidempotency.Header, "unused-key")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("health route status = %d, want 200", response.Code)
		}
	})

	t.Run("missing declaration", func(t *testing.T) {
		candidate := idempotencyTestSpec(t)
		delete(candidate.Paths.Value("/widgets").Post.Extensions, idempotencyExtension)
		if _, err := newIdempotencyRegistry(candidate, []IdempotencyOperation{operation}); err == nil {
			t.Fatal("missing OpenAPI declaration was accepted")
		}
	})
	t.Run("missing registration", func(t *testing.T) {
		if _, err := newIdempotencyRegistry(idempotencyTestSpec(t), nil); err == nil {
			t.Fatal("OpenAPI declaration without registration was accepted")
		}
	})
	t.Run("duplicate registration", func(t *testing.T) {
		if _, err := newIdempotencyRegistry(idempotencyTestSpec(t), []IdempotencyOperation{operation, operation}); err == nil {
			t.Fatal("duplicate registration was accepted")
		}
	})
	t.Run("duplicate OpenAPI operation ID", func(t *testing.T) {
		candidate := idempotencyTestSpec(t)
		duplicate := *candidate.Paths.Value("/widgets").Post
		candidate.Paths.Set("/widgets-copy", &openapi3.PathItem{Post: &duplicate})
		if _, err := newIdempotencyRegistry(candidate, []IdempotencyOperation{operation}); err == nil {
			t.Fatal("duplicate OpenAPI operation ID was accepted")
		}
	})
	t.Run("fingerprint version mismatch", func(t *testing.T) {
		candidate := testIdempotencyOperation()
		candidate.Contract.FingerprintVersions = []string{"v2"}
		if _, err := newIdempotencyRegistry(idempotencyTestSpec(t), []IdempotencyOperation{candidate}); err == nil {
			t.Fatal("version mismatch was accepted")
		}
	})
	t.Run("permanent duplicate risk", func(t *testing.T) {
		candidate := testIdempotencyOperation()
		candidate.Contract.DuplicateRisk = httpidempotency.DuplicateRiskPolicy{Permanent: true}
		spec := idempotencyTestSpecForContract(t, candidate.Contract)
		if _, err := newIdempotencyRegistry(spec, []IdempotencyOperation{candidate}); err != nil {
			t.Fatalf("permanent duplicate risk was rejected: %v", err)
		}
	})
	for _, testCase := range []struct {
		name   string
		mutate func(map[string]any)
	}{
		{name: "missing duplicate risk", mutate: func(extension map[string]any) { delete(extension, "duplicate_risk") }},
		{name: "finite duplicate risk without duration", mutate: func(extension map[string]any) { extension["duplicate_risk"] = map[string]any{"kind": "finite"} }},
		{name: "permanent duplicate risk with duration", mutate: func(extension map[string]any) {
			extension["duplicate_risk"] = map[string]any{"kind": "permanent", "duration": "2h"}
		}},
		{name: "unknown duplicate risk kind", mutate: func(extension map[string]any) { extension["duplicate_risk"] = map[string]any{"kind": "forever"} }},
		{name: "unknown duplicate risk field", mutate: func(extension map[string]any) {
			extension["duplicate_risk"] = map[string]any{"kind": "finite", "duration": "2h", "other": true}
		}},
		{name: "superseded duplicate risk scalar", mutate: func(extension map[string]any) { extension["duplicate_risk_ttl"] = "2h" }},
		{name: "conflicting duplicate risk", mutate: func(extension map[string]any) { extension["duplicate_risk"] = map[string]any{"kind": "permanent"} }},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			candidate := idempotencyTestSpec(t)
			extension := candidate.Paths.Value("/widgets").Post.Extensions[idempotencyExtension].(map[string]any)
			testCase.mutate(extension)
			if _, err := newIdempotencyRegistry(candidate, []IdempotencyOperation{operation}); err == nil {
				t.Fatal("invalid duplicate-risk declaration was accepted")
			}
		})
	}
	t.Run("external effect mutation", func(t *testing.T) {
		candidate := testIdempotencyOperation()
		candidate.Contract.ExternalEffect = "uncovered"
		if _, err := newIdempotencyRegistry(idempotencyTestSpec(t), []IdempotencyOperation{candidate}); err == nil {
			t.Fatal("uncovered external effect was accepted")
		}
	})
	t.Run("fractional retry after", func(t *testing.T) {
		candidate := testIdempotencyOperation()
		candidate.Contract.RetryAfter += time.Millisecond
		spec := idempotencyTestSpec(t)
		declaration, err := idempotencyContractExtension(candidate.Contract)
		if err != nil {
			t.Fatalf("idempotency extension: %v", err)
		}
		spec.Paths.Value("/widgets").Post.Extensions[idempotencyExtension] = declaration
		if _, err := newIdempotencyRegistry(spec, []IdempotencyOperation{candidate}); err == nil {
			t.Fatal("fractional retry after was accepted")
		}
	})
	t.Run("header declaration", func(t *testing.T) {
		candidate := idempotencyTestSpec(t)
		candidate.Paths.Value("/widgets").Post.Parameters = nil
		if _, err := newIdempotencyRegistry(candidate, []IdempotencyOperation{operation}); err == nil {
			t.Fatal("operation without Idempotency-Key was accepted")
		}
	})
	t.Run("key max bytes declaration", func(t *testing.T) {
		for _, maxLength := range []*uint64{nil, new(uint64)} {
			candidate := idempotencyTestSpec(t)
			if maxLength != nil {
				*maxLength = 63
			}
			candidate.Paths.Value("/widgets").Post.Parameters[0].Value.Schema.Value.MaxLength = maxLength
			if _, err := newIdempotencyRegistry(candidate, []IdempotencyOperation{operation}); err == nil {
				t.Fatal("operation with missing or mismatched key maximum was accepted")
			}
		}
	})
	t.Run("unknown declaration field", func(t *testing.T) {
		candidate := idempotencyTestSpec(t)
		candidate.Paths.Value("/widgets").Post.Extensions[idempotencyExtension].(map[string]any)["unknown"] = true
		if _, err := newIdempotencyRegistry(candidate, []IdempotencyOperation{operation}); err == nil {
			t.Fatal("declaration with an unknown field was accepted")
		}
	})
}

func idempotencyTestSpec(tb testing.TB) *openapi3.T {
	return idempotencyTestSpecForContract(tb, testIdempotencyOperation().Contract)
}

func idempotencyTestSpecForContract(tb testing.TB, contract httpidempotency.Contract) *openapi3.T {
	tb.Helper()
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromData([]byte(idempotencyTestOpenAPI))
	if err != nil {
		tb.Fatalf("load test OpenAPI: %v", err)
	}
	if err := spec.Validate(tb.Context()); err != nil {
		tb.Fatalf("validate test OpenAPI: %v", err)
	}
	extension, err := idempotencyContractExtension(contract)
	if err != nil {
		tb.Fatalf("idempotency extension: %v", err)
	}
	spec.Paths.Value("/widgets").Post.Extensions = map[string]any{idempotencyExtension: extension}
	return spec
}

func idempotencyContractExtension(contract httpidempotency.Contract) (map[string]any, error) {
	duplicateRisk := map[string]any{"kind": "permanent"}
	if !contract.DuplicateRisk.Permanent {
		duplicateRisk = map[string]any{"kind": "finite", "duration": contract.DuplicateRisk.Duration.String()}
	}
	encodedRisk, err := json.Marshal(duplicateRisk)
	if err != nil {
		return nil, err
	}
	declaration := idempotencyDeclaration{
		APIVersion:          contract.APIVersion,
		KeyMaxBytes:         contract.KeyMaxBytes,
		FingerprintVersions: contract.FingerprintVersions,
		ResultCodecs:        contract.ResultCodecs,
		ReplayStatuses:      contract.ReplayStatuses,
		StableHeaders:       contract.StableHeaders,
		ResultMaxBytes:      contract.ResultMaxBytes,
		ReplayTTL:           contract.ReplayTTL.String(),
		DuplicateRisk:       encodedRisk,
		InProgressWait:      contract.InProgressWait.String(),
		RetryAfter:          contract.RetryAfter.String(),
		ExternalEffect:      string(contract.ExternalEffect),
	}
	encoded, err := json.Marshal(declaration)
	if err != nil {
		return nil, err
	}
	var extension map[string]any
	return extension, json.Unmarshal(encoded, &extension)
}

func testIdempotencyOperation() IdempotencyOperation {
	return IdempotencyOperation{
		Contract: httpidempotency.Contract{
			OperationID:         "createWidget",
			APIVersion:          "v1",
			KeyMaxBytes:         64,
			FingerprintVersions: []string{"v1"},
			ResultCodecs:        []string{"create-widget/v1"},
			ReplayStatuses:      []int{http.StatusCreated},
			StableHeaders:       []string{"location"},
			ResultMaxBytes:      1024,
			ReplayTTL:           time.Hour,
			DuplicateRisk:       httpidempotency.DuplicateRiskPolicy{Duration: 2 * time.Hour},
			InProgressWait:      time.Second,
			RetryAfter:          time.Second,
			ExternalEffect:      httpidempotency.ExternalEffectNone,
		},
		Authorize: func(context.Context, *http.Request) (httpidempotency.Scope, bool) {
			return httpidempotency.Scope{Authority: "authority-a", OperationID: "createWidget", APIVersion: "v1"}, true
		},
		Admit: func(context.Context, httpidempotency.Scope) httpidempotency.Decision {
			return httpidempotency.Decision{Outcome: httpidempotency.OutcomeExecute}
		},
	}
}
