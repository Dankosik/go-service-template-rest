# Dependency-Ordered Task Ledgers

## Behavior Change Thesis
When loaded for unclear task order or `[P]` markers, this file makes the model derive dependencies from approved design artifacts and source-of-truth flow instead of marking everything parallel or starting with generated/derived files because they look easy.

## When To Load
Load this when task order, dependencies, `[P]` markers, generated artifacts, migrations, or source-of-truth-first sequencing is unclear.

## Decision Rubric
- Derive ordering from `design/component-map.md`, `design/sequence.md`, `design/ownership-map.md`, and triggered conditional artifacts, not generic architecture habits.
- Put source-of-truth, schema, contract, generator-input, or config tasks before derived consumers.
- Use `[P]` only after shared prerequisites are satisfied and only for disjoint surfaces with independent proof.
- Let the first dependency-establishing task say `Depends on: none` when it truly starts the chain.
- If a downstream task exposes a missing ownership, compatibility, migration, or source-of-truth decision, stop planning that branch and reopen the right earlier phase.

## Imitate
```markdown
- [ ] T010 [Phase 1] Update the source-of-truth artifact identified in `design/ownership-map.md`. Depends on: none. Proof: targeted diff read against `spec.md` decisions.
- [ ] T011 [Phase 1] Refresh the first derived surface listed in `design/component-map.md` from the updated source artifact. Depends on: T010. Proof: repository command named in `plan.md` or `rtk git diff --check` when no generator applies.
- [ ] T012 [Phase 1] [P] Update documentation cross-links that consume the same approved wording but do not feed later generated output. Depends on: T010. Proof: `rtk rg -n "<approved-term>" <doc-surface-from-design>`.
- [ ] T013 [Checkpoint] Confirm no downstream task requires a missing ownership or compatibility decision. Depends on: T011, T012. Proof: manual read of `design/ownership-map.md` and `plan.md` blockers.
```

Copy the dependency chain and the cautious `[P]` placement after `T010`. Replace the placeholder surfaces with the task-local design surfaces.

## Reject
```markdown
- [ ] T010 [Phase 1] [P] Regenerate derived artifacts. Depends on: none. Proof: none.
- [ ] T011 [Phase 1] [P] Update the source-of-truth artifact. Depends on: none. Proof: tests pass.
- [ ] T012 [Phase 1] Fix whatever breaks.
```

This fails because it marks unsafe parallelism, reverses source-of-truth order, and hides downstream repair inside an unbounded task.

## Agent Traps
- Do not add `[P]` to prove the plan is "parallelizable"; absence of false parallelism is safer.
- Do not require exact file paths where the approved design intentionally narrows only to a package or generated artifact surface.
- Do not treat generated-file sequencing as optional when the repo's source-of-truth relationship is explicit.
- Do not use broad `go test ./...` as the only proof when the task depends on source-to-derived ordering; add a diff or generation check when applicable.
