# Event-Driven Billing Money Architecture Test Plan

Status: repaired review-ready technical design companion for billing-issued spending leases
Trigger: required before planning because validation spans lease money math,
protected HTTP, database invariants, proxy durable allocation, fencing,
Redpanda replay, checkpoint/close, privacy, performance, and rollout safety.

This artifact defines proof obligations for later planning. It does not run
tests or create implementation tasks.

## Scope

In scope:

- protected lease issue/replenish/readback/close HTTP behavior;
- billing Postgres spending lease, child debit lineage, checkpoint, inbox,
  outbox, admission-control, and ledger invariants;
- proxy durable lease allocator and terminal submission proof obligations;
- multi-proxy fencing and stale generation rejection;
- Redpanda terminal, lease checkpoint/close, billing fact, and rejection event
  behavior;
- runtime protobuf schema validation for current-scope Redpanda events;
- reconciliation and stale lease/debit recovery;
- auth/privacy/observability assertions;
- performance budgets from `spec.md`;
- rollout and compatibility validation.

Out of scope:

- payment/top-up and payment-evidence runtime tests;
- direct per-request reserve fallback tests as target behavior;
- public `/v1*` proxy behavior except where cutover proves no visible contract
  regression.

## Proof Obligations

### Money Math And PostgreSQL Invariants

- USD atom parser/formatter and rounding vectors for lease issue, replenish,
  child debit cap, final charge, release, write-off, reversal, compensation,
  and close.
- Ledger conservation over lease holds, lease replenishments, debit charges,
  capacity releases, write-offs, reversals, operator adjustments, imports, and
  reconciliation corrections.
- `available_usd_atoms = settled_usd_atoms - reserved_usd_atoms` under lease
  issue/replenish/settlement/close.
- Non-negative available/reserved/settled/pending components.
- Account-row locking and concurrent lease issuance cannot take available below
  zero.
- Concurrent lease replenishment and close cannot release or reserve capacity
  twice.
- Terminal settlement charges no more than child cap and no more than aggregate
  billing-issued lease budget.
- Proxy over-debit evidence caps customer charge at valid lease authority and
  opens reconciliation/write-off for excess.
- Same idempotency key plus same fingerprint replays stored lease outcome;
  changed fingerprint conflicts without money mutation.
- Same child debit plus same operation/terminal fingerprint replays stored
  outcome; changed fingerprint conflicts without money mutation.

### Protected HTTP Contracts

- OpenAPI contract validation for protected routes and generated server wiring.
- Protected money routes are not under OpenAPI `security: []`.
- 401 for missing/invalid service auth.
- 403 for missing route scope or account/caller mismatch.
- 400 for malformed body, invalid decimal, missing idempotency, invalid
  deadline, invalid lease owner, or invalid close proof shape.
- 402 insufficient funds for requested lease capacity.
- 409 idempotency, lease, or debit conflict.
- 422 stale pricing, unsupported currency, account not spendable, stale policy
  evidence, unsupported use class, invalid expiry/cutoff, or stale fence.
- 429 account/lease contention, account-scoped throttle, or overload.
- 503 not ready before acceptance, including missing/expired global
  admission-control lease.
- Missing, expired, malformed, `throttle`, or `fail_closed`
  `billing_admission_controls` rows reject new lease capacity with no ledger
  effect.
- Ambiguous timeout retry/readback uses the same command identity and does not
  mint duplicate lease capacity.
- Lease issue/replenish never returns `not_ready` as paid-admission success.
- Operation/lease readback can return `not_ready` only for durably accepted
  async terminal or close outcomes.

### Runtime Event Schema Contracts

- Protobuf schemas under `api/proto/events/v1/` lint and generate derived Go
  DTOs without drift.
- Compatibility checks reject breaking changes to event identity, amount
  meaning, finality, required proof, producer authenticity, and replay
  semantics.
- Generated DTOs are used only by Redpanda adapters; app-level tests use
  app-owned command/fact types.
- Golden fixtures are synthetic and privacy-safe.

### Proxy Durable Lease Allocation

Planning must include cross-repo proof in `gonka-proxy`:

- lease grant is persisted before any child debit allocation;
- child debit allocation atomically decrements local remaining authority and
  creates terminal submission obligation;
- multiple proxy processes for one allocator owner coordinate through durable
  row-lock/CAS, not process memory;
- stale generation/fence cannot allocate new child debits;
- duplicate child debit with changed fingerprint is rejected locally and later
  by billing if observed;
- proxy restart rebuilds active lease/debit state from durable rows;
- proxy durable store outage fails paid admission before external execution;
- exhausted/expired/revoked/stale-fenced lease fails closed or requests
  replenishment; it does not call direct per-request reserve;
- request teardown/stream abort after external effect records finalize,
  write-off, or ambiguous terminal fact safely.

### Redpanda Inbox/Outbox Replay

- Duplicate terminal event same fingerprint returns stored inbox/business
  outcome.
- Same event ID changed fingerprint conflicts without money mutation.
- Same child debit/usage terminal replay returns stored terminal outcome.
- Completion/write-off terminal conflict opens reconciliation and does not
  double-mutate.
- Lease checkpoint same sequence/fingerprint replays; changed fingerprint
  conflicts.
- Close checkpoint with incomplete proof keeps disputed capacity reserved and
  opens reconciliation.
