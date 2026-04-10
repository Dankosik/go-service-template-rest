## Execution Shape

- Shape: `lightweight local`
- Research mode: `local`
- Why: the task is non-trivial and needs preserved research plus a stable implementation plan, but subagent fan-out is not required and was not explicitly requested by the user. The main risk is not coding complexity; it is choosing the right spec patterns to import without breaking this repository's `spec.md` + `plan.md` split.

## Research Lanes

1. `repo-contract` lane
   - Goal: load the repository's spec-first workflow, artifact model, and skill-authoring conventions.
   - Ownership: local orchestrator research.

2. `external-patterns` lane
   - Goal: study current spec-format practices from BMAD, GitHub Spec Kit, Superpowers, and Spec-Driven Workflow.
   - Ownership: local orchestrator research with preserved notes in `research/external-spec-patterns.md`.

3. `local-challenge` lane
   - Goal: pressure-test naming, ownership boundaries, and whether the new skill should translate patterns into repo-native `spec.md` instead of copying foreign templates directly.
   - Ownership: local orchestrator challenge before finalizing `spec.md`.

## Order / Parallelism

- Run `repo-contract` and `external-patterns` research in parallel.
- Synthesize the overlap into candidate decisions for the new skill.
- Run a local challenge pass on overlap with existing skills such as `spec-first-brainstorming` and `go-design-spec`.
- Finalize `spec.md`.
- Write `plan.md`.
- Implement the skill bundle and only then update discoverability and mirrors.

## Fan-In And Decision Path

1. Confirm the repository's hard invariants around `spec.md`, `plan.md`, and workflow state ownership.
2. Compare external frameworks for:
   - section structure,
   - coverage prompts,
   - planning gates,
   - validation/readiness checks,
   - anti-placeholder controls.
3. Keep only patterns that strengthen this repository's workflow without introducing a competing artifact model.
4. Decide:
   - skill name,
   - skill boundaries,
   - whether a reference file is needed,
   - whether eval fixtures are needed,
   - which discoverability surfaces must change.

## Implementation Control Loop

- Policy: phased delivery
- Phase model:
  1. create research-backed canonical skill bundle
  2. update discoverability and sync mirrors
  3. run structural validation
- Between phases:
  - reconcile the skill against repository invariants,
  - check for overlap drift with adjacent skills,
  - verify mirrors and JSON assets before closeout.

## Expected Later Artifacts

- `plan.md`: yes
- `research/*.md`: yes
- `test-plan.md`: no
