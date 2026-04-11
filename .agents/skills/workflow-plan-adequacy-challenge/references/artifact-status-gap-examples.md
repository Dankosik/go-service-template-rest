# Artifact Status Gap Examples

## When To Load
Load this when artifact expectations or statuses are missing, stale, too vague, or inconsistent across `workflow-plan.md`, `workflow-plans/<phase>.md`, and the task's execution shape.

## Authoritative Inputs
- `AGENTS.md`: the master records artifact status and whether later `design/`, `plan.md`, `tasks.md`, `test-plan.md`, or `rollout.md` artifacts are expected.
- `docs/spec-first-workflow.md`: expected statuses include `approved`, `draft`, `missing`, `blocked`, `waived`, or `not expected` where appropriate; do not create just-in-case artifacts.

## Good Findings
- `Gap`: `tasks.md` is expected in the master, but status is absent and planning handoff says implementation may start.
  `Why It Matters`: Non-trivial implementation could begin without the executable task ledger the repo requires by default.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `clarify_artifact_status`.
  `Exact Orchestrator Addition`: In `workflow-plan.md`, add `tasks.md: missing; blocker: planning must create or explicitly waive tasks.md before implementation readiness`; in `workflow-plans/planning.md`, add `Stop rule: do not hand off to implementation until tasks.md is approved or eligible waiver recorded`.
- `Gap`: `rollout.md` is listed as missing, but no trigger explains why rollout detail is expected.
  `Why It Matters`: The plan may be forcing a just-in-case artifact instead of task-specific control.
  `Classification`: `non_blocking_but_record`.
  `Recommended Action`: `clarify_artifact_status`.
  `Exact Orchestrator Addition`: Add `rollout.md: not expected; rationale: no migration, mixed-version, delivery sequencing, or rollback choreography change`.
- `Gap`: `design/` is marked approved in the master, but the technical-design phase file still says `design/ownership-map.md` is draft.
  `Why It Matters`: Planning could start while a required design artifact is still incomplete.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `clarify_artifact_status`.
  `Exact Orchestrator Addition`: Align both files to `design/: draft; missing approval: ownership-map.md`; keep the design content in `design/`, not in workflow control.

## Bad Findings
- "Add the whole `tasks.md` checklist to `workflow-plan.md`." Bad because the master tracks status and routing, not executable task state.
- "Create `test-plan.md` just to be complete." Bad because conditional artifacts need real triggers.
- "The plan is good once all statuses are green." Bad because the challenger does not approve the plan.

## Blocker Classification Examples
- `blocks_phase_handoff`: required non-trivial artifact is missing, stale, or claimed approved while a phase-local blocker remains.
- `blocks_specific_lane`: one optional generated phase-control file has unclear status, but the current phase can repair that file without blocking other lanes.
- `non_blocking_but_record`: an optional artifact is correctly not expected but should record the trigger rationale so later sessions do not guess.

## Exact Orchestrator Additions
- `workflow-plan.md`: `Artifacts: spec.md approved|draft|blocked|waived; design/ approved|draft|missing|waived; plan.md approved|draft|missing; tasks.md expected approved|draft|missing|waived|not expected; test-plan.md not expected|missing|approved; rollout.md not expected|missing|approved`.
- `workflow-plans/<phase>.md`: `Artifact updates this phase: <artifact status only>; Completion marker: <artifact status required to leave phase>; Stop rule: do not create or duplicate content outside the artifact that owns it`.

## Exa Source Links
Exa MCP was attempted before authoring, but this environment returned a 402 credits-limit error. These fallback links are calibration only; repository-local docs remain authoritative.
- [Asana progress report template](https://asana.com/templates/progress-report) for separating current status, risks, and next action items.
- [Asana status report template](https://asana.com/templates/status-report) for keeping stakeholders aligned on status, blockers, and next steps.
- [Smartsheet project handover templates](https://www.smartsheet.com/content/project-handover-templates) for handover sections that include status, scope, risks, constraints, assumptions, and remaining work.
