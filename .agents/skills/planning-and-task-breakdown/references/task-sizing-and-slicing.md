# Task Sizing And Slicing

## When To Load
Load this when a phase or task is too broad, too horizontal, hard to review, hard to verify, or likely to require more than one focused coding session.

Sizing is subordinate to approved design. Split to reduce risk and proof ambiguity; do not split by inventing missing design choices.

## Good Plan Snippet

```markdown
## Phase Plan

### Phase 2: Small Reviewable Behavior Slice
Objective: land one approved behavior slice end to end across only the surfaces listed in `design/component-map.md`.
Depends On: Phase 1 source-of-truth update complete.
Task Ledger Link / IDs: T020-T024.
Acceptance Criteria:
- the slice can be reviewed without reading unrelated future slices
- the slice has a focused proof path
- any remaining work is explicitly out of scope for this phase
Planned Verification:
- targeted package or artifact command named by the design
- `rtk git diff --check`
Exit Criteria:
- next phase can start from a working baseline, not from half-integrated scaffolding
```

Why it is good: it narrows the slice and makes remaining work visible instead of stuffing it into one large phase.

## Bad Plan Snippet

```markdown
## Phase 2: Finish Feature

Implement all remaining behavior, update all docs, add every test, and handle future cleanup.
```

Why it is bad: it hides size, mixes independent surfaces, and makes acceptance too vague.

## Good Task Ledger Example

```markdown
- [ ] T020 [Phase 2] Update the narrow package or file surface named for this slice in `design/component-map.md`. Depends on: T010. Proof: targeted test or diff check named in `plan.md`.
- [ ] T021 [Phase 2] Add or update the smallest regression coverage that proves this slice's accepted behavior. Depends on: T020. Proof: targeted test command from the affected package.
- [ ] T022 [Phase 2] [P] Update the reference/example file that documents this exact slice, if the design lists it as a derived doc surface. Depends on: T020. Proof: `rtk rg -n "<approved behavior term>" <doc-surface-from-design>`.
- [ ] T023 [Checkpoint] Confirm no task title in Phase 2 still needs "and" to be accurate. Depends on: T020, T021, T022. Proof: manual ledger read.
```

Why it is good: each task is small, proof-bound, and either sequential or explicitly parallel after a shared dependency.

## Bad Task Ledger Example

```markdown
- [ ] T020 [Phase 2] Update handlers, storage, generated contracts, docs, tests, validation, and rollout. Depends on: none. Proof: go test ./...
```

Why it is bad: the task title needs a long list to be accurate, mixes independent concerns, and uses broad verification as a substitute for slicing.

## Non-Findings To Avoid

- Do not split a task just because it touches two files when those files form one invariant-preserving change.
- Do not reject a short horizontal enabling task when the approved design says it is the foundation needed before vertical slices can start.
- Do not force user-visible vertical slices onto purely internal refactors; still require small, proof-bound reviewable slices.
- Do not inflate `tasks.md` with micro-steps that a coder can safely decide inside one small task.

## Exa Source Links
External links found with Exa and used only to calibrate slicing and sizing heuristics:

- Feature slicing: https://techleadhandbook.org/agile/feature-slicing/
- Work in thin vertical slices: https://www.tomdalling.com/blog/mentoring/work-in-thin-vertical-slices/
- Scrum Guide, refinement into smaller precise items: https://scrumguides.org/scrum-guide.html
