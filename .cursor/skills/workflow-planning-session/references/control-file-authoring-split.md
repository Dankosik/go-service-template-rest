# Control File Authoring Split

## Behavior Change Thesis
When loaded for symptom "I am writing or repairing workflow-control files and content is bleeding between master and phase-local files," this file makes the model put cross-phase status in `workflow-plan.md` and session-local orchestration in `workflow-plans/workflow-planning.md` instead of duplicating details, drifting into `spec.md`/`design/`/`plan.md`, or creating conflicting control sources.

## When To Load
Load this when the active decision is where a workflow-control detail belongs. If the active uncertainty is execution shape, research lanes, artifact status, or adequacy challenge routing, load that narrower reference instead.

## Decision Rubric
- `workflow-plan.md` owns cross-phase control: current phase, phase status, execution shape, research mode, session boundary, next-session start, blockers/assumptions, artifact status, phase-plan links, adequacy status, and later phase-file policy.
- `workflow-plans/workflow-planning.md` owns only this workflow-planning session: order of work, lane table for the next research session, challenge path, parallelism, fan-in rule, completion marker, blockers local to routing, next action, and stop rule.
- Master may summarize lanes by link or count; the phase file owns lane details.
- Phase file may mention artifact expectations only to keep the local handoff consistent; the master owns the status table.
- Neither file owns final domain decisions, research notes, `spec.md`, `design/`, `plan.md`, `tasks.md`, tests, migrations, or implementation status.
- For tiny direct-path work, the best authoring split is often no files: record the inline waiver and stop.

## Imitate

Master-file handoff:

```markdown
## Routing
Current phase: workflow-planning
Phase status: complete
Execution shape: full orchestrated
Research mode: fan-out, next session
Session boundary reached: yes
Ready for next session: yes
Next session starts with: research

## Phase Workflow Plans
- `workflow-plans/workflow-planning.md`: complete; lane details and stop rule recorded there

## Adequacy Challenge
Status: complete
Resolution: blocking findings reconciled
```

What to copy: the master is a cross-phase resume surface, not a lane workbook.

Phase-file handoff:

```markdown
## Local Orchestration
- Order: repair master, repair this phase file, run adequacy challenge, reconcile findings.
- Planned lanes for next session: L1 API, L2 data, L3 security, L4 reliability, L5 QA.
- Parallelizable work: L1-L5 are parallel in the next research session, not this one.
- Fan-in rule: compare assumptions and evidence before candidate synthesis.

## Completion Marker
Complete when master and phase file agree on execution shape, research mode, artifact expectations, blockers, adequacy status, and next-session start.

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
Artifact status: everything is approved except code.
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
