# Adoption v1 closeout design

## Material flows

### Repository initialization

`scripts/init-module.sh` derives identity and optional database/outbound
surfaces, then stops. Workflow-owned paths are neither profile inputs nor
mutation targets. `template-init-check` snapshots those paths in both a small
fixture and a complete checkout and compares path plus content hashes after
initialization.

### Migration execution and recovery

`cmd/migrate` creates the signal context, loads typed configuration, then adds
the configured overall deadline for migration execution. It passes statement
and lock budgets into `internal/infra/postgresmigrate`.

The migration package validates budgets before opening PostgreSQL, configures
the existing pgx migration driver statement timeout, configures the existing
`golang-migrate` lock timeout, and retains graceful-stop signaling between
migrations. It checks the caller context before and after each migration
operation so cancellation cannot become success. Resource close failures stay
joined with the operation failure.

`migrate.ErrDirty` remains the underlying error identity. The local wrapper
adds the exact dirty version and recovery guidance. Recovery is deliberately
outside normal execution: inspect the failed SQL and actual schema, restore or
repair it, then use an approved database console to set the verified version
and clear the dirty flag under exclusive control. No automatic call to
`Migrate.Force` is added.

### Generated PostgreSQL POST proof

CI copies and initializes the full template as `DATABASE=postgres`, applies a
fixture patch, then runs the existing OpenAPI and sqlc generators. The fixture
owns only its temporary domain, migration, query, adapter, wiring, and
integration test. The test starts the repository's pinned PostgreSQL image,
applies the fixture migration, sends HTTP requests through the generated strict
server, inspects durable rows, and exercises the same hand-written repository
with a real rollback and commit.

This proves the adoption workflow without adding a placeholder production
resource to the base service.

### Optional reference service

Make and drift scripts construct the reference OpenAPI path/package list only
when its canonical YAML and package exist. Service-owned paths always run.
`template-init-check` deletes the example only inside a temporary initialized
checkout and reruns the service OpenAPI gate.

## Ownership

| Responsibility | Owner |
| --- | --- |
| Initialization inputs and workflow preservation | `scripts/init-module.sh`, proved by `scripts/ci/template-init-check.sh` |
| Typed migration budgets | `internal/config` and its snapshot/validation tests |
| Signal and overall deadline | `cmd/migrate/run.go` and `run_test.go` |
| Driver bounds, cancellation result, dirty diagnostic, cleanup | `internal/infra/postgresmigrate` and unit/integration tests |
| Recovery instructions | `docs/railway-deployment-profile.md` |
| Optional example discovery | `Makefile` and `scripts/ci/generated-drift-check.sh` |
| POST adoption fixture | `scripts/ci/fixtures/postgres-post-feature.patch` and `template-init-check.sh` |

No new package, interface, or dependency is required. Generated Go remains
owned by OpenAPI/sqlc inputs and is created only inside the temporary proof
checkout.

## Failure and rollout rules

- Invalid migration budgets fail configuration or options validation before a
  database connection.
- Lock acquisition stops waiting at its configured budget.
- A running SQL migration is bounded by the driver statement timeout.
- An overall deadline or signal stops subsequent migration work and returns a
  context error even when the underlying graceful stop returns nil.
- Dirty state blocks later runs. The deployment remains failed until an
  operator repairs and explicitly clears it.
- Existing services receive only new configuration defaults and safer
  migrator behavior; application request processing and schema authority do
  not change.
