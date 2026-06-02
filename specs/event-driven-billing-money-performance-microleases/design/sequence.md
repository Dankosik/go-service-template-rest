# Sequence And Failure Design

Status: review-ready
Consumes: `overview.md` and `../spec.md`

## S1. Microlease Issue Or Replenish

1. Proxy resolves caller attribution and account scope through existing identity
   and API-key surfaces.
2. Proxy obtains pricing snapshot evidence for the intended use class from
   pricing-service and computes `max_child_cap_usd_atoms`.
3. Proxy checks its durable grant state. If remaining capacity is above the
   refill threshold and the grant is before debit cutoff, it does not call
   billing.
4. If refill is needed, proxy calls protected billing HTTP with one stable
   idempotency key, request fingerprint, account scope, proxy owner, requested
   cap formula inputs, pricing snapshot evidence, use class, deadline, and
   trace metadata.
5. Billing validates protected caller identity, deadline, schema version,
   account state, pricing freshness/use-class compatibility, admission controls,
   active exposure cap, safety floor, and idempotency.
6. Billing opens one short PostgreSQL transaction:
   - locks the account balance row;
   - locks or creates idempotency state;
   - denies if available balance cannot reserve the full microlease exposure;
   - inserts or updates the microlease grant;
   - writes the reserve ledger effect;
   - updates `reserved_usd_atoms` and derived `available_usd_atoms`;
   - stores the replay outcome and outbox fact.
7. Billing returns the stored outcome. A timeout after possible acceptance is
   ambiguous; proxy retries or reads back with the same operation identity.
8. Proxy persists the grant before any child debit uses it.

Failure behavior:

- No billing commit means no spend authority.
- Missing or stale pricing evidence denies issuance.
- Missing or stale admission controls deny issuance.
- Insufficient available balance denies issuance.
- Same idempotency key plus changed fingerprint returns conflict.
- Billing or network outage does not allow direct per-request reserve fallback.

## S2. Active Paid Admission

1. Proxy receives a paid request and maps it to an account, pricing basis, and
   expected max child cap.
2. Optional process memory precheck denies immediately when cached remaining
   capacity, cutoff, local backlog, or strict-mode state is unhealthy.
3. Memory precheck success is only a hint. Proxy still starts a durable local
   allocation transaction.
4. Proxy durable allocator validates owner, generation/fence, expiry, debit
   cutoff, remaining capacity, request fingerprint, pricing snapshot, account
   binding, and local terminal backlog gates.
5. Proxy durable allocator atomically:
   - inserts one `debitAuthorizationId`;
   - subtracts child cap from durable remaining capacity;
   - stores request and pricing fingerprints;
   - creates a durable terminal obligation with deadline;
   - records safe correlation fields.
6. External paid execution may start only after that durable transaction
   commits.
7. If durable allocation fails, the request fails closed or tries to replenish
   before execution. It must not execute from memory, Redis, or cached balance.

Failure behavior:

- Proxy durable store unavailable: fail closed before execution.
- Stale fence or duplicate child ID: fail closed and open local repair.
- Memory token consumed but durable allocation fails: restore/drop cache state;
  no external execution occurred.
- Low balance, stale pricing, high variance, abuse/manual review, terminal lag,
  stale child age, or reconciliation backlog breach: strict/fail-closed.

## S3. Strict Mode

Strict mode uses the same authority model:

- bypass memory prechecks or treat them as deny-only;
- use smaller requested caps and shorter TTLs when billing issuance is allowed;
- require durable child debit allocation under a valid microlease before
  execution;
- deny new microlease capacity when pricing, account, allocator, lag, backlog,
  or manual-review gates are unhealthy;
- fail closed when no valid durable microlease capacity exists.

Strict mode does not call direct per-request billing reserve for migrated paid
cohorts. If that becomes required, specification reopens.

## S4. Terminal Settlement

1. After external execution reaches a terminal state, proxy writes terminal
   evidence against the durable child debit.
2. The terminal row includes terminal kind, child cap, realized charge,
   write-off/release amount, terminal fingerprint, pricing snapshot identity,
   safe execution references, and qualified inference evidence when available.
