package httpx

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/httpidempotency"
	"github.com/getkin/kin-openapi/openapi3"
)

func TestIdempotentOperationDeclaration(t *testing.T) {
	t.Parallel()

	spec := idempotencySpec()
	if enabled, err := validateIdempotentOperations(spec); err != nil || !enabled {
		t.Fatalf("validateIdempotentOperations(valid) = %v, %v", enabled, err)
	}

	operation := spec.Paths.Value("/widgets").Post
	operation.Parameters = nil
	if _, err := validateIdempotentOperations(spec); err == nil || !strings.Contains(err.Error(), httpidempotency.Header) {
		t.Fatalf("missing key error = %v", err)
	}
	operation.Parameters = idempotencyKeyParameter()
	operation.Extensions[idempotencyExtension] = false
	if _, err := validateIdempotentOperations(spec); err == nil || !strings.Contains(err.Error(), "must declare") {
		t.Fatalf("false declaration error = %v", err)
	}
}

func TestIdempotentOperationRejectsAnonymousSecurityAlternative(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		set  func(*openapi3.T, *openapi3.Operation)
	}{
		{
			name: "operation",
			set: func(_ *openapi3.T, operation *openapi3.Operation) {
				security := openapi3.SecurityRequirements{{"bearerAuth": {}}, {}}
				operation.Security = &security
			},
		},
		{
			name: "global",
			set: func(spec *openapi3.T, _ *openapi3.Operation) {
				spec.Security = openapi3.SecurityRequirements{{"bearerAuth": {}}, {}}
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			spec := idempotencySpec()
			test.set(spec, spec.Paths.Value("/widgets").Post)
			if _, err := validateIdempotentOperations(spec); err == nil || !strings.Contains(err.Error(), "not protected") {
				t.Fatalf("anonymous security alternative error = %v", err)
			}
		})
	}
}

func TestCaptureIdempotencyKeyPreservesWireMultiplicity(t *testing.T) {
	t.Parallel()

	handler := httpidempotency.CaptureKey(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := httpidempotency.NewRequestFromContext(
			r.Context(),
			httpidempotency.Scope{Caller: "caller", Operation: "create"},
			struct{}{},
		)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/widgets", nil)
	request.Header.Add(httpidempotency.Header, "key-1")
	request.Header.Add(httpidempotency.Header, "key-2")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func idempotencySpec() *openapi3.T {
	decimalResponses := openapi3.NewResponses()
	for _, status := range []string{"400", "401", "403", "409", "422", "500", "503", "504"} {
		decimalResponses.Set(status, &openapi3.ResponseRef{Value: openapi3.NewResponse().WithDescription("problem")})
	}
	operation := &openapi3.Operation{
		OperationID: "createWidget",
		Extensions:  map[string]any{idempotencyExtension: true},
		Parameters:  idempotencyKeyParameter(),
		Responses:   decimalResponses,
	}
	paths := openapi3.NewPaths()
	paths.Set("/widgets", &openapi3.PathItem{Post: operation})
	return &openapi3.T{Paths: paths, Security: openapi3.SecurityRequirements{{"bearerAuth": {}}}}
}

func idempotencyKeyParameter() openapi3.Parameters {
	maxLength := uint64(httpidempotency.MaxKeyBytes)
	return openapi3.Parameters{&openapi3.ParameterRef{Value: &openapi3.Parameter{
		Name: httpidempotency.Header, In: "header", Required: true,
		Schema: &openapi3.SchemaRef{Value: &openapi3.Schema{Type: &openapi3.Types{"string"}, MaxLength: &maxLength}},
	}}}
}
