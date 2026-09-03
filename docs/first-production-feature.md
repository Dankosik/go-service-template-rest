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
them. Implement the generated strict-server operation in the service's own
package as `<feature>_handlers.go` and inject it as `httpx.Handlers.API`;
`internal/infra/http` is shared template surface that adding an operation does
not edit. Map transport data to feature
types. Own domain error identities and their `failure.Mapper` beside the feature;
return those errors for the composition root's shared HTTP/gRPC mapping, or a
generated typed Problem response when the operation itself owns the rejection.
Framework-level transport failures continue to use the shared hand-written
Problem catalog.

Wrap with `failure.Op("store row", err)` rather than `fmt.Errorf` wherever a use
case has several steps that fail the same way. Both preserve the sentinel
identity `errors.Is` matches on, but only `Op` reaches the record a transport
writes for an unclassified failure: that record prints the class chain and no
message text, so a plain `fmt.Errorf` layer renders as `*fmt.wrapError` and an
operator cannot tell which step broke. Pass a literal — an interpolated
identifier puts request data in the one record the boundary sanitized.

A request the OpenAPI validator rejects answers 400 with `invalid_params`, which
names each failed member and the constraint it broke. That comes from the
contract and never carries the submitted value, so a feature adds nothing to get
it; see `requestViolations` in `internal/infra/http` for the rule that governs
what may appear there.
Set `httpx.Handlers.API` to the concrete feature handler in
`cmd/service/internal/bootstrap`; do not register a parallel manual API route.
Append the feature's `failure.Mapper` to the local `domainErrors` slice in
`cmd/service/internal/bootstrap/run.go`; the same slice feeds the HTTP and gRPC
boundaries. The two registrations fail differently when forgotten: a missing
`API` stops the process at startup, while a missing mapper is silent and answers
a sanitized 500 or `INTERNAL` in place of the status the contract documents.

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

For each runtime key, update the typed field and `koanf` tag, its default when
appropriate, and its validation — all three in the section's own
`internal/config/<section>_config.go` — plus the relevant example. Secret values belong only in `APP__...` environment
variables. Add snapshot and invalid-value tests. Unknown keys already fail, so
a misspelled deployment variable cannot silently fall back to a default.

## 5. Add PostgreSQL only when selected

Initialize the service with `DATABASE=postgres` before building a persistence
feature. Neither `migrations/` nor `internal/infra/postgres/queries/` exists yet:
Create each one with its first real file rather than inheriting an empty
placeholder.

Goose applies migrations by the fixed-width numeric prefix. Every change is one
file with explicit directions:

```text
migrations/000001_orders_create.sql
```

```sql
-- +goose Up
CREATE TABLE orders (...);

-- +goose Down
DROP TABLE orders;
```

Keep one logical change per file with a `Down` section that genuinely reverses
`Up`. Go additive first — add a column nullable, backfill, constrain it in a
later migration — so rollback does not strand rows. Every migration is
transactional: `-- +goose NO TRANSACTION`, environment substitution, Go
migrations, nested directories, symlinks, and non-canonical filenames are
rejected before a database connection is opened. `sqlc.yaml` reads the
directory and derives schema from Goose `Up` sections.

Each query file names itself and its result shape:

```sql
-- name: GetOrder :one
SELECT id, customer_id, total_cents FROM orders WHERE id = $1;
```

Add deterministic Goose migrations under `migrations/`, SQL query
sources under `internal/infra/postgres/queries`, and regenerate with:

```bash
make sqlc-generate
make sqlc-check
make migration-check
make migration-validate
```

Keep generated SQL types behind a hand-written Postgres repository that maps
to feature-owned types. Construct its root queries with `sqlcgen.New(pool)`.
The use case owns the transaction boundary and joins generated methods through
the one callback seam:

```go
return postgres.InTx(ctx, pool, pgx.TxOptions{}, func(tx pgx.Tx) error {
	txQueries := queries.WithTx(tx)
	if err := txQueries.InsertOrder(ctx, order); err != nil {
		return err
	}
	return txQueries.InsertOrderEvent(ctx, event)
})
```

`InTx` owns begin, bounded rollback, commit, and unknown-commit
classification. Do not begin or finish the transaction manually and do not
hide transaction state in context. `internal/infra/postgresmigrate` belongs
only to `cmd/migrate` and must not enter the service dependency graph.

