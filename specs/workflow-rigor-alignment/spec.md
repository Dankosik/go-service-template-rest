## Context

This repository already has a strong Go-native agent and skill portfolio, but the workflow contract is still looser than the `gonka-proxy` reference around workflow governance, fan-out discipline, artifact roles, inventory sufficiency, and skill routing.

The requested change is to bring the workflow, AGENTS contract, and skill/subagent system up to comparable rigor while keeping the result Go-native and explicitly adapted to this repository's orchestrator/subagent-first model.

The work also needs to absorb the useful ideas behind the upstream `agent-skills` framing, specification, and planning patterns without importing their skill-driven ownership model, slash-command flow, or non-Go repository assumptions verbatim.

## Scope / Non-goals

In scope:
- reconcile [AGENTS.md](/Users/daniil/Projects/Opensource/go-service-template-rest/AGENTS.md), [docs/spec-first-workflow.md](/Users/daniil/Projects/Opensource/go-service-template-rest/docs/spec-first-workflow.md), and workflow/discoverability docs into one coherent contract
- selectively port the strongest workflow-governance ideas from the Gonka reference repo
- decide how the three upstream skills should land here: existing skill adaptation, new repo-local skill, or documented workflow pattern
- make `.agents/skills` genuinely canonical and define mirror expectations for active runtime directories
- update skill, subagent, and artifact guidance so orchestrator responsibilities, subagent responsibilities, skill responsibilities, and artifact responsibilities are clearly separated

Non-goals:
- importing Gonka product/domain rules, TypeScript-specific workflow details, or unrelated repo invariants
- installing or depending on the upstream plugin runtime directly
- creating one-to-one clones of upstream skills when a cleaner adaptation or documentation pattern is better
- rewriting unrelated deep-dive docs outside the active runtime surfaces needed for the workflow cleanup
- touching unrelated implementation code outside the workflow, documentation, skill, or mirror/tooling surfaces needed for this change

## Constraints

- Final decisions stay with the orchestrator; subagents remain read-only and advisory.
- This is non-trivial repo work, so workflow planning must be explicit before subagent fan-out and coding.
- The result must stay Go-native and compatible with the existing project agent portfolio in `.codex/agents/` and `.claude/agents/`.
- There must be no competing source of truth for workflow rules, canonical skills, or artifact roles.
- Existing user changes in the worktree are not ours to revert; integration must work around them.

## Decisions

1. Use the `full orchestrated` execution shape for this change.
2. Treat this as `fan-out` research after the initial local comparison pass, because the task spans workflow contract design, skill-catalog ownership, mirror/tooling behavior, and validation/discoverability.
3. Create and maintain [workflow-plan.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/workflow-rigor-alignment/workflow-plan.md) before any subagent fan-out, and create a separate [plan.md](/Users/daniil/Projects/Opensource/go-service-template-rest/specs/workflow-rigor-alignment/plan.md) before coding because the implementation will span several documentation and tooling surfaces.
4. Port Gonka workflow rigor selectively: keep workflow-system improvements that strengthen orchestration quality here, but reject Gonka-specific domain, stack, or runtime assumptions.
5. Adapt the upstream framing, specification, and planning ideas into this repository's orchestrator/subagent-first system rather than installing them as literal workflow owners.

## Open Questions / Assumptions

- [assumption] The upstream skill texts retrieved from `addyosmani/agent-skills` GitHub pages are sufficient to map the three requested skills accurately for this repo adaptation.
- [assumption] Existing untracked `*-workspace` directories under `.agents/skills` and mirrored runtime directories are local workbench artifacts, not canonical runnable skills that should drive repository policy.
- [assumption] The repo's active discoverability surfaces for this change are the root docs, skill catalog/distribution docs, the canonical skill tree, the runtime mirrors, and the skill-sync tooling.
- [open question] Whether this change should tighten the skill-sync script so only top-level runnable skill directories are mirrored, instead of copying every directory beneath `.agents/skills`.
