# Workflow Planning

Phase: workflow-planning
Status: complete
Owner: orchestrator
Master plan: `../workflow-plan.md`

## Session Outcome

Create workflow-control routing for the new event-driven billing money architecture effort, classify the execution shape, plan the research/subagent lanes, record artifact expectations, run the required read-only adequacy challenge, and stop before research.

Success criteria:
- master and phase-local workflow files agree on shape, research mode, artifact expectations, blockers, adequacy status, next session, and stop rule;
- the next session can start research without recreating this routing from chat;
- no research notes, `spec.md`, design files, `tasks.md`, code, migrations, generated SQL, runtime adapters, or tests are created.

## Evidence Read In This Phase

- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/PRD.md`
- `docs/critical-billing-context.md`
- `docs/repo-architecture.md`
- `specs/billing-money-core/workflow-plan.md`
- `specs/billing-money-core/spec.md`
- `specs/billing-money-core/design/data-model.md`
- `specs/billing-money-core/design/event-ingestion-redpanda.md`
- `specs/billing-money-core/workflow-plans/technical-design-review.md`

The prior `billing-money-core` artifacts were inspected as historical context only. They do not constrain the new architecture workflow where they deferred Redpanda usage/payment ingestion or assumed synchronous usage calls as the current direction.

## Execution Shape

Shape: full orchestrated.

Reason:
- money, billing, reservations, balances, quotas, persisted state, distributed events, retries, reconciliation, proxy performance, cross-service boundaries, rollout, and strict phase boundaries are all triggered.
- the user explicitly requested production-ready architecture, not a quick MVP or implementation shortcut.

## Local Orchestration

Order for this session:
1. Read workflow authority and task context.
2. Create the master workflow plan and this phase-local workflow-planning file.
3. Run one read-only workflow-plan adequacy challenger.
4. Reconcile adequacy findings by updating workflow-control files or leave the phase blocked.
5. Stop with a copy-pastable prompt for exactly one next phase.

Parallelizable work:
- No domain research lanes run in this workflow-planning session.
- Research lanes R1-R9 are parallelizable in the next research session after `workflow-plans/research.md` is created.

Fan-in rule for the next research session:
- Each lane must return facts, inferences, assumptions, risks, open points, and handoff implications.
- The orchestrator synthesizes lane findings into one research summary and records conflicts or missing inputs as specification blockers or research reopen targets.
- Lane output is advisory evidence, not final architecture authority.

## Planned Research Lanes For Next Session

| Lane | Role | Owned Question | Skill | Expected Output |
| --- | --- | --- | --- | --- |
| R1 | `architecture-agent` | What exact current `gonka-proxy` money paths, billing contracts, pricing/admission calls, and balance-writer responsibilities must the new architecture replace or interoperate with? | `go-architect-spec` | Current-state and boundary findings with source paths/contracts; no design decisions. |
| R2 | `domain-agent` | Which candidate architectures can actually enforce no-negative customer balances across reserve, finalize, write-off, reversal, duplicate, delayed, out-of-order, stale, replay, and ambiguous-outcome cases? | `go-domain-invariant-spec` | Invariant-focused option analysis, blockers, and required decisions. |
| R3 | `distributed-agent` | What Redpanda usage-event, command-event, inbox/outbox, idempotency, ordering, and replay patterns are viable without relying on broker exactly-once semantics for money correctness? | `go-distributed-architect-spec` | Distributed-consistency risks, viable patterns, rejected unsafe patterns, and evidence needs. |
| R4 | `data-agent` | What Postgres ledger, balance, hold, operation, idempotency, inbox/outbox, reconciliation, and projection state is required for each viable architecture option? | `go-data-architect-spec` | Data/source-of-truth questions, transaction boundaries, migration/cutover risks, and proof needs. |
| R5 | `api-agent` | What synchronous HTTP/internal API contracts and asynchronous event contracts would each viable option require between proxy, billing, pricing, identity/API-key, and future payments? | `api-contract-designer-spec` | Contract-surface options, provider-contract verification needs, status/error semantics, and compatibility risks. |
| R6 | `performance-agent` | What proxy request-path latency, throughput, contention, admission-cache, allowance-window, or preauthorization tradeoffs decide whether each option is acceptable? | `go-performance-spec` | Performance budgets, measurement questions, candidate bottlenecks, and proof obligations. |
| R7 | `reliability-agent` | How must billing, proxy, Redpanda, and Postgres behave under outages, retries, backpressure, stale reservations, process restarts, and partial failures for each viable option? | `go-reliability-spec` | Fail-closed/degraded-mode options, recovery requirements, timeout/retry policy questions, and rollout risks. |
| R8 | `security-agent` | What trust-boundary, authn/authz, tenant-isolation, secret-handling, privacy, abuse-resistance, and safe-telemetry constraints must gate the architecture? | `go-security-spec` | Security/privacy constraints and approval blockers only. |
| R9 | `qa-agent` | What validation strategy is required to prove money correctness, event replay safety, no-negative balances, concurrency behavior, privacy, and proxy performance? | `go-qa-tester-spec` | Test-plan inputs, proof layers, and validation blockers. |

The next research session may merge or split lanes only if it records a concrete scoped-down or expanded rationale in `workflow-plans/research.md`.

## Later Challenge And Review Expectations

- Workflow plan adequacy challenge: required in this phase after draft workflow files exist.
- Formal spec clarification challenge: expected during specification because this is broad protected-domain work. Use the default multi-lens challenge unless the specification phase records an eligible scoped-down rationale.
- Technical design review: mandatory after separate design depth is written and before planning.
- Task-ledger review/readiness: mandatory after `tasks.md` is drafted and before implementation.

## Completion Marker

Complete only when:
- `../workflow-plan.md` and this file both mark workflow planning complete;
- adequacy challenge is complete and reconciled, or the phase is explicitly blocked;
- research mode and planned lanes are explicit;
- next-session read order, objective, expected outputs, blockers/assumptions, and stop rule are explicit;
- no later-phase work has started.

## Stop Rule

Stop after workflow planning. Do not run R1-R9 research lanes, write `research/*.md`, write or approve `spec.md`, create design files, write `tasks.md`, run tests, create migrations, generate SQL, or edit runtime code in this session.

## Adequacy Challenge Result

Status: complete.

Read-only challenger:
- Role: `challenger-agent`
- Skill: `workflow-plan-adequacy-challenge`
- Scope: master workflow plan and this phase-local workflow-planning file only.

Finding:
- A1: master workflow plan had the next-session context bundle still pending, which blocked honest handoff.

Resolution:
- Repaired in `../workflow-plan.md` by adding the concrete next-session context bundle and marking workflow planning complete.
- The copy-pastable next-session prompt is rendered in final chat only, per repository handoff rules.

Open adequacy blockers:
- None.

## Next Action

Start the research phase in the next session. Create `workflow-plans/research.md`, run and reconcile R1-R9 as read-only research lanes unless a recorded scoped-down or expanded rationale changes the lane set, preserve compact findings under `research/`, update workflow state, and stop before specification.
