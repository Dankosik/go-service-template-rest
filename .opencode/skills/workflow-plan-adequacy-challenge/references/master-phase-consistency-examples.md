# Master Phase Consistency Examples

## Behavior Change Thesis
When loaded for symptom master and active phase files disagree on routing state, this file makes the model require aligned control fields in both files instead of likely mistake trusting the clearer artifact or chat intent.

## When To Load
Load this when `workflow-plan.md` and the active `workflow-plans/<phase>.md` disagree about current phase, phase status, blockers, readiness, session boundary, ready-for-next-session state, next action, or the next-session start point.

## Decision Rubric
- Classify as `blocks_phase_handoff` when the mismatch could start, resume, or close the wrong phase.
- Classify as `blocks_specific_lane` when only one generated phase-control file or lane route is stale and the active phase can keep repairing around it.
- Classify as `non_blocking_but_record` when wording differs but both files route to the same current phase, blocker, and next-session start.
- Ask for the smallest routing repair in the owning file or both files. Do not ask to copy `spec.md`, `design/`, `tasks.md`, or optional `plan.md` content into workflow control.

## Imitate
### Wrong active phase
`Gap`: Master says `Current phase: technical-design`, but the active phase file is `workflow-plans/planning.md`.

Why to copy: it ties the mismatch to a concrete failure, skipping design approval and starting task breakdown from the wrong phase.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `add_missing_routing`
- `Exact Orchestrator Addition`: In `workflow-plan.md`, set `Current phase: technical-design`; set `Phase workflow plans: technical-design active; planning pending`; in `workflow-plans/technical-design.md`, set `Next action: finish and approve design bundle; Stop rule: do not begin planning in this session`.

### False handoff readiness
`Gap`: Master says `Session boundary reached: yes`, but the phase file says `Phase status: in_progress` with unresolved blockers.

Why to copy: it blocks the handoff because the master invites a later session forward while the phase-local file still says stop.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `record_blocker_or_reopen`
- `Exact Orchestrator Addition`: In `workflow-plan.md`, set `Session boundary reached: no`, `Ready for next session: no`, and add a routing-only blocker summary.

### Missing next-session start
`Gap`: Master says `Ready for next session: yes` without naming where the next session starts.

Why to copy: resume should not require chat archaeology.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `add_missing_routing`
- `Exact Orchestrator Addition`: Add `Next session starts with: planning, beginning with tasks.md breakdown from approved spec.md + design/`.

## Reject
- "The plan is approved after this is fixed." This crosses challenger authority.
- "Copy the full design sequence into `workflow-plans/technical-design.md`." This turns the phase workflow file into a second design artifact.
- "Add more detail about everything." This does not name the mismatched field or the smallest repair.

## Agent Traps
- Do not ignore a master/phase mismatch because chat intent looks obvious.
- Do not repair by inventing a new phase file path; ask the orchestrator to align the recorded route.
- Do not treat status synonyms as a finding when the route, blocker, and next-session start are already unambiguous.
