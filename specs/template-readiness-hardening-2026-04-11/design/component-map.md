# Component Map

## `env/` And `internal/config/`

- Align `env/.env.example` with config defaults and validators.
- Add fixture-style config validation for `.env.example`.
- Improve secret-like key matching.
- Derive config-test env reset keys from config known keys where practical.
- Potentially add explicit readiness/shutdown budget fields if needed by reliability fixes.

Stable: config remains the owner of snapshot building and validation.

## `cmd/service/internal/bootstrap/`

- Keep service composition and dependency wiring here.
- Handle `NewRouter` error if router construction becomes fallible.
- Pass explicit readiness gate/admission dependencies to HTTP handlers.
- Extract narrow network policy and dependency rejection helpers.
- Add cleanup stack/teardown handling to dependency admission.
- Document or type degraded dependency status.

Stable: `cmd/service/main.go` stays thin.

## `internal/app/health` And `internal/domain`

- Make `internal/app/health` the readiness probe interface owner, or consciously route through `internal/domain`.
- Preferred: keep `health.Probe`, remove/update the unused domain readiness example, and document when `internal/domain` should be introduced.

Stable: readiness behavior belongs in app-level health service; concrete probes live in adapters/bootstrap.

## `internal/infra/http`

- Remove production fallback dependency creation.
- Add explicit readiness/admission gate for `/health/ready`.
- Make readiness timeout configurable or passed from config.
- Sanitize strict request error details.
- Add route-owner guard around `/metrics` manual/generated overlap.
- Preserve fail-closed CORS policy.

Stable: normal API route registration remains generated via `internal/api`.

## `api/openapi` And `internal/api`

- Remove or reconcile unused `bearerAuth`.
- Keep `security: []` only if docs explicitly treat current endpoints as public system/sample endpoints.
- If `/metrics` remains in OpenAPI, document and test the runtime root-router exception.
- Regenerate `internal/api` only from OpenAPI changes.

Stable: `api/openapi/service.yaml` is the REST source of truth.

## `internal/infra/postgres`, `env/migrations`, And `test/`

- Decide the concrete treatment of `ping_history` after checking sqlc behavior.
- Prefer no hidden production sample schema.
- Make migration examples deterministic unless intentionally idempotent.
- Clarify repository/unit/integration test placement.

Stable: migrations and queries are the source for sqlc-generated code.

## Docs And Makefile

- Update structure docs with bootstrap wiring, endpoint recipe, security decision step, test placement, persistence flow, and outbound adapter security expectations.
- Add feature validation targets to `make help`.
- Keep command details in `docs/build-test-and-development-commands.md`.

