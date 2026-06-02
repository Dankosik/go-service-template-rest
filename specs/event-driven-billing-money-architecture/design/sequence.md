# Runtime Sequence And Failure Behavior

Status: repaired review-ready technical design for billing-issued spending leases
Consumes: `../spec.md`, `component-map.md`, `data-model.md`, `contracts/`

## Lease Issuance Or Replenishment

1. `gonka-proxy` authenticates the public caller and obtains API-key,
   identity, account, and policy evidence.
2. `gonka-proxy` obtains immutable USD-compatible pricing snapshot evidence
   from `pricing-service` before requesting new lease capacity.
3. The proxy selects a stable `proxyLeaseOwnerId` for its durable allocator
   shard and computes the command `idempotencyKey` and
   `operationFingerprint` from account scope, requested lease amount, pricing
   evidence, policy constraints, owner, generation request, expiry/cutoff, and
   use class.
4. The proxy calls protected billing HTTP to issue or replenish a spending
   lease. This is not per-request reserve; one command may amortize capacity
   across many child debits.
5. HTTP middleware authenticates the service principal, checks route scope,
   validates deadline/body size, attaches correlation, and delegates to
   `internal/app/money`.
6. The app service validates account binding, pricing evidence shape, USD
   compatibility, idempotency key, fingerprint, requested amount, owner, expiry
   bounds, and admission-control prerequisites. It does not call pricing,
   identity, API-key-service, Redpanda, or the worker while holding the
   transaction.
7. The Postgres transaction:
   - reads global and account-scoped `billing_admission_controls` for
     `paid_usage_admission` before reserving new capacity;
   - rejects without money mutation when controls are missing, expired, stale,
     malformed, `throttle`, or `fail_closed`;
   - resolves and locks the account balance row;
   - creates or locks the lease command idempotency record;
   - replays same fingerprint when already committed;
   - rejects changed fingerprint as conflict without money mutation;
   - creates or updates `spending_leases` with issued/replenished capacity,
     owner, generation/fence, expiry, debit cutoff, pricing/policy constraints,
     and stored outcome lineage;
   - inserts a ledger effect such as `spending_lease_hold` that increases
     reserved USD and reduces available USD;
   - writes the stored operation outcome and outbox facts;
   - commits.
8. Billing returns a durable lease outcome with `spendingLeaseId`,
   generation/fence, issued/replenished amount, remaining reserved amount,
   expiry/cutoff, stored outcome identity, and safe balance reference.
9. The proxy persists the lease grant before any paid request can allocate from
   it. If this local persist fails after billing accepted the command, the
   proxy retries/readbacks the same lease command and then closes or
   reconciles the grant; it must not mint a separate lease for the same
   requested capacity.

### Lease Command Failure Classes

| Failure | Behavior |
| --- | --- |
| Billing unavailable before lease acceptance | Proxy may use other active valid lease capacity. If none is available, paid admission fails closed. |
| PostgreSQL unavailable before lease commit | No durable lease capacity. Proxy may retry same identity inside deadline or fail closed. |
| Deadline expires after possible acceptance | Ambiguous to proxy. Proxy retries/readbacks the same command identity and must not request duplicate capacity. |
| Insufficient funds, account not spendable, unsupported currency, stale pricing, account mismatch, missing idempotency | Stored or direct business rejection. No new lease capacity. |
| Admission-control row missing, stale, expired, malformed, or `fail_closed` | New lease issuance/replenishment rejected with safe 503 class and no ledger effect. |
| Admission-control row `throttle` or account-scoped load shed | New lease issuance/replenishment rejected with safe 429 class and no ledger effect. |
| Same idempotency key and same fingerprint | Return stored lease outcome. |
| Same idempotency key and changed fingerprint | Conflict; no money mutation. |

`not_ready` is allowed only for readback of a durably accepted async outcome. It
is not a successful paid-admission result when no valid local lease capacity
exists.

