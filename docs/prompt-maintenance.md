# Prompt Maintenance

Read this owner only before editing agent instructions, tool descriptions, or
skills.

## Current Evidence

Matt Pocock's [Writing for
agents](https://github.com/mattpocock/skills/blob/main/skills/productivity/writing-for-agents/SKILL.md)
and [Skill
mechanics](https://github.com/mattpocock/skills/blob/main/skills/productivity/writing-for-agents/SKILL-MECHANICS.md)
own the vocabulary for context pointers, information hierarchy, completion
criteria, leading words, and pruning. OpenAI's [GPT-5.6 prompting
guidance](https://developers.openai.com/api/docs/guides/prompt-guidance-gpt-5p6)
owns current Codex model guidance. Anthropic's [new rules of context engineering
for Claude 5](https://claude.com/blog/the-new-rules-of-context-engineering-for-claude-5-generation-models),
[Claude Code documentation](https://code.claude.com/docs), [project-memory
guidance](https://code.claude.com/docs/en/memory), and [prompting best
practices](https://platform.claude.com/docs/en/build-with-claude/prompt-engineering/claude-prompting-best-practices)
own current Claude Code guidance. Anthropic's [system-prompt
history](https://platform.claude.com/docs/en/release-notes/system-prompts) is
observational evidence for claude.ai and its mobile apps, not an API or Claude
Code contract.

Re-derive prior-generation constraints against the current target model and
harness. [Agent Harness](agent-harness.md) owns which native controls apply.

## Instruction Contract

Use the repository [Task Contract](../AGENTS.md#task-contract) as the
outcome-first schema: live context, outcome, success criteria, constraints and
authority, non-obvious tool routing, required proof or output, then stop
conditions. State each durable instruction once in its narrowest owner and link
to it elsewhere. Phrase it as an observable trigger, action, completion
criterion, or stop condition. Prefer allowed behavior; reserve prohibitions for
safety, authorization, or a decisive exclusion. Name required content and what
may be omitted instead of using broad tone or brevity labels.

Keep bootstrap files compact and move conditional detail behind existing load
gates; imported text still consumes startup context. A repeated approval rule
causes unnecessary approval requests, so link its owner. Hold every skill,
subagent, and tool description to [Skill Authoring](skill-authoring.md#invocation):
leading word, distinct triggers, owned outcome, and decisive exclusion. Expose
only material the task can act on. Reasoning effort and response verbosity are
harness controls, not prose requests to think harder or answer at length.

## Change And Proof

This section owns pruning for every repository instruction, including skills.
Change one instruction group at a time and prefer removal. Delete a weaker
statement when the behavior already has an owner. Retain an example or style
rule only when it encodes a product requirement or closes a measured gap, then
compare realistic trigger, near-miss, and completion cases. For context routing,
compare bootstrap and mandatory read sets before and after; word count proves
context cost, not behavior.

Instruction edits prove only an instruction-level mitigation. Claim changed
behavior only after a live evaluation exercises the target model, harness,
trigger, and completion case. A new model generation reopens accepted no-op,
constraint, and example decisions; repeat the removal-first comparison rather
than carrying prior-generation scaffolding forward.

Load the [instruction evaluation runbook](../evals/instructions/README.md) only
when a live baseline/candidate comparison is authorized. Structural checks and
word counts remain shape and context-cost evidence, not model-behavior proof.

## Ownership

- `AGENTS.md` owns repository-wide rules; phase files own phase method.
- Template-owned instruction paths remain free of service-specific names,
  paths, targets, owners, and invariants. [Template Sync](template-sync.md) owns
  propagation.
- [Agent Harness](agent-harness.md) owns shared carrier invariants; its selected
  adapter owns native controls, models, and reasoning effort.
- `.codex/agents/*.toml`, `.claude/agents/*.md`, and `.qwen/agents/*.md` mirror
  harness-neutral specialist roles and change together.
- External-tool summaries link the vendor contract beside the claim; omitted
  vendor clauses remain evidence gaps.
- [Skill Authoring](skill-authoring.md) owns behavioral adapters.
- The workflow's shared files retain persistence, delegation, review, and
  handoff ownership; this document does not restate them.
