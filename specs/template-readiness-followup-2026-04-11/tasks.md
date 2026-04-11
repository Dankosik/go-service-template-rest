# Task Ledger

## Phase 1: HTTP Contract And Security Decision Guardrails

- [x] T001 [Phase 1] Add a local generated chi options/error-handler helper in `internal/infra/http` so `api.ChiServerOptions.ErrorHandlerFunc` logs sanitized request-error class and writes the generic malformed-request Problem response. Proof: targeted `go test ./internal/infra/http -count=1`.
- [x] T002 [Phase 1] Add or update HTTP tests proving generated wrapper error handling uses `application/problem+json` and does not expose raw attacker-controlled details. Depends on: T001.
- [x] T003 [Phase 1] Add explicit operation-level security decision metadata to `api/openapi/service.yaml` for `ping`, liveness, readiness, and metrics. Mark `/metrics` as operational-private-required. Proof: `make openapi-check` if generated output changes.
- [x] T004 [Phase 1] Add an OpenAPI contract test, likely under `internal/infra/http/openapi_contract_test.go`, that fails when any operation lacks the explicit security decision marker. Depends on: T003.
- [x] T005 [Phase 1] Extend the OpenAPI contract test to require real security plus 401/403 Problem responses for future operations marked protected, while allowing current explicitly public/operational baseline operations. Depends on: T004.
- [x] T006 [Phase 1] Add browser-callable endpoint checklist docs to `docs/project-structure-and-module-organization.md` and/or `docs/repo-architecture.md`; state that CORS fail-closed is not CSRF protection. Proof: docs review.

Phase 1 proof note: `go test ./internal/infra/http -count=1` passed. `make openapi-check` passed with a temporary git index containing the current generated `internal/api` output, leaving the real staging area unchanged.

## Phase 2: Trust Boundary And Redaction Hardening

- [x] T007 [Phase 2] Update bootstrap network policy parsing to distinguish missing `NETWORK_PUBLIC_INGRESS_ENABLED` from explicitly set false. Proof: bootstrap tests.
- [x] T008 [Phase 2] Enforce explicit public-ingress declaration for non-local wildcard binds in `cmd/service/internal/bootstrap`; if declared true, keep existing exception metadata requirement. Depends on: T007.
- [x] T009 [Phase 2] Update docs to explain private-ingress assertion, public-ingress exception metadata, and `/metrics` private scrape/internal-listener/auth requirement. Depends on: T008.
- [x] T010 [Phase 2] Change `internal/infra/http.Recover` so panic logs include panic type/class but not the raw recovered value. Add a negative test containing `secret-value`. Proof: `go test ./internal/infra/http -count=1`.
- [x] T011 [Phase 2] Change `internal/infra/telemetry.parseOTLPHeaders` errors so malformed entries do not include raw header values. Add redaction tests for authorization/API-key-like input. Proof: `go test ./internal/infra/telemetry -count=1`.
- [x] T012 [Phase 2] Clarify YAML secret placeholder policy in `docs/configuration-source-policy.md`: empty placeholders allowed only for schema/default visibility; non-empty secret-like values rejected. Add or adjust config tests for that rule.

Phase 2 proof note: `go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/telemetry -count=1` passed.

## Phase 3: Config, Readiness, And Dependency Admission Semantics

- [x] T013 [Phase 3] Replace exact `http.shutdown_timeout == 30s` validation in `internal/config/validate.go` with relationship/range validation, unless implementation deliberately chooses to document the exact lock instead. Preferred proof: config tests for valid tuned shutdown and invalid drain/write combinations.
- [x] T014 [Phase 3] Add aggregate sequential readiness budget validation for enabled readiness probes in `internal/config`. Include Redis store mode if it participates in readiness. Depends on: T013 only if shared validation helpers change.
- [x] T015 [Phase 3] Update `docs/configuration-source-policy.md` to describe shutdown/readiness/telemetry grace relationships and clarify that CLI flags are loader controls today.
- [x] T016 [Phase 3] Add docs rule that `/health/live` remains process-only and external dependency checks belong in readiness.
- [x] T017 [Phase 3] Add a "new runtime dependency" checklist to `docs/repo-architecture.md` or `docs/project-structure-and-module-organization.md`: config, network policy, criticality, retry/budget, readiness, cleanup, metrics labels, degraded-mode contract, and bootstrap tests.
- [x] T018 [Phase 3] Extract package-local dependency name/mode/stage constants, or a narrow telemetry rejection helper, in `cmd/service/internal/bootstrap` only where it protects metrics/log label contracts. Do not merge per-dependency init functions.
- [x] T019 [Phase 3] Remove unused `startupLifecycleStartedAt` plumbing if it still has no real elapsed-time behavior. Proof: bootstrap tests.
- [x] T020 [Phase 3] Remove `runtimeIngressAdmissionGuard.violationOnce` or implement a real once-only side effect; preferred follow-up is removal. Proof: bootstrap tests.
- [x] T021 [Phase 3] Add config key drift protection in `internal/config` tests, preferably deriving leaf keys from `Config` `koanf` tags and comparing them with defaults/known keys. Avoid production reflection mapping.

Phase 3 proof note: `go test ./internal/config ./internal/app/health ./cmd/service/internal/bootstrap -count=1` passed. `make check` passed.

## Phase 4: Contributor Placement, Persistence, And Clone Polish

- [x] T022 [Phase 4] Add `internal/domain/doc.go` documenting that the package is reserved for shared stable contracts and should stay empty until needed.
- [x] T023 [Phase 4] Add a docs-only app-facing persistence port sketch showing `internal/app/<feature>` owns the port when inversion is needed and `internal/infra/postgres` implements it. Do not add a generic runtime port package.
- [x] T024 [Phase 4] Add `make migration-validate` / `make docker-migration-validate` to the main Postgres persistence checklist in `docs/project-structure-and-module-organization.md` and the native/zero-setup feature workflow in `docs/build-test-and-development-commands.md`.
- [x] T025 [Phase 4] Add conditional generated-helper drift guidance for `mocks-drift-check` and `stringer-drift-check` in the feature workflow docs.
- [x] T026 [Phase 4] Remove the stray `Hello from claude code` line from `README.md`.
- [x] T027 [Phase 4] Check `make help`; update it only if OpenAPI, SQLC, integration-test, Docker validation, migration validation, or generated-helper drift targets are not discoverable enough for template users.

Phase 4 proof note: `make help` was checked and updated to surface migration validation and generated-helper drift targets. `go test ./... -count=1` passed. `make check` passed.

## Final Validation

- [x] T028 [Validation] Run `go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/telemetry ./internal/app/... -count=1`.
- [x] T029 [Validation] Run `make check`.
- [x] T030 [Validation] Run `make openapi-check` if T003 changes OpenAPI source or generated output.
- [x] T031 [Validation] Run `make sqlc-check` or `make docker-sqlc-check` only if SQLC/migration surfaces change. Not applicable: no `env/migrations`, `internal/infra/postgres/queries`, or `internal/infra/postgres/sqlcgen` files changed.
- [x] T032 [Validation] Run `make test-integration` and `make migration-validate` only if migration-backed runtime behavior changes. Not applicable: no migration-backed runtime behavior changed.

Final validation proof note: `go test ./cmd/service/internal/bootstrap ./internal/config ./internal/infra/http ./internal/infra/telemetry ./internal/app/... -count=1` passed. `make check` passed. `make openapi-check` passed using a temporary git index for the current generated `internal/api` output, leaving the real staging area unchanged. SQLC/migration/integration checks were not run because their conditional triggers were not met.
