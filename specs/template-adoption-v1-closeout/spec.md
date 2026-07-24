# Adoption v1 closeout

Status: ready

## Outcome

Close the remaining adoption-critical gaps without changing the repository's
agent workflow:

- initialized services always retain the complete workflow tree and files
  byte-for-byte;
- PostgreSQL migrations have explicit runtime, statement, and lock budgets,
  preserve cancellation, report a dirty version with a safe recovery path, and
  never force a version automatically;
- CI proves one generated service can add a PostgreSQL-backed POST operation
  from OpenAPI and migration/query sources through a hand-written repository;
- `examples/reference-service` remains useful but can be deleted without
  breaking service-owned OpenAPI generation or validation.

## Acceptance criteria

1. `scripts/init-module.sh` has no `AGENT_WORKFLOW` option, branch, or output.
   Initialization does not delete, rewrite, or profile `.agents`, `.codex`,
   `.claude`, `.qwen`, `specs`, `AGENTS.md`, `CLAUDE.md`, or `QWEN.md`.
2. `make template-init-check` compares the workflow-owned files before and
   after initialization and fails on any content or path change.
3. Database and outbound HTTP selection remain independent profile decisions.
4. The migrator defaults to a 5-minute run budget, a 2-minute per-statement
   budget, and a 15-second migration-lock budget. Typed configuration can
   override each value; invalid or internally inconsistent budgets fail before
   a database operation.
5. Cancellation or deadline expiry cannot be reported as a successful
   migration. A failed migration that leaves the database dirty reports the
   exact version, states that automatic force is disabled, and points to the
   recovery runbook.
6. Recovery requires an operator to inspect and repair the database before
   manually clearing dirty state. The template exposes no automatic or startup
   force path.
7. CI initializes a temporary `DATABASE=postgres` service, applies a fixture
   that adds a POST JSON contract, migration, sqlc query, domain code,
   hand-written PostgreSQL mapping, and HTTP wiring, then proves on real
   PostgreSQL:
   - valid input persists and returns the generated success response;
   - malformed, invalid, and unknown-field input is rejected without a write;
   - transaction rollback discards a repository write and commit preserves it.
8. OpenAPI generation, lint, validation, drift, and generated-package tests
   include `examples/reference-service` when it exists and remain equally
   effective for the service contract when the example directory is absent.

## Constraints and non-goals

- Do not modify the agent workflow's instructions, skills, evals, routing, or
  harness behavior.
- Do not add another workflow profile or compatibility alias for
  `AGENT_WORKFLOW`.
- Do not add an ORM, migration framework, transaction abstraction, retry
  framework, or permanent demo business domain to the base service.
- Do not run migrations in application startup.
- Do not claim a Railway platform timeout that its current documentation does
  not define; the migrator owns its own explicit budgets.
- Do not remove the reference service from this repository. Make its ownership
  optional for derived services.

## Evidence and authority

- `scripts/init-module.sh` owns derivation behavior.
- `api/openapi/service.yaml`, `migrations/*.up.sql`, and
  `internal/infra/postgres/queries/*.sql` remain canonical sources for the
  service contract, schema, and generated SQL.
- `cmd/migrate` owns process signals and the overall migration deadline;
  `internal/infra/postgresmigrate` owns driver statement/lock bounds, dirty
  diagnostics, cleanup, and cancellation semantics.
- Railway runs `/migrate` as a pre-deploy command. Current Railway
  documentation says a failed command blocks deployment and is not retried,
  but does not publish a platform timeout; repository configuration therefore
  owns the bounded wait.

## Residual risk and reopen conditions

- A 2-minute statement budget can be too short for a large production table.
  The service owner must replace the default with a rehearsed, workload-owned
  value before such a migration.
- `golang-migrate` cancellation is graceful between migrations; an in-flight
  SQL operation is bounded separately by the statement timeout. Reopen the
  mechanism if a future migration needs a stricter end-to-end deadline than
  these two bounds can provide.
- The CI feature is an adoption fixture, not a supported domain module. Reopen
  only if a real generated service cannot reproduce its flow.
