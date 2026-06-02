# Event-Driven Billing Money Architecture Workflow Plan

Mode: full orchestrated
Status: technical design review complete with PASS; planning ready
Current phase: technical design review
Phase status: fresh lease-packet review complete with PASS
Owner: orchestrator

## Objective

Produce the production-ready billing architecture that moves balance
reservation, settlement, and money mutation authority out of `gonka-proxy` and
into `billing-service`, while preserving strict customer-money invariants and
minimizing proxy paid-request latency.

The active approved target is billing-issued spending leases:

- billing-service issues bounded account-scoped leases and reserves the full
  USD exposure in Postgres before proxy may spend them;
- `gonka-proxy` admits paid requests only by durably allocating child debit
  authorizations from active billing-minted lease generations;
- terminal settlement remains asynchronous through proxy durable submission,
  Redpanda transport, billing durable inbox, Postgres ledger effects, billing
  outbox facts, and reconciliation.

This workflow supersedes `specs/billing-money-core/` for the current usage
architecture question. Older `billing-money-core` artifacts remain historical
context only.

## Current State Summary

The original approved specification and technical design selected protected
per-request billing reserve before every paid execution. Follow-up technical
design review passed that old packet.

Before `tasks.md` was written, the user changed the performance requirement and
reopened specification. The approved reopened `spec.md` now selects
billing-issued account-scoped spending leases. The old design, old
`test-plan.md`, old `rollout.md`, and old technical design review PASS are
historical context only.

The technical-design session repaired the task-local design bundle for the lease
architecture and stopped before technical design review. The fresh read-only
technical design review passed the repaired billing-issued spending lease packet
and planning may now start.

## Why Full Orchestrated

Full orchestrated is required because this work touches protected domains:

- customer money, billing, reservations, leases, holds, write-offs, reversals,
  quotas, and entitlements;
- persisted ledger state, migrations, historical balance import/cutover,
  replay, and reconciliation;
- distributed Redpanda event flow, retries, duplicate delivery, delayed and
  out-of-order events, inbox/outbox, process restarts, and worker lifecycle;
- cross-service boundaries with `gonka-proxy`, `pricing-service`, future
  `payments-service`, identity/API-key attribution, and operator tooling;
- proxy request-path performance, fail-closed behavior, dependency outage
  handling, rollout, rollback, and mixed-version behavior.

## Hard Requirements To Preserve

- Billing must be the authoritative USD customer-money source of truth.
- Normal usage must not take an account negative.
- Lease issuance, child debit authorization, finalization, write-off,
  reversal, reconciliation, and visible balances must be mathematically exact.
- Every money-affecting operation must be durable, replay-stable, idempotent,
  explainable, and auditable.
- Redpanda is transport and replay infrastructure, not a substitute for
  database-backed money correctness.
- Proxy request-path overhead may be minimized only by spending from
  billing-issued reserved lease capacity through durable child debits.
- Usage that already happened but later cannot be settled must have explicit
  write-off, compensation, or reconciliation. It must not become silent balance
  mutation or retroactive overcharge.
- Logs, research notes, design files, tasks, events, and telemetry must not
  record raw prompts, completions, SSE chunks, bearer tokens, API keys, DSNs,
  payment secrets, raw webhook bodies, dynamic proof URLs, or unbounded provider
  payloads.

## Artifact State

