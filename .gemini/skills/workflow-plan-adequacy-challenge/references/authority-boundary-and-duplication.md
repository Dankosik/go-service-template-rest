# Authority Boundary And Duplication

## Behavior Change Thesis
When loaded for symptom workflow-control files duplicate canonical artifact content, this file makes the model recommend status-only routing or `trim_duplicate_authority` instead of likely mistake creating a second `spec.md`, `design/`, or `tasks.md`.

## When To Load
Load this as smell triage only when `workflow-plan.md`, `workflow-plans/<phase>.md`, or a finding would copy or summarize too much from canonical artifacts. Do not load it as the default reference when a narrower status, lane, stop-rule, or readiness gap matches.

## Decision Rubric
- `workflow-plan.md` owns cross-phase routing, artifact status, blockers, adequacy challenge status, next-session routing, and implementation-readiness status.
- `workflow-plans/<phase>.md` owns phase-local orchestration: lanes, order/parallelism, fan-in/challenge path, completion marker, stop rule, next action, blockers, and local adequacy challenge resolution.
- `spec.md` owns final decisions; `design/` owns technical design; `tasks.md` owns executable task state and implementation handoff.
- If workflow control contains canonical artifact content, recommend trimming it to a status, link/path, blocker, or next action. Do not ask for a larger duplicate summary.

## Imitate
### Phase plan became a second design file
`Gap`: `workflow-plans/technical-design.md` contains the full component map and sequence instead of a design artifact status plus next action.

Why to copy: the problem is duplicate authority, not missing design content.

Use:
- `Classification`: `blocks_specific_lane` if it affects only the generated phase-control file; `blocks_phase_handoff` if handoff relies on the duplicate content as the only design record
- `Recommended Action`: `trim_duplicate_authority`
- `Exact Orchestrator Addition`: Move design detail back to `design/`; in `workflow-plans/technical-design.md`, keep `Artifact status: design/ draft|approved`, `Completion marker: required design artifacts approved`, and `Next action: update master with design status`.

### Master contains task ledger
`Gap`: `workflow-plan.md` embeds the implementation checklist from `tasks.md`.

Why to copy: the master should support resume routing, not become executable task state.

Use:
- `Classification`: `non_blocking_but_record` if `tasks.md` exists and routing is clear; `blocks_phase_handoff` if the duplicate checklist is the only task ledger for non-trivial work
- `Recommended Action`: `trim_duplicate_authority`
- `Exact Orchestrator Addition`: Replace the checklist with `tasks.md: approved at tasks.md; Next session starts with: T001`.

### Finding asks for copied artifact content
`Gap`: Proposed finding says to copy all accepted risks and validation cases from `tasks.md` into `workflow-plans/planning.md`.

Why to copy: the safe repair is a compact pointer and gate result, not duplicated planning detail.

Use:
- `Classification`: classify the underlying missing handoff rule, not the temptation to duplicate content
- `Recommended Action`: `trim_duplicate_authority`
- `Exact Orchestrator Addition`: In `workflow-plans/planning.md`, record `Gate result: CONCERNS; proof obligation: see tasks.md readiness note; Stop or handoff rule: implementation may start with named concern only`.

## Reject
- "Copy the whole spec decision table into `workflow-plan.md` so the next session has context." The spec is the decision record.
- "Put every implementation task in the phase stop rule." The stop rule is not the task ledger.
- "Create a new summary artifact to reconcile the duplicate content." The repair is to restore ownership, not add another source of truth.

## Agent Traps
- Do not confuse "resume-friendly" with "self-contained copy of every artifact."
- Do not remove the only useful routing field while trimming detail; preserve status, blocker, next action, and owning path.
- Do not mark duplication as harmless when later handoff would rely on the duplicate instead of the canonical artifact.
