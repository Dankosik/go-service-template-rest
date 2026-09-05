# Agent Harness

Harness-neutral index for native delegation and durable execution controls.
`AGENTS.md` owns authority and workflow phases own why a control is used; the
selected adapter owns how the current harness provides it.

## Delegation Interface

Give a delegated agent only the semantic fields in the [Subagent Brief
Template](subagent-brief-template.md). Isolate concurrent mutation only when it
is cheaper than waiting for the current checkout. Concurrent mutation may share
a checkout only when mutable owners and exclusive-lock resources do not overlap.
Overlapping work stays serial. Read-only lanes may share the current checkout.
Workers inside one Lead may share the Lead's checkout only when writable
responsibility and exclusive locks are disjoint. Do not create worktrees for
sequential work, cheap disjoint units, or bounded read-only review.

Count live Leads, mutable descendants, and in-flight review or validation lanes
against capacity. Leave spare slots for unlock and landing. A silent identified
lane is not progress; replace or absorb it.

Carry model, reasoning effort, isolation, native identity, and task lifecycle in
tool fields rather than prompt prose. Read Exclusive locks and Accept-when to
choose Lead capability. Use the adapter's balanced configuration only when
locks are `none`, the focused check is a named mechanical command, and Boundary
has no protected-risk reason. Raise to the strongest configuration when the
lock is not `none`, the oracle is weak, the unit is cross-cutting, an earlier
causal attempt failed, or review exposed a previously unknown invariant. If
those signals disagree, raise. After a causal focused-proof or review miss,
raise capability for the remaining repair of that unit; do not keep the cheaper
configuration that missed the invariant. Child work uses the cheapest
configuration permitted by the adapter for its authority and fixed brief.
Preserve an exact user-selected model.

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
resumable and the selected adapter's native authorization and capability
requirements are met.

All adapters preserve the same ledger, unit ownership, review, and transition
semantics. Select topology from callable native controls, not the product name.
When a full-ledger carrier is unavailable, report that capability gap instead
of replacing the accepted workflow with a smaller one.
