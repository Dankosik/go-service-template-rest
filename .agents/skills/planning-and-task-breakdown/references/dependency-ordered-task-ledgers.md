# Dependency-Ordered Task Ledgers

## When To Load
Load this when task ordering, dependencies, `[P]` markers, generated artifacts, source-of-truth-first sequencing, or "what can run in parallel" is unclear.

Do not infer dependencies from generic architecture habits. Use the approved `design/component-map.md`, `design/sequence.md`, `design/ownership-map.md`, and any triggered conditional design artifact.

## Good Plan Snippet

```markdown
## Phase Plan

### Phase 1: Source-Of-Truth Update Before Derived Surfaces
Objective: update the approved source-of-truth artifact first, then refresh or align only the derived surfaces named in the design.
Depends On: `design/ownership-map.md` identifies the source-of-truth artifact and derived consumers.
Task Ledger Link / IDs: T010-T014.
Acceptance Criteria:
- no derived surface changes before its source artifact is updated
- every `[P]` task touches a disjoint surface and has an independent proof command
Planned Verification:
- source and derived artifacts are diff-reviewed in dependency order
- `rtk git diff --check`
Review / Checkpoint:
- stop if a derived surface appears to require a decision not present in `spec.md + design/`
```

Why it is good: the phase explains why order matters without naming an unapproved architecture.

## Bad Plan Snippet

```markdown
## Phase Plan

Do all files in parallel. Start with generated output because it is easy, then update the source later if needed.
```

Why it is bad: it reverses source-of-truth order and marks parallelism without proof that surfaces are disjoint.

## Good Task Ledger Example

```markdown
- [ ] T010 [Phase 1] Update the source-of-truth artifact identified in `design/ownership-map.md`. Depends on: none. Proof: targeted diff read against `spec.md` decisions.
- [ ] T011 [Phase 1] Refresh the first derived surface listed in `design/component-map.md` from the updated source artifact. Depends on: T010. Proof: repository command named in `plan.md` or `rtk git diff --check` when no generator applies.
- [ ] T012 [Phase 1] [P] Update documentation cross-links that consume the same approved wording but do not feed later generated output. Depends on: T010. Proof: `rtk rg -n "<approved-term>" <doc-surface-from-design>`.
- [ ] T013 [Checkpoint] Confirm no downstream task requires a missing ownership or compatibility decision. Depends on: T011, T012. Proof: manual read of `design/ownership-map.md` and `plan.md` blockers.
```

Why it is good: T012 is parallel only after the shared dependency lands, and the checkpoint is dependency-aware.

## Bad Task Ledger Example

```markdown
- [ ] T010 [Phase 1] [P] Regenerate derived artifacts. Depends on: none. Proof: none.
- [ ] T011 [Phase 1] [P] Update the source-of-truth artifact. Depends on: none. Proof: tests pass.
- [ ] T012 [Phase 1] Fix whatever breaks.
```

Why it is bad: it marks unsafe parallelism, hides a real dependency, and gives an unbounded repair task.

## Non-Findings To Avoid

- Do not flag a missing `[P]` marker when the dependency graph is unclear; absence of parallelism is safer than false parallelism.
- Do not require exact file paths where the approved design intentionally narrows only to a package or generated artifact surface.
- Do not treat generated-file sequencing as optional when the repo's source-of-truth relationship is explicit.
- Do not flag "Depends on: none" for the first task when it truly establishes the dependency foundation.

## Exa Source Links
External links found with Exa and used only to calibrate decomposition and dependency-tracking patterns:

- PMI WBS dependency/decomposition discussion: https://www.pmi.org/learning/library/developing-elaborating-work-breakdown-structures-7241
- Scrum Guide, ordered/refined backlog and actionable Sprint Backlog context: https://scrumguides.org/scrum-guide.html
- Feature slicing dependency examples: https://techleadhandbook.org/agile/feature-slicing/
