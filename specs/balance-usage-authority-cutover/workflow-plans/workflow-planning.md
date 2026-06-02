# Balance And Usage Authority Cutover Workflow Planning

Phase: workflow-planning
Status: complete
Owner: orchestrator
Date: 2026-06-01

## Session Outcome

This session created the workflow-control shell for the new
`balance-usage-authority-cutover` task and stopped before research,
specification, technical design, planning, or implementation.

The completed microlease packet remains preserved context:

- `specs/event-driven-billing-money-performance-microleases/spec.md`;
- `specs/event-driven-billing-money-performance-microleases/tasks.md`;
- `specs/event-driven-billing-money-performance-microleases/design/`;
- `specs/event-driven-billing-money-performance-microleases/test-plan.md`;
- `specs/event-driven-billing-money-performance-microleases/rollout.md`.

No files from that packet were changed during this workflow-planning session.

## Local Orchestration

Order completed:

1. Read repository workflow rules and the user-provided task frame.
2. Confirm the work is protected money/API/data/security/worker/rollout and
   cannot use direct path or lean local.
3. Verify the completed microlease packet is prior context, not the active task
   ledger for the new broader balance/usage authority cutover.
4. Create the new task-local directory:
   `specs/balance-usage-authority-cutover`.
5. Write the master `workflow-plan.md`.
6. Write this phase-local workflow-planning file.
7. Run local adequacy self-check and record the subagent authorization
   constraint.
8. Stop at the workflow-planning boundary.

## Research Handoff

Next session starts with: research.

Research mode: fan-out planned by evidence lane, with local fallback if the
active tool policy still blocks subagent spawning without explicit user
authorization.

Planned lanes:

| Lane | Role | Owned Question | Skill | Output |
| --- | --- | --- | --- | --- |
| L1 | `explorer` or local orchestrator | Existing billing-service surfaces and remaining gaps for account, balance, usage, worker, and config authority. | `research-session` | Billing-service current-state map and decision inputs. |
| L2 | `explorer` or local orchestrator | Current `gonka-proxy` balance/usage/local-money/fallback paths that must map to billing-service. | `research-session` | Proxy path map with exact files/routes/tests. |
| L3 | `api-agent` | Required internal REST resources, idempotency rules, status mappings, generated contract constraints, and compatibility risks. | `api-contract-designer-spec` | API options and blockers only. |
| L4 | `data-agent` | Source-of-truth, schema, SQLC, transaction, retention, and reconciliation questions for balance/usage authority. | `go-data-architect-spec` | Data options and blockers only. |
| L5 | `distributed-agent` | Cross-service saga, inbox/outbox, Redpanda, retry, replay, ordering, and ambiguous-outcome semantics. | `go-distributed-architect-spec` | Distributed-flow risks and decisions only. |
| L6 | `security-agent` | Service auth, tenant/account attribution, privacy, abuse controls, and trust boundaries. | `go-security-spec` | Security constraints and proof obligations only. |
| L7 | `reliability-agent` | Timeouts, fail-closed behavior, worker lifecycle, readiness, rollback, degraded mode, and recovery. | `go-reliability-spec` | Reliability constraints and reopen triggers only. |
| L8 | `qa-agent` | Unit, integration, contract, worker, replay, privacy, performance, and cross-repo proof obligations. | `go-qa-tester-spec` | Test strategy inputs only. |
| L9 | `performance-agent` | Hot-path budgets and benchmark proof for proxy cutover without unbacked memory or Redis spend authority. | `go-performance-spec` | Performance budgets and measurement obligations only. |

Parallelizable in the next session:

- L1 and L2 can run first or in parallel because they provide the local current
  state and proxy current state.
- L3 through L9 can run after enough current-state evidence exists, or locally
  as separate read-only passes if subagent spawning remains unavailable.

Fan-in rule:

- Separate facts, inferences, assumptions, blockers, and open points.
- Reconcile lane findings into research notes only; do not approve
  specification decisions during research.
- Any conflict involving money authority, proxy fallback, top-up/payment scope,
  privacy, or cross-repo write permission must be routed to specification as a
  must-decide-now issue.

## Adequacy Self-Check

Status: PASS.

Checked:

- full orchestrated shape matches protected-domain triggers;
- next phase is research, not implementation;
- lane ownership is by evidence question with one skill per lane;
- later `spec.md`, `design/`, technical design review, `tasks.md`,
  `test-plan.md`, and `rollout.md` are expected but not created in this phase;
- completed microlease artifacts are context and preserved constraints;
- no code, tests, migrations, generated files, `spec.md`, design, task ledger,
  test plan, or rollout plan were written.

Formal read-only adequacy subagent not spawned:

- reason: current multi-agent tool policy allows `spawn_agent` only when the
  user explicitly asks for subagents, delegation, or parallel agent work;
- consequence: next-session research must either obtain explicit authorization
  for fan-out or record a local-research scoped-down rationale.

Blocking findings: none.

## Completion Marker

Complete when:

- `workflow-plan.md` records full orchestrated shape, research mode, artifact
  expectations, adequacy status, session boundary, blockers, assumptions, and
  next-session context;
- this file records local orchestration, lane plan, fan-in rule, adequacy
  self-check, and stop rule;
- both files agree that the next session starts with research;
- no later-phase artifact or implementation work has started.

Completion status: complete.

## Stop Rule

Stop now.

Do not start research, spawn research lanes, write `research/*.md`, write
`spec.md`, write `design/`, write `tasks.md`, write `test-plan.md`, write
`rollout.md`, edit code, run implementation tests as proof, or mutate any
existing completed workflow packet in this session.
