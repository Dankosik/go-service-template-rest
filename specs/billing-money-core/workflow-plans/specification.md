# billing-money-core Specification Reopen Phase

Phase: specification reopen
Pass type: current funding-source correction after user clarification
Status: complete
Owner: orchestrator
Latest outcome: approved current sign-up-bonus and usage-only money scope;
payment/top-up evidence and Redpanda payment ingestion are deferred from current
planning

## Historical Scope: Redpanda Evidence Versioning

This phase owns only the specification boundary exposed by REDPANDA-TDR-F01:
the billing/payments normalized payment evidence identity and versioning
contract for Redpanda ingestion.

In scope:
- decide whether billing requires immutable payment evidence IDs per
  billing-applicable economic meaning or accepts payments-service versioned
  evidence under one stable `paymentEvidenceId`;
- update `specs/billing-money-core/spec.md` with the canonical decision;
- route the next phase to technical design repair.

Out of scope:
- code, migrations, generated SQL, runtime adapters, and `tasks.md`;
- runtime event contract schemas;
- repairing `design/event-ingestion-redpanda.md` or `design/data-model.md` in
  this phase;
- deciding REDPANDA-TDR-F02 or REDPANDA-TDR-F03 implementation mechanics.

## Inputs Read

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `specs/billing-money-core/workflow-plan.md`
- `specs/billing-money-core/workflow-plans/technical-design.md`
- `specs/billing-money-core/workflow-plans/technical-design-review.md`
- `specs/billing-money-core/spec.md`
- `specs/billing-money-core/design/data-model.md`
- `specs/billing-money-core/design/event-ingestion-redpanda.md`
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`
- `/Users/daniil/Projects/GonkaGate/payments-service/docs/service-foundation/data-model-and-migration.md`
- `/Users/daniil/Projects/GonkaGate/payments-service/research/02-internal-money-flow/findings.md`
- `/Users/daniil/Projects/GonkaGate/gonka-proxy/src/contracts/internal-money/billing/v1/index.ts`
- User clarification on 2026-06-01 that payment/top-up is not implemented now
  and the only current balance increase is the `$10.00` sign-up bonus.

## Readiness And Evidence

Readiness outcome: ready.

The prior technical design review already isolated one specification-level
approval question. Current provider-contract evidence was sufficient to decide
it:

- payments-service planning authority defines `paymentEvidenceId` as the stable
  payments-owned normalized-evidence lineage;
- payments-service requires monotonic `evidenceVersion` when versioned
  normalized economic meaning changes under that same lineage;
- evidence writeback selectors to billing include `paymentEvidenceId` plus
  `evidenceVersion`, not the bare stable lineage id;
- the current `gonka-proxy` internal-money billing v1 `topupApplyEvidence`
  schema lacks `evidenceVersion`, so later contract design must repair that
  version-scoped selector instead of treating the existing v1 request as final
  authority against payments-service's current target contract.

## Clarification Gate

Formal challenge trigger: protected money/data/cross-service contract domain.

Gate shape: scoped-down local specification clarification.

Lens: API/data/source-of-truth consequence of the REDPANDA-TDR-F01 payment
evidence selector mismatch.

Scoped-down rationale: the approval risk was concentrated in one already
reviewed question: whether billing's evidence selector is bare
`paymentEvidenceId` or version-scoped. Multi-lens fan-out would duplicate the
completed technical-design-review evidence instead of surfacing independent
specification questions.

Resolution:
- `paymentEvidenceId` remains a stable payments-owned evidence lineage and must
  not rebind across attempt, top-up, or account scope;
- `evidenceVersion` is required for payments-service evidence handoff;
- `(paymentEvidenceId, evidenceVersion)` is billing's evidence application,
  inbox lineage, idempotency, and replay selector;
- a new version is a new immutable evidence claim for evaluation and cannot
  silently rewrite a prior ledger effect.

Clarification status: resolved.
Targeted research reopened: no.
User decision required: no.

## Historical Spec Status: Redpanda Evidence Versioning

`specs/billing-money-core/spec.md`: approved for the narrow Redpanda
evidence-versioning addendum.

Approved decision:
- Billing accepts payments-service versioned normalized evidence. It does not
  require payments-service to mint immutable new `paymentEvidenceId` values for
  each corrected or superseding normalized economic meaning.

Spec-level consequences:
- billing evidence application, Redpanda inbox lineage, business idempotency,
  replay, and support readback must carry the version-scoped selector;
- duplicate same-version delivery returns stored outcome when the fingerprint
  matches;
- same version with a changed fingerprint is conflict and cannot mutate money;
- different versions under one evidence lineage are evaluated through explicit
  evidence kind/finality and prior-evidence lineage, never by rewriting posted
  ledger effects.

## Historical Next Action

Historical next action was `reopen_phase`.
Historical next phase was technical design repair.

Technical design must repair:
- REDPANDA-TDR-F01 consequences in `design/event-ingestion-redpanda.md` and any
  affected `design/data-model.md` sections;
- REDPANDA-TDR-F02 poison/quarantine durable identity for events with malformed
  or missing producer event identity or semantic identity fields;
- REDPANDA-TDR-F03 committed-offset `retry_scheduled` recovery owner and
  lifecycle.

That repair and follow-up technical design review were completed before the
current funding-source correction. They are now superseded for current planning
where they treat payment/top-up evidence ingestion as active scope.

## Current Funding-Source Correction

Pass type: specification reopen after user clarification.

User-provided product fact:
- customer payment/top-up is not implemented now;
- users cannot currently add balance themselves;
- the only current balance-increase path is the `$10.00` sign-up bonus credited
  at registration/account admission;
- after that, balance is spent through usage reserve/finalize/write-off paths.

Scope decision:
- Current active billing scope is sign-up bonus grants plus usage money
  decrement/release behavior.
- Payment-provider top-ups, payments-service evidence handoff, payment
  presentation sync, normalized payment evidence application, and Redpanda
  payment-evidence ingestion are future/conditional context only.
- Prior version-scoped payment evidence selector decisions may remain useful
  future context, but they must be revalidated when payment/top-up scope is
  explicitly reopened.

Spec status:
- `specs/billing-money-core/spec.md` is approved for the current
  sign-up-bonus/usage-only runtime scope.
- The prior Redpanda payment-evidence addendum is superseded for current
  planning.

Clarification gate:
- Formal trigger: protected money/data/domain scope.
- Gate shape: scoped-down local clarification.
- Scoped-down rationale: the approval-changing question is a direct product
  boundary correction supplied by the user: whether payments/top-up are active
  now. No separate research lane is needed to decide this current-scope fact.
- Resolution: approved with technical-design repair required.

Next action: `reopen_phase`.
Next phase: technical design repair for current sign-up-bonus and usage-only
money scope.

Technical design must repair:
- sign-up bonus grant identity, one-time credit semantics, idempotency, ledger
  effect, constraints/indexes, support readback, reconciliation, and tests;
- current usage reserve/finalize/write-off design to ensure it does not depend on
  payment/top-up activation;
- payment/top-up evidence and Redpanda payment-evidence ingestion treatment as
  future/conditional context, not current planning input.

Follow-up technical design review remains mandatory after repair. Planning and
event-ingestion or money-core task-ledger work remain blocked until that review
records `PASS` or eligible `CONCERNS`.

## Stop Rule

Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: technical design repair for current sign-up-bonus and
usage-only money scope.

Do not create `tasks.md`, write code, create migrations, generate SQL,
implement runtime adapters, or design runtime event contract schemas in this
session.
