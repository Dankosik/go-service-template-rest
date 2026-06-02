# Event-Driven Billing Money Architecture Rollout Plan

Status: repaired review-ready technical design companion for billing-issued
spending leases
Trigger: required before planning because the accepted architecture needs lease
path cutover, proxy durable allocator rollout, bridge exit criteria,
mixed-version behavior, rollback/failback, and no-dual-writer proof.

This artifact defines rollout choreography for later planning. It is not a task
ledger.

## Target State

- billing-service is the single customer-money writer for migrated cohorts.
- billing-service issues/replenishes bounded spending leases and reserves the
  full lease USD exposure in Postgres.
- `gonka-proxy` no longer writes authoritative balance/reservation/deduction
  state for migrated cohorts.
- paid execution starts only after proxy durably allocates a child debit from
  an active billing-issued lease and stores the terminal submission obligation.
- terminal settlement is carried by Redpanda terminal facts into billing durable
  inbox.
- lease checkpoint/close facts release unused capacity only after billing
  validates durable proof.
- billing emits derived facts from transactional outbox.
- stale leases, stale debits, terminal ambiguity, and proxy over-debit are
  repaired from billing Postgres state and proxy durable lease/debit lineage,
  not from raw request payloads.
- new lease issuance/replenishment is controlled by Postgres
  `billing_admission_controls`; stale, missing, `throttle`, or `fail_closed`
  state blocks new capacity before a lease hold is created.

## Rollout Sequence

### R0. Expand Without Traffic

- Add protected OpenAPI contract and auth middleware, but keep routes disabled
  or limited to non-production callers.
- Add spending lease, child debit settlement, checkpoint, inbox/outbox,
  admission-control, lineage, indexes, and constraints.
- Seed `billing_admission_controls` with default paid-usage admission closed
  until worker/operator health proof opens it.
- Add protobuf event contract inputs under `api/proto/events/v1/` and
  generation/drift/compatibility checks before event adapters are enabled.
- Add worker binary/config with Redpanda disabled by default.
- Add proxy durable lease allocator, terminal-submission store, and relay
  disabled by default.
- Add feature flags or cohort gates only as rollout controls, not as permanent
  architecture splits.

Safety checks:

- migrations validate;
- OpenAPI generation/checks pass;
- SQLC generation/checks pass;
- proto generation/drift/compatibility checks pass;
- protected-route auth config fails closed when missing;
- admission-control missing/stale/default-closed state rejects lease issuance
  without creating reserved exposure;
- no public ingress is opened for protected money routes;
- worker disabled state does not claim readiness for event processing.

### R1. Import And Shadow

- Import or map existing proxy balances into explicit billing USD ledger state
  for candidate cohorts.
- Run shadow balance readback and parity checks.
- Exercise lease issue/readback/close in non-production or test cohort mode
  where possible.
- Exercise proxy durable child debit allocation in shadow mode without external
  customer-money authority.
- Keep proxy-local writer authoritative until cutover gate passes for a cohort.

Safety checks:

- parity results are support-safe and do not include raw prompts, completions,
  tokens, or secrets;
- import mismatches create reconciliation cases or block cohort enablement;
- no dual writer is enabled.

### R2. Enable Billing Lease Issuance For Cohorts

- Enable proxy calls to protected billing lease issue/replenish/readback for
  selected cohorts.
- Proxy persists lease grants before any child debit can spend them.
- Proxy local child debit allocation becomes the only paid hot path for
  migrated cohorts.
- External execution starts only after child debit allocation and terminal
  submission obligation are durable.
- Proxy-local money writes and direct per-request reserve fallback are disabled
  for the migrated cohort before billing is declared authoritative.

Safety checks:

- lease issuance and cold replenishment latency stay within budgets;
- active-lease child allocation latency stays within budgets;
- insufficient funds/stale pricing/auth failures fail closed;
- proxy ambiguous lease timeout retries/readbacks use same command identity;
- no new proxy-local authoritative balance mutation for cohort.

### R3. Enable Terminal And Checkpoint Settlement

- Enable proxy terminal relay to `usage.execution.terminal.v1`.
- Enable proxy lease checkpoint/close relay to `usage.lease.checkpoint.v1`.
- Enable billing terminal/checkpoint consumers and inbox retry worker for
  selected cohorts.