When `examples/reference-service` is present, the executable reference for that
shape is the `pgAdapter` in
`examples/reference-service/postgres_unitofwork_integration_test.go`: it is the
in-memory repository's replacement, satisfying the same feature-owned interfaces
against a real pool, and it composes two repository calls in one transaction.
Read it before writing the first `internal/infra/postgres/<feature>_repository.go`.
For a repository integration test, `pgtest.Migrated` supplies the isolated,
migrated DSN and registers database cleanup; open the pool with the production
PostgreSQL adapter and close it through the test cleanup.

The PostgreSQL profile is required, not a degraded mode:
`APP__POSTGRES__ENABLED` must remain `true` and the DSN must be configured
before either the service or migrator starts. Bootstrap retains the initialized
`*pgxpool.Pool`; feature composition injects that concrete library type into
the hand-written repository. The template fixes connection, readiness,
statement, migration, and cleanup budgets. A deployment sets only the DSN and,
when capacity evidence requires it, `postgres.max_open_conns`.

For a service initialized with `DATABASE=none`, re-derive from the template or
bring in the PostgreSQL profile deliberately; do not copy only a pool file and
omit migrations, CI, image, and deployment ownership.

## 6. Call an external HTTP service

Create a provider adapter under `internal/infra/<provider>`. Reuse
`internal/infra/httpclient` for one fixed public HTTPS authority. A deployment
with an existing private HTTPS route uses `NewPrivateHTTPS` and supplies its
private DNS suffix; there is no platform-specific default. The fixed client
ignores `HTTP_PROXY`/`HTTPS_PROXY` on purpose, because a proxy would dial on
the client's behalf and bypass the post-DNS address gate; a provider that must
be reached through a mandatory egress proxy uses a plain `net/http` client.
Construction also requires provider-wide response-header, decoded-body, and
request-concurrency ceilings. The provider adapter chooses those values,
operation deadlines, and any smaller body limit; `DoWithPolicy` enforces the
smaller non-streaming pair without weakening the transport ceiling.
Authentication, parsing, retry eligibility, provider errors, and telemetry stay
in the provider adapter or official SDK. There is no generic streaming escape
hatch: a real streaming operation must add explicit duration, idle, byte/rate,
and concurrency bounds. Dynamic or user-controlled URLs require a
feature-specific SSRF design.

Generate the provider client from its authoritative versioned OpenAPI schema,
then give it the fixed-target client through oapi-codegen's generated seam:

```go
bounded, err := httpclient.NewExternalHTTPS(
    cfg.OrdersBaseURL,
    cfg.OrdersTransportLimits,
)
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

```go
connection, err := grpcclient.New(
	"dns:///orders.railway.internal:9000",
    grpcclient.Options{
        TransportCredentials: credentials.NewTLS(tlsConfig),
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
connection uses native DNS, reconnects, `pick_first`, transparent retry, and W3C
Trace Context. Resolver service config is disabled, so application retry,
client-side health, or a different load-balancing policy cannot appear without
a separate dependency decision. Pass provider-owned dynamic credentials through
`Options.PerRPCCredentials`. The template sets no client keepalive policy.
<!-- profile:grpc:end -->

## 7. Wire, observe, and prove

Composition belongs in `cmd/service/internal/bootstrap`. Every acquired
resource needs cleanup on later startup failure and bounded shutdown.
Readiness should include only dependencies whose loss makes the instance
unable to serve; liveness must remain process-local.

Before production promotion, replace the unresolved entries in
[`production-contract.md`](production-contract.md) with service-owned scope,
dependency, capacity, durability, trust, SLO, and recovery decisions. Do not
invent missing platform values; keep promotion blocked until their owner
supplies them.

Application traffic uses the configured HTTP listener. Prometheus exposition
uses the separate `observability.metrics.addr` diagnostics listener and is
never available from the application listener. Its shipped `:9090` default
binds every interface, so deployment must keep it private; enabling `pprof`
requires stricter network or authentication policy. Add low-cardinality feature
metrics or spans only where they answer an operational question.

Use focused tests while iterating. Optional package-sized iteration is
`make prove PKG=./internal/<feature> FILES='internal/<feature>/*.go'`. On the
integrated candidate run `make verify` once. Use `ALLOW_FULL=1 make check` only
when the intended claim spans the full repository.

Run the matching container, PostgreSQL, migration, or deployment leaf when the
change touches it. Before merge, inspect the generated diff and
verify that the OpenAPI YAML, migrations/query sources, and generated outputs
still have one unambiguous source of truth.
