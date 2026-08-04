# Operational branches

Read only the sections named by the request. The core lease, effect, failure, and authority rules remain in [SKILL.md](../SKILL.md).

## Capacity, fairness, and backpressure

Set global and per-job-type concurrency from downstream capacity. Add per-tenant limits or fair selection when one tenant could monopolize workers. Define rate-limit ownership, admission behavior under saturation, queue-age SLOs, and the signal that reduces claims. Preserve priority without starvation.

Extend the crash matrix with admission rejection, throttled claims, and dependency saturation. Test selection under continuous high-priority load and prove concurrency remains inside the named dependency budgets.

## Civil schedules and misfires

Store civil intent or instant rule, IANA time-zone ID, schedule occurrence ID, time-zone data policy, DST gap/fold behavior, misfire policy, catch-up bound, overlap policy, jitter, start/end bounds, and cancellation behavior. A daily civil schedule is not a fixed 24-hour interval. Derive billing identity from the intended occurrence, not the actual start time.

Extend the matrix with occurrence creation, delayed enqueue, outage recovery, overlap, and cancellation. Test named-zone spring gaps and autumn folds, bounded catch-up, and runs longer than their interval.

## Long-running jobs and cancellation

- Commit a replay-safe chunk effect before storing the next unprocessed cursor and checkpoint version.
- Treat heartbeat as liveness/lease renewal, not proof that a checkpoint or effect committed.
- Poll cancellation at safe boundaries, stop creating new effects, and record request, observation, and completion.
- Resume from the durable cursor with the same job and effect identities.
- Throttle at claim and chunk boundaries so backpressure and cancellation remain responsive.

Extend the matrix with each chunk effect/checkpoint boundary, heartbeat loss, lease expiry, cancellation, and resume. Prove crash and cancellation cannot skip or multiply a chunk effect.

## Deploy, drain, and versioning

Stop claiming first. Renew and finish bounded in-flight work or cooperatively checkpoint/release it, with an explicit hard-deadline behavior. Keep compatible workers or routing until queued payloads, checkpoints, schedules, and durable histories no longer require them.

Test current and previous payload/checkpoint versions, hard shutdown recovery, replay or pinning contracts, and the criterion for retiring old workers.

## Recovery, retention, and observability

Detect stuck work from state plus queue age, lease/heartbeat age, runtime, attempts, and unchanged checkpoint. Make manual retry a new audited attempt against the same business-effect key. Reconcile accepted business records, queued/running jobs, and durable effects to find missing, duplicated, or ambiguous work.

Retain job history, results, payload metadata, errors, attempts, cancellation audit, and effect/deduplication keys for explicit product, privacy, and compliance periods. Effect and deduplication retention exceeds the longest retry, redrive, misfire, and late-delivery window.

Observe enqueue rate, ready depth and age, claims and lease renewals, attempts and retry budget, runtime/checkpoint progress, failures/quarantine/cancellation, effect replays and reconciliation drift, fairness/rate saturation, capacity, and deploy drainage.

## Load and deploy proof

Run load tests for renewal under pause, fairness, rate limiting, backpressure, and dependency saturation. Run deploy tests for old/new worker compatibility, drain, replay, and resumption. Keep these tests out of narrower requests that do not exercise the corresponding branch.
