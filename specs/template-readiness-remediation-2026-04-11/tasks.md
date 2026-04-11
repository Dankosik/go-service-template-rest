# Task Ledger

## Phase 1: Docs And Contributor Guidance

- [x] T001 Docs: add a short worked feature path to `docs/project-structure-and-module-organization.md`, covering simple read-only endpoint, Postgres-backed endpoint, and background job/worker placement. Proof: `Worked Feature Paths` table covers those three paths without adding a new architecture taxonomy.
- [x] T002 Docs: add a short domain-type decision rule to `docs/project-structure-and-module-organization.md`. Proof: docs say feature-local types stay in `internal/app/<feature>` until a shared stable contract is actually needed.
- [x] T003 Docs: add feature telemetry placement guidance to `docs/project-structure-and-module-organization.md`. Proof: docs distinguish HTTP-edge instrumentation, shared `internal/infra/telemetry`, low-cardinality labels, and feature-local instrumentation.
- [x] T004 Docs: add a compact test placement matrix to `docs/project-structure-and-module-organization.md`. Proof: matrix maps app/domain, HTTP contract/policy, bootstrap, Postgres repository, migration/container, and broad integration tests to owning locations.
- [x] T005 Docs: add online migration safety guidance near the existing deterministic migration rule in `docs/project-structure-and-module-organization.md`. Proof: guidance distinguishes migration rehearsal from production lock/backfill/mixed-version safety.
- [x] T006 Docs: add list-limit guidance to the Postgres persistence recipe. Proof: docs require API/app contracts or repositories to clamp bounded list limits before SQL `LIMIT`.
- [x] T007 Docs: add a repo-local transaction recipe to the Postgres persistence guidance. Proof: docs mention caller context, tx-scoped sqlc queries, bounded rollback cleanup, commit context, and no generic transaction helper until real repetition exists.
- [x] T008 Docs: add DB-required feature bootstrap guidance. Proof: docs say DB-backed features must validate required Postgres config, construct repositories only after an initialized pool exists, inject through app-owned ports, and test disabled/ready/cleanup paths.
- [x] T009 Docs: clarify Redis/Mongo as guard-only extension stubs in `docs/project-structure-and-module-organization.md` and/or `docs/configuration-source-policy.md`. Proof: both docs say no real cache/store/database adapter exists until a feature owns `internal/infra/redis` or `internal/infra/mongo`.
- [x] T010 Docs: add an "Adding a config key" recipe to `docs/configuration-source-policy.md`. Proof: recipe points to `internal/config/types.go`, `defaults.go`, `snapshot.go`, validation when needed, env/config docs, and config tests without listing every key.
- [x] T011 Docs: add a strict-server endpoint checklist to `internal/api/README.md`. Proof: checklist names `api/openapi/service.yaml`, generation, `api.StrictServerInterface`, `strictHandlers.<Operation>`, `Handlers` wiring, contract tests, and `make openapi-check`.
- [x] T012 Docs: include route-label proof guidance for future parameterized endpoints. Proof: docs say parameterized routes must prove logs, metrics, and spans use route templates rather than concrete IDs.
- [x] T013 Docs/commands: correct OpenAPI validation wording in `README.md` and `docs/build-test-and-development-commands.md`. Proof: `make openapi-check` and `BASE_OPENAPI=<base> make openapi-breaking` have distinct meanings.
- [x] T014 Docs: add a short README "start here for feature work" link or pointer without duplicating the workflow docs. Proof: README sends contributors to the placement guide quickly.
- [x] T015 Docs: refresh the stale project tree in `docs/project-structure-and-module-organization.md`. Proof: tree no longer lists absent `docs/llm/...`, includes `specs/`, and classifies generated/report artifacts without becoming too noisy.

Phase 1 validation:

