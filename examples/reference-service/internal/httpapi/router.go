package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/example/go-service-template-rest/examples/reference-service/internal/article"
	"github.com/example/go-service-template-rest/examples/reference-service/internal/openapi"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
)

func NewRouter(articles *article.Service) (http.Handler, error) {
	if articles == nil {
		return nil, errors.New("reference router: article service is required")
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
		ErrorHandlerWithOpts: func(
			_ context.Context,
			_ error,
			w http.ResponseWriter,
			_ *http.Request,
			_ oapimiddleware.ErrorHandlerOpts,
		) {
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

func writeProblem(w http.ResponseWriter, body openapi.Problem) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(int(body.Status))
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// The response status is already committed; the caller cannot recover.
		return
	}
}
