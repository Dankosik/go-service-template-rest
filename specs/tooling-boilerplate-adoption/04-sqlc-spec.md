# Spec 04: SQL Data Access Boilerplate Reduction via `sqlc`

## Problem
The repository already uses Postgres (`pgx`) and SQL migrations. As business features grow, handwritten query + scan + mapping code will become a major boilerplate source and a correctness risk.

## Goals
1. Adopt `sqlc` for type-safe query code generation from SQL files.
2. Keep explicit SQL ownership and reviewability.
3. Preserve current architecture boundaries (`internal/app` depends on domain interfaces; concrete DB access stays in `internal/infra`).
4. Keep migration-first workflow as source of schema truth.

## Non-Goals
- No ORM adoption.
- No replacement of migration tooling.
- No automatic repository redesign outside touched features.

## Decisions (Normative)
1. `sqlc` is the only approved SQL query code generator in this repository.
2. Schema source for generation is migration SQL in `env/migrations/`.
3. Query source is explicit `.sql` files under an infra-owned directory.
4. Generated code stays in an infra-owned package; app/domain layers do not import sqlc package directly.
5. `sqlc` is pinned via `go.mod` tool directives and executed via `go tool sqlc`.
6. Generated artifacts are versioned in git and protected by drift checks.

## Package and File Layout
Proposed baseline layout:
```text
internal/infra/postgres/sqlc.yaml
internal/infra/postgres/queries/*.sql
internal/infra/postgres/sqlcgen/*.go
```

Allowed schema inputs:
- `env/migrations/*.up.sql`

Contract boundary:
- `internal/app` consumes repository interfaces.
- `internal/infra/postgres` wraps sqlc-generated queries and maps to domain types where needed.

## Query and Transaction Rules
1. Every query has a named SQL file entry and generated method.
2. Multi-step write flows use explicit transaction boundaries in infra wrappers.
3. Query semantics (locking, ordering, pagination) are expressed in SQL, not hidden in Go helpers.
4. No raw SQL string literals in app/service layer.

## Implementation Plan

### WP-1: Tool Bootstrap
- Add `sqlc` tool directive to `go.mod`.
- Add `sqlc.yaml` with PostgreSQL + `pgx/v5` generation settings.
- Add make targets:
  - `make sqlc-generate`
  - `make sqlc-check` (generate + drift check)

### WP-2: First Vertical Slice
- Implement one real repository use-case through sqlc (start with simple table path, e.g., `ping_history`).
- Add infra adapter methods delegating to generated querier.
- Keep public behavior unchanged.

### WP-3: Test Baseline
- Unit tests for infra wrapper behavior and error mapping.
- Integration tests against Postgres testcontainers for at least one read and one write path.

### WP-4: CI and Drift Protection
- Include `sqlc-check` in local/CI quality chain.
- Require generated code updates in same PR as query/schema changes.

### WP-5: Incremental Migration Strategy
- Apply on touch for new or refactored DB flows.
- No forced big-bang migration of all existing DB code.

## Validation
Mandatory evidence after implementation:
1. `make sqlc-generate`
2. `make sqlc-check`
3. `make test`
4. `make test-integration`
5. `make lint`

Acceptance checks:
- Query changes without regenerated artifacts must fail drift check.
- App layer contains no direct SQL or sqlc imports.

## Rollout Strategy
1. Start with one low-risk query set to validate config/layout.
2. Expand to new features first, then opportunistic migration of touched legacy paths.
3. Keep rollback simple: generated code can be regenerated from committed SQL + migrations.

## Risks and Mitigations
- Risk: schema/query mismatch breaks generation.
  - Mitigation: schema source is migrations; generation runs in CI.
- Risk: leaking generated types into domain API.
  - Mitigation: keep sqlc package infra-local and map to domain contracts at boundary.
- Risk: transaction misuse in wrappers.
  - Mitigation: document explicit transaction rules and cover with integration tests.

## Definition of Done
1. `sqlc` is pinned and callable through `go tool`.
2. `sqlc` config, query directory, and generated package are in place.
3. At least one production path uses sqlc-generated queries through infra wrappers.
4. CI/local checks detect sqlc generation drift.
5. Architecture boundaries remain intact (`internal/app` not coupled to generated DB package).
