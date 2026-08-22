# Agent Harness

Harness-neutral index for native delegation and durable execution controls.
`AGENTS.md` owns authority and workflow phases own why a control is used; the
selected adapter owns how the current harness provides it.

## Delegation Interface

Give a delegated agent only the semantic fields in the [Subagent Brief
Template](subagent-brief-template.md). Select `shared` or `worktree` isolation
only when the harness supports it and current collision or candidate-state risk
justifies the choice.

Carry model, reasoning effort, isolation, native identity, and task lifecycle in
tool fields rather than prompt prose. The Acceptance-Unit Lead uses the
adapter's quality-first configuration; child work uses the cheapest
configuration likely to close its fixed brief. Raise capability for complex
reasoning, a weak oracle, protected domains, or high consequence. Preserve an
exact user-selected model.

Choose roles by capability: `evidence-agent`, `specialist-agent`,
`worker-agent`, `reviewer-agent`, or `adjudicator-agent`. Put domain expertise or
a phase review lens in the brief's `Method`; use model/effort fields for a
critical quality tier instead of creating another semantic role.

## Select One Adapter

| Current harness | Adapter |
| --- | --- |
| Codex App or Codex CLI | [Codex](agent-harness/codex.md) |
| Claude Code CLI, desktop, web, or IDE | [Claude Code](agent-harness/claude-code.md) |
| Qwen Code CLI or IDE | [Qwen Code](agent-harness/qwen-code.md) |
| Grok Build CLI, TUI, or ACP | [Grok Build](agent-harness/grok-build.md) |
| Cursor IDE, CLI, or Cloud Agents | [Cursor](agent-harness/cursor.md) |
| OpenCode CLI, TUI, desktop, or IDE | [OpenCode](agent-harness/opencode.md) |

Do not mix control planes inside one outcome. A task, subagent, worktree,
model, or Goal is a carrier; it never expands authorization or transfers unit
ownership. Use a durable Goal only when the work is genuinely long-running or
resumable and the installed harness supports it.

All adapters preserve the same ledger, unit ownership, review, and transition
semantics. Select topology from callable native controls, not the product name.
When a full-ledger carrier is unavailable, report that capability gap instead
of replacing the accepted workflow with a smaller one.
