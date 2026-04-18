# Context

The template currently validates migrations in CI but does not apply them as part of Railway deployment. That leaves a manual production step where schema and service rollouts can drift, which is especially brittle because Railway keeps overlapping deployments alive during promotion.

# Scope / Non-goals

In scope:
- add a release-safe automatic migration step for Railway deployments
- keep migration execution inside the template's current stack and deployment model
- document the operator prerequisites for GitHub-triggered Railway deploys

Non-goals:
- manage Railway GitHub service linkage or `Wait for CI` entirely from repository code
- make destructive or contract-only schema changes automatically safe in one deploy
- redesign schema ownership or change migration file format/tooling

# Constraints

- `railway.toml` is the non-secret deployment-policy source of truth.
- The canonical runtime image remains `build/docker/Dockerfile`.
- The final image is distroless, so the migration path must not depend on `/bin/sh`.
- Postgres deployments use the template's typed `postgres.dsn` contract (`APP__POSTGRES__DSN`) rather than ambient `PG*` configuration.
- One controlled migrator owns production migration execution; routine service startup must not compete with it.

# Decisions

1. Railway deployments will run migrations through `deploy.preDeployCommand` in `railway.toml`.
2. The template will ship a dedicated `/migrate` binary in the runtime image and copy `env/migrations/` into the image so Railway pre-deploy can execute without a shell.
3. The migration runner will load the same `APP__...` config namespace as the service and will no-op when Postgres is disabled.
4. The service binary will stay migration-free at normal startup. Migration execution happens once per deployment, before the new release is promoted.
5. GitHub push auto-deploy remains an operator-owned Railway service setting. The template will document the required GitHub repo connection and `Wait for CI` toggle rather than pretending repo code can enforce them.
6. Automatic pre-deploy migration does not waive mixed-version safety. While Railway keeps overlap and draining windows enabled, same-deploy schema changes must stay expand-compatible. Destructive or contract-only migrations require staged rollout discipline.

# Open Questions / Assumptions

- [assumption] The target Railway service deploys from a connected GitHub repository branch and operators can enable `Wait for CI` in Railway settings when CI-gated deploys are desired.
- [accepted_risk] Railway does not retry failed pre-deploy commands. The operational recovery path is to fix the cause and trigger a new deployment, while the old deployment remains active.

# Task Breakdown / Handoff Link

- Implementation handoff lives in `tasks.md`.
- Operator sequencing and mixed-version notes live in `rollout.md`.

# Validation

- `go test ./cmd/migrate ./internal/infra/postgres` passed.
- `go test ./...` passed.
- `go test -tags=integration ./test -run '^TestPostgresMigrateUpAppliesAndReplaysMigrations$' -count=1` passed.
- `APP__POSTGRES__ENABLED=true APP__POSTGRES__DSN=postgres://app:app@127.0.0.1:<random-port>/app?sslmode=disable go run ./cmd/migrate` passed against a temporary Docker Postgres and applied the repo migrations.
- `make migration-validate` passed and rehearsed `up -> down 1 -> up 1` through the repository-owned Docker path.
- `make guardrails-check` passed.
- `docker build -f build/docker/Dockerfile -t go-service-template-rest:migrate-check .` passed.

# Outcome

The template now ships a dedicated Railway pre-deploy migrator, includes migration files in the runtime image, and records the one-migrator policy in `railway.toml` plus guardrails/docs. Railway GitHub autodeploy and `Wait for CI` remain operator-managed settings, but the repo now encodes the deploy-time migration step that was previously manual.
