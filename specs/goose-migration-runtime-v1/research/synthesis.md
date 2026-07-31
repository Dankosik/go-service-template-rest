# Goose migration runtime evidence

status: ready

## Decision

Use `github.com/pressly/goose/v3` v3.27.3 as the PostgreSQL migration
engine for the base template. Retain a narrow repository-owned `/migrate`
binary for configuration, fail-closed source discovery, mandatory locking,
structured reporting, and the production-only `up` boundary.

This decision applies to the template and newly initialized services. It does
not authorize rewriting migration files or metadata in an already deployed
derived repository.

## Current repository baseline

- The repository currently pins `github.com/golang-migrate/migrate/v4`
  v4.19.1 and wraps it in `internal/infra/postgresmigrate`.
- Railway invokes `/migrate` as a pre-deploy command. Application startup does
  not run schema migrations.
- The production binary exposes only `up`; `MigrateDown` exists for disposable
  integration proof.
- The configured defaults are a five-minute orchestration budget, two-minute
  statement budget, and fifteen-second migration-lock budget.
- A missing migration directory fails closed, while an existing empty
  directory is the valid state before the first owned migration.
- The template currently has no owned migration corpus, so changing the source
  format does not rewrite released history in this repository.

## Decision-changing evidence

### Cancellation and locking

Goose's current `Provider` API accepts `context.Context` for connection,
status, lock, transaction, and migration operations. Its PostgreSQL session
locker retries `pg_try_advisory_lock`, which does not leave one indefinitely
blocked lock query behind. Locking is disabled by default and therefore must be
mandatory in the repository-owned constructor.

The current `golang-migrate` pgx/v5 driver calls `Ping`,
`pg_advisory_lock`, `pg_advisory_unlock`, and several metadata operations
without the caller's context. The core `LockTimeout` can return while the
driver's blocking advisory-lock query is still active. Keeping that engine
would require server-side or local-driver compensation for a lifecycle defect
inside the dependency.

Sources:

- <https://pressly.github.io/goose/documentation/provider/>
- <https://github.com/pressly/goose/blob/v3.27.3/lock/postgres.go>
- <https://github.com/pressly/goose/blob/v3.27.3/provider_run.go>
- <https://github.com/golang-migrate/migrate/blob/v4.19.1/database/pgx/v5/pgx.go>

### Transaction and failure semantics

Goose runs each SQL migration in one transaction by default and makes the
exception explicit with `-- +goose NO TRANSACTION`. It returns per-migration
results and a `PartialError` naming work completed before a failure.

The Provider collects file metadata eagerly but parses each SQL migration
lazily immediately before applying that file. Goose's CLI `validate` command
uses the same internal parser without a database, so it belongs in the
pre-merge source gate; the production adapter can still reject repository
annotations and naming before it opens PostgreSQL without copying Goose's
parser.

Goose does not persist a `dirty` marker before a non-transactional migration.
A failed `NO TRANSACTION` file can therefore leave partial database effects
without a durable engine-owned fence. The base template therefore rejects that
annotation instead of adding a second, repository-owned migration state
machine. PostgreSQL operations that cannot run transactionally require a
separate restart-safe release-operation design when a concrete need appears.

Source:

- <https://pressly.github.io/goose/documentation/annotations/>
- <https://github.com/pressly/goose/blob/v3.27.3/provider_errors.go>
- <https://github.com/pressly/goose/blob/v3.27.3/provider_run.go>

### History and integrity

Goose records the currently applied version set rather than only the maximum
version, and rejects a newly introduced lower unapplied version by default.
Neither Goose nor `golang-migrate` stores migration-file checksums. Goose also
does not reject an applied database version merely because its source file was
removed. Repository history therefore remains the authority for file
immutability and needs an append-only merge gate regardless of engine choice.

`sqlc` accepts a migration directory as schema input and parses Goose files by
reading only their Up sections. Because it processes migration files
lexicographically, the repository needs one fixed-width numeric filename
grammar shared by Goose, sqlc, CI, and documentation.

Source:

- <https://docs.sqlc.dev/en/latest/howto/ddl.html#handling-sql-migrations>

### Operability and maintenance

Goose v3.27.3 is the current signed release as of 2026-07-30 and supports the
repository's Go 1.26 toolchain. Its Provider returns source, direction,
duration, state, and failure information suitable for structured pre-deploy
logs. Both candidate projects are mature and MIT licensed; maintenance status
does not reverse the decision.

Sources:

- <https://github.com/pressly/goose/releases/tag/v3.27.3>
- <https://github.com/golang-migrate/migrate/releases/tag/v4.19.1>

## Constraints carried forward

- Use the Provider API, not legacy global Goose functions.
- Use SQL migrations only.
- Make PostgreSQL session locking non-optional in production construction.
- Keep database connection, statement, DDL-lock, migration-lock, and cleanup
  work inside explicit finite budgets.
- Keep out-of-order application disabled.
- Keep the production binary `up`-only.
- Preserve the absent-versus-empty migration-source distinction.
- Reject `NO TRANSACTION`; a future non-transactional release operation needs
  a separate accepted design and proof.
- Add repository-owned append-only history proof; do not claim Goose provides
  checksum integrity.
- Validate the actual owned migration corpus with `up all -> down all -> up
  all` on disposable PostgreSQL.

## Evidence limits and refresh triggers

- No live contention or process-termination probe was run during tool
  selection. Implementation acceptance must supply those probes against the
  pinned version.
- Reopen dependency behavior if the pinned Goose version changes, Provider
  locking becomes default, transaction/version-store semantics change, or a
  supported PostgreSQL major changes.
- Reopen migration-history design before converting an already deployed
  derived repository.
