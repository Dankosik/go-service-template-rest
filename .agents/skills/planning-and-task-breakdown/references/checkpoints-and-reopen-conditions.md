# Checkpoints And Reopen Conditions

## When To Load
Load this when writing review checkpoints, validation checkpoints, implementation-readiness handoff, blockers, or reopen conditions for specification, technical design, planning, or validation.

Use reopen conditions to stop unsafe guessing. They are not a license to create new workflow/process artifacts after implementation begins.

## Good Plan Snippet

```markdown
## Handoffs / Reopen Conditions

Implementation may start only when readiness is `PASS` or eligible `CONCERNS`.

Reopen `technical design` if:
- an implementation task needs a file/package ownership decision not present in `design/ownership-map.md`
- a task requires ordering that contradicts `design/sequence.md`
- a conditional artifact such as `test-plan.md` or `rollout.md` becomes necessary but was not created during planning

Reopen `specification` if:
- acceptance criteria require a product or behavior decision not already recorded in `spec.md`
```

Why it is good: it names earlier phases and tells the next session when to stop instead of improvising.

## Bad Plan Snippet

```markdown
## Risks

If something is missing, figure it out during coding. Add extra docs as needed.
```

Why it is bad: it invites post-code artifact invention and hides the reopen target.

## Good Task Ledger Example

```markdown
- [ ] T040 [Checkpoint] Verify every planned implementation phase has an existing phase-control file when the approved execution shape requires one. Depends on: phase plan drafted. Proof: `rtk rg --files specs/<task>/workflow-plans`.
- [ ] T041 [Checkpoint] Confirm readiness status and stop rule are recorded in `workflow-plan.md`, `workflow-plans/planning.md`, and compactly in `plan.md`. Depends on: T040. Proof: targeted diff read of those artifacts.
- [ ] T042 [Checkpoint] If any blocker requires a missing decision, mark readiness `FAIL` and name the earlier phase to reopen instead of writing code tasks. Depends on: T041. Proof: readiness section states the reopen target.
```

Why it is good: checkpoint work is executable and produces a clear handoff result.

## Bad Task Ledger Example

```markdown
- [ ] T040 [Checkpoint] Start implementation.
- [ ] T041 [Checkpoint] Add missing workflow docs if needed.
- [ ] T042 [Checkpoint] Reopen later.
```

Why it is bad: it starts the next phase, creates process artifacts late, and does not say which phase owns the missing context.

## Non-Findings To Avoid

- Do not flag conditional reopen conditions as blockers when they are clearly "if discovered later" stop rules.
- Do not require a checkpoint after every task; place checkpoints where risk or phase boundaries change.
- Do not treat `FAIL` readiness as a planning failure when it honestly prevents unsafe implementation.
- Do not require new `workflow-plans/validation-phase-N.md` after code starts; if it was required but missing, the correct action is to reopen planning.

## Exa Source Links
External links found with Exa and used only to calibrate checkpoint and traceability wording:

- Scrum Guide, commitments and Definition of Done: https://scrumguides.org/scrum-guide.html
- Martin Fowler on lightweight decision records and source-repository traceability: https://martinfowler.com/bliki/ArchitectureDecisionRecord.html
- MADR decision record lifecycle and confirmation hints: https://adr.github.io/madr/
