# Migration Rollout Window

## Load When
Load for release sequencing with a schema change, rollback class, backfill gating, migrator ownership, or a promotion claim that rests on "the migration ran."

## Decide
- `railway.toml` sets `preDeployCommand = ["/migrate"]` under the `database-postgres` profile. The migrator runs to completion **before** any new instance starts, so every migration must be readable and writable by the version already in production. A change that only the new code can tolerate breaks the old one.
- `overlapSeconds = 45` and `drainingSeconds = 45` size that window: old and new instances both serve for up to 45 seconds against the new schema. This is the mixed-version window — it is never zero, and no gate closes it.
- Sequencing a change so both versions stay correct is owned by [`postgres-schema-design`](../../../../../docs/universal-disciplines/postgres-schema-design/SKILL.md); its DDL reference carries the ordered expand → backfill → switch → contract steps. Delivery decides which release each step ships in and what proves it, not the DDL shape.
- Migration files are append-only, enforced by `scripts/ci/migration-history-check.sh` against the merge base and `migration-image-history-check.sh` against the published image. A contract step is always a new migration; editing an applied one fails CI rather than rewriting history.
- Concurrent migrators are already handled: `internal/infra/postgresmigrate` acquires a goose Postgres session lock with `MigrationLockTimeout`, and nothing migrates on service startup. Re-deciding migrator ownership is wasted work unless the change moves migration out of `preDeployCommand`.
- `make migration-validate` rehearses `up → down → up` on a disposable Compose Postgres and against the production image. CI runs it as a step in `container-security` whenever migrations, `cmd/`, `internal/`, the Dockerfile, the Makefile, or `go.mod` change — a wider filter than `migrations/` alone.

## Inspect
- "Adding a required column ships as nullable-with-default plus backfill this release; the `NOT NULL` tightening ships after the overlap window closes, as a separate migration." Copy the split-across-releases habit.

## Reject
- "The migration succeeded, so the release is safe." Success proves the schema moved, not that the still-running version can read and write it.
- "Local `make migration-validate` printed a Docker-unavailable skip." A skip is not rehearsal evidence.

## Reopen
- Forward-only and destructive steps still pass `up → down → up` rehearsal when the down migration merely drops what the up added; rehearsal proves reversibility of shape, not recoverability of data.
- Railway's healthcheck at `/health/ready` with `healthcheckTimeout = 180` gates promotion only; it observes the new instance, never the old one still draining against the migrated schema.

## Prove
Use the `container-security` job log showing the migration-validate step ran rather than skipped, the append-only history check output, the rollback class with its restore evidence when reversal depends on backup, and the backfill verification query.
