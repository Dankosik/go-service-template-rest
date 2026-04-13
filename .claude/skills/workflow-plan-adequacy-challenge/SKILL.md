---
name: workflow-plan-adequacy-challenge
description: "Review generated workflow-control artifacts for task-specific handoff adequacy. Use inside a read-only challenger subagent after `workflow-plan.md` and the active `workflow-plans/<phase>.md` are generated or substantially repaired, before the orchestrator treats the phase plan as sufficient for non-trivial or agent-backed work."
---

# Workflow Plan Adequacy Challenge

## Purpose
Surface the workflow-control gaps that would make phase handoff dishonest or brittle.

This skill is a read-only challenge gate over `workflow-plan.md` and the active `workflow-plans/<phase>.md`. It gives the orchestrator compact findings to reconcile before handoff; it is not a workflow phase, an approval authority, or a replacement for `spec.md`, `design/`, or `tasks.md`.

## Scope
- inspect generated or substantially repaired workflow-control artifacts for the current task
- check consistency between the master `workflow-plan.md` and active `workflow-plans/<phase>.md`
- decide whether routing, stop rules, blockers, artifact expectations, lane ownership, completion criteria, and next-session handoff are sufficient for the task's risk and execution shape
- when the active phase is planning, check that implementation-readiness status and the planning handoff rule are explicit enough for the next session
- identify missing or unclear control details that could cause bad execution, premature handoff, or later chat archaeology
- keep findings actionable enough for the orchestrator to update the workflow-control artifacts directly

## Boundaries
Do not:
- edit files, create workflow artifacts, mutate git state, approve readiness, or change the task ledger or implementation handoff
- make final product, architecture, API, data, security, reliability, rollout, planning, or validation decisions
- turn `workflow-plans/<phase>.md` into a second `spec.md`, `design/`, or `tasks.md`
- demand generic checklist fields that do not change task execution quality
- reopen settled scope unless the workflow-control artifact cannot route the current task honestly
- treat this pass as a substitute for `spec-clarification-challenge`, pre-spec challenge, technical design, or task breakdown

## Required Input Bundle
Expect a compact bundle from the orchestrator:
- task goal, scope, non-goals, constraints, risk hotspots, and success checks
- execution shape and current phase
- current master `workflow-plan.md`
- active `workflow-plans/<phase>.md`
- any generated review or validation phase-control files whose adequacy affects handoff
- relevant artifact status for `spec.md`, `design/`, `tasks.md`, `test-plan.md`, or `rollout.md`
- implementation-readiness status and any recorded `CONCERNS`, `FAIL`, or `WAIVED` rationale when the current phase is planning
- recorded direct/local waiver or skip rationale, if the orchestrator claims the challenge is not required

If the bundle is too thin to review, return a blocking input-gap finding instead of guessing.

## What To Check
Keep only gaps whose absence could change execution quality, handoff safety, or resume reliability:
- master and phase files agree on current phase, phase status, blockers, session boundary, next action, next-session start point, and the master file's always-present next-session context bundle
- research mode is explicit when research is expected; fan-out lanes name role, owned question, and one skill or `no-skill`
- phase-local file records order or parallelism, fan-in or challenge path when relevant, completion marker, stop rule, local blockers, and parallelizable work
- artifact expectations are explicit and proportional: `spec.md`, `design/`, expected `tasks.md`, triggered `test-plan.md`, `rollout.md`, and planned review/validation phase files are approved, draft, missing, blocked, waived, or not expected
- when the current phase is planning, implementation readiness is recorded as `PASS`, `CONCERNS`, `FAIL`, or `WAIVED`; `CONCERNS` names accepted risks and proof obligations; `FAIL` names the earlier phase to reopen; `WAIVED` names rationale and scope
- blockers, assumptions, accepted risks, reopen targets, and user-decision needs are visible instead of hidden in optimistic handoff text
- the workflow-control artifacts contain enough task-specific routing for the next session to start without recreating workflow planning from chat
- the phase-local plan stays routing-only and does not duplicate final decisions, technical design, optional strategy notes, or the executable `tasks.md` ledger
- tiny/direct-path or lightweight-local skips include a real rationale instead of silently bypassing control artifacts

