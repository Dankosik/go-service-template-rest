package httpapi

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"
	"github.com/getkin/kin-openapi/openapi3filter"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
)

// RejectFunc writes a transport-level rejection. The composition root supplies
// these so this package maps the feature onto its contract without importing an
// infra adapter to do it; see the reference binary for what it passes.
type RejectFunc func(http.ResponseWriter, *http.Request, error)

// Options carries what the composition root owns.
type Options struct {
	// WriteToken is the credential accepted for protected operations. It is a
	// deliberately minimal stand-in so the example can demonstrate how a
	// spec-declared security scheme becomes a runtime check; it is NOT an
	// authentication design. A real service owns identity, key rotation,
	// authorization, and audit — see docs/first-production-feature.md.
	WriteToken string
	// RejectRequest maps a validator failure onto a response: an oversized body
	// to 413, a failed security requirement to 401 with a WWW-Authenticate
	// challenge, everything else to a sanitized 400.
	RejectRequest RejectFunc
	// RejectResponse maps a generated strict-server response failure: a spent
	// request budget to 504, anything else to 500.
	RejectResponse RejectFunc
}

// NewAPIHandler wires the generated contract to the reference feature.
//
// It returns the API handler only — not a server-ready router. The middleware
// chain that makes it safe to expose (correlation, tracing, access logging, body
// limits, the request budget, load shedding, panic recovery) is applied by the
// composition root through httpx.Harden, because a feature package must not
// import concrete infra adapters. The important part is that the chain is applied
// by the shared code rather than rebuilt here: a second hand-built chain is how a
// service ends up with none of those protections while believing it inherited all
// of them.
func NewAPIHandler(articles *article.Service, opts Options) (http.Handler, error) {
	if articles == nil {
		return nil, errors.New("reference api: article service is required")
	}
	if strings.TrimSpace(opts.WriteToken) == "" {
		return nil, errors.New("reference api: write token is required")
	}
	if opts.RejectRequest == nil || opts.RejectResponse == nil {
		return nil, errors.New("reference api: request and response rejection mappers are required")
	}
	spec, err := openapi.GetSpec()
	if err != nil {
		return nil, fmt.Errorf("reference api: load OpenAPI spec: %w", err)
	}

	validator := oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
		DoNotValidateServers: true,
		// The spec's securitySchemes drive this call, so an operation marked
		// protected cannot reach a handler without passing the check.
		Options: openapi3filter.Options{
			AuthenticationFunc: func(_ context.Context, input *openapi3filter.AuthenticationInput) error {
				return authenticateBearer(input, opts.WriteToken)
			},
		},
		ErrorHandlerWithOpts: func(
			_ context.Context,
			err error,
			w http.ResponseWriter,
			r *http.Request,
			_ oapimiddleware.ErrorHandlerOpts,
		) {
			opts.RejectRequest(w, r, err)
		},
	})

	strict := openapi.NewStrictHandlerWithOptions(&handler{articles: articles}, nil, openapi.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  opts.RejectRequest,
		ResponseErrorHandlerFunc: opts.RejectResponse,
	})

	return openapi.HandlerWithOptions(strict, openapi.ChiServerOptions{
		Middlewares:      []openapi.MiddlewareFunc{validator},
		ErrorHandlerFunc: opts.RejectRequest,
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
