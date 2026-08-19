# Claude Code Harness Adapter

## Read When

Read after the [Agent Harness](../agent-harness.md) selects Claude Code and a
native Agent lane, Goal, model, effort, peer message, or programmatic run is
needed.

## Native Map

- `/goal <condition>` is the durable execution control; its evaluator sees only
  the conversation.
- An isolated Implementation Worker is a background `Agent` lane with
  `isolation: "worktree"` and an effort-carrier `subagent_type`.
- Read-only lanes use fresh background `Agent` contexts without worktree
  isolation.
- Claude Code has no Ledger Orchestrator carrier; the current root binds one
  Acceptance-Unit Lead and stops at that unit's canonical transition.

## Goal Mechanics

Vendor authority: [Keep Claude working toward a
goal](https://code.claude.com/docs/en/goal).

`/goal <condition>` starts work and permits one active condition per session.
The evaluator runs no tools and reads no files, so require the transcript to
show the named proof command and result. The condition is at most 4,000
characters and has no turn limit. A goal does not change permissions. Resume or
continue restores an active goal when supported; achieved or cleared goals do
not resume.

## Worker Mechanics

### Dispatch

Dispatch a background `Agent` with `isolation: "worktree"`, the exact frozen
base, accepted ledger revision, and the [Worker
Contract](../spec-first-workflow/phases/implementation-worker-contract.md#dispatch-contract).
Pass isolation at dispatch rather than frontmatter so the base follows the
parent's current `HEAD`. Collapse any dependency chain whose successor base the
harness cannot materialize before dispatch.

Read-only research and review use ordinary background agents and bind
`READ_ONLY_SPECIALIST` or `ACCEPTANCE_REVIEWER`. They never become acceptance
owners.

### What crosses into a worker

A Worker receives its dispatch, worktree files, repository instructions, the
parent-start status snapshot, and explicitly loaded skills. It does not receive
the root conversation, prior command output, output style, or auto-memory. Put
every execution-changing fact absent from canonical files in the dispatch.

### Monitor and correction

Record the completing Worker's agent ID and keep it reachable through the unit
transition. Send one batched correction to that same ID; it retains context.
Message an active Worker only for a safety stop or accepted-input invalidation.
Replace it only for a true stall or invalidated base. `Explore` and `Plan` are
one-shot read-only lanes and never Workers.

### Model and effort

Map the shared selection policy to Sonnet for mechanical and ordinary lanes and
Opus for Acceptance-Unit Leads and complex or high-consequence lanes. Carry the
shared effort through the installed role carrier; delete those carriers when
Claude gains a per-dispatch effort field.

### Read-only lanes

Use one independently checkable question per lane and the shared read-only
carrier. A triggered implementation review starts a fresh one-shot
`task-acceptance-agent`; use a critical reviewer only for a justified
highest-consequence boundary. Harness task lists never replace repository
`tasks.md` or its receipts.

## Cross-Session Messaging

Vendor authority: [Cross-session
messaging](https://code.claude.com/docs/en/cross-session-messaging).

`/list-agents` discovers reachable peer sessions and `SendMessage` sends plain
text by name. A peer message is evidence input, never a lane result, proof
receipt, acceptance, or ledger transition. Before opening a write lane in a
checkout this session did not create, use `/list-agents` to detect a second
writer and route to an isolated worktree or that owning session.

Keep `SendMessage` available when same-Worker correction is required. Use
`crossSessionInbound` to control peer traffic. Agent teams remain outside this
workflow while they self-claim work or maintain a second acceptance ledger.

## Programmatic Runs

Use the Claude Agent SDK when code must drive this same harness. Direct
Anthropic Messages API or managed-agent integrations are separate products and
do not substitute for repository harness controls.
