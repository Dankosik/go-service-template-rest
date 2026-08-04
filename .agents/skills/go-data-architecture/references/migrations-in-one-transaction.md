# Migrations Run In One Transaction

## Behavior Change Thesis
When loaded for a schema change against a populated table, this file makes the model split the work into additive DDL plus a separately owned backfill instead of the likely mistake "declare `-- +goose NO TRANSACTION`, build the index concurrently, and backfill in the same migration" — a plan this repository rejects before it opens a database connection.

## When To Load
Load for any change under `migrations/`: a column, constraint, index, type or enum change, or a backfill.

## Decision Rubric
- `internal/infra/postgresmigrate/source.go` rejects `-- +goose NO TRANSACTION`, its case variants, `ENVSUB`, Go migrations, and non-canonical filenames, and `scripts/ci/migration-source-check.sh` rejects them again in CI. Every admitted migration commits as one transaction, on every table, at every size. [first-production-feature.md](../../../../docs/first-production-feature.md) owns the full authoring rules.
- `CREATE INDEX CONCURRENTLY` therefore cannot run on this path. Either accept that a plain `CREATE INDEX` holds `SHARE` against writers for the whole build, or build it out of band and name it an operator action with its own authority — then say which environments the migration set no longer provisions on its own.
- Give a chunked, restartable, throttled backfill its own checkpoint outside `migrations/`, keeping the migration additive. `internal/background` supplies the supervisor and cancellation; whether it runs as a task in an existing binary or its own `cmd/` entry is `go-implementation-ownership`'s call. The alternative the runner forces is one transaction spanning the whole table.
- Split `ALTER TYPE ... ADD VALUE` from the rows that use it: on PostgreSQL 17 a value added inside a transaction is unusable until that transaction commits.
- Reach `NOT NULL` on the pinned `postgres:17` through a `NOT VALID` `CHECK (col IS NOT NULL)`, `VALIDATE CONSTRAINT`, then `SET NOT NULL`, which skips the table scan a valid check already proves. Not-null constraints became addable as `NOT VALID` only in PostgreSQL 18.
- `internal/infra/postgres/sqlc.yaml` reads `migrations/` as its schema source, so DDL and generated Go move in one change.

[postgresql-ddl.md](../../../../docs/universal-disciplines/postgres-schema-design/references/postgresql-ddl.md) owns the expand, backfill, verify, contract sequence; follow it there rather than restating it here.

## Reject
- A `Down` that drops the column a deployed reader already depends on. It reads as a reversal and destroys the only copy of the data.
- `IF NOT EXISTS` as *evidence* that an existing object matches the intended definition — a name collision is not definition equality. Reconciling an out-of-band build this way is fine once the deployed definition is checked rather than assumed.

## Validation Shape
`make migration-check`, `make migration-validate`, and `make sqlc-check`. State which step is irreversible and what the `Down` section actually restores.
