# Migration Release Safety

## When To Load
Load this when a delivery spec needs migration rehearsal, release sequencing, rollback class, mixed-version compatibility windows, backfill gating, one-migrator policy, or deployment promotion criteria.

## Local Source Of Truth
- `.github/workflows/ci.yml` runs `migration-validate` only when `env/migrations/` changes and validates on ephemeral Postgres.
- `Makefile` exposes `migration-validate` and `docker-migration-validate`.
- `scripts/dev/docker-tooling.sh` runs migration up, down one step, and up one step against a temporary Postgres network.
- `railway.toml` records non-secret deploy policy, healthcheck, overlap, draining, restart policy, and replica baseline comments.

## Enforceable Policy Examples
- Migration changes block merge unless `migration-validate` proves `up -> down 1 -> up 1` on an ephemeral database, or an exception states why rollback rehearsal is impossible and what compensating proof replaces it.
- Release specs must classify migrations as reversible, forward-only, or destructive; forward-only and destructive changes require explicit rollback and restore strategy.
- Rolling or overlapping deploys require mixed-version compatibility for every schema phase visible during the overlap window; if compatibility is unknown, delivery status is blocked and data/API ownership must decide.
- Use one controlled migrator job/process for production migration execution; do not run migrations opportunistically on every app pod or process startup unless the owning data spec proves idempotence and concurrency safety.
- Phased schema release policy should name the actual gates for `expand`, backfill or migrate, verify, and `contract`; the delivery spec should not treat `contract` as release-safe until compatibility and rollback evidence exist.

## Non-Enforceable Anti-Patterns
- "Run migrations before deploy" without naming the migrator owner, command, database target, lock behavior, and rollback class.
- Treating a successful `up` as enough for reversible migrations when the local policy has a down/up rehearsal.
- Combining schema contraction and application rollout in a single irreversible release without a mixed-version compatibility decision.
- Backfills with no budget, checkpoint, retry, or verification artifact.
- Relying on app startup migration when autoscaling, restarts, or parallel deploys can create multiple migrators.

## Evidence Artifacts
- `migration-validate` CI logs for changed migrations.
- Production migration run identifier, command, operator or automation identity, start/end time, and target database/environment.
- Backfill checkpoint and verification query output for data-moving releases.
- Rollback class recorded in the release spec, with restore drill or backup evidence when rollback depends on restore.
- Deployment health evidence tied to `railway.toml` healthcheck, overlap, draining, and restart policy when Railway is the target.

## Hand-Off Boundary
Do not design schema shape, data invariants, transaction boundaries, or event publication semantics here. Delivery can require proof and block release, but data and distributed-consistency specs own those decisions.

## Exa Source Links
- GitHub Docs: [Workflow syntax for GitHub Actions](https://docs.github.com/actions/using-workflows/workflow-syntax-for-github-actions)
- GitHub Docs: [Control the concurrency of workflows and jobs](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency)
- Kubernetes Docs: [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment)

