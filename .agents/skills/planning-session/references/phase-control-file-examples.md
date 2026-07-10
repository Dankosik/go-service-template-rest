# Phase Control File Examples

## Behavior Change Thesis
When loaded for phase-control file work, this file makes the model create only routing-focused files for named phases instead of creating just-in-case controls or duplicating `tasks.md`, `spec.md`, or `design/`.

## When To Load
Load when `ROUTING-PHASE-CONTROL` requires writing or repairing `workflow-plans/planning.md`, or when planning must create future review or validation phase-control files for named multi-session routing.

## Decision Rubric
- A `ROUTING-PHASE-CONTROL`-triggered `workflow-plans/planning.md` records phase-local orchestration: canonical `phase_state`, typed outputs, blockers, pending task-review/readiness handoff, adequacy challenge packet, stop rule, and next action. When the trigger is false, do not create the file.
- Future review or validation phase-control files are allowed only when named multi-session routing requires them before implementation starts.
- Future files start as pending routing skeletons; after the named phase starts, they remain compact routing and progress surfaces, not full execution logs or new decision records.
- Review phase-control files should name review scope, read-only lanes, finding status, compact finding disposition, orchestrator reconciliation status, accepted risks, blockers or reopen targets, validation implications, completion marker, and stop rule.
- Validation phase-control files should name closeout claim, proof inputs, command/proof scope, phase status, blockers or reopen target, completion marker, next action, and stop rule.
- If a future phase-control file needs design facts that do not exist, block planning and reopen upstream instead of filling the gap.
- Put executable tasks and the task-review/readiness handoff in `tasks.md`, consume test depth from approved `test-plan.md` when `test-design` triggered it, and put rollout choreography in triggered `rollout.md`.

## Imitate
```markdown
Phase: planning
phase_state: complete
Completion marker: `tasks.md` has artifact_expectation=expected, artifact_state=review_ready, record_validity=current; task-review/readiness handoff recorded.
Allowed writes used: `tasks.md`, existing durable `workflow-plan.md`, and `workflow-plans/planning.md` because ROUTING-PHASE-CONTROL is satisfied.
Task-review/readiness procedural_gate_state: pending.
Task-review/readiness review_verdict: pending.
handoff_readiness: ready for the recorded task-review/readiness session; implementation remains unauthorized until that distinct gate completes.
Workflow plan adequacy procedural_gate_state: pending because an ADEQUACY-* condition is true; otherwise record the local deterministic matrix audit instead of a phase gate.
Stop rule: do not begin implementation in this session.
Next action: run task-review/readiness in a later session.
```

Copy this shape: the planning file stays phase-local and handoff oriented.

```markdown
Phase: review-phase-1
phase_state: not_started
Consumes: implemented scope from `tasks.md`, approved artifact bundle, and the diff for the named checkpoint.
Entry condition: implementation checkpoint complete with fresh local proof recorded.
Review scope: changed API, persistence, and reliability surfaces named in `tasks.md`.
Read-only lanes: one focused review question per lane, one skill or `no-skill` per lane.
Reconciliation rule: orchestrator records accepted findings, accepted risks, reopen targets, and validation implications; review agents do not edit files or approve decisions.
Finding disposition shape: finding ID or short label, source lane, disposition (`accepted`, `fixed in reconciliation`, `accepted risk`, `reopen`, or `no_action`), target task or artifact when applicable, validation implication.
Stop rule: do not create tasks, design notes, or implementation patches from review output; reopen the right phase if review exposes a missing decision.
Next action: run the named read-only review lanes.
```

Copy this shape: a review phase skeleton preserves what the next session must inspect and how findings become orchestrator-owned routing, without becoming a transcript or decision record.

```markdown
Phase: validation-phase-1
phase_state: not_started
Consumes: approved artifact bundle, existing `tasks.md`, and review phase notes for the named checkpoint when present.
Closeout claim: phase complete for T001-T006.
Proof scope: commands and manual checks named in `tasks.md` plus approved `test-plan.md` if test design triggered it.
Allowed future writes: `spec.md` Validation/Outcome, existing `tasks.md` progress, and this existing validation phase file only.
Stop rule: do not implement fixes or create missing process artifacts; reopen the narrowest earlier phase if proof fails or required artifacts are missing.
Next action: run fresh validation for the named proof scope.
```

Copy this shape only when planning already justified a dedicated validation phase.

## Reject
```markdown
Phase: validation-phase-1
phase_state: not_started
Created because validation is usually useful.
```

Failure: later phase files need an approved named phase, not a generic habit.

```markdown
Phase: review-phase-1
Findings: [full raw review transcript pasted here]
Tasks:
- Fix every comment above.
```

Failure: review control should store orchestrator-reconciled status and routing, not raw transcripts or a new task ledger.

## Agent Traps
- Turning `workflow-plans/planning.md` into a duplicate of `tasks.md`.
- Creating review or validation phase files "for completeness."
- Pasting raw review findings into `workflow-plans/review-phase-N.md` instead of recording orchestrator-reconciled status.
- Writing a coding control file that describes code already changed during planning.
- Creating a missing phase-control file during implementation instead of reopening planning.
