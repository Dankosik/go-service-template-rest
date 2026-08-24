package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/getkin/kin-openapi/openapi3"
)

const idempotencyExtension = "x-idempotent"

var idempotencyReplayHeaders = []string{
	"Content-Disposition",
	"Content-Encoding",
	"Content-Language",
	"Content-Type",
	"Location",
}

// HasIdempotentOperations reports whether the generated contract activates the
// PostgreSQL component and rejects incomplete declarations before startup.
func HasIdempotentOperations() (bool, error) {
	spec, err := openapi.GetSpec()
	if err != nil {
		return false, fmt.Errorf("load OpenAPI idempotency declarations: %w", err)
	}
	return validateIdempotentOperations(spec)
}

func validateIdempotentOperations(spec *openapi3.T) (bool, error) {
	if spec == nil || spec.Paths == nil {
		return false, errors.New("http idempotency: OpenAPI spec is required")
	}
	found := false
	for _, path := range spec.Paths.Map() {
		if path == nil {
			continue
		}
		for _, operation := range path.Operations() {
			if operation == nil {
				continue
			}
			raw, declared := operation.Extensions[idempotencyExtension]
			if !declared {
				continue
			}
			if err := validateIdempotentOperation(spec, operation, raw); err != nil {
				return false, err
			}
			found = true
		}
	}
	return found, nil
}

func validateIdempotentOperation(spec *openapi3.T, operation *openapi3.Operation, raw any) error {
	enabled, ok := raw.(bool)
	if !ok || !enabled {
		return fmt.Errorf("http idempotency: operation %q must declare %s: true", operation.OperationID, idempotencyExtension)
	}
	if !operationHasSecurity(spec, operation) {
		return fmt.Errorf("http idempotency: operation %q is not protected", operation.OperationID)
	}
	if !operationHasRequiredKey(operation) {
		return fmt.Errorf("http idempotency: operation %q lacks required %s", operation.OperationID, httpidempotency.Header)
	}
	if err := validateIdempotentResponseHeaders(operation); err != nil {
		return err
	}
	for _, status := range []int{400, 401, 403, 422, 500, 503, 504} {
		if operation.Responses == nil || operation.Responses.Value(strconv.Itoa(status)) == nil {
			return fmt.Errorf("http idempotency: operation %q lacks %d response", operation.OperationID, status)
		}
	}
	return nil
}

func validateIdempotentResponseHeaders(operation *openapi3.Operation) error {
	if operation.Responses == nil {
		return nil
	}
	for status, reference := range operation.Responses.Map() {
		success := strings.EqualFold(status, "2XX")
		code, err := strconv.Atoi(status)
		if err == nil {
			success = code >= 200 && code < 300
		}
		if !success {
			continue
		}
		if reference == nil || reference.Value == nil {
			continue
		}
		for name := range reference.Value.Headers {
			canonical := http.CanonicalHeaderKey(name)
			if !slices.Contains(idempotencyReplayHeaders, canonical) {
				return fmt.Errorf("http idempotency: operation %q success response header %q is not replayable", operation.OperationID, canonical)
			}
		}
	}
	return nil
}

func operationHasSecurity(spec *openapi3.T, operation *openapi3.Operation) bool {
	requirements := spec.Security
	if operation.Security != nil {
		requirements = *operation.Security
	}
	if len(requirements) == 0 {
		return false
	}
	for _, requirement := range requirements {
		if len(requirement) == 0 {
			return false
		}
	}
	return true
}

func operationHasRequiredKey(operation *openapi3.Operation) bool {
	for _, reference := range operation.Parameters {
		parameter := reference.Value
		if parameter == nil || !strings.EqualFold(parameter.In, "header") || !strings.EqualFold(parameter.Name, httpidempotency.Header) {
			continue
		}
		return parameter.Required && parameter.Schema != nil && parameter.Schema.Value != nil &&
			parameter.Schema.Value.MaxLength != nil && *parameter.Schema.Value.MaxLength == httpidempotency.MaxKeyBytes
	}
	return false
}
