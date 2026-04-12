Objective: Investigate the ping history persistence duplicate-write issue before changing code, and verify the repository/sqlc query shape and migration surface.

User Intent And Context: The user is unsure about the exact file path and explicitly says `internal/db/ping_repo.go` may be wrong. The real problem is ping history persistence: duplicate writes started appearing after a retry path. The downstream agent should inspect the repository layer first, then the sqlc query shape, and only then decide whether migrations are needed. The user also wants the agent to say the guessed path is bogus if it does not exist, rather than pretending otherwise.

Confirmed Signals And Exact Identifiers:
- `internal/db/ping_repo.go` (explicitly called out as possibly wrong)
- `ping_history`
- `sqlc`
- migrations
- postgres adapter
- repository test
- duplicate writes
- retry path

Relevant Repository Context:
- This repo uses PostgreSQL with `pgx` and `sqlc`.
- The ping history sample lives under `internal/infra/postgres/`, not `internal/db/`.
- Likely relevant surfaces are `internal/infra/postgres/ping_history_repository.go`, `internal/infra/postgres/queries/ping_history.sql`, `internal/infra/postgres/sqlcgen/ping_history.sql.go`, `internal/infra/postgres/ping_history_repository_test.go`, and `env/migrations/000001_init.up.sql` plus `env/migrations/000002_ping_history_recent_index.up.sql`.
- There is an existing integration test surface in `test/postgres_sqlc_integration_test.go`.

Inspect First:
- `internal/infra/postgres/ping_history_repository.go`
- `internal/infra/postgres/queries/ping_history.sql`
- `internal/infra/postgres/ping_history_repository_test.go`
- `env/migrations/000001_init.up.sql`
- `env/migrations/000002_ping_history_recent_index.up.sql`
- If needed, confirm whether `internal/db/ping_repo.go` exists; treat it as likely nonexistent unless proven otherwise.

Requested Change / Problem Statement:
- Diagnose why retry handling is causing duplicate ping history writes.
- Check whether the repository adapter or sqlc query shape is allowing repeated inserts.
- Determine whether the fix belongs in the query, adapter logic, retry flow, or a migration.
- Avoid assuming generated sqlc output is the only thing to change.

Constraints / Preferences / Non-goals:
- Do not hand-edit generated sqlc output as the sole fix.
- Do not assume the guessed `internal/db/ping_repo.go` path is real.
- Investigate first; do not jump straight to implementation.

Validation / Verification:
- Use the repository test surface around ping history.
- If the fix touches sqlc sources, include a `sqlc` drift check.
- If a migration changes, validate the migration path.

Assumptions / Open Questions:
- The exact retry path that triggers duplicates is not identified in the raw input.
- It is unclear whether the bug is in idempotency, transaction boundaries, or query semantics until the repository code is inspected.