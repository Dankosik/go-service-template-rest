<!-- profile:jobs-postgres:start -->
# PostgreSQL durable background jobs

`JOBS=postgres` retains the reusable default-off job pack and `/jobs-worker`.
The worker fails closed before database I/O until a derived service supplies a
concrete builder. It neither creates a producer nor enables operator controls,
periodic scheduling, multiple classes, deployment, or capacity claims.
Every registered definition's termination envelope must fit
`http.grace_period`; the worker rejects a mismatch before opening PostgreSQL.

Worker metrics include `postgres.jobs.queue.delay` (eligible-to-claim delay)
and `postgres.jobs.attempt.duration` (handler duration), bounded Store-operation
duration/outcome, and terminal/drain events. They carry no job, payload,
producer, occurrence, effect identity, or arbitrary error; attributes use only
closed operation/outcome vocabularies.

This pack is a durable execution kernel, not a complete job platform. It has no
authenticated inspect/cancel/redrive/delete surface, terminal cleanup loop,
retention defaults, tenant fairness, admission backpressure, capacity model, or
SLO/alert thresholds. `postgres_job_actions` is a deprecated compatibility
relation from migration `000004`, not reserved capacity: current schema
admission, runtime code, roles, and proofs do not depend on it. Remove it only
in a later append-only contract migration after every N-1 worker that still
checks the original exact schema has left the rollback window.

Compatibility blocks claims only when a `ready`, `scheduled`, `retry_wait`,
`running`, or `cancel_requested` row has an unregistered exact revision.
Terminal history stays retained but does not globally stop unrelated live work;
any future redrive must recheck the target revision before changing state.
Observation cadence and readiness freshness use a worker-local monotonic
deadline derived from the observation interval, one poll interval, and one
Store-operation timeout. PostgreSQL `clock_timestamp()` remains authoritative
for persisted scheduling, leases, and transition timestamps, never local timer
arithmetic.

Before production use, the adopter must supply a concrete definition and effect
idempotency/reconciliation authority, payload classification and retention,
operator decision (including safe redrive classes), representative capacity and
PostgreSQL/vacuum evidence, N/N-1 artifact proof, termination-platform bounds,
and alerts/runbooks. Until those inputs exist, keep producer emission and any
manual recovery surface disabled.

`app.instance_id` is only the deployment-instance prefix. Each worker process
adds a random suffix so overlapping replicas and restarts never share lease or
fencing identity.

Local commands:

```sh
make run-jobs-worker
make build-jobs-worker
```

Run `/migrate` for schema changes; application and worker startup never migrate.
<!-- profile:jobs-postgres:end -->
