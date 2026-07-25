package httpapi

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"
	"github.com/getkin/kin-openapi/openapi3filter"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
)

// NewRouter wires the generated contract to the reference feature.
//
// writeToken is the credential accepted for protected operations. It is a
// deliberately minimal stand-in so the example can demonstrate how a
// spec-declared security scheme becomes a runtime check; it is NOT an
// authentication design. A real service owns identity, key rotation,
// authorization, and audit — see docs/first-production-feature.md.
func NewRouter(articles *article.Service, writeToken string) (http.Handler, error) {
	if articles == nil {
		return nil, errors.New("reference router: article service is required")
	}
	if strings.TrimSpace(writeToken) == "" {
		return nil, errors.New("reference router: write token is required")
	}
	spec, err := openapi.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("reference router: load OpenAPI spec: %w", err)
	}

	writeBadRequest := func(w http.ResponseWriter) {
		writeProblem(w, problem("bad_request", "bad request", http.StatusBadRequest, "request is malformed or invalid"))
	}
	validator := oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
		DoNotValidateServers: true,
		// The spec's securitySchemes drive this call, so an operation marked
		// protected cannot reach a handler without passing the check.
		Options: openapi3filter.Options{
			AuthenticationFunc: func(_ context.Context, input *openapi3filter.AuthenticationInput) error {
				return authenticateBearer(input, writeToken)
			},
		},
		ErrorHandlerWithOpts: func(
			_ context.Context,
			err error,
			w http.ResponseWriter,
			_ *http.Request,
			_ oapimiddleware.ErrorHandlerOpts,
		) {
			// A failed security requirement is 401, not 400: the request was
			// well formed and the credential was the problem.
			var securityErr *openapi3filter.SecurityRequirementsError
			if errors.As(err, &securityErr) {
				w.Header().Set("WWW-Authenticate", "Bearer")
				writeProblem(w, problem("unauthorized", "unauthorized", http.StatusUnauthorized, "credentials are missing or invalid"))
				return
			}
			writeBadRequest(w)
		},
	})
	strict := openapi.NewStrictHandlerWithOptions(&handler{articles: articles}, nil, openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeBadRequest(w)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeProblem(w, problem("internal_error", "internal server error", http.StatusInternalServerError, "request failed"))
		},
	})

	return openapi.HandlerWithOptions(strict, openapi.ChiServerOptions{
		Middlewares: []openapi.MiddlewareFunc{validator},
		ErrorHandlerFunc: func(w http.ResponseWriter, _ *http.Request, _ error) {
			writeBadRequest(w)
		},
	}), nil
}

// authenticateBearer accepts exactly the configured demonstration credential.
// The comparison is constant time so a wrong token cannot be recovered by
// timing, which is the one property worth copying from this function.
func authenticateBearer(input *openapi3filter.AuthenticationInput, expected string) error {
	if input == nil || input.SecurityScheme == nil {
		return errors.New("missing security scheme")
	}
	if !strings.EqualFold(input.SecurityScheme.Type, "http") ||
		!strings.EqualFold(input.SecurityScheme.Scheme, "bearer") {
		return fmt.Errorf("unsupported security scheme %q", input.SecuritySchemeName)
	}

	header := input.RequestValidationInput.Request.Header.Get("Authorization")
	presented, ok := strings.CutPrefix(header, "Bearer ")
	if !ok {
		return errors.New("bearer credential is missing")
	}
	if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(presented)), []byte(expected)) != 1 {
		return errors.New("bearer credential is invalid")
	}
	return nil
}

func writeProblem(w http.ResponseWriter, body openapi.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(int(body.Status))
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// The response status is already committed; the caller cannot recover.
		return
	}
}
