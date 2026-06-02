# billing-money-core Workflow Plan

Mode: full orchestrated
Status: follow-up technical design review complete with PASS for current sign-up-bonus and usage-only scope
Current phase: follow-up technical design review
Phase status: complete with PASS
Next phase: planning for current sign-up-bonus and usage-only money-core ledger
Owner: orchestrator

## Objective

Design the production-ready core of `billing-service` so it resolves the `gonka-proxy` money-math-audit problem in the new microservice: USD customer ledger, exact fixed-scale money math, fast reserve/finalize/write-off paths, durable idempotency, reconciliation, and test-first proof for customer-money correctness.

## Why Full Orchestrated

This work touches protected domains:

- money and billing correctness;
- persisted ledger data and migrations;
- cross-service contracts with `gonka-proxy`, `pricing-service`, and `payments-service`;
- retries, reconciliation, idempotency, and ambiguous external effects;
- performance-sensitive request-path gates;
- rollout and cutover from proxy-local money writes.

## Artifact State

| Artifact | State | Notes |
| --- | --- | --- |
| `docs/PRD.md` | existing | Product responsibility baseline. |
| `docs/critical-billing-context.md` | created | Repo-level critical context transferred from `gonka-proxy`. |
| `research/money-math-and-settlement-context.md` | created | Phase-local evidence and synthesis. |
| `spec.md` | approved for current sign-up-bonus and usage-only scope | User clarification supersedes active payment/top-up planning: users cannot currently add balance; the only current balance increase is the `$10.00` sign-up bonus. Payment/top-up evidence and Redpanda payment ingestion are future/conditional. |
| `workflow-plans/specification.md` | complete for current funding-source correction | Records the user-provided product fact, scoped-down clarification rationale, approved spec status, and handoff to technical-design repair. |
| `workflow-plans/technical-design.md` | complete for current repair | Records the sign-up bonus grant data-model repair, payment/top-up deferral, and handoff to follow-up technical design review. |
| `design/data-model.md` | repaired and reviewed with PASS for current planning | Adds current sign-up bonus grant identity, ledger effect, idempotency, constraints/indexes, support readback, reconciliation, and tests. Top-up/payment evidence remains dormant future context. |
| `design/event-ingestion-redpanda.md` | historical/future-conditional for current planning | Current planning must not consume payment-evidence Redpanda ingestion, inbox/outbox deltas, or runtime event schemas until async event ingestion or payment/top-up scope is explicitly reopened. |
| `design/contracts/` | conditional later | HTTP/OpenAPI contracts remain out of scope. Payment/top-up contracts are future/conditional and must be reopened before design or planning. |
| `design/dependency-graph.md` | not expected in this pass | No package/module dependency design is in scope for this specification correction. |
| `test-plan.md` | not expected as separate artifact in this pass | Current sign-up bonus and usage test obligations live in `design/data-model.md`; executable task planning is later. |
| `rollout.md` | conditional later | Runtime rollout is not part of this specification correction. |
| broader `design/` | blocked/out of scope for this pass | Public API contracts, service architecture, runtime adapters, and worker implementation remain outside this specification phase. |
| `workflow-plans/technical-design-review.md` | current follow-up review complete with PASS | Contains historical data-model and Redpanda review records plus the current sign-up-bonus/usage-only follow-up PASS. Planning is permitted to start for the current repaired scope. |
| `tasks.md` | approved for prior data-model slice only; planning required for current scope | Existing ledger must not be broadened or executed for sign-up bonus/current-scope changes until planning repairs or replaces it and task-ledger review/readiness passes. |

## Research Decisions Captured

- Customer ledger target is USD, not GNK/ngonka.
- `balanceNgonka` and `lockedRateUsd` are migration or display compatibility only.
- GNK inventory is a treasury asset, not per-customer balance backing.
- Reserve and finalize should be USD-based and bound to pricing snapshot lineage.
- Money idempotency must be database-backed with stored outcomes and conflict fingerprints.
- Redis and in-process maps are not correctness stores.
- `request_id` is correlation only; current settlement uses `usageOperationId`,
  `signupBonusGrantId`, `settlementEffectId`, and qualified `inferenceId`.
  `(paymentEvidenceId, evidenceVersion)` is future/conditional payment context.
- The first implementation must be test-heavy and benchmarked; tests are not a cleanup phase.

