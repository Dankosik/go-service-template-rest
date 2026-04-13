# Phase Control File Examples

## Behavior Change Thesis
When loaded for phase-control file work, this file makes the model create only routing-focused files for named phases instead of creating just-in-case controls or duplicating `tasks.md`, `spec.md`, or `design/`.

## When To Load
Load when writing or repairing `workflow-plans/planning.md`, or when planning must create future review or validation phase-control files for named multi-session routing.

## Decision Rubric
- `workflow-plans/planning.md` records phase-local orchestration: status, outputs, blockers, readiness, adequacy challenge, stop rule, next action.
- Future review or validation phase-control files are allowed only when named multi-session routing requires them before implementation starts.
- Future files start as pending routing skeletons; after the named phase starts, they remain compact routing and progress surfaces, not full execution logs or new decision records.
- Review phase-control files should name review scope, read-only lanes, finding status, compact finding disposition, orchestrator reconciliation status, accepted risks, blockers or reopen targets, validation implications, completion marker, and stop rule.
- Validation phase-control files should name closeout claim, proof inputs, command/proof scope, phase status, blockers or reopen target, completion marker, next action, and stop rule.
- If a future phase-control file needs design facts that do not exist, block planning and reopen upstream instead of filling the gap.
- Put executable tasks and the implementation handoff in `tasks.md`, test depth in triggered `test-plan.md`, and rollout choreography in triggered `rollout.md`.

## Imitate
```markdown
Phase: planning
Phase status: complete
Completion marker: `tasks.md` approved; readiness gate recorded.
Allowed writes used: `tasks.md`, `workflow-plan.md`, `workflow-plans/planning.md`.
Implementation readiness: PASS.
Workflow plan adequacy challenge: completed; blocking findings reconciled.
Stop rule: do not begin implementation in this session.
Next action: start T001 in a later session.
```

Copy this shape: the planning file stays phase-local and handoff oriented.

```markdown
Phase: review-phase-1
Phase status: pending
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
Phase status: pending
Consumes: approved artifact bundle, existing `tasks.md`, and review phase notes for the named checkpoint when present.
Closeout claim: phase complete for T001-T006.
Proof scope: commands and manual checks named in `tasks.md` plus triggered `test-plan.md` if present.
Allowed future writes: `spec.md` Validation/Outcome, existing `tasks.md` progress, `workflow-plan.md`, and this existing validation phase file only.
Stop rule: do not implement fixes or create missing process artifacts; reopen the narrowest earlier phase if proof fails or required artifacts are missing.
Next action: run fresh validation for the named proof scope.
```

Copy this shape only when planning already justified a dedicated validation phase.

## Reject
```markdown
Phase: validation-phase-1
Phase status: pending
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