| Artifact | State | Trigger / Notes |
| --- | --- | --- |
| `workflow-plan.md` | updated | Master control file routes from fresh technical design review PASS to planning. |
| `workflow-plans/workflow-planning.md` | complete | Historical routing for workflow-planning session. |
| `workflow-plans/research.md` | complete | Historical research lane execution and fan-in. |
| `workflow-plans/specification.md` | complete after reopen | Reopened specification readiness, formal clarification rerun, fan-in resolution, completion marker, and stop rule. |
| `research/` | complete | Compact current provider/consumer evidence and fan-in synthesis for specification input. |
| `spec.md` | approved after reopen | Canonical decision record. It selects billing-issued spending leases. |
| Formal spec clarification challenge | complete after rerun | Five read-only `spec-clarification-challenge` lenses were rerun and reconciled. |
| `workflow-plans/technical-design.md` | complete after lease repair | Phase-local design state and handoff to review. |
| `design/overview.md` | repaired review-ready | Entry point and bundle index for billing-issued leases. |
| `design/component-map.md` | repaired review-ready | Components, generated surfaces, proxy durable allocator, worker, stable non-touches, and bridge placement. |
| `design/sequence.md` | repaired review-ready | Lease issuance/replenishment, child debit allocation, terminal settlement, checkpoint/close, expiry, reconciliation, and backpressure. |
| `design/ownership-map.md` | repaired review-ready | Billing/proxy/pricing/API-key/identity/Redpanda/Postgres ownership boundaries. |
| `design/data-model.md` | repaired review-ready | Spending leases, child debit lineage, checkpoints, inbox/outbox, admission controls, migration shape, replay, retention, and privacy. |
| `design/dependency-graph.md` | repaired review-ready | Runtime/package dependency graph, external dependency shape, and coupling controls. |
| `design/contracts/protected-http.md` | repaired review-ready | Protected lease issue/replenish/readback/close contracts and status mapping. |
| `design/contracts/redpanda-events.md` | repaired review-ready | Terminal, checkpoint/close, billing facts, protobuf authority, authenticity, retention, and evolution. |
| `test-plan.md` | repaired review-ready | Lease/debit/fencing/proxy durability/event/privacy/performance/rollout proof obligations. |
| `rollout.md` | repaired review-ready | Lease-path rollout, no-dual-writer gates, direct-reserve fallback disablement, rollback/failback, and bridge exit. |
| `workflow-plans/technical-design-review.md` | fresh lease-packet PASS | Current review record. Prior old-packet FAIL/PASS is historical context only. |
| `tasks.md` | missing, expected next | Planning is ready to create and review the task ledger. |
| Post-code review or validation phase files | conditional | Planning must decide later after fresh design review. |

## Blockers, Assumptions, And Reopen Targets

Blockers:

- None for starting planning.
- Implementation remains blocked until planning creates `tasks.md` and
  task-ledger review/readiness passes.

Accepted assumptions:

- Current evidence is static repository and sibling-repository source/contracts.
  No live deployment, production DB, or traffic evidence was used.
- `pricing-service` can provide or attest USD-compatible immutable pricing
  snapshot evidence before implementation planning approves lease issuance and
  child debit allocation.
- Current web-search-like paid operations can map into the shared
  lease/debit/finalize/write-off/reversal model.

Reopen specification if:

- pricing-service cannot provide or attest USD-compatible immutable pricing
  snapshot evidence;
- a current web-search paid path cannot map into the shared
  lease/debit/finalize/write-off/reversal model;
- the target performance envelope cannot be met with billing-issued leases and
  durable local debit allocation without moving money authority to proxy or
  cache;
- design introduces direct per-request reserve fallback, risk-based branchy
  paid admission, pure cached balance admission, payment/top-up scope, payment
  evidence ingestion, changed pricing/account/API-key authority, weaker proxy
  terminal durability, weaker billing Postgres money authority, or weaker
  outage/privacy policy.

Reopen technical design if planning cannot task package boundaries, protected
HTTP/event contracts, lease/debit data-model deltas, worker lifecycle, proxy
durable lease/debit allocation, rollout gates, or validation proof without
choosing a new design.

Reopen research if a later phase requires live deployment evidence, production
latency distributions, current DB rows, or materially changed sibling-provider
contracts.

## Routing Status

Current phase: technical design review.
Phase status: fresh lease-packet review complete with PASS.
Session boundary reached: yes.
Ready for next session: yes.
Next session starts with: planning.

## Next Session Context Bundle

The next planning session should read:

1. `AGENTS.md` and `docs/spec-first-workflow.md` for phase boundaries,
   artifact rules, planning rules, task-ledger review/readiness, and stop
   rules.
2. `specs/event-driven-billing-money-architecture/workflow-plan.md` for
   current workflow state and planning routing.
3. `specs/event-driven-billing-money-architecture/workflow-plans/technical-design-review.md`
   because it records the fresh lease-packet PASS and planning-input
   obligations.
4. `specs/event-driven-billing-money-architecture/workflow-plans/technical-design.md`
   because it records the completed lease design repair and review handoff.
5. `specs/event-driven-billing-money-architecture/workflow-plans/specification.md`
   because it records the reopened specification pass and formal clarification
   fan-in.
6. `specs/event-driven-billing-money-architecture/spec.md` because it is the
   canonical approved decision record.
7. `specs/event-driven-billing-money-architecture/design/overview.md`,
   `design/component-map.md`, `design/sequence.md`, `design/ownership-map.md`,
   `design/data-model.md`, `design/dependency-graph.md`,
   `design/contracts/protected-http.md`, and
   `design/contracts/redpanda-events.md` because planning must task the reviewed
   design without inventing new architecture.
