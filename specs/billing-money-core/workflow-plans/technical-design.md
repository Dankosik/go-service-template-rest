# billing-money-core Technical Design Phase

Phase: technical design
Pass type: current sign-up-bonus and usage-only technical design repair
Status: complete; follow-up technical design review required
Owner: orchestrator

## Scope

This repair pass owns the current-scope correction after the approved
2026-06-01 funding-source clarification:

- customer payment/top-up is not implemented now;
- users cannot currently add balance themselves;
- the only current balance-increase path is the `$10.00` sign-up bonus credited
  at registration/account admission;
- usage reserve, finalize, write-off, reversal/compensation, reconciliation,
  legacy import, and explicit correction behavior remain current money scope;
- payment/top-up evidence, payments-service writeback, and Redpanda
  payment-evidence ingestion are future/conditional context only.

In scope:
- repair `design/data-model.md` for sign-up bonus grant identity, one-time
  credit semantics, ledger effect, idempotency, constraints/indexes, support
  readback, reconciliation, and tests;
- preserve usage reserve/finalize/write-off design without depending on
  payment/top-up activation;
- repair `design/event-ingestion-redpanda.md` only enough to mark it
  historical/future-conditional for current planning;
- update workflow routing to the mandatory follow-up technical design review.

Out of scope:
- implementation code;
- migration SQL;
- generated SQL access code;
- `tasks.md`;
- HTTP/OpenAPI contract design;
- runtime adapters, workers, bootstrap wiring, or broader service architecture;
- runtime event contract schemas;
- payment/top-up product or provider-contract design.