## Specification Decision Closure

Closed for the data-model slice:

1. USD amount representation: fixed-scale `usd_atoms`.
2. Canonical account scope and account lifecycle: service-owned account scopes with user-backed day-one rows and organization-ready schema.
3. Ledger and balance source of truth: append-only ledger entries plus transactionally maintained balance rows.
4. Reserve/finalize/write-off state model: usage operations plus durable holds and terminal outcome uniqueness.
5. Idempotency key namespace and fingerprint rules: account/kind/key scoped durable records with stored outcomes and changed-fingerprint conflict.
6. Current funding-source boundary: one `$10.00` sign-up bonus grant is the only
   current balance-increase path; payment/top-up evidence is future/conditional.
7. Reconciliation data scope: durable cases for stale reservations, ambiguous
   terminal states, sign-up bonus conflict, missing inference evidence, legacy
   import mismatch, and operator adjustment.
8. Data-model performance posture: account-row lock, O(1) grant/usage operation
   lookups, no cross-service calls in DB transactions.
9. Test obligations: parser/formatter, rounding, conservation, non-negative
   balance, sign-up bonus idempotency, usage idempotency, concurrency, Postgres
   constraints/locks, reconciliation, and benchmarks.
10. Historical Redpanda normalized payment evidence identity/versioning addendum:
    payments-service `paymentEvidenceId` is a stable evidence lineage, while
    `(paymentEvidenceId, evidenceVersion)` is billing's evidence application,
    inbox lineage, idempotency, and replay selector for payments-service
    handoff. A new evidence version is a new immutable evidence claim for
    evaluation; it cannot silently rewrite a prior ledger effect.
11. Current funding-source correction:
    customer payment/top-up is not implemented now, users cannot currently add
    balance themselves, and the only current balance-increase path is the
    `$10.00` sign-up bonus credited at registration/account admission. Payment
    evidence and Redpanda payment ingestion are future/conditional, not current
    planning input.

Still out of scope for this data-model slice:

- Internal service-to-service auth and private ingress.
- HTTP/OpenAPI route contracts and payload schemas.
- Worker runtime and full service architecture.

## Historical Data-Model Technical Design Closure

Closed for the earlier technical-design slice. This historical closure is
superseded for current planning where it treats top-up/payment evidence as
active scope; consume the current sign-up-bonus repair section below for active
planning:

1. Concrete PostgreSQL table set for accounts, balances, ledger entries,
   idempotency records, operation outcomes, usage operations, holds, terminal
   outcomes, qualified inference evidence, top-ups, payment attempts, normalized
   payment evidence, reconciliation cases, audit events, and legacy imports.
2. Concrete column types, including `BIGINT` USD atom amount columns and
   constrained `TEXT` state/kind columns.
3. Database constraints and indexes for account lookup, non-negative balance,
   ledger delta shape, idempotency replay/conflict, usage terminal uniqueness,
   payment evidence dedupe, reconciliation case dedupe, stale reservation scans,
   and support readbacks.
4. Account-first row-locking rules and one short transaction per money command.
5. Hot-path query shapes for reserve, finalize, write-off, top-up evidence, and
   strict balance readback.
6. Support/reconciliation readback surfaces and keyset pagination expectations.
7. Legacy import compatibility surfaces that preserve `balanceNgonka` and
   `lockedRateUsd` as evidence only.
8. Data-model test obligations for money representation, constraints, ledger
   conservation, idempotency, concurrency, reconciliation/import, and
   performance evidence.

Technical design review initially decided `FAIL` in
`workflow-plans/technical-design-review.md`. The targeted technical-design
repair pass updated `design/data-model.md` for the named findings:

1. TDR-F01: reconciliation duplicate-open-case dedupe now maps each approved
   reason to concrete lineage keys, including usage, top-up/payment-attempt,
   payment evidence, settlement effect, qualified inference evidence, ledger,
   and legacy import row lineage.
2. TDR-F02: enum-like constrained text coverage now includes an explicit
   inventory plus concrete checks for the previously missing
   `payment_attempts.state`, `operation_outcomes.primary_resource_type`,
   `operation_outcomes.operation_kind`, `idempotency_records.retention_class`,
   `ledger_entries.created_by_kind`, and adjacent closed state/kind/class
   fields.
