# Implementation Plan

## Strategy

Implement in four phases. Each phase should be reviewable on its own and should leave the repository in a validating state.

## Phase 1: Clone And Config Correctness

Goal: remove quickstart-breaking drift and config policy gaps before changing architecture seams.

Work:

- Align `.env.example` shutdown timeout with validated defaults.
- Add a config fixture test for `.env.example`.
- Improve secret-like config key detection and tests.
- Derive config-test environment reset keys from config known keys where practical.

Proof:

- `go test ./internal/config -count=1`
- `make check`

## Phase 2: Composition, Readiness, And Lifecycle Ownership

Goal: make bootstrap the only composition root and make readiness/shutdown semantics explicit.

Work:

- Make HTTP router construction fail on missing required dependencies instead of creating fallbacks.
- Update bootstrap and tests for fallible router construction.
- Resolve readiness probe interface ownership.
- Add external readiness gate based on startup admission.
- Make readiness timeout and shutdown budget semantics explicit in config and validation where needed.
- Add dependency admission cleanup stack/status structure.
- Extract bootstrap-local helpers for ingress policy and dependency rejection telemetry.

Proof:

- `go test ./cmd/service/internal/bootstrap ./internal/app/health ./internal/infra/http -count=1`
- `go test ./internal/config -count=1` if config fields change
- `make check`

## Phase 3: HTTP, Security, Metrics, And Generated Route Boundaries

Goal: remove misleading auth hints and make generated/manual route ownership explicit.

Work:

- Remove or reconcile unused OpenAPI `bearerAuth`; prefer removal for this task.
- Add docs for endpoint security decision requirements.
- Make `/metrics` route ownership explicit and add a guard test for manual/generated overlap.
- Sanitize strict request error details.
- Preserve fail-closed CORS behavior.
- Regenerate OpenAPI artifacts if the contract changes.

Proof:

- `go test ./internal/infra/http -count=1`
- `make openapi-check`
- `make check`

## Phase 4: Persistence Sample, Docs, Make Help, And Test Placement

Goal: make future business-feature placement obvious.

Work:

- Resolve `ping_history` status: remove from default runtime path if possible; otherwise label as explicit template sample.
- Make default migration examples deterministic where retained.
- Update structure docs with bootstrap wiring, endpoint recipe, app/domain/interface rule, Postgres/sqlc flow, security decision step, outbound adapter expectations, and test placement.
- Expand `test/README.md`.
- Add feature validation targets to `make help`.
- Update command docs only if Make target behavior changes.

Proof:

- `make sqlc-check` or `make docker-sqlc-check`
- `make openapi-check` if OpenAPI docs/contract changed in this phase
- `make test-integration` if migrations/repository integration behavior changed
- `make check`

## Reopen Conditions

Reopen planning before implementation continues if:

- removing `ping_history` breaks sqlc generation in a way that changes the chosen data approach,
- auth implementation becomes required instead of docs/contract cleanup,
- metrics exposure moves to a separate listener or changes deployment topology,
- startup admission needs to gate all app traffic rather than only readiness.