- Enable billing outbox relay for derived facts.
- Enable worker renewal of `billing_admission_controls` for selected cohorts
  only after lag/backlog, checkpoint processing, and stale lease/debit scans are
  healthy.
- Monitor terminal lag, checkpoint lag, stale leases, stale child debits,
  admission-control state age, quarantine/conflict rate, outbox pending age,
  and account/lease contention.

Safety checks:

- duplicate terminal events do not duplicate charges;
- Redpanda outage keeps proxy terminal/checkpoint submissions durable and
  outbox rows pending;
- worker crash after DB commit before offset commit redelivers safely;
- stale lease/debit reconciliation opens cases before accepted age budget is
  missed;
- critical lag, stale controls, or stale exposure breach flips new capacity to
  throttle or fail closed before more exposure is minted.

### R4. Bridge Exit And Contract

- Disable old shared-balance bridge routes by default for migrated cohorts.
- Remove or hard-disable proxy-local writer paths for migrated cohorts.
- Prove no dual writer remains.
- Prove direct per-request reserve fallback is disabled for migrated cohorts.
- Keep old read-only compatibility only where support requires it and where it
  cannot mutate money.

Exit criteria:

- proxy uses target billing-service OpenAPI for lease issue/replenish/readback
  for migrated cohorts;
- proxy-local money writes are disabled for those cohorts;
- paid hot path uses durable child debits from billing-issued leases;
- terminal and checkpoint facts flow through proxy durable submission and
  Redpanda;
- billing inbox/outbox/reconciliation metrics are healthy;
- parity checks show no divergent balance mutation;
- bridge routes are removed or disabled by default.

## Mixed-Version Rules

- New billing routes may be deployed before proxy uses them, but must fail
  closed without auth/scope config.
- Proxy durable lease allocator and terminal-submission support must be
  deployed before any cohort can spend from billing-issued leases.
- Billing terminal/checkpoint consumers can be deployed disabled before events
  are produced.
- Redpanda topics can exist before producers/consumers are enabled.
- Admission-control rows default to fail closed until worker or protected
  operator authority renews an `open` lease.
- Billing outbox relay can publish derived facts after source rows exist;
  consumers must dedupe.
- Old proxy writer and new billing writer must not both mutate the same
  cohort's authoritative money state.
- Old per-request reserve bridge behavior must not be active as a paid fallback
  for migrated cohorts.

## Rollback And Failback

Before a cohort is billing-writer enabled:

- rollback can disable new calls and keep proxy-local writer authoritative if
  no billing writer mutation occurred for that cohort.

After a cohort is billing-writer enabled:

- rollback may fail paid admission closed;
- rollback may continue spending already-minted valid leases through cutoff only
  if proxy durable terminal submission and billing reconciliation remain
  healthy;
- rollback may stop issuing/replenishing new leases by closing admission
  controls;
- rollback must not reactivate proxy-local balance mutation or direct
  per-request reserve for the same migrated money scope without explicit
  reconciliation and approval;
- any already allocated child debit must settle, write off, or reconcile
  through billing.

Forward recovery is preferred for terminal or checkpoint delays: replay proxy
terminal/checkpoint submissions, process inbox retries, and repair
reconciliation cases.

## Operational Gates

Do not expand cohorts when any of these are true:

- lease issuance/replenishment p95/p99 exceeds accepted budget without
  explanation;
- active-lease child allocation latency exceeds accepted budget;
- terminal or checkpoint event critical lag persists beyond budget;
- admission-control lease is missing, expired, stale, or fail closed;
- stale lease/debit reconciliation cannot keep up;
- inbox conflict/quarantine rate indicates producer contract drift;
- outbox pending age exceeds alert thresholds;
- account-row or lease-row contention causes sustained paid-admission failures;
- protected auth or producer ACL verification is degraded;
- privacy checks find raw sensitive payloads in logs, traces, metrics,
  inbox/outbox, audit, or reconciliation rows.

## Bridge Ownership

If a compatibility bridge is required:

- bridge owner: proxy integration layer plus billing-service target OpenAPI
  contract;
- target owner: billing-service OpenAPI;
- bridge must preserve target account scope, lease/debit identities,
  idempotency keys, fingerprints, stored outcomes, and privacy constraints;
- bridge must not preserve direct per-request reserve fallback or proxy-local
  balance mutation for migrated cohorts;
- bridge removal proof is part of rollout exit.

The bridge is not an alternate source of truth and cannot be a long-lived
contract owner.
