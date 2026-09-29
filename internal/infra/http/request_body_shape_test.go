package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/example/go-service-template-rest/internal/reqctx"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	oapimiddleware "github.com/oapi-codegen/nethttp-middleware"
)

const streamedBodyContract = `
openapi: 3.0.3
info: {title: streamed body, version: "1"}
components:
  securitySchemes:
    bearerAuth: {type: http, scheme: bearer}
  schemas:
    Item:
      type: object
      additionalProperties: false
      required: [id, name]
      properties:
        id: {type: integer, format: int64}
        name: {type: string}
        count: {type: integer, format: int32}
        ratio: {type: number}
        ok: {type: boolean}
        tags: {type: array, items: {type: string}}
        meta:
          type: object
          properties:
            level: {type: integer}
paths:
  /items/{id}:
    post:
      security: [{bearerAuth: []}]
      parameters:
        - {name: id, in: path, required: true, schema: {type: integer, format: int32}}
        - {name: X-Ids, in: header, schema: {type: array, items: {type: integer}, default: [1, 2]}}
      requestBody:
        required: true
        content:
          application/json:
            schema: {$ref: "#/components/schemas/Item"}
      responses:
        "200": {description: ok}
  /constrained:
    post:
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              properties:
                slug: {type: string, maxLength: 3}
      responses:
        "200": {description: ok}
  /a/{x}/c:
    post:
      parameters: [{name: x, in: path, required: true, schema: {type: string}}]
      requestBody:
        required: true
        content:
          application/json:
            schema: {type: object, required: [s], properties: {s: {type: string}}}
      responses:
        "200": {description: ok}
  /a/b/{y}:
    post:
      parameters: [{name: y, in: path, required: true, schema: {type: string}}]
      requestBody:
        required: true
        content:
          application/json:
            schema: {type: object, properties: {n: {type: integer}}}
      responses:
        "200": {description: ok}
`

