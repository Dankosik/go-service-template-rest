# Worker Runtime Design

Status: review-ready
Date: 2026-06-02

## Runtime Owner

`cmd/billing-worker` owns all durable async work for the cutover. The HTTP
service owns request/response mapping only. Enabled production runtime must not
use no-op worker tasks.

## Required Roles

The existing `internal/app/microleaseworker` role model is retained:

| Role | Concrete responsibility | Store/adapter owner |
| --- | --- | --- |
| `terminal_consumer` | Consume terminal events, apply terminal outcomes, quarantine conflicts, commit offsets after store effect. | `internal/infra/redpanda` + Postgres repository |
| `checkpoint_consumer` | Consume checkpoint progress, update checkpoint/readback state, flag close gaps and stale child exposure. | New Redpanda checkpoint adapter + Postgres repository |
| `close_consumer` | Consume close/shutdown/repair proof, release only proven unallocated capacity, open reconciliation for unresolved child cap. | New Redpanda close adapter + Postgres repository |
| `inbox_retry` | Retry received/conflict/quarantined/reconcile-required inbox records that are eligible for replay or redrive. | Postgres repository + app reconciliation policy |
| `outbox_relay` | Claim billing outbox rows, validate fingerprint, publish support-safe facts, mark published or retry. | Existing Redpanda outbox relay + concrete producer |
| `stale_reconciliation` | Scan stale usage holds, microleases, child debits, terminal lag, close gaps, import mismatches, and event conflicts. | App reconciliation + Postgres repository |
| `admission_control_renewal` | Compute and renew global/account/use-class admission states from lag, stale exposure, backlog, and reconciliation gates. | App microlease rollout/admission policy + Postgres repository |

## Bootstrap Requirements

When worker runtime is disabled:

- command exits cleanly;
- migrated paid cohorts must not be enabled;
- HTTP readbacks expose runtime-disabled/not-ready state where relevant.

When worker runtime is enabled:

- config validation requires Postgres, Redpanda, service auth config, microlease
  config, and Redis not used as spend authority;
- bootstrap opens Postgres and Redpanda clients;
- dependency probes include Postgres and Redpanda;
- all seven required roles receive concrete tasks;
- readiness becomes true only after probes pass and task construction is
  complete.

Enabled-but-no-op task construction is a production blocker.

## Task Behavior

Each task:

- accepts context cancellation;
- uses bounded concurrency;
- emits low-cardinality metrics by role, result, and reason class;
- uses app-owned commands and Postgres transactions for money effects;
- keeps retry/backoff outside HTTP handlers;
- does not log raw payloads, secrets, tokens, prompts, completions, SSE chunks,
  dynamic proof URLs, or provider payloads.

Recommended concurrency:

- terminal consumer: bounded greater than one only if row-lock tests prove
  account/child ordering remains safe;
- checkpoint and close consumers: default one per partition/business identity
  until proof supports more;
- stale reconciliation: bounded batch size from config;
- outbox relay: bounded claim limit and retry delay.

## Readiness And Admission Gates

Worker readiness gates migrated paid admission through:

- terminal lag bucket;
- stale microlease and stale child debit age;
- inbox conflict/quarantine backlog;
- outbox retry backlog;
- reconciliation backlog severity;
- admission control freshness;
- Redpanda healthcheck status.

Critical state sets admission controls to `fail_closed`. Warning state may set
`strict` or `throttle` only if the selected mode still preserves durable
microlease and child-debit authority.

## Failure And Recovery

- Terminal consumer applies or quarantines before offset commit.
- Store retryable failures keep the offset uncommitted.
- Quarantine stores support-safe reason class and opens reconciliation where
  needed.
- Outbox publish failure schedules retry without rolling back committed ledger
  state.
- Worker shutdown cancels tasks and leaves uncommitted work for replay.
- Reconciliation scans repair stale states by creating explicit write-off,
  release, reversal, compensation, or reconciliation outcomes through the same
  app service paths used by normal terminal handling.

## Implementation Inputs For Planning

Planning must include tasks to:

- replace `disabledRuntimeTasks()` when worker is enabled;
- add concrete Redpanda checkpoint and close consumers;
- add concrete Redpanda client producer/consumer wrappers if absent;
- add Postgres repository methods for worker claims, stale scans, admission
  renewal, inbox retry, and readback summaries;
- wire worker telemetry and readiness;
- prove signal-aware shutdown and no goroutine leaks where package patterns
  support it.
