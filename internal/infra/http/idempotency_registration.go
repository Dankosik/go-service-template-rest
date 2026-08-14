package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/getkin/kin-openapi/openapi3"
)

const idempotencyExtension = "x-http-idempotency"

// IdempotencyOperation pairs the static OpenAPI declaration with the endpoint's
// current authorization and per-authority admission decisions.
type IdempotencyOperation struct {
	Contract  httpidempotency.Contract
	Authorize func(context.Context, *http.Request) (httpidempotency.Scope, bool)
	Admit     func(context.Context, httpidempotency.Scope) httpidempotency.Decision
}

type registeredIdempotencyOperation struct {
	contract  httpidempotency.Contract
	authorize func(context.Context, *http.Request) (httpidempotency.Scope, bool)
	admit     func(context.Context, httpidempotency.Scope) httpidempotency.Decision
}

type idempotencyRegistry struct {
	operations map[string]registeredIdempotencyOperation
}

type idempotencyDeclaration struct {
	APIVersion          string          `json:"api_version"`
	KeyMaxBytes         int             `json:"key_max_bytes"`
	FingerprintVersions []string        `json:"fingerprint_versions"`
	ResultCodecs        []string        `json:"result_codecs"`
	ReplayStatuses      []int           `json:"replay_statuses"`
	StableHeaders       []string        `json:"stable_headers"`
	ResultMaxBytes      int             `json:"result_max_bytes"`
	ReplayTTL           string          `json:"replay_ttl"`
	DuplicateRisk       json.RawMessage `json:"duplicate_risk"`
	InProgressWait      string          `json:"in_progress_wait"`
	RetryAfter          string          `json:"retry_after"`
	ExternalEffect      string          `json:"external_effect"`
}

type duplicateRiskDeclaration struct {
	Kind     string  `json:"kind"`
	Duration *string `json:"duration"`
}

func newIdempotencyRegistry(spec *openapi3.T, configured []IdempotencyOperation) (idempotencyRegistry, error) {
	if spec == nil || spec.Paths == nil {
		return idempotencyRegistry{}, errors.New("http idempotency: OpenAPI spec is required")
	}
	configuredByID, err := configuredIdempotencyOperations(configured)
	if err != nil {
		return idempotencyRegistry{}, err
	}
	registered, err := declaredIdempotencyOperations(spec, configuredByID)
	if err != nil {
		return idempotencyRegistry{}, err
	}
	if len(registered) != len(configuredByID) {
		for operationID := range configuredByID {
			if _, ok := registered[operationID]; !ok {
				return idempotencyRegistry{}, fmt.Errorf("http idempotency: registration %q has no OpenAPI declaration", operationID)
			}
		}
	}
	return idempotencyRegistry{operations: registered}, nil
}

func configuredIdempotencyOperations(configured []IdempotencyOperation) (map[string]IdempotencyOperation, error) {
	configuredByID := make(map[string]IdempotencyOperation, len(configured))
	for _, operation := range configured {
		if err := operation.Contract.Validate(); err != nil {
			//nolint:wrapcheck // Preserve the accepted Contract validation diagnostic.
			return nil, err
		}
		if operation.Authorize == nil || operation.Admit == nil {
			return nil, fmt.Errorf("http idempotency: operation %q lacks authorization or admission", operation.Contract.OperationID)
		}
		if _, duplicate := configuredByID[operation.Contract.OperationID]; duplicate {
			return nil, fmt.Errorf("http idempotency: operation %q is registered twice", operation.Contract.OperationID)
		}
		configuredByID[operation.Contract.OperationID] = operation
	}
	return configuredByID, nil
}

func declaredIdempotencyOperations(spec *openapi3.T, configured map[string]IdempotencyOperation) (map[string]registeredIdempotencyOperation, error) {
	registered := make(map[string]registeredIdempotencyOperation, len(configured))
	for _, pathItem := range spec.Paths.Map() {
		if pathItem == nil {
			continue
		}
		for _, operation := range pathItem.Operations() {
			if err := registerIdempotencyOperation(spec, operation, configured, registered); err != nil {
				return nil, err
			}
		}
	}
	return registered, nil
}

