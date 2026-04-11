# Component Map

## Documentation Components

### `docs/project-structure-and-module-organization.md`

Planned changes:

- Add a compact worked feature path before or inside `Where to Put New Code`.
- Add the domain-type decision rule.
- Add the test placement matrix.
- Add feature telemetry placement guidance.
- Add online migration safety guidance near the deterministic migration rule.
- Add list-limit, transaction, and DB-required-feature guidance where it fits the Postgres persistence recipe.
- Add Redis/Mongo stub language in the integration section.
- Refresh stale project-tree details, including `specs/` and generated/report artifacts, if it stays reviewable.

Stable:

- Existing layer ownership rules should remain.
- Do not add a new domain taxonomy.

### `README.md`

Planned changes:

- Add or adjust a short feature-work entry point linking to the structure doc.
- Correct the `make openapi-check` description so compatibility proof is not implied unless `openapi-breaking` is included.

Stable:

- Do not duplicate the full spec-first workflow again.

### `docs/build-test-and-development-commands.md`

Planned changes:

- Update `openapi-runtime-contract-check` docs after the Makefile regex/name change.
- Clarify `openapi-check` versus `openapi-breaking`.
- If `.artifacts/test/*` remains tracked, document why; otherwise align docs with generated-report output behavior.

Stable:

- Existing command catalog remains the detailed command reference.

### `docs/configuration-source-policy.md`

Planned changes:

- Add a short "Adding a config key" recipe.
- Make extension-only keys explicit when they exist without full runtime behavior.

Stable:

- Secret policy and precedence rules remain unchanged.

### `internal/api/README.md`

Planned changes:

- Add a strict-server extension checklist for future endpoints.
- Mention future parameterized route-label proof when new parameterized endpoints appear.

Stable:

- Generated API bindings remain derived from `api/openapi/service.yaml`.

## Code And Test Components

### `Makefile`

Planned changes:

- Widen `openapi-runtime-contract-check` to include HTTP policy tests that enforce generated-route ownership, fallback behavior, manual root-route exceptions, and route labels.

Stable:

- `openapi-breaking` stays a separate compatibility proof because it requires `BASE_OPENAPI`.

### `internal/config/config_test.go`

Planned changes:

- Add snapshot round-trip proof for all known config leaf keys.
- Fix or harden the repository-path mutation test if it remains in scope for implementation.
- Avoid production reflection mapping.

Stable:

- Existing explicit parse and validation tests remain.

### Postgres Test And Helper Improvements

- Strengthen `internal/infra/postgres/ping_history_repository_test.go` and `test/postgres_sqlc_integration_test.go` assertions for payload and timestamp mapping.
- Move test-only repository construction helper out of `internal/infra/postgres/ping_history_repository.go` or otherwise remove it from the production helper surface.

### `cmd/service/internal/bootstrap`

Planned changes:

- Reduce startup dependency label drift in `startup_common.go` / `startup_dependencies.go` with a small same-package label/spec shape.

Stable:

- Keep dependency-specific control flow local; do not introduce a lifecycle callback framework.

### `internal/infra/http`

Planned changes:

- Tighten manual root-route exception policy.
- Fix or harden `TestServerRunAndShutdown`.
- Document the `httpx` package-name convention or leave a deliberate no-op decision in task closeout.

## Generated Artifacts

No generated artifacts are expected for the planned changes unless the implementation unexpectedly changes OpenAPI or sqlc sources. That should not happen.
