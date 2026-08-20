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
own current Claude guidance. Product system-prompt history is observational
evidence, not a harness contract.

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

Write durable instructions as observable triggers, actions, completion criteria,
or stop conditions. State allowed behavior and reserve prohibitions for safety,
authority, or decisive exclusions. Expose only material the current task can
act on.

`AGENTS.md` owns global authority and invariants; the workflow router owns path
and phase selection; phase/shared files own conditional method; task artifacts
own accepted decisions. `.agents/roles` owns harness-neutral role semantics and
`scripts/agent-roles-sync.sh` generates the Codex, Claude, and Qwen carriers.
[Skill Authoring](skill-authoring.md) owns skill invocation, metadata,
body/reference boundaries, and catalog constraints. [Template
Sync](template-sync.md) owns propagation.

## Change And Proof

Change one instruction group at a time and prefer removal. Preserve exact hard
boundaries and move text before rewriting its meaning. Compare bootstrap and
mandatory read sets before and after; word count proves context cost, not
behavior.

Instruction edits prove only an instruction-level mitigation. Claim changed
behavior only after a live evaluation exercises the target model, harness,
trigger, and completion case. Load the [instruction evaluation
runbook](../evals/instructions/README.md) only when a live baseline/candidate
comparison is authorized. Structural checks prove ownership, generation, links,
and shape—not model behavior.