## Reference Routing
References are compact rubrics and example banks, not exhaustive checklists. Load at most one reference by default: choose the file that matches the suspected adequacy gap. Load multiple references only when the same bundle clearly spans independent decision pressures, such as both a lane-ownership gap and an implementation-readiness gap. Treat repository-local `AGENTS.md` plus `docs/spec-first-workflow.md` as authoritative.

| Symptom | Load | Behavior Change |
| --- | --- | --- |
| Master and active phase disagree about current phase, status, blockers, readiness, session boundary, or next-session start. | `references/master-phase-consistency-examples.md` | Makes the challenger require aligned routing fields in both workflow-control files instead of trusting the clearer artifact or chat intent. |
| Artifact expectations or statuses are missing, stale, overbroad, or not proportional to the execution shape. | `references/artifact-status-gap-examples.md` | Makes the challenger ask for status/rationale repair only, instead of demanding just-in-case artifacts or copying artifact content into workflow control. |
| Research mode, lane role, owned question, single skill, order/parallelism, or fan-in path is missing or muddy. | `references/lane-ownership-and-research-mode.md` | Makes the challenger split vague agent work into lane-level routing instead of telling the orchestrator to "use more agents." |
| Completion marker, stop rule, phase boundary, or "do not start next phase" handoff is weak. | `references/stop-rule-and-completion-marker.md` | Makes the challenger protect the phase boundary with a concrete stop/handoff rule instead of letting the session drift into the next phase. |
| Planning-phase implementation readiness is missing, misclassified, or lacks accepted risks, proof obligations, or reopen routing. | `references/planning-readiness-gap-examples.md` | Makes the challenger route readiness gaps to `PASS`, `CONCERNS`, `FAIL`, or `WAIVED` repair instead of treating "ready to code" prose as enough. |
| Finding severity is unclear, especially `blocks_phase_handoff` versus `blocks_specific_lane` versus `non_blocking_but_record`. | `references/non-blocking-vs-blocking-findings.md` | Makes the challenger classify by execution impact instead of making every imperfection block the phase or every gap a nit. |
| Workflow-control files duplicate `spec.md`, `design/`, or `tasks.md`, or a finding would ask them to do so. | `references/authority-boundary-and-duplication.md` | Makes the challenger recommend `trim_duplicate_authority` or a status-only repair instead of creating a second source of truth. |

## Classification
Use exactly one per finding:
- `blocks_phase_handoff` when the current phase cannot honestly be marked complete or ready for next session until repaired, explicitly waived, or accepted as risk
- `blocks_specific_lane` when one lane, artifact, blocker, or generated phase-control file needs repair but the whole phase may continue around it
- `non_blocking_but_record` when the concern should be recorded or accepted but does not block handoff

## Recommended Action
Use the smallest specific action that repairs the control gap:
- `add_missing_routing`
- `clarify_artifact_status`
- `clarify_readiness_status`
- `clarify_lane_ownership`
- `clarify_stop_or_completion_rule`
- `record_blocker_or_reopen`
- `record_skip_or_accepted_risk`
- `trim_duplicate_authority`
- `no_action`

## Deliverable Shape
Return:
- `Adequacy Summary`
- `Findings`
- `Handoff Recommendation`
- `Confidence`

For each finding include:
- `Gap`
- `Why It Matters`
- `What Could Fail`
- `Classification`
- `Recommended Action`
- `Exact Orchestrator Addition`
- `Evidence`

If no finding survives the filter, say there are no blocking adequacy gaps and name the evidence boundary. Avoid saying the plan is approved; only the orchestrator can decide handoff.

## Stop Condition
Stop when:
- every handoff-blocking workflow-control gap has been surfaced or the input gap is clearly blocking
- each finding has a classification, recommended action, and exact addition or clarification
- low-value checklist items have been pruned
- the output is short enough for the orchestrator to reconcile without reading a second workflow plan

## Anti-Patterns
- approving or rejecting the workflow plan as an authority
- padding findings with generic "add more detail" advice
- asking for `spec.md` or `tasks.md` content to be copied into a phase workflow plan
- treating all tiny tasks as if they need the full non-trivial artifact bundle
- ignoring a master/phase mismatch because the intended next step seems obvious from chat
- using the challenge to perform domain research, specification clarification, technical design, or task breakdown
