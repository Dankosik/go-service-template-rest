# Phase Strategy Examples

## When To Load
Load this when a planning pass needs examples for phase boundaries, session stops, checkpoints, phased versus single-pass execution, or the relationship between `plan.md` strategy and `tasks.md` ledger detail.

Use these snippets only after reading the approved task-local `spec.md + design/`. They calibrate shape, not decisions.

## Good Plan Snippet

```markdown
## Phase Plan

### Phase 1: Canonical Planning Contract
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
- `rtk rg -n "plan.md|tasks.md|Implementation Readiness" AGENTS.md docs/spec-first-workflow.md`
- `rtk git diff --check`
Review / Checkpoint:
- stop after contract surfaces are aligned and compare them before planning downstream skill edits
Exit Criteria:
- downstream task ledger work can derive order from the approved contract without re-planning the phase
```

Why it is good: the phase has one purpose, names its dependency on approved artifacts, keeps task details in `tasks.md`, and creates a real stop point.

## Bad Plan Snippet

```markdown
## Phase 1: Implement Everything

Update docs, skills, code, tests, rollout, and validation. Do whatever looks necessary. Add any missing details while coding.
```

Why it is bad: it mixes decisions, execution, validation, and possible rollout without approved triggers or a checkpoint.

## Good Task Ledger Example

```markdown
- [ ] T001 [Phase 1] Update the canonical contract files listed in `design/component-map.md` to carry the approved planning boundary wording. Depends on: none. Proof: `rtk rg -n "plan.md|tasks.md|Implementation Readiness" AGENTS.md docs/spec-first-workflow.md`.
- [ ] T002 [Phase 1] Compare those contract files for drift and remove duplicate wording that would make either file a competing source of truth. Depends on: T001. Proof: manual side-by-side diff read plus `rtk git diff --check`.
- [ ] T003 [Checkpoint] Record whether Phase 2 may start in the next session or whether `technical design` must be reopened. Depends on: T002. Proof: `workflow-plan.md` readiness/status update.
```

Why it is good: task IDs map to the phase, each task has proof, and the checkpoint does not pretend implementation may continue automatically.

## Bad Task Ledger Example

```markdown
- [ ] T001 [Phase 1] Implement planning changes across the repo.
- [ ] T002 [Phase 1] Add tests and rollout.
- [ ] T003 [Phase 1] Finish.
```

Why it is bad: it is not dependency ordered, does not name change surfaces, invents rollout work, and gives no proof obligation.

## Non-Findings To Avoid

- Do not flag sequential phases as "not parallel enough" when dependencies or checkpoint risk make ordering real.
- Do not require a separate rollout phase if the approved spec/design does not trigger rollout work.
- Do not require every phase to be independently releasable when the plan explicitly names the coupling and keeps the checkpoint honest.
- Do not treat an explicit session stop as overhead; for non-trivial work it is the handoff rule.

## Exa Source Links
External links found with Exa and used only to calibrate decomposition patterns:

- Feature slicing: https://techleadhandbook.org/agile/feature-slicing/
- Work in thin vertical slices: https://www.tomdalling.com/blog/mentoring/work-in-thin-vertical-slices/
- Scrum Guide, Sprint Backlog and Definition of Done context: https://scrumguides.org/scrum-guide.html