3. TDR-F03: ledger delta-pattern rules now explicitly zero every balance
   component an effect type is not allowed to mutate, including
   `usage_charge` setting `pending_delta_usd_atoms = 0`.

The follow-up technical-design review in
`workflow-plans/technical-design-review.md` records `PASS` for the repaired
packet. Planning may start for the approved data-model slice.

## Prior Data-Model Planning Completion

Prior phase: planning.
Prior phase status: complete with task-ledger review PASS for the original data-model-only slice.
Created ledger: `specs/billing-money-core/tasks.md`.
Task-ledger review: PASS.
Implementation readiness: PASS.
Accepted concerns from technical design review: none.
Required design artifact: `design/data-model.md` is reviewed with follow-up PASS after targeted TDR-F01/TDR-F02/TDR-F03 repair.
Conditional artifacts: `design/contracts/`, `design/dependency-graph.md`, `test-plan.md`, and `rollout.md` remain not expected for this data-model-only implementation slice for the reasons recorded above.
Blockers: none for implementation.
Reopen route: none now. Reopen `technical design` if implementation needs unapproved schema/constraint/index/dedupe/lock/query semantics; reopen `specification` if implementation needs a source-of-truth, unit, settlement-identity, idempotency, privacy, or slice-boundary change; reopen `planning` if task order or proof needs repair without changing approved decisions.
Prior session boundary reached: yes.
Prior ready-for-next-session marker: yes.
Prior next session was: implementation from approved `tasks.md`.
Prior next-session context bundle: read `tasks.md` first, then the artifacts named by its Goal Contract and Implementation Handoff.

That prior implementation handoff applied only to the approved data-model ledger and named proof. The current Redpanda addendum supersedes the active next-session route for event-ingestion work; do not execute event-ingestion inbox/outbox work from the prior data-model-only ledger.

## Historical Redpanda Event-Ingestion Technical Design Addendum

Historical phase: follow-up technical design review after REDPANDA-TDR-F01/F02/F03
repair.
Phase status: complete with PASS.
Created design artifact: `specs/billing-money-core/design/event-ingestion-redpanda.md`.
Design status at that time: repaired and accepted for planning.
Technical design review status: prior review complete with FAIL; follow-up
review complete with PASS.
Planning status at that time: permitted to start.
Current status: superseded for current planning by the sign-up-bonus/usage-only
specification correction.

Scope completed in this addendum:
- Redpanda as Kafka-compatible transport/event backbone, not the billing source
  of truth.
- Billing Postgres ledger remains authoritative for money effects.
- Broker-level exactly-once is explicitly not the billing correctness guarantee.
- First consumed event classes: usage terminal completion, usage
  failure/timeout/write-off, normalized payment evidence, and optional
  reconciliation/admin repair signals.
- First emitted event classes: committed ledger effects, operation outcomes,
  reconciliation-required signals, and rejected/conflict signals.
- Concrete topic names, partition keys, consumer groups, lag semantics, inbox
  idempotency, event transaction order, failure handling, reconciliation,
  performance posture, observability, tests, and data-model delta.
- Version-scoped normalized payment evidence selector
  `(paymentEvidenceId, evidenceVersion)` in inbox lineage, event operation
  identity, business idempotency, replay/conflict rules, reconciliation, support
  readback, tests, and later contract consequences.
- Durable poison/quarantine identity by broker receipt coordinates when producer
  `eventId` or semantic identity is malformed or missing.
- Committed-offset `retry_scheduled` recovery owned by a billing inbox retry
  worker with lease/generation lifecycle.

Data-model delta introduced by the addendum:
- new `billing_event_inbox` table;
- new `billing_event_outbox` table;
- repaired `payment_evidence_lineages` / `payment_evidence` versioned model and
  versioned evidence references where a row points to one normalized evidence
  claim;
- optional event lineage links from `idempotency_records`,
  `operation_outcomes`, and `reconciliation_cases`.

Writes intentionally not performed:
- no code;
- no migrations;
- no generated SQL;
- no runtime adapter or worker implementation;
- no event contract schemas;
- no `tasks.md` changes.

Blockers:
- none remaining inside technical design repair.

Review blockers closed by follow-up technical design review:
- REDPANDA-TDR-F01: design carries `(paymentEvidenceId, evidenceVersion)` through
  event ingestion and affected data-model context.
