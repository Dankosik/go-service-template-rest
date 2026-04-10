## Execution Shape

- Shape: `full orchestrated`
- Research mode: `fan-out`
- Why: the task crosses repository conventions, skill authoring conventions, repo-profile selection, and prompt-quality risk. It also explicitly requests subagent use when available.

## Research Lanes

1. `repo-profile` lane
   - Goal: summarize the repository’s product/domain, stack, architecture, workflows, naming/style, and validation surfaces relevant to prompt composition.
   - Ownership: read-only subagent research.

2. `skill-conventions` lane
   - Goal: inspect canonical skill layout, SKILL authoring patterns, mirror strategy, and likely best-fitting shape for this new repository-local skill.
   - Ownership: read-only subagent research.

3. `challenge` lane
   - Goal: pressure-test candidate design assumptions once local + subagent research is synthesized, focusing on hallucination risk, over-context injection, and maintainability drift.
   - Ownership: read-only challenger pass after candidate decisions exist.

## Order / Parallelism

- Run `repo-profile` and `skill-conventions` in parallel.
- Keep local orchestrator research on concrete repository files in parallel with those lanes.
- Fan in results into candidate decisions.
- Run the `challenge` lane on the candidate design before writing the implementation plan.

## Fan-In And Decision Path

1. Compare local repository evidence with both research lanes.
2. Extract stable repo facts that are worth embedding or dynamically loading.
3. Decide the minimal artifact set:
   - canonical skill path
   - any supporting references/examples
   - any lightweight validation fixtures
4. Run pre-spec challenge on the candidate design.
5. Reconcile challenge findings into `spec.md`.
6. Write a phased `plan.md` before implementation.

## Implementation Control Loop

- Policy: phased delivery
- Phase model:
  1. implement canonical skill + supporting references/examples
  2. sync mirrors into runtime directories
  3. validate with realistic sample inputs
- Between phases:
  - review/reconcile locally against repository conventions
  - run targeted validation before advancing

## Expected Later Artifacts

- `plan.md`: yes
- `research/*.md`: optional, only if preserving synthesized repo profile materially helps the skill
- `test-plan.md`: no, unless validation scope grows unexpectedly
