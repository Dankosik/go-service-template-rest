# Acceptance Criteria And Proof Obligations

## When To Load
Load this when acceptance criteria, planned verification, manual checks, implementation-readiness `CONCERNS`, or proof obligations feel vague or disconnected from the task ledger.

Acceptance criteria should say what must be true. Proof obligations should say how the session will know.

## Good Plan Snippet

```markdown
## Phase Plan

### Phase 3: Verification And Handoff
Objective: prove the planned changes satisfy the approved decision record without broadening scope.
Task Ledger Link / IDs: T030-T034.
Acceptance Criteria:
- every changed surface named in `design/component-map.md` is either updated or explicitly deferred in `plan.md`
- `tasks.md` proof commands match the changed surfaces
- readiness is `PASS` or `CONCERNS` with named accepted risks and proof obligations
Planned Verification:
- targeted command for each changed package or artifact surface
- `rtk git diff --check`
- manual read for artifact-boundary drift
Review / Checkpoint:
- stop before implementation if any proof requires an unapproved design, rollout, or compatibility decision
```

Why it is good: it separates acceptance from proof and ties both to approved artifacts.

## Bad Plan Snippet

```markdown
## Phase 3: Validate

Acceptance: looks good.
Proof: run tests.
Readiness: should be fine.
```

Why it is bad: it gives no task-specific condition, no command scope, and no readiness evidence.

## Good Task Ledger Example

```markdown
- [ ] T030 [Phase 3] Run the targeted verification command named for the changed package or artifact surface. Depends on: implementation tasks complete. Proof: paste command name and pass/fail result into the validation surface required by the workflow.
- [ ] T031 [Phase 3] Run `rtk git diff --check` for whitespace and patch hygiene. Depends on: implementation tasks complete. Proof: command exits 0.
- [ ] T032 [Phase 3] Re-read `plan.md` acceptance criteria against the final diff and record any deferred item as a blocker or accepted `CONCERNS` obligation. Depends on: T030, T031. Proof: readiness status updated in the owning artifact.
```

Why it is good: proof is concrete, not just "tests pass", and failed proof routes to blockers or `CONCERNS`.

## Bad Task Ledger Example

```markdown
- [ ] T030 [Phase 3] Check everything.
- [ ] T031 [Phase 3] Make sure quality is good.
- [ ] T032 [Phase 3] Mark PASS.
```

Why it is bad: it cannot be executed or reviewed, and it assumes the readiness result.

## Non-Findings To Avoid

- Do not require a new automated test for a docs-only or skill-only change when manual diff/read checks are the right proof.
- Do not require `rtk go test ./...` for every tiny local documentation edit; match proof scope to risk and change surface.
- Do not reject `CONCERNS` when it names accepted risks and proof obligations clearly.
- Do not treat acceptance criteria and Definition of Done as the same layer; keep task-specific acceptance separate from repository-wide quality gates.

## Exa Source Links
External links found with Exa and used only to calibrate acceptance and proof wording:

- Scrum Guide, Definition of Done: https://scrumguides.org/scrum-guide.html
- Scrum.org, Definition of Done versus Acceptance Criteria: https://www.scrum.org/resources/blog/what-difference-between-definition-done-and-acceptance-criteria
- Scrum.org, What is a Definition of Done: https://www.scrum.org/resources/what-definition-done