- REDPANDA-TDR-F02: design defines durable quarantine identity for malformed or
  missing producer/semantic identity using
  `event_receipt_identity = offset:<topic>:<partition>:<offset>`.
- REDPANDA-TDR-F03: design defines the billing inbox retry worker as owner for
  committed-offset `retry_scheduled` rows and records claim/backoff/escalation
  lifecycle.

Reopen route:
- Historical route was planning for event-ingestion work. Current route is
  technical design repair for sign-up bonus grants and payment/top-up deferral.

## Historical Specification Reopen: Redpanda Evidence Versioning

Phase: specification reopen.
Phase status: complete.
Repair target: REDPANDA-TDR-F01.
Phase-local record: `specs/billing-money-core/workflow-plans/specification.md`.

Decision:
- Billing accepts payments-service's versioned normalized evidence handoff.
- `paymentEvidenceId` remains the stable payments-owned normalized-evidence
  lineage and must not rebind across top-up, attempt, or account scope.
- `evidenceVersion` is required for payments-service evidence handoff and forms
  the billing application selector together with `paymentEvidenceId`.
- `(paymentEvidenceId, evidenceVersion)` is the selector that technical design
  must carry through `payment_evidence`, `billing_event_inbox`,
  event-originated business idempotency, replay/conflict handling, and support
  readback.
- A new evidence version under the same evidence lineage is a new immutable
  evidence claim for evaluation; it cannot silently rewrite an existing ledger
  effect. Money-changing correction must be an explicit reversal/refund/
  adjustment effect or reconciliation/manual-review path.

Provider-contract evidence:
- payments-service data-model planning states that normalized payment evidence
  stores stable `paymentEvidenceId` plus monotonic `evidenceVersion`, with new
  versions only when normalized economic meaning changes under the same lineage;
- payments-service billing writeback selectors for evidence include
  `paymentEvidenceId` plus `evidenceVersion`, not the bare stable lineage id;
- the current `gonka-proxy` internal-money billing v1 `topupApplyEvidence`
  schema lacks `evidenceVersion`, so later contract repair must add or otherwise
  encode the version-scoped selector instead of forcing payments-service to mint
  new evidence IDs per versioned meaning.

Clarification gate:
- scoped-down local specification clarification complete;
- lens: API/data/source-of-truth consequence of the REDPANDA-TDR-F01 payment
  evidence selector mismatch;
- scoped-down rationale: the prior technical design review already isolated one
  approval-critical specification question, and provider-contract evidence was
  sufficient to decide it without reopening broader scope;
- result: approve the spec addendum with technical-design obligations, not a
  user decision or research reopen.

Specification reopen session boundary: reached.
Specification reopen next session was: technical design repair for Redpanda
event ingestion.

Session boundary reached: yes.
Ready for next session: yes.
Historical next session at that time: follow-up technical design review for the
repaired Redpanda event-ingestion addendum.

## Technical Design Repair Completion

Phase: technical design repair.
Phase status: complete.
Phase-local record: `specs/billing-money-core/workflow-plans/technical-design.md`.

Repair summary:
- `design/event-ingestion-redpanda.md` now carries the approved
  `(paymentEvidenceId, evidenceVersion)` selector through inbox lineage, event
  operation identity, business idempotency, replay/conflict handling,
  reconciliation, tests, and later contract consequences.
- `design/data-model.md` now distinguishes `payment_evidence_id` stable lineage
  from versioned `payment_evidence` rows, adds the versioned selector to
  ledger/reconciliation/audit/idempotency/support/test design, and preserves
  lineage readback by `payment_evidence_id`.
- REDPANDA-TDR-F02 is repaired by using broker receipt coordinates and
  `event_receipt_identity` for poison/quarantined events without valid producer
  event ID or semantic identity.
- REDPANDA-TDR-F03 is repaired by assigning committed-offset
  `retry_scheduled` rows to a billing inbox retry worker with claim,
  lease/generation, backoff, escalation, shutdown, and stale-claim recovery
  lifecycle.

Follow-up technical design review was complete with PASS for the historical
Redpanda payment-evidence packet. That PASS is superseded for current planning by
the sign-up-bonus/usage-only specification correction.

## Follow-Up Technical Design Review Completion

Phase: follow-up technical design review.
Phase status: complete.
Phase-local record:
`specs/billing-money-core/workflow-plans/technical-design-review.md`.