- [x] `git diff --check` passed after removing trailing whitespace.
- [x] Targeted trailing-whitespace grep over the untracked Phase 1 task/control artifacts found no matches.
- [x] Targeted docs grep confirmed `make openapi-check` is no longer described as a compatibility check in README/command docs; `BASE_OPENAPI=<base> make openapi-breaking` is documented as the breaking-change compatibility proof.
- [x] Targeted docs grep confirmed the refreshed structure tree no longer references absent `docs/llm/...`.
- [x] Targeted docs grep confirmed Phase 1 additions for worked feature paths, Redis/Mongo guard-only stubs, route-template proof, and the config-key recipe.
- [x] Existing workflow-control files were updated only to mark the Phase 1 session boundary and point the next session at implementation Phase 2.
- [x] Go test commands were not run for Phase 1 because the completed work changed docs and task metadata only; code/test guardrail validation is reserved for Phase 2 and Phase 3 tasks that change those surfaces.

## Phase 2: Config And HTTP Guardrails

- [x] T016 Config tests: add same-package proof in `internal/config/config_test.go` that every known config leaf key reaches the built `Config` snapshot. Proof: `go test ./internal/config -run '^TestBuildSnapshotMapsEveryKnownConfigLeafKey$' -count=1` and `go test ./internal/config -count=1` passed.
- [x] T017 Config cleanup: remove or clarify `enforceSecretSourcePolicy`'s ignored local-environment parameter. Proof: `enforceSecretSourcePolicy` no longer accepts a local-environment parameter, local and non-local secret-file rejection remains environment-independent, and `go test ./internal/config -count=1` passed.
- [x] T018 Config test hygiene: fix or harden `TestNonLocalDefaultRootsDoNotAllowRepositoryConfigDir` so it does not write temporary files into the real repository path, or record a clear justification and stronger cleanup. Proof: the test now writes a repo-shaped config path under `t.TempDir()`, and `go test ./internal/config -run '^TestNonLocalDefaultRootsDoNotAllowRepositoryConfigDir$' -count=1` passed.
- [x] T019 HTTP command gate: widen `openapi-runtime-contract-check` in `Makefile` or rename the relevant HTTP tests so route policy, manual root-route exception, and route-label tests run under the target. Proof: relevant HTTP policy, manual root-route, access-log route-label, metrics route-label, and OTel span-name tests were renamed under `TestOpenAPIRuntimeContract`; `go test ./internal/infra/http -list '^TestOpenAPIRuntimeContract'` listed the intended set; `make openapi-runtime-contract-check` and `make openapi-check` passed. Related write: `Makefile` now limits OpenAPI drift detection to `internal/api/openapi.gen.go`, with `docs/build-test-and-development-commands.md` updated to match, because `internal/api/README.md` is hand-written package guidance from Phase 1 and must not make `openapi-check` fail as generated drift.
- [x] T020 HTTP route policy: tighten manual root-route exception handling for all manual root routes, not only generated overlaps, or document the explicit allow policy and required tests. Proof: `documentedManualRootRouteExceptions` now covers every manual root route, stale documented exceptions are rejected, and `go test ./internal/infra/http -run 'ManualRootRoute' -count=1` passed.
- [x] T021 HTTP lifecycle test hygiene: fix or harden `TestServerRunAndShutdown` so it does not teach reserve-close-listen polling or unbounded default-client behavior. Proof: the lifecycle test is now `TestServerServeAndShutdown`, uses an owned listener with `Serve`, uses a bounded `http.Client`, and `go test ./internal/infra/http -run '^TestServerServeAndShutdown$' -count=1` passed.
- [x] T022 HTTP package-name convention: close the `httpx` naming finding by documenting the convention in a package doc or structure doc, or explicitly recording a no-op decision in this task's closeout. Proof: `internal/infra/http/doc.go` documents why the import path stays `internal/infra/http` while the package name is `httpx`; `go test ./internal/infra/http -count=1` passed.

Phase 2 validation:

