# Balance And Usage Authority Cutover Research

Phase: research
Status: complete
Owner: orchestrator
Date: 2026-06-01

## Session Outcome

This research phase completed local read-only research for the lane questions
recorded in `workflow-plans/workflow-planning.md`.

No subagents were spawned because the user did not explicitly authorize
subagent fan-out for this session. The scoped-down mode was local orchestrator
research against the same L1-L9 lane questions, with preserved evidence notes
under `research/`.

No `spec.md`, design, `tasks.md`, code, migrations, generated files, tests, or
rollout artifacts were written in this phase.

## Research Artifacts

| Lane | Status | Note |
| --- | --- | --- |
| L1 billing-service current state | complete | `research/l1-billing-service-current-state.md` |
| L2 gonka-proxy current state | complete | `research/l2-gonka-proxy-money-paths.md` |
| L3 API and compatibility inputs | complete | `research/l3-l5-api-data-distributed-inputs.md` |
| L4 data authority inputs | complete | `research/l3-l5-api-data-distributed-inputs.md` |
| L5 distributed flow inputs | complete | `research/l3-l5-api-data-distributed-inputs.md` |
| L6 security/privacy inputs | complete | `research/l6-l9-security-reliability-qa-performance-inputs.md` |
| L7 reliability/rollout inputs | complete | `research/l6-l9-security-reliability-qa-performance-inputs.md` |
| L8 QA/proof inputs | complete | `research/l6-l9-security-reliability-qa-performance-inputs.md` |
| L9 performance inputs | complete | `research/l6-l9-security-reliability-qa-performance-inputs.md` |
| Fan-in synthesis | complete | `research/local-research-synthesis.md` |

## Scoped-Down Rationale

The workflow plan preferred read-only fan-out because this is money, API, data,
auth, worker, performance, and cross-service cutover work. The user did not
explicitly authorize subagents in this session, so research stayed local and
read-only. The local pass still preserved each planned lane's owned question,
exact source paths, evidence limits, conflicts, and specification handoff
implications.

## Fan-In Summary

Facts to carry into specification:

- `billing-service` currently exposes generated internal microlease HTTP routes,
  not the broader account resolve, balance read, usage reserve/finalize/write-off
  and reversal HTTP surface needed by the cutover.
- The billing-service runtime currently builds its HTTP router without a
  concrete microlease service, so generated microlease handlers would return the
  handler-level `503` path when called in the current service bootstrap.
- The billing worker command exists, but its current bootstrap supplies no-op
  runtime tasks for terminal, checkpoint, close, inbox retry, outbox relay,
  stale reconciliation, and admission renewal.
- The durable money schema contains `billing_accounts`, `account_balances`,
  usage operation/hold/outcome tables, immutable ledger tables, microlease
  tables, event inbox/outbox, and reconciliation tables.
- `gonka-proxy` still has local `User.balanceNgonka`, in-memory reservations,
  local balance transactions, completion deduction paths, local usage logging,
  and web-search pre-dispatch reservation paths.
- The proxy's existing shared-balance bridge posts to `/api/v1/usage/*` and
  `/api/v1/account-effects/*`, while billing-service currently exposes
  `/internal/billing/v1/microleases/*` and `/internal/billing/v1/operations/readback`.
- The proxy has microlease allocator and migrated-cohort policy code, but source
  search found those wired only to tests/support code, not live completion
  request execution paths.

Specification must decide:

- whether this cutover standardizes on a microlease-first runtime API, adds the
  broader usage/account HTTP surface, or defines both with exact ownership;
- the canonical account identity and account resolve behavior from proxy user
  identity to billing `account_scope_key`;
- the exact balance read contract, including active microlease exposure and
  stale/ambiguous operation visibility;
- idempotency keys, fingerprints, replay behavior, conflict mapping, and stored
  outcome readback for every usage lifecycle command;
- service authentication compatibility, because billing-service expects scoped
  JWT service auth while the current proxy shared-balance bridge is configured
  as a bearer auth-key client;
- worker and Redpanda runtime ownership, since adapters and events exist but the
  billing worker bootstrap is not wired to real tasks;
- proxy migration scope: either approved cross-repo edits or an exact
  `gonka-proxy` implementation handoff.

## Evidence Limits

- This phase was read-only research except for writing the requested workflow
  and research notes.
- No tests, builds, migrations, contract generators, or live deployment checks
  were run as validation proof.
- The `billing-service` checkout was already dirty before this research phase;
  unrelated existing changes were treated as current workspace context and were
  not reverted.
- `gonka-proxy` was inspected locally only. No proxy files were changed.

## Completion Marker

Research is complete when:

- this file records local research mode, fan-in, evidence limits, and stop rule;
- lane evidence notes exist under `research/`;
- `workflow-plan.md` routes the next session to specification;
- no later-phase artifact or implementation surface is created.

Completion status: complete.

## Stop Rule

Stop now. The next phase is specification only.
