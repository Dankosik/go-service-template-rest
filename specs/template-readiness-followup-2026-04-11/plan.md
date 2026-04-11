# Implementation Plan

## Strategy

Implement in four bounded phases. Each phase should leave the repository reviewable and testable. Do not start a later phase if an earlier phase exposes a design decision that changes security, ingress, readiness concurrency, or generated-code ownership.

## Phase 1: HTTP Contract And Security Decision Guardrails

Goal: make future endpoint behavior explicit and testable before new business routes are added.

Work:

- Add generated chi wrapper `ErrorHandlerFunc` parity so generated parse errors return sanitized Problem responses.
- Add operation-level OpenAPI security decision metadata for existing operations.
- Add a contract test that requires every operation to carry a security decision and enforces protected-operation rules without adding fake auth.
- Document browser-callable endpoint requirements without enabling CORS/session/CSRF runtime.

Proof:

- `go test ./internal/infra/http -count=1`
- `make openapi-check` if OpenAPI or generated output changes

## Phase 2: Trust Boundary And Redaction Hardening

Goal: remove the highest-risk template surprises around public exposure and secret leakage.

Work:

- Require explicit public-ingress declaration for non-local wildcard binds.
- Keep `/metrics` operational-private-required in docs and OpenAPI decision metadata.
- Redact raw panic recovered values in HTTP recovery logs.
- Redact malformed OTLP header entries from telemetry setup errors.
- Clarify YAML secret placeholder policy in docs and tests.

Proof:

- `go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/telemetry -count=1`
- `make openapi-check` if `/metrics` contract metadata changes in this phase

## Phase 3: Config, Readiness, And Dependency Admission Semantics

Goal: make lifecycle/config conventions clear enough for new runtime dependencies.

Work:

- Replace exact `http.shutdown_timeout == 30s` validation with relationship/range validation, or explicitly document the exact lock if kept. Preferred: make it configurable within validated bounds.
- Add aggregate readiness budget validation for enabled sequential probes.
- Add docs rule that `/health/live` must not check external dependencies.
- Add runtime dependency checklist to docs.
- Add package-local dependency label/stage constants or a narrow helper where label duplication risks metrics drift.
- Remove unused `startupLifecycleStartedAt` plumbing if still unused.
- Remove `runtimeIngressAdmissionGuard.violationOnce` unless implementing a real once-only side effect.
- Add config key drift protection for defaults/types/snapshot coverage.

Proof:

- `go test ./internal/config ./internal/app/health ./cmd/service/internal/bootstrap -count=1`
- `make check`

## Phase 4: Contributor Placement, Persistence, And Clone Polish

Goal: make the extension path obvious for future business code and remove small clone-readiness noise.

Work:

- Add `internal/domain/doc.go` explaining the shared-contract seam, or update docs to say the package is create-on-demand. Preferred: add `doc.go`.
- Add docs-only app-facing persistence port sketch.
- Add migration rehearsal guidance to the main Postgres feature checklist and feature workflow.
- Add generated helper drift guidance for mockgen/stringer changes.
- Remove the stray `README.md` line.
- Update `Makefile` help only if the needed feature validation targets are still not discoverable.

Proof:

- `go test ./...`
- `make check`
- `make sqlc-check` / `make docker-sqlc-check` only if SQLC surfaces change
- `make migration-validate` / `make docker-migration-validate` only if migrations change

## Final Validation

Run after all implementation phases:

- `make check`
- `make openapi-check` if any OpenAPI source or generated output changed
- `go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/telemetry ./internal/app/... -count=1`
- `make sqlc-check` or `make docker-sqlc-check` only if SQLC/migration surfaces changed
- `make test-integration` and `make migration-validate` only if migration-backed runtime behavior changed

## Reopen Conditions

Reopen planning before continuing if:

- operation-level security markers require a Redocly plugin or external linter beyond the existing toolchain,
- ingress exposure cannot be inferred safely from `app.env` and `http.addr`,
- readiness aggregation changes from sequential to parallel probing,
- a real auth, tenant, browser session, or CSRF runtime becomes necessary,
- `/metrics` needs a separate listener or auth design,
- config key drift protection needs production reflection rather than tests/package-local constants,
- implementation discovers that the old `template-readiness-hardening-2026-04-11` artifact conflicts with current code in a way that changes this follow-up scope.
