# Task Ledger

## Phase 1: Clone And Config Correctness

- [x] T001 [Phase 1] Align `env/.env.example` `APP__HTTP__SHUTDOWN_TIMEOUT` with the validated baseline in `internal/config/defaults.go` and `env/config/default.yaml`. Proof: `go test ./internal/config -count=1`.
- [x] T002 [Phase 1] Add a config fixture test that parses `env/.env.example`, sets `APP__...` env vars through `t.Setenv`, and proves `config.LoadDetailed` succeeds. Depends on: T001.
- [x] T003 [Phase 1] Tighten `isSecretLikeConfigKey` / related policy in `internal/config/load_koanf.go` for common future key shapes: `client_secret`, `jwt_secret`, `api_key`, `private_key`, and top-level `token`. Proof: targeted config tests.
- [x] T004 [Phase 1] Replace the hand-maintained `resetConfigEnv` list in `internal/config/config_test.go` with a derived namespace-key reset where practical, plus explicit non-namespace keys. Proof: `go test ./internal/config -count=1`.

## Phase 2: Composition, Readiness, And Lifecycle Ownership

- [x] T005 [Phase 2] Change `internal/infra/http.NewRouter` / `newStrictHandlers` so production router construction requires explicit dependencies and does not create fallback app services or metrics. Update tests to use explicit test helpers.
- [x] T006 [Phase 2] Update `cmd/service/internal/bootstrap.Run` to handle the router-construction shape from T005 and keep all concrete wiring in bootstrap.
- [x] T007 [Phase 2] Resolve readiness contract ownership: prefer `internal/app/health.Probe`; remove/update `internal/domain/readiness.go` and docs accordingly. Proof: `go test ./internal/app/health -count=1`.
- [x] T008 [Phase 2] Add an external readiness gate so `/health/ready` returns not ready until startup admission marks ready, without deadlocking the internal startup admission check. Proof: HTTP/bootstrap readiness tests.
- [x] T009 [Phase 2] Make readiness timeout configurable or explicitly passed from config, and validate/document its relationship with dependency healthcheck budgets.
- [x] T010 [Phase 2] Couple shutdown/write/readiness propagation budgets through validation or documented config policy; include telemetry flush in process-grace documentation.
- [x] T011 [Phase 2] Add cleanup-stack/status handling in dependency admission so partially initialized resources close if later dependency startup fails.
- [x] T012 [Phase 2] Extract same-package ingress policy validation helper for `EnforceIngress` and `ValidateIngressRuntime`.
- [x] T013 [Phase 2] Extract same-package dependency probe rejection telemetry helper while keeping degraded/fail-open branches explicit.

## Phase 3: HTTP, Security, Metrics, And Generated Route Boundaries

- [x] T014 [Phase 3] Remove or reconcile unused OpenAPI `bearerAuth`; prefer removal unless real auth is introduced. Regenerate `internal/api` if OpenAPI changes.
- [x] T015 [Phase 3] Add endpoint security decision guidance to `docs/project-structure-and-module-organization.md` and related docs.
- [x] T016 [Phase 3] Make `/metrics` generated/manual route ownership explicit; add a route-owner guard test that allows only documented root exceptions.
- [x] T017 [Phase 3] Sanitize strict-handler request error details in `internal/infra/http/router.go` / `problem.go`; add a negative test if a current request shape can trigger parser errors.
- [x] T018 [Phase 3] Preserve and document fail-closed CORS behavior; do not enable browser CORS in this task.

## Phase 4: Persistence Sample, Docs, Make Help, And Test Placement

- [x] T019 [Phase 4] Decide the concrete `ping_history` treatment after checking sqlc behavior: remove from default runtime path if possible; otherwise explicitly label as template sample. Record the chosen outcome in docs. Outcome: retained as explicit template SQLC sample because the current sqlc config fails when `queries/*.sql` is empty; documented in `docs/project-structure-and-module-organization.md` and labeled in repository comments.
- [x] T020 [Phase 4] Make retained migrations deterministic by removing unnecessary `IF NOT EXISTS` / `IF EXISTS`, or document any intentional idempotent exceptions. Proof: `make docker-migration-validate`.
- [x] T021 [Phase 4] Update persistence docs with the flow `migration -> queries/*.sql -> sqlcgen -> postgres repository -> app-owned port if needed -> bootstrap wiring`.
- [x] T022 [Phase 4] Expand `test/README.md` with integration-test placement rules: build tag, package name, root/subdirectory policy, Docker behavior, migration-backed helpers, and `REQUIRE_DOCKER`.
- [x] T023 [Phase 4] Add test-placement guidance by layer to `docs/project-structure-and-module-organization.md`.
- [x] T024 [Phase 4] Fix docs that say new integrations wire in `cmd/service/main.go`; they should wire in `cmd/service/internal/bootstrap` for the service binary.
- [x] T025 [Phase 4] Add outbound integration security expectations to docs: target source, allowlist, timeout, redirect, DNS/IP-class behavior, and separate review for dynamic URLs.
- [x] T026 [Phase 4] Add feature validation targets to `make help`: OpenAPI, SQLC, integration tests, and Docker equivalents. Proof: `make help`.

## Final Validation

- [x] T027 [Validation] Run `make check`.
- [x] T028 [Validation] Run `make openapi-check` if OpenAPI/generated HTTP surfaces changed. Proof: passed with a temporary git index containing the current OpenAPI source/generated pair so the real staging area stayed unchanged.
- [x] T029 [Validation] Run `make sqlc-check` or `make docker-sqlc-check`; use Docker if native sqlc still fails with `pg_query_go`. Proof: `make docker-sqlc-check`.
- [x] T030 [Validation] Run `make test-integration` if persistence/migration integration behavior changed.