- [x] `go test ./internal/config -count=1` passed.
- [x] `go test ./internal/infra/http -count=1` passed.
- [x] `make openapi-runtime-contract-check` passed and ran the widened `TestOpenAPIRuntimeContract` test set.
- [x] `make openapi-check` passed after OpenAPI drift detection was narrowed to generated output instead of hand-written package docs.
- [x] `git diff --check` passed.
- [x] Existing workflow-control files were updated to mark the Phase 2 session boundary and point the next session at implementation Phase 3.

## Phase 3: Data, Bootstrap, Artifacts, And Validation

- [x] T023 Postgres sample assertions: strengthen `internal/infra/postgres/ping_history_repository_test.go` and `test/postgres_sqlc_integration_test.go` assertions for payload and timestamp mapping. Proof: unit list assertions now cover payload and `CreatedAt`; integration read/write assertions now cover created/listed payloads and timestamps; `go test ./internal/infra/postgres -run 'TestPingHistoryRepository' -count=1` and `go test -tags=integration ./test/... -run 'TestPingHistoryRepositorySQLCReadWrite' -count=1` passed with Docker available.
- [x] T024 Postgres helper surface: move `newPingHistoryRepositoryWithQuerier` out of production code or otherwise remove it as a visible runtime extension seam. Proof: helper now lives only in `internal/infra/postgres/ping_history_repository_test.go`; `go test ./internal/infra/postgres -run 'TestPingHistoryRepository' -count=1` passed.
- [x] T025 Startup maintainability: reduce startup dependency label drift from same-typed string clusters in `cmd/service/internal/bootstrap` with a small same-package label/spec shape, without introducing a lifecycle framework. Proof: `startupDependencyProbeLabels` derives probe operation/stage names from one dependency name per Postgres/Redis/Mongo dependency; related write `cmd/service/internal/bootstrap/startup_dependency_labels.go` was required because the existing label constants are owned there; `go test ./cmd/service/internal/bootstrap -count=1` passed.
- [x] T026 Artifacts cleanup: resolve tracked generated report outputs under `.artifacts/test/*` by untracking/ignoring them, or record a deliberate tracked-sample decision. Proof: `.gitignore` now ignores `.artifacts/test/`, tracked report outputs are deleted for untracking in the next commit, and `git status --short .artifacts .gitignore` shows `D .artifacts/test/junit.xml`, `D .artifacts/test/test2json.json`, and `M .gitignore`.
- [x] T027 Coverage audit: keep `research/finding-coverage.md` in sync with implementation closeout. Proof: `research/finding-coverage.md` now has Phase 3 closeout notes mapping T023-T028, artifact cleanup, and no-overreach guardrails.
- [x] T028 Validation: run targeted validation commands from `plan.md`, then `make check` if feasible. Proof: `go test ./internal/infra/postgres -run 'TestPingHistoryRepository' -count=1`, `go test ./cmd/service/internal/bootstrap -count=1`, `go test ./internal/config -count=1`, `go test ./internal/infra/http -count=1`, `go test ./internal/infra/postgres -count=1`, `go test -tags=integration ./test/... -run 'TestPingHistoryRepositorySQLCReadWrite' -count=1`, `make openapi-runtime-contract-check`, `make openapi-check`, `make test`, `make check`, and `git diff --check` passed.

Phase 3 validation:

- [x] `go test ./internal/infra/postgres -run 'TestPingHistoryRepository' -count=1` passed.
- [x] `go test ./cmd/service/internal/bootstrap -count=1` passed.
- [x] `go test ./internal/config -count=1` passed.
- [x] `go test ./internal/infra/http -count=1` passed.
- [x] `go test ./internal/infra/postgres -count=1` passed.
- [x] `go test -tags=integration ./test/... -run 'TestPingHistoryRepositorySQLCReadWrite' -count=1` passed with Docker available.
- [x] `make openapi-runtime-contract-check` passed.
- [x] `make openapi-check` passed.
- [x] `make test` passed.
- [x] `make check` passed.
- [x] `git diff --check` passed.
