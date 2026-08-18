<!-- profile:jobs-postgres:start -->
# PostgreSQL background jobs

`JOBS=postgres` retains River, its Goose-owned schema migration, and the
`/jobs-worker` process. Business code defines a typed `river.JobArgs`, implements
`river.Worker[T]`, registers it once with `river.AddWorkerSafely`, and inserts it
through `Client.InsertTx` in the business transaction.
<!-- profile:webhooks-durable:start -->
`WEBHOOKS=durable` supplies the template's one concrete River worker; without
that profile the nil-builder fail-closed contract remains unchanged.
<!-- profile:webhooks-durable:end -->

The template does not implement a second claim, lease, retry, heartbeat, rescue,
scheduling, concurrency, shutdown, readiness, telemetry, or operator engine.
River owns those mechanics. The worker adds River's OpenTelemetry plugin to the
repository's existing providers and exposes the standard process diagnostics.

The reusable binary deliberately has no job kind. A derived service replaces its
nil worker builder and registers its business workers before PostgreSQL is opened.
`jobs.max_workers` is the only jobs-specific runtime setting; the source template
uses `0` to stay inert, and a selected deployment starts at `1` unless capacity
evidence supports more.

River's configured baseline is its explicit one-minute job timeout and 25-attempt
retry default. Business job types may narrow attempts, timeout, queue, uniqueness,
or one-off schedule through River's native `InsertOpts` and worker methods. The
baseline uses one default queue and no periodic jobs, River UI, or River Pro.

River provides enqueue uniqueness, not effect idempotency. Workers run with
at-least-once semantics, so an external side effect still needs a stable provider
idempotency key, a repeat-safe invariant, or reconciliation owned by the business
feature. Keep payloads to stable identifiers rather than secrets or unnecessary
personal data; OSS retention is client-wide.

Migration `000004` remains append-only legacy history. Shared migration
`000008_river.sql` vendors River v0.44.0 migrations 001-007 into the canonical
Goose sequence for jobs, outbox, and webhooks. Existing
deployments must stop legacy production, drain every nonterminal `postgres_jobs`
row, apply the additive River schema, and then switch producer and worker code.
If legacy work cannot drain, do not deploy this cutover without a separately
designed row bridge. Legacy tables remain during the rollback window.

The worker runs River in `PollOnly` mode because the shared PostgreSQL pool
enforces finite statement budgets; an unbounded `LISTEN` would conflict with
that connection policy.

River's OSS UI is not part of this profile and is publicly accessible by default.
Any deployment of it needs externally owned authentication and network policy.
Per-queue retention, durable periodic schedules, global concurrency limits, and
the Pro dead-letter queue require an explicit River Pro purchase and reopen.

Local commands:

```sh
make run-jobs-worker
make build-jobs-worker
```

Run `/migrate` for schema changes; application and worker startup never migrate.
<!-- profile:jobs-postgres:end -->
