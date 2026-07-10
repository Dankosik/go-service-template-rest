# Control File Authoring Split

## Behavior Change Thesis
When loaded for symptom "I am writing or repairing workflow-control files and content is bleeding between master and phase-local files," this file makes the model put cross-phase status in `workflow-plan.md` and session-local orchestration in `workflow-plans/workflow-planning.md` instead of duplicating details, drifting into `spec.md`/`design/`/`tasks.md`, or creating conflicting control sources.

## When To Load
Load this when the active decision is where a workflow-control detail belongs. If the active uncertainty is execution shape, research lanes, artifact status, or adequacy challenge routing, load that narrower reference instead.

## Decision Rubric
- `workflow-plan.md` owns cross-phase control: matched `SHAPE-*` evidence, routing identity/validity, typed current phase/session/handoff state, research expectation/mode, next-session start, blockers/assumptions, typed artifact state, phase-plan links, adequacy state, and later phase-file policy.
- When `ROUTING-PHASE-CONTROL` is satisfied, `workflow-plans/workflow-planning.md` owns only this session: the same routing identity, order of work, next-phase lane table, challenge path, parallelism, fan-in rule, completion marker, local blockers, next action, and stop rule. Otherwise no phase file is created and compact state stays in the master.
- Master and phase control must carry the same current `(routing_scope,routing_revision)` before `handoff_readiness=ready`; mismatch or staleness blocks handoff.
- Master may summarize lanes by link or count; a triggered phase file owns lane details.
- A triggered phase file may mention artifact expectations only to keep the local handoff consistent; the master owns the status table.
- Neither file owns final domain decisions, research notes, `spec.md`, `design/`, `tasks.md`, tests, migrations, or implementation status.
- For tiny direct-path work, the best authoring split is no files: record the current-session direct envelope and `artifact_expectation=not_expected` consequences, then stop.

## Imitate

Master-file handoff:

```markdown
## Routing
Current phase: workflow-planning
phase_state: complete
execution_shape: full_orchestrated
Research mode: fan-out, next session
session_boundary: reached
handoff_readiness: ready
Next session starts with: research

## Phase Workflow Plans
- `workflow-plans/workflow-planning.md`: complete; lane details and stop rule recorded there

## Adequacy Challenge
procedural_gate_state: complete
Resolution: blocking findings reconciled
```

What to copy: the master is a cross-phase resume surface, not a lane workbook.

Phase-file handoff:

```markdown
## Local Orchestration
- Order: repair master, repair this phase file, run adequacy challenge, reconcile findings.
- Planned concurrent lanes for next session: L1 API, L2 data, L3 security.
- Deferred dependent checks: reliability and QA stay in root synthesis unless fan-in exposes a new independent question; any later lane runs sequentially.
- Parallelizable work: L1-L5 are parallel in the next research session, not this one.
- Fan-in rule: compare assumptions and evidence before candidate synthesis.

## Completion Marker
Complete when the master and every triggered phase file agree on execution shape, research mode, artifact expectations, blockers, adequacy state, and next-session start.

## Stop Rule
Stop before starting L1-L5 research.
```

What to copy: the phase file is a session-local operating note with a hard stop.

## Reject

```markdown
workflow-plan.md:
- API decision: use cursor pagination.
- Data decision: add an exports table.
- Implementation Phase 1: create migration and handler.
```

Failure: turns workflow control into spec/design/planning output.

```markdown
workflow-plans/workflow-planning.md:
Artifact state: everything is approved except code.
```

Failure: invents cross-phase status and bypasses the master control surface.

```markdown
Both files:
Full lane table, artifact matrix, blockers, next action, and raw adequacy transcript copied verbatim.
```

Failure: duplication creates drift; record the durable summary once and link/summarize across files.

## Agent Traps
- Creating a phase file because the skill exists even though direct-path inline routing is enough.
- Letting the phase file become the only source of truth for cross-phase readiness.
- Pasting raw subagent findings instead of the orchestrator's reconciled status.
- Writing "complete" in the master while the phase file still says blocking findings are open.