## Proxy Local Child Debit Allocation

1. For each paid request, proxy authenticates the caller and obtains immutable
   pricing snapshot evidence appropriate for the child debit.
2. Proxy chooses an active lease whose account scope, owner, generation/fence,
   use class, pricing constraints, debit cutoff, and local health policy match
   the request.
3. Inside one durable proxy transaction or compare-and-swap boundary, the
   allocator:
   - verifies the lease generation/fence is current and not expired/revoked;
   - verifies local remaining authority is at least the child cap;
   - creates a unique `debitAuthorizationId` linked to the
     `spendingLeaseId`, generation/fence, `usageOperationId`, child cap,
     operation fingerprint, pricing snapshot identity/fingerprint, allocation
     sequence, terminal deadline, and safe caller lineage;
   - decrements local remaining authority by the child cap;
   - creates the terminal submission obligation for that child debit.
4. External paid execution may start only after the child debit and terminal
   submission obligation are durable.
5. If the proxy durable allocator is unavailable, if the lease is exhausted,
   stale-fenced, expired, revoked, or if terminal backlog violates local policy,
   the proxy requests/awaits lease replenishment or fails paid admission closed.
   It does not call billing for direct per-request reserve.

## Terminal Event Settlement

1. After execution completes, fails, aborts, times out, or becomes ambiguous,
   the proxy updates the durable child debit/terminal row with safe terminal
   evidence:
   - terminal kind `finalize`, `write_off`, `reversal`, or `compensation` when
     applicable;
   - terminal fingerprint;
   - final charge or write-off/release classification;
   - metered facts fingerprint or cancellation/timeout facts fingerprint;
   - qualified inference evidence references when available;
   - safe correlation identifiers.
2. Proxy relay publishes `usage.execution.terminal.v1` from the durable row and
   marks it published only after Redpanda acknowledgement.
3. `cmd/billing-worker` consumes the event under a current-scope consumer group.
4. The adapter validates topic, schema version, producer authority, event
   identity, account scope, lease/debit lineage, generation/fence,
   fingerprints, and safe fields. It does not log raw event payloads.
5. Billing inserts or locks `billing_event_inbox` before any money mutation.
6. In one Postgres transaction, billing:
   - locks the account balance row;
   - locks the parent `spending_leases` row;
   - creates or locks event-originated idempotency by `usageOperationId` and
     terminal operation kind;
   - creates or replays the child debit settlement lineage;
   - validates lease ID, owner, generation/fence, account scope,
     `debitAuthorizationId`, child cap, operation fingerprint, pricing
     fingerprint, and terminal fingerprint;
   - charges no more than the child cap and no more than aggregate valid lease
     authority;
   - reduces lease reserved exposure by the settled child cap or verified
     release amount;
   - inserts explicit ledger effects, terminal outcome, operation outcome,
     reconciliation case when required, and outbox facts;
   - updates the inbox outcome;
   - commits.
7. The worker commits the Redpanda offset only after Postgres commit succeeds.

### Terminal Event Outcomes

| Event condition | Billing outcome |
| --- | --- |
| Same `(topic, eventId)` and same fingerprint | Stored inbox/business outcome replay. |
| Same `(topic, eventId)` with changed fingerprint | Inbox conflict, rejected fact emitted, no money mutation. |
| Same `debitAuthorizationId` and same operation/terminal fingerprint | Stored child/terminal outcome replay. |
| Same child debit ID with changed fingerprint | Conflict/reconciliation, no second money mutation. |
| Finalize for valid child cap under active/expired original lease | Charge capped by child and lease authority, release unused child capacity, store terminal outcome. |
| Write-off/release for valid child cap | Release or write off under explicit ledger effect, store terminal outcome. |
| Completion after write-off or write-off after completion | Conflict/reconciliation, no second mutation. |
| Missing, stale, or invalid parent lease/debit lineage | Ambiguous-terminal reconciliation, no customer charge beyond verified authority. |
| Proxy over-debit beyond lease budget | Charge only valid child/lease authority; excess becomes write-off/compensation/reconciliation. |

