# Dependency Graph

Status: review-ready
Date: 2026-06-02

## Package Direction

Approved dependency shape:

```text
cmd/service/internal/bootstrap
  -> internal/config
  -> internal/infra/postgres
  -> internal/app/billingauthority
  -> internal/app/microlease
  -> internal/app/reconciliation
  -> internal/infra/http

internal/infra/http
  -> internal/api
  -> internal/app/billingauthority
  -> internal/app/microlease

internal/app/billingauthority
  -> internal/domain/money
  -> internal/app/microlease
  -> internal/app/reconciliation
  -> app-owned ports

internal/infra/postgres
  -> internal/infra/postgres/sqlcgen
  -> internal/domain/money
  -> app-owned types

cmd/billing-worker/internal/bootstrap
  -> internal/config
  -> internal/infra/postgres
  -> internal/infra/redpanda
  -> internal/app/microleaseworker
  -> internal/app/billingauthority

internal/infra/redpanda
  -> internal/api/events/v1
  -> app-owned event command ports
```

## New App Ports

The billing-authority app service should own narrow ports such as:

- account reader/import-state reader;
- balance reader and active-exposure reader;
- idempotency/stored-outcome store;
- usage operation and terminal store;
- microlease issue/read/close/terminal store;
- reconciliation case reader/claimer/writer;
- admission-control reader/writer;
- clock/ID generator abstractions only where tests need determinism.

Ports should be grouped by use-case needs, not by one generic repository
interface. Avoid interface-per-method sprawl; same-package concrete helpers are
acceptable until a real adapter boundary exists.

## Generated Flow

REST:

```text
api/openapi/service.yaml
  -> make openapi-generate
  -> internal/api/openapi.gen.go
  -> internal/infra/http compile/runtime contract tests
```

Events:

```text
api/proto/events/v1/*.proto
  -> make proto-generate
  -> internal/api/events/v1/*.gen.go
  -> internal/infra/redpanda adapter tests
```

SQL:

```text
env/migrations/*.sql
internal/infra/postgres/queries/*.sql
  -> make sqlc-generate
  -> internal/infra/postgres/sqlcgen/*.go
  -> repository and integration tests
```

Generated outputs are never edited by hand.

## Runtime Dependency Graph

HTTP runtime dependencies:

```text
HTTP route
  -> service JWT/JWKS verifier
  -> route-scope middleware
  -> generated strict handler
  -> billing-authority app service
  -> Postgres repository
  -> telemetry observer
```

Worker dependencies:

```text
billing-worker
  -> Postgres probe
  -> Redpanda probe
  -> terminal/checkpoint/close consumers
  -> inbox retry
  -> outbox relay
  -> stale reconciliation
  -> admission-control renewal
```

External service dependencies:

- Pricing-service is not called from inside billing money transactions. Pricing
  snapshot evidence is supplied by callers or upstream contract flow and stored
  as immutable lineage.
- API-key-service is not called by billing for final admission. It may supply
  attribution/policy context to proxy, and `spend_limit_check_required` means
  caller-side billing/proxy spend checks still run.
- Payments-service is not called by this cutover.
- Gonka-proxy is not called by billing-service in request path; it calls billing
  and publishes events.

## Coupling Controls

- Do not share TypeScript proxy contract code into billing-service runtime.
  Use it only as read-only evidence while authoring OpenAPI.
- Do not import generated OpenAPI types into app packages.
- Do not put Redpanda client code behind app service methods.
- Do not make Postgres repository publish Redpanda events directly; use outbox.
- Do not make HTTP handlers run retries or reconciliation loops.
- Do not add a generic `internal/domain` package unless two real consumers need
  the same stable contract.

## Review Checks

Technical design review should verify:

- no app package imports infra or generated transport packages;
- operation-readback scope mismatch is resolved in both OpenAPI and middleware
  design;
- worker task construction is possible without circular imports;
- proxy client contract changes are isolated to proxy adapter seams;
- no direct dependency from billing-service to proxy runtime code is introduced.
