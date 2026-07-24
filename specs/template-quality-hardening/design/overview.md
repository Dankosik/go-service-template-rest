# Template quality hardening design
status: ready

## Runtime decisions

### PostgreSQL environment boundary

`internal/infra/postgres` keeps ownership of DSN preflight. Replace the broad
`PG*` prefix check with an explicit set of non-empty environment variables
recognized by the current pgx/libpq parser. The test process must clear only
that same owned set, so unrelated host variables remain visible to regression
tests.

### Ingress policy

`cmd/service/internal/bootstrap` remains the network-policy owner.
`EnforceIngress` performs both declaration validation and the terminal
public-ingress rejection. Delete ingress exception parsing, state, runtime
expiry checks, and the later metrics-exposure gate. Egress exception parsing
and enforcement keep their current owner and behavior.

### Config loading and normalization

`internal/config.LoadDetailedWithContext` passes the caller's context through
the synchronous load and validation stages without manufacturing stage
deadlines. `cmd/service/internal/bootstrap.withStageBudget` uses `context`
directly for real context-aware probe/server stages.

App, HTTP, and observability validators accept pointers, trim the values they
own, then validate the normalized values. Snapshot decoding stays responsible
only for shape/type decoding and unknown-key discovery.

### PostgreSQL startup

`startup_dependencies.go` keeps retry and concrete PostgreSQL lifecycle
ownership. The flow is:

1. skip when PostgreSQL is disabled;
2. resolve and enforce the egress target;
3. verify remaining startup budget;
4. create the probe context and span;
5. connect with the existing bounded retry policy;
6. classify context/budget failures, record telemetry, and close partial state;
7. publish the pool as the readiness probe and cleanup owner.

No generic probe specification, result envelope, or abort hook survives.

## Reference feature flow and ownership

`examples/reference-service` is an isolated executable example:

```text
OpenAPI request validator
  -> generated strict-server adapter
  -> article HTTP handler
  -> article use case
  -> consumer-owned Repository port
  -> in-memory adapter
```

- `api/openapi.yaml` is the example wire-contract authority.
- `internal/openapi/openapi.gen.go` is generated output and participates in the
  existing OpenAPI drift gate.
- `internal/article` owns the entity, `ErrNotFound`, consumer-side repository
  port, and use case.
- `internal/article/memory` owns the immutable in-memory adapter.
- `internal/httpapi` owns HTTP/domain mapping and Problem bodies.
- `main.go` owns only construction, middleware, route wiring, and process
  startup.

The article endpoint is deliberately read-only and public because it exposes
only static example content. It has no durable state, identity, or side
effects. It is not reachable from the production service.

## Cleanup and proof ownership

- Delete obsolete ingress configuration documentation and tests together with
  the code.
- Delete config stage-budget tests and the exported shared budget helper;
  retain parent-context and elapsed-stage coverage.
- Replace generic dependency-probe tests with behavior tests against the
  concrete PostgreSQL flow.
- Extend the existing generated drift script and OpenAPI checks to include the
  example contract and generated package.
- Focused package tests own local behavior; `make openapi-check`, race checks,
  `make lint`, and `make check` own repository-level acceptance.