Gate status: PASS.
Accepted concerns: none.
Reopen target: none.

Review closure:
- REDPANDA-TDR-F01 is closed: the reviewed packet carries
  `(paymentEvidenceId, evidenceVersion)` through spec, data-model, event inbox,
  business idempotency, replay/conflict, reconciliation, support readback, tests,
  and later contract consequences.
- REDPANDA-TDR-F02 is closed: poison/quarantine receipts whose producer event ID
  or semantic identity is malformed have durable broker-coordinate identity and
  offset-commit/redrive rules.
- REDPANDA-TDR-F03 is closed: committed-offset `retry_scheduled` rows are owned
  by the billing inbox retry worker with claim, lease/generation, backoff,
  escalation, shutdown, and stale-claim recovery rules.
- Adjacent assumptions remain planning-ready: Postgres remains money truth,
  Redpanda remains transport, outbox owns emitted events, exact event schemas and
  runtime packaging remain later-phase work with fixed constraints, and the
  existing data-model-only ledger is not an event-ingestion implementation
  handoff.

Historical next route at that time: planning for the Redpanda event-ingestion
addendum. This route is superseded by the current funding-source correction
below.

## Specification Reopen: Current Funding-Source Correction

Phase: specification reopen.
Phase status: complete.
Phase-local record: `specs/billing-money-core/workflow-plans/specification.md`.

User-provided product fact:
- customer payment/top-up is not implemented now;
- users cannot currently add balance themselves;
- the only current balance-increase path is the `$10.00` sign-up bonus credited
  at registration/account admission;
- after that, balance changes through usage reserve/finalize/write-off and
  explicit correction/reversal paths.

Spec decision:
- Current active billing scope is sign-up bonus grant credit plus usage money
  decrement/release behavior.
- Sign-up bonus credit is a first-class ledger effect with durable grant
  identity, idempotency, support readback, reconciliation, and tests.
- Customer payment/top-up product flow, payment provider sessions,
  payments-service evidence handoff, payment presentation sync, normalized
  payment evidence application, and Redpanda payment-evidence ingestion are
  future/conditional and must not be planned from the current spec.
- The prior Redpanda payment-evidence addendum and follow-up PASS are historical
  and superseded for current planning until payment/top-up scope is explicitly
  reopened.

Clarification gate:
- formal trigger: protected money/data/domain scope;
- gate shape: scoped-down local clarification;
- scoped-down rationale: the approval-changing fact is a direct product boundary
  correction supplied by the user, so no additional research lane is needed to
  decide whether payment/top-up is active now;
- result: approved with technical-design repair required.

Historical next route:
- Reopen `technical design` next.
- Repair `design/data-model.md` and affected event-ingestion/design context for
  current sign-up-bonus and usage-only scope.
- Do not create `tasks.md`, code, migrations, generated SQL, runtime adapters,
  workers, or runtime event schemas in the technical-design repair session.

That technical-design repair is now complete.

## Technical Design Repair: Current Sign-Up Bonus And Usage-Only Scope

Phase: technical design.
Phase status: complete.
Phase-local record: `specs/billing-money-core/workflow-plans/technical-design.md`.

Design repair summary:
- `design/data-model.md` now defines `signup_bonus_grants` as the current
  one-time balance-increase model.
- `signup_bonus_grant_id` is the billing-owned grant identity, with safe
  registration/account-admission lineage via `admission_authority` and
  `admission_reference_id`.
- `signup_bonus_policy_version` and uniqueness by account, subject, and
  admission lineage prevent duplicate grants for the same approved policy.
- The current grant amount is fixed at `1,000,000,000 usd_atoms` (`$10.00`);
  future amount changes require specification reopen and compatible schema
  evolution.
- `signup_bonus_credit` is the explicit ledger effect; the bonus is not a
  top-up, payment, operator adjustment, or migration import.
- `operation_kind = 'signup_bonus_grant'` owns durable idempotency, replay,
  changed-fingerprint conflict, stored outcome, and support readback.
- `signup_bonus_conflict` is the current reconciliation reason for duplicate or
  changed-fingerprint grant ambiguity.
- Current data-model tests now cover exact bonus atoms, grant uniqueness,
  idempotency replay/conflict, concurrent duplicate delivery, reconciliation
  dedupe, ledger conservation, and O(1) grant lookup.
