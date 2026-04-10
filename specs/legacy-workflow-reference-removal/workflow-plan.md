# Workflow Plan

## Execution Shape

- Shape: `lightweight local`
- Research mode: `local`
- Why: the task is bounded to repository documentation cleanup and does not need specialist subagent fan-out once the affected files are inventoried.

## Research Guardrails

- Keep the work focused on active workflow instructions and directly conflicting documentation surfaces.
- Do not silently preserve old wording behind "historical" or "legacy" disclaimers.
- Treat runnable `SKILL.md` files as the canonical skill surface.

## Local Work Sequence

1. Create a small decision record and coder-facing plan for the cleanup.
2. Update the active contract docs:
   - `AGENTS.md`
   - `docs/spec-first-workflow.md`
3. Remove obsolete documentation-only workflow/design-note files:
   - the workflow rewrite notes
   - the old skill-doc and adaptation layer
4. Update remaining discoverability and runtime docs:
   - `.agents/skills/api-contract-designer-spec/SKILL.md`
   - `.agents/skills/api-contract-designer-spec/evals/evals.json`
5. Clean affected spec records that still point at the removed layer.
6. Re-scan the repository and record validation evidence in `spec.md`.

## Implementation Control Loop

- Delivery mode: sequential
- Checkpoints:
  1. active contract surfaces updated
  2. obsolete docs removed
  3. remaining surfaces aligned
  4. validation scans clean

## Expected Artifacts

- Required now:
  - [spec.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/legacy-workflow-reference-removal/spec.md)
  - [workflow-plan.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/legacy-workflow-reference-removal/workflow-plan.md)
- Required before completion:
  - [plan.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/legacy-workflow-reference-removal/plan.md)
