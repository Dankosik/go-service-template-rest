# Workflow Planning Phase

## Session Objective

Create the durable routing packet for repairing all eleven B01 findings, prove the selected shape and artifact expectations are adequate, and hand off exactly one research phase without beginning it.

## Phase State

- Current phase: `workflow-planning`.
- Phase status: `complete`.
- Execution shape: `full orchestrated`.
- Research mode: `fan-out`, next session only.
- Master: `specs/execution-shape-routing-hardening/workflow-plan.md`.
- Session boundary reached: `yes`.
- Ready for next session: `yes`.
- Subagent gate: `complete`; read-only adequacy lane `A1` returned, both blocking findings were reconciled, and the orchestrator recorded the readiness consequence and reopen target below.
- Local blockers: none.

## Decision Frontier

This phase decides only:

- that all eleven B01 findings are the accepted repair scope;
- that full-orchestrated, multi-session control is required;
- which evidence questions the next research session owns;
- which artifacts are expected, conditional, or not expected;
- where this session stops.

It does not choose the repaired wording, status model, execution actor, compatibility strategy, eval implementation, or mirror policy.

## Next Research Lanes

| Lane | Role | Owned question | Finding coverage | Skill | Inspect-first evidence | Expected output | Mode |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `R1` | workflow authority researcher | What single authority and routing model can make shape classification, dedicated workflow planning, direct-code execution, and subagent authorization coherent without weakening existing safety gates? | `B01-F01`, `B01-F02`, `B01-F07` | `no-skill` | `AGENTS.md`, router, artifact model, shared handoff contract, workflow-planning skill, worker rules | Anchored contradiction matrix, viable repair models, rejected alternatives, compatibility consequences, and reopen points; no edits. | read-only |
| `R2` | workflow state researcher | What typed state/transition model can unify artifact expectation/lifecycle, phase/gate/session states, reclassification, research-skip routing, phase-file triggers, and artifact-first resume? | `B01-F03`, `B01-F04`, `B01-F05`, `B01-F08`, `B01-F10` | `no-skill` | artifact model, router resume rules, shared handoff, workflow-planning skill and references, workflow-status skill/evals | Anchored current-state map, candidate state machines, legal transitions, stale-state behavior, compatibility consequences, and proof obligations; no edits. | read-only |
| `R3` | enforcement researcher | What semantic guardrails, adequacy checks, eval cases, CI hooks, and mirror-availability contract are needed to make the repaired routing behavior regression-protected? | `B01-F06`, `B01-F09`, `B01-F11` | `no-skill` | adequacy skill/references, challenger agent, required guardrails, Make/CI, sync scripts, eval manifests, gitignore | Coverage matrix with fail-before cases, canonical/mirror ownership, exact enforcement gaps, candidate minimal checks, and missing-proof boundaries; no edits. | read-only |
| `R4` | pattern-fit researcher | Which established workflow/state-machine/status-model patterns fit this repository's routing problem, and which should be rejected before a custom repair contract is approved? | cross-cutting `B01-F02`-`B01-F06`, `B01-F09`-`B01-F10` | `no-skill` | current routing/state contracts plus concrete external pattern descriptions and real-use examples | Pattern Fit Diligence record: candidates, evidence, applicability, repository fit, rejected patterns, and custom-design justification if needed; no edits. | read-only |

## Order And Parallelism

1. Start `R1`-`R4` in parallel after the research session loads the accepted packet.
2. Do not let any lane edit repository files or make final policy decisions.
3. Wait for every required lane; a missing lane keeps research fan-in blocked.
4. The orchestrator compares conflicting assumptions and writes durable lane evidence plus `research/synthesis.md`.
5. Synthesis must map every `B01-F01`-`B01-F11` ID to evidence, candidate decisions, rejected alternatives, proof needs, and the owning specification section.
6. Stop research after durable fan-in; do not write `spec.md` in the research session.

## Fan-In Rule

- Final authority: orchestrator.
- Do not accept a repair model that closes one finding by reopening another without explicit disposition.
- Preserve canonical ownership: `AGENTS.md` for hard invariants, router for loading, shared docs for detailed mechanics, skills for procedure, agent configs for mode routing, and scripts/evals for enforcement.
- Distinguish current confirmed facts, candidate repair decisions, compatibility constraints, proof obligations, and missing evidence.
- If the research packet cannot define one coherent target state for all eleven findings, keep research blocked and name the smallest unresolved owner decision.

## Workflow Plan Adequacy Challenge

- Lane: `A1`.
- Role: read-only challenger.
- Owned question: Are the master and phase-local workflow plans sufficient and internally consistent for a full-orchestrated research handoff that preserves every B01 finding without starting research early?
- Skill: `workflow-plan-adequacy-challenge`.
- Inspect first:
  - `specs/execution-shape-routing-hardening/workflow-plan.md`;
  - `specs/execution-shape-routing-hardening/workflow-plans/workflow-planning.md`;
  - `AGENTS.md`;
  - `docs/spec-first-workflow.md`;
  - `docs/spec-first-workflow/shared/artifact-model.md`.
- Read-only enforcement: no file edits, git mutation, research, specification, design, tasking, or implementation.
- Expected output: adequacy summary, classified findings, smallest exact orchestrator additions, handoff recommendation, and confidence.
- Status: `complete`.
- Result: the selected full-orchestrated shape and `B01-F01`-`B01-F11` coverage were confirmed. Two blocking control findings were reconciled: final phase/gate/blocker state is now explicit in both files, and the research handoff now loads the owning research plus shared subagent/handoff contracts before activating `workflow-plans/research.md` and spawning `R1`-`R4`.
- Orchestrator fan-in: both findings repaired; no waiver, accepted risk, or unresolved severity conflict remains.
- Readiness consequence: the next session may start research only through the recorded artifact-first loading and phase-local control activation.
- Reopen target: `workflow-planning` if the next session cannot preserve the recorded shape, finding coverage, lane set, artifact expectations, or stop rule.

## Completion Marker

This phase is complete only when:

- master and phase-local files agree on objective, shape, research mode, artifact expectations, blockers, next-session start, and stop rule;
- every B01 finding maps to a research owner;
- the adequacy challenge has returned and every blocking finding is repaired, explicitly accepted as risk, or leaves the phase blocked;
- both files record `Phase status: complete`, `Workflow-plan adequacy challenge: complete`, `Subagent gate: complete`, and `Blockers: none`;
- lane `A1` records `Status: complete`, its reconciled result, readiness consequence, and reopen target;
- the master records `Session boundary reached: yes`, `Ready for next session: yes`, and `Next session starts with: research`;
- no research, specification, design, test design, task breakdown, implementation, or validation work has started.

## Next Action

Stop this session and return a chat-only research-session prompt. The prompt must include the exact repository-owned `Subagent authorization:` line from `docs/spec-first-workflow/shared/subagents-and-handoff.md`. The next session must first load `docs/spec-first-workflow/phases/research.md` and that shared handoff contract, create or activate `workflow-plans/research.md`, then run `R1`-`R4` and stop before specification.

## Stop Rule

Stop after adequacy reconciliation and workflow-control verification. Do not start `R1`-`R4`, create research files, write `spec.md`, create design or test artifacts, draft `tasks.md`, or edit implementation surfaces in this session.
