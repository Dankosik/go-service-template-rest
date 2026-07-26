package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"

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
	// Authenticate validates the security requirement this contract declares and
	// publishes the caller it proved, so handlers can authorize without reading
	// the credential a second time. The composition root builds it with
	// httpx.Authenticated; no credential material reaches this package.
	//
	// Nil rejects every protected operation with 401, which is the correct
	// default for a contract that declares a scheme nothing implements yet.
	Authenticate openapi3filter.AuthenticationFunc
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
			AuthenticationFunc: opts.Authenticate,
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
