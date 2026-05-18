# Phase Strategy Examples

## Behavior Change Thesis
When loaded for unclear phase boundaries or session stop points, this file makes the model choose a target-state task ledger with real checkpoints instead of partial phased delivery, a giant "implement everything" phase, or a ceremony-only checkpoint.

## When To Load
Load this when choosing phase boundaries, session stops, review/validation checkpoints, or single-pass versus phased execution.

## Decision Rubric
- A checkpoint owns a coherent risk boundary, not a folder, role, or wish list.
- Prefer a reviewable target-state increment with a named stop rule. Split only when proof, dependency order, or a user-approved rollout constraint makes the split real.
- Use `tasks.md` for the executable handoff; keep strategy inside small, executable task slices instead of creating a separate strategy artifact.
- Add review, validation, rollout, or reconciliation checkpoints only when approved scope or risk triggers them.
- If a phase needs an architecture, API, data, security, reliability, rollout, or ownership decision not already approved, reopen the earlier phase instead of planning through it.

## Imitate
```markdown
### Checkpoint: Canonical Planning Contract
Objective: update only the authoritative planning-contract surfaces named in `design/component-map.md`.
Depends On: approved `spec.md`, approved core `design/` bundle.
Task Ledger Link / IDs: T001-T003.
Acceptance Criteria:
- the listed contract surfaces describe the same planning boundary
- no example text introduces a new architecture, API, data, security, reliability, or rollout decision
Change Surface:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
Planned Verification:
- `rtk rg -n "tasks.md|Implementation Readiness" AGENTS.md docs/spec-first-workflow.md`
- `rtk git diff --check`
Review / Checkpoint:
- stop after contract surfaces are aligned and compare them before planning downstream skill edits
Exit Criteria:
- downstream task ledger work can derive order from the approved contract without re-planning the phase
```

Copy the single objective, approved-artifact dependency, explicit change surface, and exit criterion. Do not copy the file names unless the task-local design names those surfaces.

## Reject
```markdown
### Phase 1: Implement Everything
Update docs, skills, code, tests, rollout, and validation. Do whatever looks necessary. Add any missing details while coding.
```

This fails because it mixes decisions, implementation, proof, and possible rollout without approved triggers or a real stop point.

## Agent Traps
- Do not treat sequential phases as "not parallel enough" when dependencies or checkpoint risk make ordering real.
- Do not require a separate rollout phase if approved decisions plus required design context do not trigger rollout work.
- Do not turn known in-scope cleanup into a future phase. If it is required for the target state, keep it in the accepted ledger or reopen design.
- Do not require every checkpoint to be independently releasable when the plan explicitly names the coupling and keeps the checkpoint honest.
- Do not treat an explicit session stop as overhead; for non-trivial work it is the handoff rule.
