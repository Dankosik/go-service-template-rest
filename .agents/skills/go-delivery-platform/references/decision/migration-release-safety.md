# Migration Release Safety

## Behavior Change Thesis
When loaded for symptom "a release includes schema migrations or data-moving rollout," this file makes the model choose rehearsal, rollback classification, mixed-version compatibility, and one-migrator ownership instead of likely mistake "run migrations before deploy" or treating a successful `up` as enough.

## When To Load
Load for migration rehearsal, release sequencing, rollback class, mixed-version compatibility windows, backfill gates, one-migrator policy, or promotion criteria tied to migrations.

## Local Source Of Truth
- `.github/workflows/ci.yml` runs `migration-validate` on ephemeral Postgres when `env/migrations/` changes.
- `Makefile` exposes `migration-validate` and `docker-migration-validate`.
- `scripts/dev/docker-tooling.sh` runs migration up, down one step, and up one step against temporary Postgres.
- `railway.toml` records healthcheck, overlap, draining, restart policy, and replica baseline comments that affect rollout windows.

## Decision Rubric
- Migration changes block merge unless CI proves `up -> down 1 -> up 1` on ephemeral Postgres, or an exception states why rollback rehearsal is impossible and what compensating proof replaces it.
- Release specs must classify migrations as reversible, forward-only, or destructive; forward-only and destructive changes require explicit rollback/restore strategy.
- Rolling or overlapping deploys require mixed-version compatibility across the overlap window. If compatibility is unknown, delivery is blocked and data/API ownership must decide.
- Use one controlled migrator job/process for production execution. Do not run migrations opportunistically on every app pod or process startup unless the owning data spec proves idempotence and concurrency safety.
- Phased schema release must name gates for expand, backfill/migrate, verify, and contract; do not mark contract release-safe until compatibility and rollback evidence exist.

## Imitate
- "For migration PRs, require CI `migration-validate` with `MIGRATION_DSN` against ephemeral Postgres and evidence of `up -> down 1 -> up 1`." Copy the exact rehearsal sequence.
- "A destructive contract step is release-blocked until the data spec owns compatibility, restore, and verification; delivery records the blocker rather than approving the schema decision." Copy the handoff behavior.
- "Backfill release requires budget, checkpoint, retry, and verification artifact before promotion." Copy the data-moving proof shape.

## Reject
- "Run migrations before deploy." This omits migrator owner, target DB, lock/concurrency behavior, rollback class, and proof.
- "Local `make migration-validate` printed a skip because Docker was unavailable, so migrations are validated." A skip is not proof.
- "Rolling deploy plus schema contraction in one release." This collapses compatibility, rollback, and deployment windows into one irreversible step.

## Agent Traps
- Do not let app startup migrations become the default when autoscaling or restarts can create concurrent migrators.
- Do not equate successful `up` with reversible-migration proof.
- Do not design schema shape, transaction safety, or event semantics here; require proof and route decisions to data or distributed-consistency specs.

## Validation Shape
Use `migration-validate` CI logs, production migration run identifier, command, automation/operator identity, target database/environment, backfill checkpoint output, verification queries, rollback class, and restore/backup evidence when rollback depends on restore.

## Hand-Off Boundary
Do not design schema shape, data invariants, transaction boundaries, or event publication semantics here. Delivery can require proof and block release, but data and distributed-consistency specs own those decisions.