func registerIdempotencyOperation(spec *openapi3.T, operation *openapi3.Operation, configured map[string]IdempotencyOperation, registered map[string]registeredIdempotencyOperation) error {
	if operation == nil || operation.Extensions == nil {
		return nil
	}
	raw, opted := operation.Extensions[idempotencyExtension]
	if !opted {
		return nil
	}
	configuredOperation, ok := configured[operation.OperationID]
	if !ok {
		return fmt.Errorf("http idempotency: OpenAPI operation %q has no registration", operation.OperationID)
	}
	declaration, err := decodeIdempotencyDeclaration(operation.OperationID, raw)
	if err != nil {
		return err
	}
	if !sameIdempotencyContract(declaration, configuredOperation.Contract) {
		return fmt.Errorf("http idempotency: OpenAPI declaration for %q differs from registration", operation.OperationID)
	}
	if err := validateIdempotencyOperationShape(spec, operation, declaration); err != nil {
		return err
	}
	if _, duplicate := registered[operation.OperationID]; duplicate {
		return fmt.Errorf("http idempotency: OpenAPI operation %q is declared twice", operation.OperationID)
	}
	registered[operation.OperationID] = registeredIdempotencyOperation{
		contract:  configuredOperation.Contract.Clone(),
		authorize: configuredOperation.Authorize,
		admit:     configuredOperation.Admit,
	}
	return nil
}

func decodeIdempotencyDeclaration(operationID string, raw any) (httpidempotency.Contract, error) {
	encoded, err := json.Marshal(raw)
	if err != nil {
		return httpidempotency.Contract{}, fmt.Errorf("http idempotency: encode declaration for %q: %w", operationID, err)
	}
	var declaration idempotencyDeclaration
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&declaration); err != nil {
		return httpidempotency.Contract{}, fmt.Errorf("http idempotency: decode declaration for %q: %w", operationID, err)
	}
	parseDuration := func(name, value string) (time.Duration, error) {
		duration, err := time.ParseDuration(value)
		if err != nil {
			return 0, fmt.Errorf("http idempotency: declaration %q has invalid %s: %w", operationID, name, err)
		}
		return duration, nil
	}
	replayTTL, err := parseDuration("replay_ttl", declaration.ReplayTTL)
	if err != nil {
		return httpidempotency.Contract{}, err
	}
	duplicateRisk, err := decodeDuplicateRiskDeclaration(operationID, declaration.DuplicateRisk, replayTTL)
	if err != nil {
		return httpidempotency.Contract{}, err
	}
	inProgressWait, err := parseDuration("in_progress_wait", declaration.InProgressWait)
	if err != nil {
		return httpidempotency.Contract{}, err
	}
	retryAfter, err := parseDuration("retry_after", declaration.RetryAfter)
	if err != nil {
		return httpidempotency.Contract{}, err
	}
	contract := httpidempotency.Contract{
		OperationID:         operationID,
		APIVersion:          declaration.APIVersion,
		KeyMaxBytes:         declaration.KeyMaxBytes,
		FingerprintVersions: declaration.FingerprintVersions,
		ResultCodecs:        declaration.ResultCodecs,
		ReplayStatuses:      declaration.ReplayStatuses,
		StableHeaders:       declaration.StableHeaders,
		ResultMaxBytes:      declaration.ResultMaxBytes,
		ReplayTTL:           replayTTL,
		DuplicateRisk:       duplicateRisk,
		InProgressWait:      inProgressWait,
		RetryAfter:          retryAfter,
		ExternalEffect:      httpidempotency.ExternalEffectDisposition(declaration.ExternalEffect),
	}
	if err := contract.Validate(); err != nil {
		//nolint:wrapcheck // Preserve the accepted declaration validation diagnostic.
		return httpidempotency.Contract{}, err
	}
	return contract, nil
}

