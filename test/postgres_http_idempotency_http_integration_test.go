//go:build integration

package integration_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/failure"
	"github.com/example/go-service-template-rest/internal/httpidempotency"
	httpx "github.com/example/go-service-template-rest/internal/infra/http"
	"github.com/example/go-service-template-rest/internal/infra/postgresidempotency"
	"github.com/example/go-service-template-rest/internal/openapi"
	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/jackc/pgx/v5"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
)

func TestPostgresHTTPIdempotencyAuthenticatedGeneratedHandlerPath(t *testing.T) {
	fixture := newHTTPIDFixture(t, "idempotency-http-path")
	executor, err := postgresidempotency.NewExecutor(
		fixture.store,
		func(tx pgx.Tx) effectRepository { return postgresEffectRepository{tx: tx} },
		httpidempotency.JSONCodec[openapi.HealthLive200TextResponse](http.StatusOK),
	)
	if err != nil {
		t.Fatalf("NewExecutor(): %v", err)
	}
	handler := httpIDAuthenticatedHandler(t, executor)

	first := doHTTPIDRequest(handler, "caller-a", "key-a", "first")
	if first.Code != http.StatusOK || first.Body.String() != "created:first" {
		t.Fatalf("first response = %d %q", first.Code, first.Body.String())
	}
	replay := doHTTPIDRequest(handler, "caller-a", "key-a", "first")
	if replay.Code != http.StatusOK || replay.Body.String() != first.Body.String() {
		t.Fatalf("replay response = %d %q", replay.Code, replay.Body.String())
	}
	changed := doHTTPIDRequest(handler, "caller-a", "key-a", "changed")
	if changed.Code != http.StatusUnprocessableEntity {
		t.Fatalf("changed response = %d, want %d", changed.Code, http.StatusUnprocessableEntity)
	}
	otherCaller := doHTTPIDRequest(handler, "caller-b", "key-a", "other")
	if otherCaller.Code != http.StatusOK || otherCaller.Body.String() != "created:other" {
		t.Fatalf("other caller response = %d %q", otherCaller.Code, otherCaller.Body.String())
	}
	unauthorized := doHTTPIDRequest(handler, "", "key-a", "first")
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized response = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}
	fixture.assertEffects(t, 2)
}

func httpIDAuthenticatedHandler(
	t *testing.T,
	executor httpidempotency.Executor[effectRepository, openapi.HealthLive200TextResponse],
) http.Handler {
	t.Helper()
	loader := openapi3.NewLoader()
	spec, err := loader.LoadFromData([]byte(httpIDOpenAPI))
	if err != nil {
		t.Fatalf("load fixture OpenAPI: %v", err)
	}
	terminal := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := reqctx.PrincipalFromContext(r.Context())
		if !ok {
			http.Error(w, "principal missing", http.StatusInternalServerError)
			return
		}
		value := r.URL.Query().Get("value")
		request, err := httpidempotency.NewRequestFromContext(
			r.Context(),
			httpidempotency.Scope{Caller: principal.Issuer + "\x00" + principal.Subject, Operation: "fixtureCreate"},
			struct {
				Value string `json:"value"`
			}{Value: value},
		)
		if err != nil {
			writeHTTPIDError(w, err)
			return
		}
		response, _, err := httpidempotency.Execute(r.Context(), executor, request,
			func(ctx context.Context, effects effectRepository) (openapi.HealthLive200TextResponse, error) {
				if err := effects.Insert(ctx, value); err != nil {
					return "", err
				}
				return openapi.HealthLive200TextResponse("created:" + value), nil
			})
		if err != nil {
			writeHTTPIDError(w, err)
			return
		}
		if err := response.VisitHealthLiveResponse(w); err != nil {
			http.Error(w, "render response", http.StatusInternalServerError)
		}
	})
	validator := oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
		DoNotValidateServers: true,
		Options: openapi3filter.Options{AuthenticationFunc: httpx.Authenticated(func(
			_ context.Context,
			input *openapi3filter.AuthenticationInput,
		) (reqctx.Principal, error) {
			value := strings.TrimPrefix(input.RequestValidationInput.Request.Header.Get("Authorization"), "Bearer ")
			if value == "" {
				return reqctx.Principal{}, errors.New("fixture credential missing")
			}
			return reqctx.Principal{Issuer: "fixture", Subject: value}, nil
		})},
		ErrorHandlerWithOpts: func(
			_ context.Context,
			_ error,
			w http.ResponseWriter,
			_ *http.Request,
			_ oapimiddleware.ErrorHandlerOpts,
		) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		},
	})
	return httpidempotency.CaptureKey(validator(terminal))
}

func doHTTPIDRequest(handler http.Handler, caller, key, value string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, "/fixture?value="+value, nil)
	request.Header.Set(httpidempotency.Header, key)
	if caller != "" {
		request.Header.Set("Authorization", "Bearer "+caller)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func writeHTTPIDError(w http.ResponseWriter, err error) {
	classified, ok := httpidempotency.ClassifyError(err)
	if !ok {
		http.Error(w, failure.SanitizedDetail, http.StatusInternalServerError)
		return
	}
	switch classified.Code {
	case failure.CodeBadRequest:
		http.Error(w, classified.Detail, http.StatusBadRequest)
	case failure.CodeIdempotencyKeyMismatch:
		http.Error(w, classified.Detail, http.StatusUnprocessableEntity)
	case failure.CodeIdempotencyUnavailable, failure.CodeIdempotencyOutcomeUnknown:
		http.Error(w, classified.Detail, http.StatusServiceUnavailable)
	default:
		http.Error(w, failure.SanitizedDetail, http.StatusInternalServerError)
	}
}

const httpIDOpenAPI = `openapi: 3.0.3
info: {title: idempotency fixture, version: 1.0.0}
paths:
  /fixture:
    post:
      operationId: fixtureCreate
      security: [{bearerAuth: []}]
      parameters:
        - {in: header, name: Idempotency-Key, required: true, schema: {type: string, maxLength: 255}}
        - {in: query, name: value, required: true, schema: {type: string}}
      responses:
        "200": {description: ok}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
`