// The streaming pass may only change how a body is validated, never the answer.
// Every request must get the response the library validator gave before it, with
// the same principal and the same body reaching the handler — including on a
// secured operation whose resolver consumes the credential, as the production
// bearer resolver does.
func TestStreamedRequestBodyMatchesLibraryValidator(t *testing.T) {
	t.Parallel()

	spec := loadStreamedBodyContract(t)
	item := spec.Paths.Find("/items/{id}").Post
	bodies := structuralRequestBodies(spec)
	if _, ok := bodies[item]; !ok {
		t.Fatal("structural Item body did not compile; the streaming pass would never run")
	}
	if _, ok := bodies[spec.Paths.Find("/constrained").Post]; ok {
		t.Fatal("a maxLength schema compiled; the streaming pass would skip that constraint")
	}

	reject := func(w http.ResponseWriter, _ *http.Request, err error) {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	streamed := validatedEchoRouter(mustRequestValidator(t, spec, Authenticated(consumeBearer), reject))
	library := validatedEchoRouter(oapimiddleware.OapiRequestValidatorWithOptions(loadStreamedBodyContract(t), &oapimiddleware.Options{
		DoNotValidateServers: true,
		Options:              openapi3filter.Options{AuthenticationFunc: Authenticated(consumeBearer)},
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, r *http.Request, _ oapimiddleware.ErrorHandlerOpts) {
			reject(w, r, err)
		},
	}))

	const itemPath = "/items/7"
	for _, tt := range []struct {
		name        string
		path        string
		contentType string
		anonymous   bool
		body        string
		// streamed marks a body the pass must admit; otherwise the optimization
		// could silently fall back for every request.
		streamed bool
	}{
		{name: "valid", path: itemPath, streamed: true, body: `{"id":1,"name":"a","count":-2147483648,"ratio":1.5e3,"ok":true,"tags":["x"],"meta":{"level":3,"extra":[1]}}`},
		{name: "charset parameter", path: itemPath, streamed: true, contentType: "application/json; charset=utf-8", body: `{"id":1,"name":"a"}`},
		{name: "escaped member name", path: itemPath, streamed: true, body: `{"id":1,"\u006eame":"a"}`},
		{name: "missing credential", path: itemPath, anonymous: true, body: `{"id":1,"name":"a"}`},
		{name: "escaped unknown member", path: itemPath, body: `{"id":1,"name":"a","\u0078":1}`},
		{name: "missing required", path: itemPath, body: `{"id":1}`},
		{name: "unknown member", path: itemPath, body: `{"id":1,"name":"a","extra":1}`},
		{name: "wrong member type", path: itemPath, body: `{"id":"1","name":"a"}`},
		{name: "null member", path: itemPath, body: `{"id":1,"name":null}`},
		{name: "int32 overflow", path: itemPath, body: `{"id":1,"name":"a","count":2147483648}`},
		{name: "fractional integer", path: itemPath, body: `{"id":1.5,"name":"a"}`},
		{name: "integral exponent", path: itemPath, body: `{"id":1e3,"name":"a"}`},
		{name: "integer beyond float precision", path: itemPath, body: `{"id":9223372036854775807,"name":"a"}`},
		{name: "array item type", path: itemPath, body: `{"id":1,"name":"a","tags":[1]}`},
		{name: "duplicate member", path: itemPath, body: `{"id":1,"name":"a","name":"b"}`},
		{name: "invalid utf8", path: itemPath, body: "{\"id\":1,\"name\":\"\xff\"}"},
		{name: "trailing value", path: itemPath, body: `{"id":1,"name":"a"} {}`},
		{name: "malformed", path: itemPath, body: `{"id":1,`},
		{name: "not an object", path: itemPath, body: `[]`},
		{name: "empty body", path: itemPath, body: ``},
		{name: "other media type", path: itemPath, contentType: "text/plain", body: `{"id":1,"name":"a"}`},
		{name: "invalid path parameter", path: "/items/x", body: `{"id":1,"name":"a"}`},
		{name: "non-structural operation", path: "/constrained", body: `{"slug":"long"}`},
		{name: "overlapping templates", path: "/a/b/c", body: `{"n":1}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.streamed && !bodies[item].shape.accepts([]byte(tt.body)) {
				t.Fatal("streaming pass did not admit a valid structural body")
			}
			request := func() *http.Request {
				req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, tt.path, strings.NewReader(tt.body))
				req.Header.Set("Content-Type", tt.contentType)
				if tt.contentType == "" {
					req.Header.Set("Content-Type", "application/json")
				}
				if !tt.anonymous {
					req.Header.Set("Authorization", "Bearer token")
				}
				return req
			}
			want := httptest.NewRecorder()
			library.ServeHTTP(want, request())
			got := httptest.NewRecorder()
			streamed.ServeHTTP(got, request())
			if got.Code != want.Code || got.Body.String() != want.Body.String() {
				t.Fatalf("streamed = %d %q, library validator = %d %q", got.Code, got.Body, want.Code, want.Body)
			}
		})
	}
}

// consumeBearer removes the credential it accepts, as bearerauthn does, so a
// second authentication of the same request would fail.
func consumeBearer(_ context.Context, input *openapi3filter.AuthenticationInput) (reqctx.Principal, error) {
	request := authenticatedRequest(input)
	credential := request.Header.Get("Authorization")
	request.Header.Del("Authorization")
	if credential != "Bearer token" {
		return reqctx.Principal{}, errors.New("credential missing")
	}
	return reqctx.Principal{Subject: "caller"}, nil
}

func loadStreamedBodyContract(t *testing.T) *openapi3.T {
	t.Helper()

	spec, err := openapi3.NewLoader().LoadFromData([]byte(streamedBodyContract))
	if err != nil {
		t.Fatalf("load contract: %v", err)
	}
	if err := spec.Validate(t.Context()); err != nil {
		t.Fatalf("validate contract: %v", err)
	}
	return spec
}

// validatedEchoRouter mounts every contract path behind the validator and
// echoes the caller and the body the handler received. Chi alone would route
// POST /a/b/c to /a/b/{y}; the validator must follow its own router instead.
func validatedEchoRouter(validator func(http.Handler) http.Handler) http.Handler {
	echo := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		principal, _ := reqctx.PrincipalFromContext(r.Context())
		_, _ = io.WriteString(w, principal.Subject+" "+r.Header.Get("X-Ids")+" "+string(body)) //nolint:gosec // Test echo compared byte-for-byte; no browser renders it.
	})
	router := chi.NewRouter()
	for _, path := range []string{"/items/{id}", "/constrained", "/a/{x}/c", "/a/b/{y}"} {
		router.With(validator).Post(path, echo)
	}
	return router
}
