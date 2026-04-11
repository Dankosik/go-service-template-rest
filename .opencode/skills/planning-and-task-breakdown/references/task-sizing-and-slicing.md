# Task Sizing And Slicing

## Behavior Change Thesis
When loaded for oversized or horizontal tasks, this file makes the model split work into reviewable, proof-bound slices instead of producing a catch-all task that hides independent surfaces behind one broad test command.

## When To Load
Load this when a phase or task is too broad, too horizontal, too vague, hard to review, hard to verify, or likely to require more than one focused coding session.

## Decision Rubric
- Split to reduce review and proof ambiguity, not to create paperwork.
- Prefer vertical behavior slices when the approved design supports them; use horizontal enabling tasks only when the design says they establish a needed foundation.
- Keep tightly coupled invariant-preserving edits together when splitting would create a half-safe state.
- A task that needs multiple independent nouns joined by "and" is usually too large.
- Do not split by inventing missing design choices; reopen design when a split depends on unapproved ownership or sequence decisions.

## Imitate
```markdown
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

Copy the explicit out-of-scope boundary and the "working baseline" exit criterion. They prevent a phase from becoming a bag of leftovers.

## Reject
```markdown
- [ ] T020 [Phase 2] Update handlers, storage, generated contracts, docs, tests, validation, and rollout. Depends on: none. Proof: go test ./...
```

This fails because one task hides several independent surfaces and uses broad verification as a substitute for slicing.

## Agent Traps
- Do not split a task just because it touches two files when those files form one invariant-preserving change.
- Do not force user-visible vertical slices onto purely internal refactors; still require small, proof-bound reviewable slices.
- Do not inflate `tasks.md` with micro-steps a coder can safely decide inside one small task.
- Do not postpone integration until the end just to keep early tasks small; small scaffolding that cannot be proven is not a useful slice.
