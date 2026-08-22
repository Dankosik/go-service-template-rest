# Prompt Maintenance

Read this owner only before editing agent instructions, tool descriptions,
roles, or skills.

## Evidence Boundary

OpenAI's [GPT-5.6 prompting
guidance](https://developers.openai.com/api/docs/guides/prompt-guidance-gpt-5p6)
owns current Codex model guidance. Anthropic's [context-engineering
guidance](https://claude.com/blog/the-new-rules-of-context-engineering-for-claude-5-generation-models),
[Claude Code documentation](https://code.claude.com/docs), and [prompting best
practices](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices)
own current Claude guidance. Installed Grok native schemas own Grok controls;
the same context-engineering density rules apply when writing Grok carriers.
Cursor's [rules](https://cursor.com/docs/rules.md),
[skills](https://cursor.com/docs/skills.md), and
[subagents](https://cursor.com/docs/subagents.md) own current Cursor controls.
OpenCode's [agents](https://opencode.ai/docs/agents),
[skills](https://opencode.ai/docs/skills), [config](https://opencode.ai/docs/config),
and [CLI](https://opencode.ai/docs/cli) own current OpenCode 1 controls.
Product system-prompt history is observational evidence, not a harness
contract.

Re-derive old constraints against the current target model and harness. [Agent
Harness](agent-harness.md) owns native controls.

## Classification

Give each instruction one disposition before editing it.

| Instruction type | Disposition |
| --- | --- |
| Safety, authorization, secret, or irreversible-effect boundary | Keep explicit and fail closed in the earliest required owner. |
| Non-obvious product or team policy | Keep in its narrowest semantic owner. |
| Conditional method | Put behind its observable trigger. |
| Behavior guaranteed by a tool, schema, sandbox, or generator | Remove from prose or emit once only where the guarantee is absent. |
| Fact apparent from canonical code, contract, test, or file layout | Remove. |
| Restatement of another owner | Replace with a trigger and link. |
| Example that defines required behavior | Replace with a schema or fixture when possible; otherwise retain it with its contract. |
| Example of only the ordinary path | Remove. |

Keep a hard boundary explicit; express judgment as the broadest principle that
preserves the measured behavior. If ordinary repository context already makes a
rule derivable, do not teach it again.

## Ownership And Interfaces

One file owns one meaning. Bootstrap files contain only rules needed before
routing. Routers choose the next owner; they do not explain that owner's method.
Move repeatable fields, allowed values, tool capabilities, sandbox guarantees,
and generated carriers into interfaces rather than narrative checklists.

The instruction graph flows `bootstrap -> router -> phase -> method ->
reference`. Lower layers do not reselect higher layers. A selector selects; it
does not restate the selected owner's method. Reference indexes contain only
selection pressures, links, and the decision effect of loading them.

Roles encode capability, permission, isolation, and context freshness. Skills
encode expertise and method. A model-invoked skill needs an independent
observable trigger that changes the method; aliases and explicit workflows do
not compete in autonomous discovery.

Write durable instructions as observable triggers, actions, completion criteria,
or stop conditions. State allowed behavior and reserve prohibitions for safety,
authority, or decisive exclusions. Expose only material the current task can
act on.

| Meaning | Canonical owner |
| --- | --- |
| Authority, safety, secrets, irreversible boundaries, global invariants, Direct Work eligibility | `AGENTS.md` |
| Structured/orchestrated path, macro phases, phase selection | [Workflow Router](spec-first-workflow.md) |
| Movement, narrow reopen, and boundary handoff | [Transition](spec-first-workflow/shared/transition.md) |
| Durable movement receipt fields | [Transition Result V1](spec-first-workflow/interfaces/transition-result-v1.md) |
| One phase's unique decision | Its phase owner |
| Artifact persistence | [Artifacts](spec-first-workflow/shared/artifacts.md) |
| Status values and transitions | [Artifact Lifecycle V1](spec-first-workflow/interfaces/artifact-lifecycle-v1.md) |
| Proof semantics | [Evidence Contract](spec-first-workflow/shared/evidence-contract.md) |
| Proof result fields | [Evidence Result V1](spec-first-workflow/interfaces/evidence-result-v1.md) |
| Independent-review trigger and lifecycle | [Review](spec-first-workflow/shared/review.md) |
| Review result fields and verdict values | [Review Result V1](spec-first-workflow/interfaces/review-result-v1.md) |
| Read-only lane eligibility | [Read-Only Delegation](spec-first-workflow/shared/read-only-delegation.md) |
| Read-only lane result fields | [Lane Result V1](spec-first-workflow/interfaces/lane-result-v1.md) |
| Prompt packaging and no-duplication contract | [Prompt Composition](prompt-composition.md) |
| Resume after interruption and terminal cleanup | Their separate shared owners |
| Domain judgment | `.agents/skills/<domain>` |
| Domain decision result fields | [Decision Result V1](spec-first-workflow/interfaces/decision-result-v1.md) |
| Harness-neutral role scope | `.agents/roles` |
| Output fields and allowed values | `docs/spec-first-workflow/interfaces/` |
| Accepted task decisions and execution state | `specs/<task>/` |
| Repository/runtime facts | Code, OpenAPI, tests, generated sources, and repository architecture |
| Generated carriers | `scripts/agent-roles-sync.sh` from canonical roles |

[Skill Authoring](skill-authoring.md) owns skill invocation, metadata,
body/reference boundaries, and catalog constraints. [Template
Sync](template-sync.md) owns propagation.

## Change And Proof

Change one instruction group at a time and prefer removal. Preserve exact hard
boundaries and move text before rewriting its meaning. Compare bootstrap and
mandatory read sets before and after; word count proves context cost, not
behavior.

Also compare active semantic owners and catalog branches on representative
routes. A shorter file that adds another selector has increased context cost.

Instruction edits prove only an instruction-level mitigation. Claim changed
behavior only after a live evaluation exercises the target model, harness,
trigger, and completion case. Load the [instruction evaluation
runbook](../evals/instructions/README.md) only when a live baseline/candidate
comparison is authorized. Structural checks prove ownership, generation, links,
and shape—not model behavior.
