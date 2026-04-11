# Implementation Sequence

## Phase 1: Docs And Contributor Guidance

1. Update `docs/project-structure-and-module-organization.md` with the vertical feature path, migration safety guidance, and Redis/Mongo stub language.
2. Add the adjacent doc gaps there as well: domain-type rule, telemetry placement, test placement matrix, list-limit guidance, transaction recipe, DB-required-feature bootstrap guidance, and project-tree refresh.
3. Update `docs/configuration-source-policy.md` with an "Adding a config key" recipe and Redis/Mongo extension-stub clarity when appropriate.
4. Update `internal/api/README.md` with the strict-server endpoint checklist and parameterized route-label proof note.
5. Update `README.md` only enough to link contributors to the feature-work path and correct OpenAPI check wording.
6. Update `docs/build-test-and-development-commands.md` after the final Makefile command shape is known.

Why first: docs define the intended future behavior and reduce ambiguity before test/command edits.

## Phase 2: Config And HTTP Guardrails

1. Add config snapshot round-trip proof in `internal/config/config_test.go`.
2. Fix or harden `TestNonLocalDefaultRootsDoNotAllowRepositoryConfigDir`.
3. Widen `openapi-runtime-contract-check` in `Makefile`.
4. If the implementation uses test renames instead of a wider regex, rename tests without changing behavior.
5. Tighten manual root-route exception coverage.
6. Fix or harden `TestServerRunAndShutdown`.
7. Remove or clarify `enforceSecretSourcePolicy`'s ignored parameter.

Why second: config and HTTP tests/commands enforce the most central template guardrails and can be validated without mixing in data/bootstrap cleanup.

## Phase 3: Data, Bootstrap, Artifacts, And Validation

1. Strengthen Postgres sample assertions and move test-only repository helper out of production code.
2. Reduce startup dependency label drift with a local typed label/spec shape.
3. Resolve `.artifacts/test/*` tracking.
4. Update `tasks.md` and `research/finding-coverage.md` with final task evidence.

Why third: these items are important, but less tightly coupled to the docs and HTTP/config gates.

## Validation

Run targeted validation first:

- `go test ./internal/config -run 'TestKnownConfigKeysMatchSnapshotTagsAndDefaults|Test.*Snapshot' -count=1`
- `make openapi-runtime-contract-check`
- `go test ./internal/infra/http -run '^(TestOpenAPIRuntimeContract|TestRouterHTTPPolicy|TestManualRootRouteGeneratedOverlapsAreDocumented|TestRouteTemplateUsedForOTelSpanName)' -count=1` if not covered exactly by the Makefile target output.
- `go test ./cmd/service/internal/bootstrap -count=1` if startup label code changes.
- `go test ./internal/infra/http -run '^TestServerRunAndShutdown$' -count=1` if the server lifecycle test changes.

Run broader validation when feasible:

- `make openapi-check`
- `make test`
- `make check`

Run Postgres-focused validation when Postgres sample assertions/helper placement change:

- `go test ./internal/infra/postgres -run 'TestPingHistoryRepository' -count=1`
- `go test -tags=integration ./test/... -run 'TestPingHistoryRepositorySQLCReadWrite' -count=1` when Docker is available.