8. `specs/event-driven-billing-money-architecture/test-plan.md` and
   `rollout.md` because planning must carry proof and rollout obligations into
   `tasks.md`.
9. `docs/repo-architecture.md`, `docs/critical-billing-context.md`,
   `docs/PRD.md`, and `docs/build-test-and-development-commands.md` for
   repository boundaries, money invariants, worker/contract seams, rollout
   constraints, and repo-owned validation entrypoints.

Next phase: planning only.
Expected output: `tasks.md` with a Goal-ready ledger, task-ledger
review/readiness result, updated workflow state, and next-session prompt.
Constraints: do not implement runtime code, migrations, generated SQL, runtime
schemas, adapters, tests, generated artifacts, runtime event schemas, or code;
do not broaden scope into direct reserve fallback, branchy admission, pure cached
balance, process-local spend allowance, payment/top-up scope, changed
pricing/account/API-key authority, weaker terminal durability, weaker billing
Postgres money authority, or weaker outage/privacy policy.
Stop rule: complete planning and task-ledger review/readiness only, then stop
before implementation.

## Recommended Next-Session Prompt

```text
Work in `/Users/daniil/Projects/GonkaGate/billing-service`.

Next phase: planning for `specs/event-driven-billing-money-architecture`.

Read first:
- `AGENTS.md` and `docs/spec-first-workflow.md` for phase boundaries, artifact rules, planning rules, task-ledger review/readiness, and stop rules.
- `specs/event-driven-billing-money-architecture/workflow-plan.md` for current workflow state and planning routing.
- `specs/event-driven-billing-money-architecture/workflow-plans/technical-design-review.md` because it records the fresh lease-packet PASS and planning-input obligations.
- `specs/event-driven-billing-money-architecture/workflow-plans/technical-design.md` because it records the completed lease design repair and review handoff.
- `specs/event-driven-billing-money-architecture/workflow-plans/specification.md` because it records the reopened specification pass and formal clarification fan-in.
- `specs/event-driven-billing-money-architecture/spec.md` because it is the canonical approved decision record.
- `specs/event-driven-billing-money-architecture/design/overview.md`, `design/component-map.md`, `design/sequence.md`, `design/ownership-map.md`, `design/data-model.md`, `design/dependency-graph.md`, `design/contracts/protected-http.md`, and `design/contracts/redpanda-events.md` because planning must task the reviewed design without inventing new architecture.
- `specs/event-driven-billing-money-architecture/test-plan.md` and `rollout.md` because planning must carry their proof and rollout obligations into `tasks.md`.
- `docs/repo-architecture.md`, `docs/critical-billing-context.md`, `docs/PRD.md`, and `docs/build-test-and-development-commands.md` for repository boundaries, money invariants, worker/contract seams, rollout constraints, and repo-owned validation entrypoints.

Objective:
Create and review `specs/event-driven-billing-money-architecture/tasks.md` for the approved billing-issued spending lease architecture. The ledger must cover the approved `spec.md`, repaired design bundle, fresh technical-design-review PASS planning obligations, `test-plan.md`, and `rollout.md`, with no open questions or implementation-time design decisions.

Constraints:
- Do not implement runtime code, migrations, generated SQL, runtime adapters, tests, runtime event schemas, generated artifacts, or code in the planning phase.
- Do not broaden scope into direct per-request reserve fallback, branchy paid admission, pure cached balance authority, process-local spend allowance, Redpanda reserve commands, payment/top-up flows, payment evidence ingestion, changed pricing/account/API-key authority, weaker terminal durability, weaker billing Postgres money authority, or weaker outage/privacy policy.
- If planning cannot task a proof obligation without choosing new architecture, reopen technical design. If tasking would require a spec-scope change, reopen specification.

Expected output:
- `specs/event-driven-billing-money-architecture/tasks.md` with Goal-ready task ledger, dependencies, proof obligations, evidence slots, and explicit constraints.
- Task-ledger review/readiness result: `PASS`, eligible `CONCERNS` with named proof obligations, or `FAIL` with reopen target.
- Updated workflow state and the next-session prompt. If readiness passes, the next prompt must be an implementation prompt composed with `codex-goal-prompt-composer`.

Stop rule:
Complete planning and task-ledger review/readiness only, then stop before implementation.
```