3. Proxy outbox publishes the terminal fact to Redpanda with stable event ID and
   business identity.
4. Billing consumer reads the event, validates producer authenticity, and
   records inbox/idempotency before committing offset.
5. Billing transaction validates parent microlease, account, owner/fence, child
   identity, fingerprint, cap, terminal finality, and duplicate/conflict state.
6. Billing posts ledger effects no greater than the child cap and no greater
   than aggregate parent authority, updates child terminal projection, updates
   microlease settlement totals, stores outcome, and writes outbox facts.
7. Billing commits offset only after durable DB effect or durable quarantine.

Failure behavior:

- Duplicate event with same fingerprint replays stored outcome.
- Duplicate event with changed fingerprint conflicts and opens reconciliation.
- Realized cost over child cap charges at most child cap and records explicit
  write-off/compensation/reconciliation.
- Aggregate over parent cap charges at most parent cap and opens
  over-debit reconciliation.
- Missing parent/child lineage quarantines or opens ambiguous-terminal
  reconciliation without money mutation.
- Billing consumer lag keeps exposure reserved and blocks replenishment through
  admission controls.

## S5. Checkpoint And Close

Proxy emits checkpoint/close facts when capacity is near cutoff, at expiry,
after exhaustion, during shutdown, and on operator repair.

Required close proof:

- account scope;
- microlease ID;
- owner and generation/fence;
- checkpoint or close sequence;
- allocated child high-water mark;
- allocated child count;
- allocated child cap sum;
- terminal submitted count;
- terminal published count;
- terminal accepted count when known;
- unresolved child count and cap sum;
- local remaining capacity;
- checkpoint fingerprint over the allocator snapshot.

Billing release rule:

- Expiry stops new child debits but does not release money.
- Billing may release only capacity proven never allocated by a valid close
  proof from the current owner/fence.
- Capacity corresponding to unresolved allocated child caps remains reserved
  until terminal settlement or reconciliation.
- Close proof with gaps, changed fingerprint, invalid fence, or impossible cap
  sums opens reconciliation and denies release.

## S6. Expiry And Reconciliation

1. Proxy stops new debits at debit cutoff.
2. Proxy attempts close at or before expiry.
3. Billing worker scans for expired microleases, stale child debits, stale close
   proof, terminal deadline breaches, and lag critical breaches.
4. Worker opens or updates reconciliation cases within the 5 minute SLA.
5. Reconciliation applies explicit release, write-off, reversal, compensation,
   or manual-review effects from durable evidence only.

Failure behavior:

- TTL alone is never proof of unused capacity.
- Unresolved exposure stays reserved and visible.
- Operator actions use audited support-safe notes and bounded reason codes.
- No raw prompts, completions, SSE chunks, tokens, API keys, DSNs, payment
  secrets, raw event payloads, or dynamic proof URLs are stored.

## S7. Outage Matrix

| Condition | Behavior |
| --- | --- |
| Billing unavailable before issue commit | No new capacity; proxy may retry same identity or fail closed. |
| Billing timeout after possible acceptance | Ambiguous; proxy retries/readbacks same identity. No duplicate grant identity. |
| Existing valid microlease and billing unavailable | Proxy may spend only until cap/cutoff/local backlog gates; no new capacity minted. |
| Proxy durable store unavailable | Fail closed before external execution. |
| Proxy crash after durable child debit before execution | Recovery either resumes only with terminal obligation intact or closes/reconciles without customer charge. |
| Proxy crash after execution before publish | Durable terminal obligation survives and retries publish; billing exposure stays reserved. |
| Redpanda unavailable after execution | Proxy durable outbox retains terminal fact; billing exposure stays reserved. |
| Billing consumer lag | New issue/replenish denied or reduced by admission controls; active exposure remains reserved. |
| Redis unavailable | No effect in first target because Redis is absent. If later introduced, uncertainty must degrade to strict/fail-closed. |
| Memory cache lost | Rebuild from durable proxy grants/debits; no money capacity is lost or minted. |
| Pricing snapshot stale | No new issue/replenish or child debit requiring that evidence. |
