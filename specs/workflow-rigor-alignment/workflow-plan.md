# Workflow Plan

## Execution Shape

- Shape: `full orchestrated`
- Research mode: `fan-out`
- Why: this change crosses workflow governance, skill ownership, mirror/tooling policy, validation expectations, and documentation/discoverability. A local-only pass would under-cover material seams.

## Research Guardrails

- The initial local comparison pass is complete.
- Once specialist lanes are active, the main flow stays focused on routing, fan-in, synthesis, challenge resolution, and repository fact-gathering that is truly undelegable.
- All delegated lanes are read-only. No delegated lane may write code, edit repository files, mutate git state, or alter the implementation plan.
- Each lane owns one narrow problem and one primary skill.

## Inventory Sufficiency Check

Before synthesis is treated as stable:
- verify which top-level directories under `.agents/skills` are real runnable skills versus workbench artifacts
- verify which runtime mirror directories are active and how `scripts/dev/sync-skills.sh` currently treats non-skill directories
- verify that each canonical skill expected to stay active has a real top-level `SKILL.md`
- verify that agent mirrors stay aligned across `.codex/agents/*.toml` and `.claude/agents/*.md` for the surfaces touched by this change

## Planned Lanes

1. `architecture-agent(workflow-contract)` using primary skill `go-architect-spec`
   Goal:
   compare the target workflow contract against the Gonka reference and recommend which governance ideas should be ported, which should be rejected, and how the workflow/artifact/subagent/skill boundaries should read in this Go repository.
   Inputs:
   `AGENTS.md`, `docs/spec-first-workflow.md`, `README.md`, Gonka reference docs, upstream skill notes.
   Output:
   candidate contract changes, rejected imports, and open seams needing reconciliation.

2. `delivery-agent(skill-canon-and-mirrors)` using primary skill `go-devops-spec`
   Goal:
   audit canonical skill ownership, mirror expectations, sync/check tooling behavior, and discoverability docs so `.agents/skills` becomes operationally canonical without competing truths.
   Inputs:
   `.agents/skills`, runtime mirror directories, `scripts/dev/sync-skills.sh`, relevant docs and README sections.
   Output:
   recommended canonical policy, mirror rules, tooling/doc changes, and inventory sufficiency findings.

3. `qa-agent(validation-and-discoverability)` using primary skill `go-qa-tester-spec`
   Goal:
   define the smallest sufficient validation proof set and identify where terminology drift or discoverability gaps would make the final workflow hard to follow in practice.
   Inputs:
   workflow docs, README, skill-distribution docs, current validation expectations from the user.
   Output:
   validation obligations, terminology drift risks, and doc examples/checkpoints needed for clarity.

## Fan-In And Challenge Path

1. Run the three specialist lanes in parallel.
2. Compare outputs as claims, not as authority.
3. Resolve conflicts and write candidate decisions into [spec.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/workflow-rigor-alignment/spec.md).
4. Run `challenger-agent` against the candidate decisions before implementation planning.
5. Reopen targeted specialist research only if the challenger exposes a planning-critical seam.
6. After challenge reconciliation, write [plan.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/workflow-rigor-alignment/plan.md) before coding.

## Implementation Control Loop

- Delivery mode: phased by default
- Expected phase policy:
  1. contract and workflow docs
  2. canonical skill + mirror/discoverability/tooling alignment
  3. final consistency cleanup and validation evidence
- Exact phase boundaries may tighten after fan-in, but implementation remains sequential with review/reconcile/validate checkpoints between phases.

## Expected Artifacts

- Required now:
  - [spec.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/workflow-rigor-alignment/spec.md)
  - [workflow-plan.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/workflow-rigor-alignment/workflow-plan.md)
- Required before coding:
  - [plan.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/workflow-rigor-alignment/plan.md)
- Optional:
  - `research/*.md` only if one of the specialist outputs is valuable enough to preserve outside the main spec