## Inputs Read

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`
- `specs/billing-money-core/spec.md`
- `specs/billing-money-core/workflow-plan.md`
- `specs/billing-money-core/workflow-plans/specification.md`
- `specs/billing-money-core/workflow-plans/technical-design.md`
- `specs/billing-money-core/workflow-plans/technical-design-review.md`
- `specs/billing-money-core/design/data-model.md`
- `specs/billing-money-core/design/event-ingestion-redpanda.md`
- repository schema/code references were inspected only to confirm the prior
  data-model slice had top-up/payment-evidence schema context and no current
  sign-up bonus schema/runtime implementation; no code was changed.

## Artifacts

| Artifact | Status | Notes |
| --- | --- | --- |
| `spec.md` | approved input | Current scope is sign-up bonus grants plus usage money decrement/release behavior. Payment/top-up and Redpanda payment-evidence ingestion are future/conditional. |
| `design/data-model.md` | repaired; follow-up review required | Adds `signup_bonus_grants`, `signup_bonus_credit`, `signup_bonus_grant` idempotency/outcome semantics, grant constraints/indexes, support readback, reconciliation, and test obligations. Top-up/payment evidence tables are marked dormant future context. |
| `design/event-ingestion-redpanda.md` | demoted to historical/future-conditional context | Current planning must not consume Redpanda payment-evidence ingestion or runtime event schemas from this addendum. It should be inspected in review only to verify the demotion is clear. |
| `design/contracts/` | conditional later | No HTTP/OpenAPI, payment/top-up, or runtime event contract design is approved in this pass. |
| `design/dependency-graph.md` | not expected in this pass | No package/module dependency design is in scope for this document-only repair. |
| `test-plan.md` | not expected as separate artifact in this pass | Current sign-up bonus and usage data-model test obligations are embedded in `design/data-model.md`; executable task planning is later. |
| `rollout.md` | conditional later | Runtime rollout, migration sequencing, and mixed-version behavior are not part of this technical-design repair. |
| `tasks.md` | blocked for current scope | Existing prior ledger must not be broadened or executed for sign-up bonus/current-scope changes until follow-up technical design review, planning, and task-ledger review/readiness complete. |

## Repair Summary

`design/data-model.md` now records the current balance-increase model:

- `signup_bonus_grant_id` is the billing-owned stable grant identity.
- `admission_authority` and `admission_reference_id` carry safe
  registration/account-admission lineage without designing a runtime provider
  contract in this phase.
- `signup_bonus_policy_version` scopes the one-time grant and prevents duplicate
  account/policy grants.
- `grant_amount_usd_atoms = 1000000000` enforces the current `$10.00` approved
  policy; any later amount change requires specification reopen and compatible
  schema evolution.
- `ledger_entries(effect_type = 'signup_bonus_credit')` is the explicit settled
  credit effect; the bonus is not a top-up, payment, operator adjustment, or
  migration import.
- Durable idempotency uses `operation_kind = 'signup_bonus_grant'` and
  fingerprints account, grant, policy version, admission reference, and amount.
- Duplicate same-fingerprint delivery returns the stored outcome; duplicate
  account/admission delivery with a changed fingerprint opens or reads
  `signup_bonus_conflict` and cannot credit twice.
- Support readback includes sign-up bonus grants with linked ledger,
  idempotency, stored outcome, audit, and reconciliation rows.
- Data-model tests now include exact `$10.00` atom proof, one-grant-per-account
  and one-grant-per-admission constraints, idempotency replay/conflict,
  concurrent duplicate grant delivery, reconciliation dedupe, ledger
  conservation, and O(1) grant hot-path lookup.

`design/event-ingestion-redpanda.md` is repaired for current routing:

- its status is historical/future-conditional;
- normalized payment evidence consumption is explicitly not current scope;
- payment evidence ingestion, payment/top-up credit, payments-service writeback,
  inbox/outbox event-ingestion deltas, and runtime event schemas require an
  explicit future reopen before planning;
- current sign-up bonus application is owned by `design/data-model.md` and does
  not depend on Redpanda payment evidence.

## Blockers And Reopen Conditions

Current blockers inside technical design repair: none.

Follow-up technical design review must verify:

- the sign-up bonus grant model is concrete enough for planning without hidden
  schema, idempotency, reconciliation, or test decisions;
- payment/top-up and Redpanda payment-evidence ingestion are clearly demoted to
  future/conditional context and cannot leak into current task planning;
- usage reserve/finalize/write-off data-model semantics remain planning-ready
  after the current funding-source repair.

Reopen `specification` if review finds that the sign-up bonus source identity,
policy version, amount policy, account-admission boundary, or payment/top-up
scope is still contradictory or undecided.

Reopen `technical design` if review finds local schema, constraint, index,
idempotency, support-readback, reconciliation, or test-obligation gaps that do
not require a specification change.

Planning remains blocked until the follow-up technical design review records
`PASS` or eligible `CONCERNS`.

## Completion Marker

Latest technical design repair completion criteria:

- `design/data-model.md` carries current sign-up bonus grant identity, ledger
  effect, idempotency, constraints/indexes, support readback, reconciliation,
  and tests.
- `design/event-ingestion-redpanda.md` no longer presents payment-evidence
  ingestion as current planning input.
- Workflow state routes the next session to follow-up technical design review.
- No code, migrations, generated SQL, runtime adapters, runtime event schemas,
  or `tasks.md` changes were added.

Completion marker: met.

## Stop Rule

Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: follow-up technical design review for the current
sign-up-bonus and usage-only repair.

Do not begin planning, `tasks.md`, implementation, migrations, generated code,
runtime event contract work, payment/top-up design, or adapter/worker work in
this session.

## Recommended Next-Session Prompt

```text
Work in `/Users/daniil/Projects/GonkaGate/billing-service`.

Next phase: follow-up technical design review for the current sign-up-bonus and usage-only repair.

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
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`

Goal:
Run the follow-up technical design review for the repaired current-scope packet. Verify that the sign-up bonus grant data model is planning-ready, that usage reserve/finalize/write-off remains current and independent of payment/top-up activation, and that payment/top-up evidence, payments-service writeback, and Redpanda payment-evidence ingestion are future/conditional rather than current planning input.

Expected output:
- update `specs/billing-money-core/workflow-plans/technical-design-review.md` with the reviewed packet, findings, orchestrator resolution, and gate verdict of `PASS`, `CONCERNS`, or `FAIL`;
- update `specs/billing-money-core/workflow-plan.md` to route the next phase based on that verdict;
- produce the next recommended prompt.

Stop rule:
Complete follow-up technical design review only, then stop with updated workflow state and the next recommended prompt. Do not create `tasks.md`, write code, create migrations, generate SQL, implement runtime adapters or workers, design runtime event schemas, or reopen payment/top-up scope unless the review verdict requires reopening an earlier phase.
```
