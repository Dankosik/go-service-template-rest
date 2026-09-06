# Agent Harness

Harness-neutral index for native delegation and durable execution controls.
`AGENTS.md` owns authority and workflow phases own why a control is used; the
selected adapter owns how the current harness provides it.

## Delegation Interface

Across checkout or actor changes, retain the selected workflow revision and
instruction locators in the existing handoff, and verify the effective model
and effort through native state. Compare only the relevant local owners before
using a stale checkout's policy. Resolve material drift with the continuation
owner before dependent work; do not silently mix policies, bulk-sync an active
checkout, or treat a pinned revision as overriding current user/native authority.
Unchanged owners need no repeated inventory or bootstrap turn.

Give a delegated agent only the semantic fields in the [Subagent Brief
Template](subagent-brief-template.md). Isolate concurrent mutation only when it
is cheaper than waiting for the current checkout. Concurrent mutation may share
a checkout only when mutable owners and exclusive-lock resources do not overlap.
Overlapping work stays serial. Read-only lanes may share the current checkout.
Workers inside one Lead may share the Lead's checkout only when writable
responsibility and exclusive locks are disjoint. Do not create worktrees for
sequential work, cheap disjoint units, or bounded read-only review.

Count live Leads, mutable descendants, and in-flight review or validation lanes
against capacity. Leave spare slots for unlock and landing.

Silence alone does not establish a stall. Use native failure state, a missed
agreed checkpoint, or repeated attempts without new evidence to trigger
[Parent-Owned Recovery](spec-first-workflow/shared/transition.md#parent-owned-recovery).
Preserve useful work and reconcile the affected subtree before replacement or
scope reassignment.

Carry model, reasoning effort, isolation, native identity, and task lifecycle in
tool fields rather than prompt prose. Exclusive locks determine scheduling and
isolation, not reasoning capability. Select capability from the remaining
semantic judgment, implementation uncertainty, protected risks, and interacting
invariants under the selected adapter. Use its balanced configuration for
bounded implementation with closed behavior and no protected-risk reason.
A test plan or named check is not a prerequisite for delegation or model choice.
Raise capability for unresolved judgment or interacting invariants, or after a causal
attempt or review exposes a missed invariant. If capability signals disagree,
raise under the adapter.
Expected failing tests and mechanically diagnosed setup or syntax errors do
not by themselves trigger escalation; unexplained or repeated causal failures do.
Retain the raised capability for the remaining repair of that unit. Child work
uses the cheapest configuration permitted by the adapter for its authority and
fixed brief. Preserve an exact user-selected model.

Choose roles by capability: `evidence-agent`, `specialist-agent`,
`worker-agent`, `reviewer-agent`, or `adjudicator-agent`. Put domain expertise or
a phase review lens in the brief's `Method`; use model/effort fields for a
critical quality tier instead of creating another semantic role.

## Context And Lifetime

Lifetime follows responsibility, not model size:

| Owner | Lifetime |
| --- | --- |
| Orchestrator | The ledger; recover from canonical state when context is no longer reliable. |
| Architect or phase owner | Its macro phase; a different phase or independent design uses a fresh actor. |
| Lead | Its unit, with related-unit reuse only under the Planning Ledger Contract. |
| Execution lane | One bounded brief and corrections of that same result, including on smaller models. |
| Evidence lane | One bounded evidence brief and related follow-ups under the conditions below. |

Start every execution or evidence lane, including nested descendants, with a
fresh history using the selected adapter's native controls. Supply accepted
behavior, necessary decisions, source locators, writable scope, and the expected
outcome through the existing Subagent Brief. Executors choose their tests while
coding; final-validation briefs carry execution commands after assembly. Clean history excludes the parent's conversation; it does not
remove applicable instructions, tools, permissions, or current repository files.
Reviewer freshness remains governed by shared Review.

Keep implementation corrections with the same worker while its outcome,
writable responsibility, and proof boundary remain unchanged and its context
remains useful. Once the parent consumes its completed result, end that worker's
assignment; resume it only for a correction within the same brief.

An evidence agent may answer a related follow-up for the same parent decision
within the same evidence boundary when its context remains useful and
independence is unnecessary. Supply the changed question and inputs; recheck
earlier assumptions affected by that delta.

Use a fresh agent for a different implementation outcome or writable
responsibility, an unrelated investigation, or a question requiring independent
judgment. Lead reuse does not extend to its execution lanes. Reviewer reuse
remains governed by [Review](spec-first-workflow/shared/review.md).

For stalled diagnosis or unreliable context, preserve the current edits and
material findings, reconcile and stop the old subtree, and give a fresh agent
the remaining brief and useful evidence. Change the failed approach through
the responsible parent; do not copy the unsuccessful conversation wholesale.
Follow-ups continue existing history and do not reset context. Do not rotate
agents by elapsed time or message count, or add a separate lifecycle manager.

## Nested Execution

The Orchestrator owns the ledger; Leads return Implemented, and one assigned
delivery owner owns final validation and acceptance. A worker
may delegate a strict subset of its accepted brief only when the selected
adapter permits descendants and Implementation's benefit, ownership, and lock
conditions hold. Missing product or architecture decisions return to their
owner; unwritten test cases and commands do not block delegation.
At a native depth or tool limit, the nearest capable parent dispatches the
subset or completes it within its own authority; do not invent a new acceptance
unit or an unsupported carrier. Use only as much nesting as the work needs.

Each delegating parent owns its subtree: retain child identities, give them
disjoint writable scopes, consume and integrate their results, and join or stop
all descendants before returning, freezing, or releasing that scope. A parent
cannot edit a delegated writable scope concurrently. Count the whole active
tree against capacity and leave room for review and completion. Reconcile and
stop the entire affected subtree before replacing a parent or reassigning its
scope; parent interruption alone does not prove descendant termination.

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
