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

<!-- profile:grpc:start -->
### Native gRPC operation

When the accepted contract is gRPC instead of, or beside, REST, define it under
`api/proto`, generate `internal/gen/proto`, implement the generated interface
as a thin adapter over the same feature service, and register it through
the `newGRPCRuntime` call in `cmd/service/internal/bootstrap/run.go`. Do not duplicate the use case
for streaming or return raw dependency statuses. Run `make proto-check` and
the focused gRPC tests from [Native gRPC](grpc.md); add
`BASE_REF=origin/main make proto-breaking` once the contract has a published
base.
<!-- profile:grpc:end -->

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

### What a rejected request tells the client

Every rejection from the request validator collapses to one response: `400` with
`code: bad_request` and the detail `request is malformed or invalid`. A bad
enum, a missing required field, and an over-length string are indistinguishable
to the caller. That is deliberate — a validation message echoes attacker-supplied
input back out — but it moves integration debugging from the client's screen into
your logs, and consumers will ask.

Naming the failing field is safe; echoing its value is not. `kin-openapi` already
carries the location, so the opt-in is to widen the Problem schema with an
`errors` array and fill it from the pointer only:

```go
var schemaErr *openapi3.SchemaError
if errors.As(err, &schemaErr) {
    // JSONPointer() is the field path, for example ["name"]. schemaErr.Value is
    // the rejected input: never copy it into the response.
    field := strings.Join(schemaErr.JSONPointer(), "/")
    _ = field
}
```

Add the matching `errors` property to the `Problem` schema in
`api/openapi/service.yaml`, regenerate, and extend `writeProblem` in
`internal/infra/http/problem.go`. Keep the framework-level problems — `404`,
`405`, `413`, `504` — detail-free; they carry no field to name.

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
feature. Neither `migrations/` nor `internal/infra/postgres/queries/` exists yet:
`project-structure-check` rejects both until they hold real content, so you
create each one with its first file rather than inheriting an empty placeholder.

`golang-migrate` applies migrations in filename order, and every change is a
pair:

```text
migrations/0001_create_orders.up.sql
migrations/0001_create_orders.down.sql
```

Keep one logical change per pair with a `down` that genuinely reverses `up`, go
additive first — add a column nullable, backfill, constrain it in a later pair —
so a rollback never strands rows, and keep `CREATE INDEX CONCURRENTLY` out of the
transaction the runner opens. `sqlc.yaml` reads `migrations/*.up.sql` as the
schema, so a new table reaches generation only once its `up` file exists.

Each query file names itself and its result shape:

```sql
-- name: GetOrder :one
SELECT id, customer_id, total_cents FROM orders WHERE id = $1;
```

Add paired deterministic migrations under `migrations/`, SQL query
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
bounds, explicit correlation-policy enforcement, response-size limits, and
idle-connection cleanup.
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

The zero propagation policy is `PropagationNone`: local client telemetry
remains, but no trace or request ID is disclosed remotely. Select
`PropagationTraceContext` for an approved W3C-only boundary, or
`PropagationTrustedService` for a service allowed to receive both W3C Trace
Context and the valid request ID already in the operation context. Private DNS
or TLS alone does not establish that trust. All modes remove caller-supplied
`traceparent`, `tracestate`, `baggage`, and `X-Request-ID` before each attempt;
baggage is never propagated.

Generate the provider client from its authoritative versioned OpenAPI schema,
then give it the bounded client through oapi-codegen's generated seam:

```go
bounded, err := httpclient.New(httpclient.Config{
    DependencyName:         "orders",
    BaseURL:                "http://orders.railway.internal:8080",
    TargetClass:            httpclient.PrivateHTTP,
    PrivateHostSuffix:      "railway.internal",
    RequestTimeout:         2 * time.Second,
    ResponseHeaderTimeout:  time.Second,
    MaxResponseHeaderBytes: 16 << 10,
    MaxResponseBodyBytes:   1 << 20,
    MaxConnsPerHost:        16,
    Propagation:            httpclient.PropagationTrustedService,
}, metrics.MeterProvider())
if err != nil {
    return err
}

orders, err := ordersv1.NewClient(
    bounded.BaseURL(),
    ordersv1.WithHTTPClient(bounded),
)
if err != nil {
    bounded.CloseIdleConnections()
    return err
}
// Bootstrap retains bounded for reuse and closes its idle pool at shutdown.
_ = orders // The adapter calls the provider's generated operations with ctx unchanged.
```

The adapter passes the operation context unchanged and maps generated
transport types and errors into feature-owned types and errors. It does not
hand-edit generated code or duplicate the provider schema locally.

<!-- profile:grpc:start -->
### Call another service over gRPC

Create one long-lived connection per fixed dependency with
`internal/infra/grpcclient`, share it between that dependency's generated
clients, and let bootstrap close the connection during shutdown. The
constructor is lazy: successful construction does not prove that the target is
reachable. Each operation still owns its deadline, retry eligibility, and
provider-error mapping.

Select propagation at the dependency's trust boundary. `PropagationNone` is
the zero value: it retains local client telemetry while sending no remote
correlation. `PropagationTraceContext` sends only W3C Trace Context.
`PropagationTrustedService` additionally sends the valid request ID already in
the operation context. Private DNS or TLS does not by itself justify the
trusted-service policy. Every policy removes stale `traceparent`,
`tracestate`, `baggage`, and `x-request-id` metadata before the call; baggage
is never propagated.

```go
connection, err := grpcclient.New(
    grpcclient.DefaultConfig("dns:///orders.railway.internal:9000"),
    grpcclient.Options{
        TransportCredentials: credentials.NewTLS(tlsConfig),
        Propagation:          grpcclient.PropagationTrustedService,
    },
)
if err != nil {
    return err
}
// Bootstrap retains connection for reuse and calls connection.Close() during
// shutdown or after any later startup failure.

healthClient := healthgrpc.NewHealthClient(connection)
_ = healthClient // Provider-generated clients use the same grpc.ClientConnInterface seam.
```

Pass the operation context unchanged to the generated method. The shared
connection deliberately ignores environment proxies and resolver-provided
service configurations, so a proxy, resolver-selected balancer, or configured
retry cannot silently bypass its metadata policy. A dependency that requires
one of those mechanisms needs a separate design. grpc-go's native transparent
retry may still occur before commitment; application retry remains an explicit
per-operation adapter decision.
<!-- profile:grpc:end -->

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