Terminal settlement targets the original lease/debit authority even after lease
expiry or newer lease issuance.

## Lease Checkpoint, Close, Expiry, And Release

1. Proxy periodically emits lease checkpoint facts with owner, generation,
   allocation high-water mark, allocated child cap sum, locally remaining
   amount, terminal submitted/published counts, open child summaries, and a
   checkpoint fingerprint.
2. Proxy emits close facts when it has durable proof that unused capacity is
   safe to release. It may also call protected HTTP close/cancel for bounded
   synchronous readback or operator-controlled close.
3. Billing stores checkpoint/close receipt in the inbox and verifies parent
   lease, owner, generation/fence, monotonic high-water mark, and fingerprint.
4. Billing releases only capacity it can prove is neither already settled nor
   still open as an allocated child. If proof is incomplete, billing keeps the
   exposure reserved and opens or updates reconciliation.
5. Lease expiry or cutoff stops new child debits. It does not silently release
   capacity that may correspond to executed work.
6. Stale expired leases, missing checkpoints, stale child debits, and terminal
   lag are handled by reconciliation and admission-control backpressure.

## Inbox Retry Worker

`cmd/billing-worker` owns committed-offset retry rows in `billing_event_inbox`.

1. Claim eligible retry rows with `FOR UPDATE SKIP LOCKED`, `claim_owner`,
   `claim_generation`, and bounded `claim_deadline_at`.
2. Re-enter the same business processor using stored inbox identity, operation
   identity, event fingerprint, and safe payload digest/reference. Do not
   reconstruct behavior from raw payload.
3. Success transitions to a terminal inbox outcome.
4. Retryable failure schedules bounded backoff.
5. Exhausted or non-retryable failure transitions to `reconcile_required` or
   `quarantined` and opens or links a reconciliation case.
6. Shutdown stops new claims and either completes the in-flight transaction or
   lets claim deadlines expire.

## Billing Outbox Relay

1. Money transactions insert `billing_event_outbox` rows in the same transaction
   as source lease, ledger, outcome, reconciliation, or inbox state.
2. Outbox relay claims pending rows with `FOR UPDATE SKIP LOCKED`.
3. Relay publishes with stable `eventId`, event key, schema version, and event
   fingerprint.
4. Relay marks `published` only after broker acknowledgement.
5. Publish retry reuses the same event identity/fingerprint. Downstream
   consumers dedupe by lease, debit, ledger, outcome, or reconciliation source
   identity, not broker delivery count.

## Startup, Readiness, Shutdown, And Backpressure

`cmd/service`:

- validates config, auth configuration, network ingress policy, Postgres
  dependency, and route-scope config before readiness;
- does not report ready for protected money routes if required auth config or
  Postgres readiness is missing;
- does not call Redpanda or the worker during lease issuance.

`cmd/billing-worker`:

- validates Redpanda brokers, topic configuration, consumer groups, producer
  ACL assumptions, Postgres dependency, and worker concurrency budgets before
  worker readiness;
- renews `billing_admission_controls` only while terminal consumption, inbox
  retry, outbox relay, lease checkpoint processing, and stale lease/debit
  reconciliation are within accepted budgets;
- writes `throttle` or `fail_closed` when lag, stale leases, stale child
  debits, reconciliation backlog, or worker-health breaches threaten reserved
  exposure;
- on shutdown stops consuming/publishing new work, finishes or abandons
  in-flight claims safely, commits offsets only after durable outcomes, and
  leaves outbox/inbox rows retryable.

New lease issuance/replenishment fails closed when billing reserve authority is
unavailable or admission controls are stale. Active valid lease capacity may be
spent only through its owner/fence/cutoff/cap and only while proxy durable
allocator and terminal submission health are intact.