- Poison event with malformed or missing event ID stores broker-coordinate
  receipt and can commit offset without raw payload.
- Unsupported schema version quarantines safely.
- Crash after DB commit before offset commit redelivers and produces no
  duplicate money effect.
- DB timeout before durable outcome does not commit offset.
- `retry_scheduled` committed-offset rows are recovered by inbox retry worker,
  including stale-claim reclamation.
- Outbox relay duplicate publish uses same event ID/fingerprint and downstream
  dedupe identity.
- Redpanda outage leaves outbox rows pending and money truth committed.

### Lease Lifecycle, Expiry, And Reconciliation

- Stale active lease opens or updates one deduped reconciliation case.
- Stale child debit after terminal deadline opens or updates one deduped
  reconciliation case.
- Missing terminal event after lag/expiry does not silently release or charge.
- Expired leases reject new child debits in proxy and release only through valid
  close proof, reconciliation, or operator repair.
- Terminal event after lease expiry still settles against original valid
  lease/debit authority.
- Terminal lag critical breach, stale lease budget breach, or stale child debit
  budget breach writes `billing_admission_controls` to `throttle` or
  `fail_closed`.
- Stopped worker, stale control lease, failed checkpoint processor, or failed
  stale-debit scanner causes new lease capacity to fail closed.
- Recovery renews `open` admission only after lag/backlog clear and
  reconciliation eligibility is within budget.
- Missing lease/debit lineage opens ambiguous reconciliation and does not
  charge customer money beyond verified authority.
- Missing qualified inference evidence opens `missing_inference_evidence`.
- Operator/admin redrive requires authority and durable audit.
- Money-changing repair re-enters idempotent billing operation and writes
  explicit ledger effect.

### Auth, Privacy, And Observability

- Startup/readiness fails closed when protected-route auth config is missing.
- Event producer authority mismatch is rejected/quarantined.
- Lease/debit artifacts are not bearer spend tokens in logs, APIs, events, or
  support exports.
- Logs/traces/metrics/inbox/outbox/audit/reconciliation rows do not contain raw
  prompts, completions, SSE chunks, bearer tokens, API keys, DSNs, payment
  secrets, raw event payloads, raw webhooks, dynamic proof URLs, or full
  provider payloads.
- Access logs do not leak account, lease, debit, or operation identifiers
  through raw path for new protected money routes.
- Metrics use only low-cardinality labels; no account IDs, API keys, request
  IDs, event IDs, inference IDs, lease IDs, debit authorization IDs, payment
  evidence IDs, or provider IDs.

### Performance And Reliability

Initial budgets from `spec.md`:

- billing lease issuance database transaction p95 under 100 ms and p99 under
  250 ms in planned benchmark workload;
- amortized proxy added latency for paid admission from an active lease p95
  under 10 ms and p99 under 25 ms, excluding external execution and cold
  replenishment;
- cold lease replenishment added latency p95 under 250 ms and p99 under 500 ms;
- terminal event processing database transaction p95 under 100 ms and p99 under
  250 ms excluding intentional same-account contention;
- terminal event lag warning and critical thresholds from
  `contracts/redpanda-events.md`;
- stale lease/debit reconciliation eligibility no later than 5 minutes after
  lease expiry, child terminal deadline, or critical lag breach;
- admission-control lease renewal interval and expiry are short enough to fail
  closed before stale exposure budgets are missed.

Benchmarks must cover:

- uncontended lease issue/replenish;
- same-account lease issuance contention;
- active-lease child debit allocation in proxy durable store;
- duplicate replay;
- lease timeout/readback;
- terminal event processing;
- terminal duplicate/conflict;
- broker lag;
- lease checkpoint/close;
- stale lease/debit recovery;
- admission-control closed/throttle/recovery behavior;
- proxy first-token or upstream-start impact.

Loosening budgets requires benchmark evidence and specification or
technical-design-review reconciliation.

### Rollout Validation

- Legacy/proxy balance import or mapping produces explicit USD ledger state.
- Shadow readback/parity passes before enabling billing writer cohorts.
- Migrated cohorts use billing lease issue/replenish and proxy durable child
  debit allocation.
- Proxy-local money writes are disabled for migrated cohorts.
- Direct per-request reserve fallback is disabled for migrated cohorts.
- Bridge paths are disabled/removed by default or proven unused.
- Rollback/failback does not silently reactivate proxy-local balance mutation
  or per-request reserve for migrated money scopes.

## Planned Command Families

Exact commands belong in `tasks.md`, but planning should draw from
repository-owned entrypoints:

- OpenAPI generation/check commands from
  `docs/build-test-and-development-commands.md`;
- SQLC generation/check commands;
- proto lint/generate/drift/compatibility commands;
- migration validation;
- Go unit and integration tests;
- race tests for concurrency-sensitive packages;
- targeted benchmarks for lease issuance, active child allocation, and worker
  event processing;
- cross-repo proxy proof commands for durable lease allocation and terminal
  submission.

## Exit Criteria

Planning may mark validation ready only when every proof obligation above is
represented in `tasks.md` or explicitly waived by technical design review as not
applicable to the accepted scope.

If planning cannot map a proof obligation to executable tasks without choosing
new architecture, reopen technical design. If a proof requires direct
per-request reserve fallback, Redpanda reserve commands, payment/top-up scope,
changed account/pricing/API-key authority, or weaker outage/privacy policy,
reopen specification.
