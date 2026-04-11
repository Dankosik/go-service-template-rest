# Ownership Map

## Documentation Ownership

- `docs/project-structure-and-module-organization.md` owns placement guidance and feature extension recipes.
- It also owns concise domain-type, telemetry-placement, test-placement, migration-safety, list-limit, transaction, DB-required-feature, and project-tree guidance when those rules are onboarding-oriented.
- `docs/configuration-source-policy.md` owns config-source, secret-source, and config-extension rules.
- `docs/build-test-and-development-commands.md` owns command semantics and local validation wording.
- `README.md` owns short contributor entry points and links to detailed docs.
- `internal/api/README.md` owns local generated-API package guidance.

## Runtime Boundary Ownership

- `internal/config` owns config snapshot shape, default keys, parse behavior, and validation.
- `cmd/service/internal/bootstrap` owns dependency admission and lifecycle wiring, but should not own Redis/Mongo cache/store semantics.
- `internal/infra/http` owns generated handler mapping, manual root-route exceptions, fallback policy, CORS/OPTIONS policy, route labels, and transport observability.
- `internal/infra/postgres` owns Postgres adapter mapping and sqlc-generated query access.
- `env/migrations` owns schema evolution source files.

## Source-Of-Truth Rules

- OpenAPI changes must start in `api/openapi/service.yaml`; do not hand-edit `internal/api/openapi.gen.go`.
- sqlc changes must start in migrations and `internal/infra/postgres/queries`; do not hand-edit `sqlcgen`.
- Config keys must be represented in `Config`, defaults, snapshot construction, validation when needed, and tests.
- Online migration safety belongs in docs and task-local design for real migrations, not in a generic migration framework.
- Manual root routes must be explicit exceptions, not a backdoor around OpenAPI.
- `.artifacts/test/*` are generated report outputs unless a deliberate tracked-sample decision is recorded.

## Abstraction Boundary

Do not add:

- generic `domain` taxonomy;
- `common` or `util` package;
- generic repository or transaction framework;
- Redis/Mongo adapter packages without a real feature consumer;
- service registry or dependency-injection framework.
- broad HTTP package rename unless it is separately justified by more than package-name preference.
