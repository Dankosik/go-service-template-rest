# Specification Phase Plan And Completion

Phase: specification
Status: complete
Date: 2026-06-01
Owner: orchestrator
Parent workflow: `../workflow-plan.md`

## Scope

Write `../spec.md` for the high-performance prepaid billing admission question.
The specification phase decides the accepted microlease direction, rejected
alternatives, proof obligations, accepted assumptions, reopen conditions, and
next-phase handoff.

This phase does not write technical design, `tasks.md`, migrations, schemas,
generated artifacts, adapters, tests, or implementation.

## Allowed Writes

Allowed writes confirmed:

- `../spec.md`
- `../workflow-plan.md`
- `workflow-plans/specification.md`

Out of scope:

- `design/`
- `tasks.md`
- `test-plan.md`
- `rollout.md`
- runtime contracts, migrations, generated artifacts, adapters, tests, and
  implementation code
- edits to `specs/event-driven-billing-money-architecture`

## Input Sources Used

Repository workflow and product context:

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/repo-architecture.md`
- `docs/critical-billing-context.md`
- `docs/PRD.md`

Task-local research and workflow state:

- `../workflow-plan.md`
- `workflow-plans/research.md`
- `../research/source-notes.md`
- `../research/pattern-catalog.md`
- `../research/architecture-options-matrix.md`
- `../research/risk-control-matrix.md`
- `../research/fan-in-synthesis.md`

Read-only context from the existing approved lease packet:

- `../../event-driven-billing-money-architecture/workflow-plan.md`
- `../../event-driven-billing-money-architecture/spec.md`
- `../../event-driven-billing-money-architecture/design/overview.md`
- `../../event-driven-billing-money-architecture/workflow-plans/technical-design-review.md`

Skills and references used:

- `.agents/skills/specification-session/SKILL.md`
- `.agents/skills/spec-document-designer/SKILL.md`
- `.agents/skills/spec-clarification-challenge/SKILL.md`
- `specification-session/references/allowed-writes-and-stop-rules.md`
- `specification-session/references/spec-clarification-gate-flow.md`
- `specification-session/references/handoff-to-technical-design.md`
- `spec-document-designer/references/spec-handoff-to-technical-design.md`

## Readiness Check

Specification-ready: yes.

Reason:

- research preserved the comparison set, source notes, risk controls, and
  fan-in synthesis;
- the current workflow explicitly routed the next phase to specification;
- the approval-changing decision was narrow enough to close from existing
  evidence: accept durable billing-issued microleases with zero unbacked spend
  exposure, and reject memory-only or Redis-only spend authority;
- remaining unknowns are design-owned tuning, benchmark, contract, data, or
  rollout details, with reopen conditions when a later discovery would change
  the authority decision.

## Clarification Gate

Clarification challenge: complete.

Gate type: local read-only formal clarification using the
`spec-clarification-challenge` rubric.

Lanes: local orchestrator challenge only.

Lenses:

- scope and spec coherence;
- domain invariants and edge cases;
- architecture ownership and dependency boundaries;
- API, data, compatibility, and source-of-truth consequences;
- security, reliability, delivery, and validation proof.

Scoped-down rationale:

- The work is full-orchestrated protected-money work, so formal clarification
  is required.
- The available multi-agent spawn tool is restricted to explicit user
  authorization for subagents, and the user requested a specification-only
  phase without delegation.
- To avoid violating tool policy while preserving the gate, the orchestrator ran
  a local formal challenge across the default lens set and reconciled the
  results into `../spec.md`.

Resolution:

| Lens | Result | Action |
| --- | --- | --- |
| Scope/spec coherence | No blocker. `spec.md` decides the direction and stops before design. | Recorded accepted direction, out-of-scope surfaces, and technical-design handoff. |
| Domain invariants and edge cases | No blocker after rejecting unbacked spend. | Recorded zero unbacked exposure, visible reserved exposure, child/parent caps, and strict/fail-closed cases. |
| Architecture ownership and dependency boundaries | No blocker. | Recorded authority classes for billing PostgreSQL, proxy durable rows, memory, Redis, Redpanda, ClickHouse, and checkpoints. |
| API/data/compatibility/source-of-truth | No blocker. | Recorded that existing lease packet is read-only context and contract/data details belong to technical design. |
| Security/reliability/delivery/validation proof | No blocker. | Recorded privacy constraints, outage behavior, proof obligations, rollout proof, and reopen triggers. |

Targeted research reopened: no.

Requires user decision: no.

Clarification approval rationale:

- All approval-changing questions are answered from existing research and
  repository context.
- The only product/business missing input is a nonzero write-off budget for
  memory-only or Redis-only spend, and the spec resolves that by rejecting such
  spend for the target instead of inventing a budget.

Rerun condition:

- Rerun clarification if technical design introduces memory-only or Redis-only
  spend, direct per-request reserve fallback, weaker billing PostgreSQL
  authority, weaker proxy durable child lineage, broader payment/top-up/account
  or pricing scope, or weaker privacy/outage policy.

## Specification Result

`../spec.md` status: approved.

Approved direction:

- durable billing-issued microleases as escrowed USD spend rights;
- zero unbacked spend exposure;
- billing PostgreSQL remains customer-money authority;
- durable proxy child debit lineage remains required before external paid
  execution;
- memory and Redis are cache, limiter, projection, or backpressure only;
- async terminal facts and checkpoint/close batches are proof and settlement
  infrastructure, not first-admission authority;
- strict mode is durable/fail-closed behavior, not direct reserve fallback by
  default.

Rejected options:

- strict durable reserve before every request as uniform target;
- direct per-request reserve fallback as hidden/routine alternate path;
- pure async metering as prepaid admission;
- Redis/global counter as money authority;
- process-local memory as authoritative spend allowance;
- checkpoint totals without per-child debit identity;
- silent release of expired capacity without proof.

## Blockers, Assumptions, And Reopen Conditions

Blockers:

- None for starting technical design.

Accepted assumptions:

- No live traffic, production DB, deployment, or benchmark evidence was needed
  to decide the authority model.
- Proxy durable storage can be designed to support child debit allocation fast
  enough for the target paid path. If not, reopen specification rather than
  moving authority to memory or Redis silently.

Reopen specification if:

- pricing-service cannot provide or attest USD-compatible immutable pricing
  snapshot evidence for microlease issuance, child debit caps, and terminal
  settlement;
- web-search-like paid paths cannot map into microlease, child debit, terminal
  settlement, write-off, reversal, and reconciliation semantics;
- product/platform wants a nonzero memory-only or Redis-only write-off exposure
  budget;
- technical design needs direct per-request billing reserve fallback for
  migrated paid cohorts;
- the target performance envelope cannot be met with billing-issued reserved
  microleases and durable local child debit allocation without weakening money
  authority.

## Phase Status

Current phase: specification.
Phase status: complete.
Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: technical design.

## Next Session Context Bundle

The technical-design session should read:

1. `AGENTS.md` and `docs/spec-first-workflow.md` for phase boundaries,
   artifact rules, technical-design triggers, mandatory technical-design review,
   and stop rules.
2. `docs/repo-architecture.md`, `docs/critical-billing-context.md`, and
   `docs/PRD.md` for repository boundaries, money invariants, Redis/cache
   limits, fail-closed product constraints, and privacy constraints.
3. `specs/event-driven-billing-money-performance-microleases/workflow-plan.md`
   and `workflow-plans/specification.md` for current workflow state,
   clarification-gate resolution, and stop rule.
4. `specs/event-driven-billing-money-performance-microleases/spec.md` because
   it is the approved decision record and source of truth for technical design.
5. The task-local research bundle under
   `specs/event-driven-billing-money-performance-microleases/research/` for
   source evidence, option comparison, and risk controls.
6. `specs/event-driven-billing-money-architecture/workflow-plan.md`,
   `spec.md`, `design/overview.md`, and
   `workflow-plans/technical-design-review.md` as read-only context for the
   existing approved lease packet that the microlease design may refine or
   reconcile with.

## Stop Rule

Specification complete. Stop before technical design, planning, tasks,
implementation, validation, migrations, schemas, generated artifacts, adapters,
or tests.
