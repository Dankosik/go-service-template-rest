# First production feature

This is the maintained path from the health-only scaffold to one useful
vertical slice. The runnable code under `examples/reference-service` shows the
package shape, but it is isolated demonstration code: do not import it into the
production service or treat its in-memory repository as production-ready.

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

Start with sibling unit tests for business invariants and error identity. The
reference `article.Service` is the closest small example of this boundary.

## 3. Evolve the REST contract

Edit `api/openapi/service.yaml`, including a stable `operationId`, concrete
request/response schemas, Problem responses, and real security declarations
when protected. Run:

```bash
make openapi-generate
make openapi-check
```

Generated files under `internal/openapi` are derived output. Do not hand-edit
them. Implement the generated strict-server operation in
`internal/infra/http/<feature>_handlers.go`, map transport data to feature
types, and map domain errors through the existing Problem response owner.
Extend `httpx.Handlers` and wire the concrete feature in
`cmd/service/internal/bootstrap`; do not register a parallel manual API route.

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

For a service initialized with `DATABASE=none`, re-derive from the template or
bring in the PostgreSQL profile deliberately; do not copy only a pool file and
omit migrations, CI, image, and deployment ownership.

## 6. Call an external HTTP service

Create a provider adapter under `internal/infra/<provider>`. Reuse
`internal/infra/httpclient` for fixed-authority URL validation, transport
bounds, trace propagation, response-size limits, and idle-connection cleanup.
Keep provider authentication, per-operation timeout, retry eligibility,
provider error mapping, and generated client ownership in the adapter. Add
tests for timeout/cancellation, oversized responses, redirects, error bodies,
and cleanup. Dynamic or user-controlled URLs require a separate SSRF design.

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