func decodeDuplicateRiskDeclaration(operationID string, raw json.RawMessage, replayTTL time.Duration) (httpidempotency.DuplicateRiskPolicy, error) {
	if len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return httpidempotency.DuplicateRiskPolicy{}, fmt.Errorf("http idempotency: declaration %q lacks duplicate_risk", operationID)
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		return httpidempotency.DuplicateRiskPolicy{}, fmt.Errorf("http idempotency: declaration %q has invalid duplicate_risk", operationID)
	}
	var declaration duplicateRiskDeclaration
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&declaration); err != nil {
		return httpidempotency.DuplicateRiskPolicy{}, fmt.Errorf("http idempotency: decode duplicate_risk for %q: %w", operationID, err)
	}

	_, hasDuration := fields["duration"]
	switch declaration.Kind {
	case "finite":
		if !hasDuration || declaration.Duration == nil {
			return httpidempotency.DuplicateRiskPolicy{}, fmt.Errorf("http idempotency: declaration %q lacks finite duplicate-risk duration", operationID)
		}
		duration, err := time.ParseDuration(*declaration.Duration)
		if err != nil {
			return httpidempotency.DuplicateRiskPolicy{}, fmt.Errorf("http idempotency: declaration %q has invalid duplicate-risk duration: %w", operationID, err)
		}
		policy := httpidempotency.DuplicateRiskPolicy{Duration: duration}
		if policy.Duration < replayTTL {
			return httpidempotency.DuplicateRiskPolicy{}, fmt.Errorf("http idempotency: declaration %q has duplicate risk before replay TTL", operationID)
		}
		return policy, nil
	case "permanent":
		if hasDuration {
			return httpidempotency.DuplicateRiskPolicy{}, fmt.Errorf("http idempotency: declaration %q gives permanent duplicate risk a duration", operationID)
		}
		return httpidempotency.DuplicateRiskPolicy{Permanent: true}, nil
	default:
		return httpidempotency.DuplicateRiskPolicy{}, fmt.Errorf("http idempotency: declaration %q has unknown duplicate-risk kind %q", operationID, declaration.Kind)
	}
}

func sameIdempotencyContract(left, right httpidempotency.Contract) bool {
	return left.OperationID == right.OperationID &&
		left.APIVersion == right.APIVersion &&
		left.KeyMaxBytes == right.KeyMaxBytes &&
		slices.Equal(left.FingerprintVersions, right.FingerprintVersions) &&
		slices.Equal(left.ResultCodecs, right.ResultCodecs) &&
		slices.Equal(left.ReplayStatuses, right.ReplayStatuses) &&
		slices.Equal(left.StableHeaders, right.StableHeaders) &&
		left.ResultMaxBytes == right.ResultMaxBytes &&
		left.ReplayTTL == right.ReplayTTL &&
		left.DuplicateRisk == right.DuplicateRisk &&
		left.InProgressWait == right.InProgressWait &&
		left.RetryAfter == right.RetryAfter &&
		left.ExternalEffect == right.ExternalEffect
}

func validateIdempotencyOperationShape(spec *openapi3.T, operation *openapi3.Operation, contract httpidempotency.Contract) error {
	if !operationHasSecurity(spec, operation) {
		return fmt.Errorf("http idempotency: operation %q is not protected", operation.OperationID)
	}
	if !operationHasRequiredKey(operation, contract.KeyMaxBytes) {
		return fmt.Errorf("http idempotency: operation %q lacks required %s", operation.OperationID, httpidempotency.Header)
	}
	statuses := append([]int{400, 401, 403, 409, 422, 429, 500, 503, 504}, contract.ReplayStatuses...)
	for _, status := range statuses {
		if operation.Responses == nil || operation.Responses.Value(strconv.Itoa(status)) == nil {
			return fmt.Errorf("http idempotency: operation %q lacks %d response", operation.OperationID, status)
		}
	}
	return nil
}

func operationHasSecurity(spec *openapi3.T, operation *openapi3.Operation) bool {
	if operation.Security != nil {
		return len(*operation.Security) > 0
	}
	return len(spec.Security) > 0
}

func operationHasRequiredKey(operation *openapi3.Operation, keyMaxBytes int) bool {
	for _, parameterRef := range operation.Parameters {
		parameter := parameterRef.Value
		if parameter != nil && strings.EqualFold(parameter.In, "header") && strings.EqualFold(parameter.Name, httpidempotency.Header) && parameter.Required &&
			parameter.Schema != nil && parameter.Schema.Value != nil && parameter.Schema.Value.MaxLength != nil && *parameter.Schema.Value.MaxLength == uint64(keyMaxBytes) { // #nosec G115 -- Contract.Validate rejects a non-positive KeyMaxBytes.
			return true
		}
	}
	return false
}