- `design/event-ingestion-redpanda.md` is now historical/future-conditional for
  current planning; Redpanda payment-evidence ingestion, inbox/outbox deltas,
  payments-service writeback, and runtime event schemas require an explicit
  future reopen.

Blockers:
- none inside technical design repair.

Follow-up technical design review is now complete with PASS.

## Follow-Up Technical Design Review: Current Sign-Up Bonus And Usage-Only Scope

Phase: follow-up technical design review.
Phase status: complete with PASS.
Phase-local record:
`specs/billing-money-core/workflow-plans/technical-design-review.md`.

Gate status: PASS.
Accepted concerns: none.
Reopen target: none.

Review closure:
- The sign-up bonus grant model is planning-ready: it has stable
  `signup_bonus_grant_id`, safe account-admission lineage, policy version,
  fixed `$10.00` / `1,000,000,000 usd_atoms` amount, explicit
  `signup_bonus_credit` ledger effect, durable `signup_bonus_grant`
  idempotency, conflict reconciliation, support readback, and test obligations.
- Usage reserve/finalize/write-off remains current and independent of
  payment/top-up activation: the repaired data model preserves durable holds,
  terminal outcomes, account-first locking, O(1) hot paths, and no
  cross-service calls inside money transactions.
- Payment/top-up evidence, payments-service writeback, Redpanda
  payment-evidence ingestion, inbox/outbox event-ingestion deltas, and runtime
  event schemas are future/conditional and not current planning input.
- The existing `tasks.md` is a prior-slice ledger only. It must be repaired or
  replaced during planning before current sign-up-bonus/usage-only
  implementation can start.

Planning may start for the current sign-up-bonus and usage-only money-core
ledger. Implementation, migrations, generated SQL, runtime adapters/workers,
payment/top-up work, and event contract work remain blocked until planning
produces a reviewed task ledger with task-ledger review/readiness `PASS` or
eligible `CONCERNS`.

Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: planning for current sign-up-bonus and usage-only
money-core ledger.
Next session context bundle:
- default resume order plus `workflow-plans/technical-design-review.md` for the
  current PASS;
- `design/data-model.md` for active sign-up bonus and usage data-model planning;
- `design/event-ingestion-redpanda.md` only to keep payment-evidence ingestion
  and runtime event schemas out of current planning.

## Recommended Next-Session Prompt

```text
Work in `/Users/daniil/Projects/GonkaGate/billing-service`.

Next phase: planning for the current sign-up-bonus and usage-only money-core ledger.

Read first:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `specs/billing-money-core/workflow-plan.md`
- `specs/billing-money-core/workflow-plans/specification.md`
- `specs/billing-money-core/workflow-plans/technical-design.md`
- `specs/billing-money-core/workflow-plans/technical-design-review.md`
- `specs/billing-money-core/spec.md`
- `specs/billing-money-core/design/data-model.md`
- `specs/billing-money-core/design/event-ingestion-redpanda.md`
- `specs/billing-money-core/tasks.md` if present, only to determine whether the
  prior ledger should be repaired or replaced for current scope
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`

Goal:
Create or repair the planning ledger from the approved current sign-up-bonus and
usage-only specification/design packet and the follow-up technical-design-review
PASS. The ledger must cover sign-up bonus grant schema/query/test work, current
usage reserve/finalize/write-off preservation, reconciliation/support readback,
and the named proof obligations without planning payment/top-up evidence,
payments-service writeback, Redpanda payment-evidence ingestion, runtime event
schemas, or runtime adapter/worker work.

Expected output:
- update `specs/billing-money-core/tasks.md` or the appropriate planning
  artifact so current sign-up-bonus and usage-only work is executable without
  hidden design choices;
- run and record the post-ledger task-review/readiness gate as `PASS`,
  eligible `CONCERNS`, or `FAIL`;
- update `specs/billing-money-core/workflow-plan.md` with the planning outcome;
- produce the next recommended prompt.

Stop rule:
Complete planning and task-ledger review/readiness only, then stop with updated
workflow state and the next recommended prompt. Do not write code, create
migrations, generate SQL, implement runtime adapters or workers, design runtime
event contract schemas, or reopen payment/top-up scope in this session unless
the planning review verdict requires reopening an earlier phase.
```
