# Master Phase Consistency Examples

## Behavior Change Thesis
When loaded for symptom master and active phase files disagree on routing state, this file makes the model require aligned control fields in both files instead of likely mistake trusting the clearer artifact or chat intent.

## When To Load
Load this when `workflow-plan.md` and the active `workflow-plans/<phase>.md` disagree about routing identity/validity, current phase, typed phase/gate/handoff state, blockers, next action, or the next-session start point.

## Decision Rubric
- Classify as `blocks_phase_handoff` when routing revisions differ, either record is stale, or the mismatch could start, resume, or close the wrong phase.
- Classify as `blocks_specific_lane` when only one generated phase-control file or lane route is stale and the active phase can keep repairing around it.
- Classify as `non_blocking_but_record` when wording differs but both files route to the same current phase, blocker, and next-session start.
- Ask for the smallest routing repair in the owning file or both files. Do not ask to copy `spec.md`, `design/`, or `tasks.md` content into workflow control.

## Imitate
### Wrong active phase
`Gap`: Master says `Current phase: go-code-ownership-design`, but the active phase file is `workflow-plans/planning.md`.

Why to copy: it ties the mismatch to a concrete failure, skipping design approval and starting task breakdown from the wrong phase.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `add_missing_routing`
- `Exact Orchestrator Addition`: In `workflow-plan.md`, set `Current phase: go-code-ownership-design`, `phase_state=active`, and typed current routing identity; record `workflow-plans/go-code-ownership-design.md` as the active phase file only when `ROUTING-PHASE-CONTROL` requires it. In that triggered file set `phase_state=active`, the same routing identity, `Next action: finish Go code ownership design`, and `Stop rule: do not begin planning in this session`.

### False handoff readiness
`Gap`: Master says `session_boundary=reached` and `handoff_readiness=ready`, but the phase file says `phase_state=active` with unresolved blockers.

Why to copy: it blocks the handoff because the master invites a later session forward while the phase-local file still says stop.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `record_blocker_or_reopen`
- `Exact Orchestrator Addition`: In `workflow-plan.md`, set `session_boundary=reached`, add a routing-only blocker summary plus the reopen target, and use `handoff_readiness=ready` only when that one reconciliation session can start; otherwise keep `handoff_readiness=blocked` and state the missing prerequisite.

### Missing next-session start
`Gap`: Master says `handoff_readiness=ready` without naming where the next session starts.

Why to copy: resume should not require chat archaeology.

Use:
- `Classification`: `blocks_phase_handoff`
- `Recommended Action`: `add_missing_routing`
- `Exact Orchestrator Addition`: Add the exact next phase, such as `Next session starts with: technical design review` when separate design depth needs review, or `Next session starts with: planning, beginning with tasks.md breakdown from specification-review-approved spec.md plus reviewed design context` after the review gate is reconciled.

## Reject
- "The plan is approved after this is fixed." This crosses challenger authority.
- "Copy the full design sequence into `workflow-plans/system-integration-design.md`." This turns the phase workflow file into a second design artifact.
- "Add more detail about everything." This does not name the mismatched field or the smallest repair.

## Agent Traps
- Do not ignore a master/phase mismatch because chat intent looks obvious.
- Do not repair by inventing a new phase file path; ask the orchestrator to align the recorded route.
- Treat noncanonical new-output synonyms as a typed-state gap; only exact closed legacy mapping is eligible for historical reads.
