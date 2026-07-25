# First production feature

This is the maintained path from the health-only scaffold to one useful
vertical slice.

When `examples/reference-service` is present, its runnable code shows the
package shape, but it is isolated demonstration code: do not import it into the
production service or treat its in-memory repository as production-ready. Note
that it uses its own `internal/httpapi` and in-memory repository rather than the
service's `internal/infra/http` and `internal/infra/postgres`, so read it for the
ownership boundaries, not as a layout to copy. Initialization removes it unless
you pass `REFERENCE_EXAMPLE=keep`; service OpenAPI gates discover it only when
present, and it remains readable
[upstream](https://github.com/Dankosik/go-service-template-rest/tree/main/examples/reference-service).

## 1. Close the behavior and trust decision

Write down the resource, operation, success response, expected failures, and
compatibility requirement before editing the contract. Decide whether the
operation is public, protected by a real identity/authorization design, or
blocked. The template intentionally supplies no placeholder authentication.

If the feature calls another service or persists data, also decide who owns the
source of truth, timeout and retry eligibility, transaction boundary,
readiness participation, and cleanup on partial startup.

## 2. Add feature-owned behavior

Create `internal/<feature>` with concrete business types and a service/use-case
type. Keep HTTP status codes, generated OpenAPI types, SQL rows, and provider
payloads outside this package. Add a narrow interface only when the feature
actually needs dependency inversion over persistence or an outbound system.

Start with sibling unit tests for business invariants and error identity. When
the reference example is present, its `article.Service` is the closest small
example of this boundary.

## 3. Evolve the REST contract

Edit `api/openapi/service.yaml`, including a stable `operationId`, concrete
request/response schemas, Problem responses, and real security declarations
when protected. Run:

```bash
make openapi-generate
go mod tidy
make openapi-check
```

The health-only baseline needs no request-binding helpers. The first operation
with a path or query parameter makes the generated code import
`github.com/oapi-codegen/runtime`, so `go mod tidy` records it like any other
new import.

Generated files under `internal/openapi` are derived output. Do not hand-edit
them. Implement the generated strict-server operation in
`internal/infra/http/<feature>_handlers.go`, map transport data to feature
types, and return the generated typed Problem response for domain errors.
Framework-level transport failures continue to use the shared hand-written
Problem catalog.
Extend `httpx.Handlers` and wire the concrete feature in
`cmd/service/internal/bootstrap`; do not register a parallel manual API route.

### Protected operations

The template ships no placeholder authentication, and the base contract declares
`security: []`. It does not follow that you have to derive the wiring: a
spec-declared `securityScheme` only becomes a runtime check through
`openapi3filter.AuthenticationFunc`, so `internal/infra/http/router.go` needs the
`Options` field its request validator currently leaves unset.

```go
validator := oapimiddleware.OapiRequestValidatorWithOptions(spec, &oapimiddleware.Options{
    DoNotValidateServers: true,
    // The spec's securitySchemes drive this call, so an operation marked
    // protected cannot reach a handler without passing the check.
    Options: openapi3filter.Options{
        AuthenticationFunc: func(ctx context.Context, input *openapi3filter.AuthenticationInput) error {
            return authenticate(ctx, input) // your identity design
        },
    },
    ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, r *http.Request, _ oapimiddleware.ErrorHandlerOpts) {
        // A failed security requirement is 401, not 400: the request was well
        // formed and the credential was the problem.
        var securityErr *openapi3filter.SecurityRequirementsError
        if errors.As(err, &securityErr) {
            w.Header().Set("WWW-Authenticate", "Bearer")
            writeProblem(w, r, problemResponse{code: problemCodeUnauthorized, detail: "credentials are missing or invalid"})
            return
        }
        handleMalformedGeneratedRequest(log, w, r, err)
    },
})
```

Leaving `AuthenticationFunc` unset is fail-closed on this path:
`openapi3filter.ValidateRequest` returns `ErrAuthenticationServiceMissing`, so a
protected operation is refused rather than admitted. Keep it that way. Do not
"fix" that error by installing `openapi3filter.NoopAuthenticationFunc` — it
returns success unconditionally, and the same default is what
[GHSA-r277-6w6q-xmqw](https://github.com/advisories/GHSA-r277-6w6q-xmqw) reported
as an authentication bypass in `ValidationHandler.Load()`, which this repository
does not use.

Two things this does not decide for you. Compare presented credentials in
constant time (`crypto/subtle.ConstantTimeCompare`), and add `unauthorized` and
any `forbidden` entries to the Problem catalog in `internal/infra/http/problem.go`
plus the matching `401`/`403` responses in the OpenAPI contract, so the failures
are declared rather than incidental. Identity, key rotation, authorization, and
audit remain your service's design. `examples/reference-service`, upstream or
kept, has a working end-to-end version of exactly this wiring.

Add boundary tests for the happy path, malformed path/query/body input,
unknown JSON fields, every documented failure status, and the chosen
authentication policy.

## 4. Add configuration

For each runtime key, update the typed field and `koanf` tag in
`internal/config/types.go`, its default when appropriate, validation, and the
relevant example. Secret values belong only in `APP__...` environment
variables. Add snapshot and invalid-value tests. Unknown keys already fail, so
a misspelled deployment variable cannot silently fall back to a default.

## 5. Add PostgreSQL only when selected

Initialize the service with `DATABASE=postgres` before building a persistence
feature. Add paired deterministic migrations under `migrations/`, SQL query
sources under `internal/infra/postgres/queries`, and regenerate with:

```bash
make sqlc-generate
make sqlc-check
make migration-validate
```

Keep generated SQL types behind a hand-written Postgres repository that maps
to feature-owned types. The use case owns the transaction boundary; pass a
transaction-scoped repository explicitly rather than hiding transaction state
in context. `internal/infra/postgresmigrate` belongs only to `cmd/migrate` and
must not enter the service dependency graph.

The PostgreSQL profile is required, not a degraded mode:
`APP__POSTGRES__ENABLED` must remain `true` and the DSN must be configured
before either the service or migrator starts. Bootstrap retains the initialized
`postgres.Pool`; feature composition uses its concrete `PGX()` pool with the
generated sqlc constructor. For a transaction, call `Begin(ctx)`, pass
`queries.WithTx(tx)` to the transaction-scoped repository, and commit or roll
back at the use-case boundary. Do not add query delegates to `postgres.Pool`.

For a service initialized with `DATABASE=none`, re-derive from the template or
bring in the PostgreSQL profile deliberately; do not copy only a pool file and
omit migrations, CI, image, and deployment ownership.

## 6. Call an external HTTP service

Create a provider adapter under `internal/infra/<provider>`. Reuse
`net/http` directly for an ordinary provider-specific client. If the repository
was initialized with `OUTBOUND_HTTP=bounded`, reuse
`internal/infra/httpclient` for fixed-authority URL validation, transport
bounds, trace propagation, response-size limits, and idle-connection cleanup.
Select `TargetClass: ExternalHTTPS` for public providers, or `PrivateHTTP` for
a service reachable only on the platform's private network. `PrivateHTTP`
requires `PrivateHostSuffix`, your platform's private DNS zone — for example
`railway.internal`, or `svc.cluster.local` on Kubernetes. There is no default
on purpose: a platform-specific default would succeed silently on one platform
and fail confusingly everywhere else. The bounded client
ignores `HTTP_PROXY`/`HTTPS_PROXY` on purpose, because a proxy would dial on
the client's behalf and bypass the post-DNS address gate; a provider that must
be reached through a mandatory egress proxy uses a plain `net/http` client.
Keep provider authentication, per-operation timeout, retry eligibility,
provider error mapping, and generated client ownership in the adapter. Let the
deployment platform enforce network egress. Add tests for
timeout/cancellation, oversized responses, redirects, error bodies, and
cleanup. Dynamic or user-controlled URLs require a separate SSRF design.

## 7. Wire, observe, and prove

Composition belongs in `cmd/service/internal/bootstrap`. Every acquired
resource needs cleanup on later startup failure and bounded shutdown.
Readiness should include only dependencies whose loss makes the instance
unable to serve; liveness must remain process-local.

Application traffic uses the configured HTTP listener. Prometheus exposition
uses `observability.metrics.addr` (loopback by default) and is never available
from the application listener. Add low-cardinality feature metrics or spans
only where they answer an operational question.

Use focused tests while iterating, then run:

```bash
go test ./internal/<feature> ./internal/infra/http
make check
```

Run `make check-full` when the change touches the container, PostgreSQL,
migrations, or deployment proof. Before merge, inspect the generated diff and
verify that the OpenAPI YAML, migrations/query sources, and generated outputs
still have one unambiguous source of truth.
