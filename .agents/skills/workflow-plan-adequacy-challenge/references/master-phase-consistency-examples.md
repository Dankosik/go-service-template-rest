# Master Phase Consistency Examples

## When To Load
Load this when `workflow-plan.md` and the active `workflow-plans/<phase>.md` disagree about current phase, phase status, blockers, session boundary, ready-for-next-session state, next action, or the next-session start point.

## Authoritative Inputs
- `AGENTS.md`: `workflow-plan.md` is the cross-phase control artifact; `workflow-plans/<phase>.md` is phase-local only.
- `docs/spec-first-workflow.md`: read the master first, then the current phase workflow plan; a missing or mismatched phase file is incomplete control, not something to reconstruct from chat.

## Good Findings
- `Gap`: Master says `Current phase: technical-design`, but active phase file is `workflow-plans/planning.md`.
  `Why It Matters`: A later session could skip design approval and start task breakdown from the wrong phase.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `add_missing_routing`.
  `Exact Orchestrator Addition`: In `workflow-plan.md`, set `Current phase: technical-design`; set `Phase workflow plans: technical-design active; planning pending`; in `workflow-plans/technical-design.md`, set `Next action: finish and approve design bundle; Stop rule: do not begin planning in this session`.
- `Gap`: Master says `Session boundary reached: yes`, but phase file says `Phase status: in_progress` with unresolved blockers.
  `Why It Matters`: The master invites a new session to move forward while the phase-local file still blocks handoff.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `record_blocker_or_reopen`.
  `Exact Orchestrator Addition`: In `workflow-plan.md`, set `Session boundary reached: no`, `Ready for next session: no`, and copy the blocker summary only as routing, not design detail.
- `Gap`: Master says `Ready for next session: yes` without naming where the next session starts.
  `Why It Matters`: Resume requires chat archaeology and can restart the wrong phase.
  `Classification`: `blocks_phase_handoff`.
  `Recommended Action`: `add_missing_routing`.
  `Exact Orchestrator Addition`: Add `Next session starts with: planning, beginning with plan.md strategy from approved spec.md + design/`.

## Bad Findings
- "The plan is approved after this is fixed." Bad because the challenger must not approve handoff.
- "Copy the full design sequence into `workflow-plans/technical-design.md`." Bad because the phase workflow file would become a second design artifact.
- "Add more detail about everything." Bad because it does not name the mismatched field or the smallest repair.

## Blocker Classification Examples
- `blocks_phase_handoff`: master and active phase point at different phases, or the master marks handoff ready while phase-local blockers remain.
- `blocks_specific_lane`: one generated `workflow-plans/review-phase-1.md` has stale next action, but the active planning phase can continue repairing it.
- `non_blocking_but_record`: wording differs, but both artifacts route to the same current phase, blocker, and next-session start.

## Exact Orchestrator Additions
- `workflow-plan.md`: `Current phase: <phase>; Phase status: in_progress|blocked|complete; Session boundary reached: yes|no; Ready for next session: yes|no; Next session starts with: <phase and first action>`.
- `workflow-plans/<phase>.md`: `Phase status: <status>; Completion marker: <phase-local marker>; Stop rule: <what not to start>; Next action: <single next routing action>; Blockers: <routing-only blockers or none>`.

## Exa Source Links
Exa MCP was attempted before authoring, but this environment returned a 402 credits-limit error. These fallback links are calibration only; repository-local docs remain authoritative.
- [Asana status report template](https://asana.com/templates/status-report) for status, blockers, and next steps in handoff-style updates.
- [Smartsheet project handover templates](https://www.smartsheet.com/content/project-handover-templates) for project status, remaining work, risks, constraints, and assumptions in handover documents.
- [Asana project closure template](https://asana.com/templates/project-closure) for documenting completion, approvals, records, and next steps at closeout.
