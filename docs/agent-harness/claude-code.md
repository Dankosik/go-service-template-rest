# Claude Code Harness Adapter

Use the installed Agent and Goal controls as native authority. The Opus 5.5
model policy requires Claude Code 2.1.280 or later; recheck native controls after
runtime changes.

## Native Map

- `/orchestrator` binds the current session as Ledger Orchestrator. Dispatch
  mutually independent ready units through `Agent` with
  `subagent_type: "acceptance-unit-lead"`, within current capacity. The native
  [Lead carrier](../../.claude/agents/acceptance-unit-lead.md) loads the existing
  role skill and returns its fixed unit as `Implemented`, without task proof
  or review gates. The root integrates these candidates serially into the local
  development tree from
  [Acceptance Result](../spec-first-workflow/interfaces/acceptance-result-v1.md)
  and unlocks dependent implementation immediately. One final delivery
  assignment owns validation and acceptance after assembly.
- Ordinary named subagents can spawn descendants through `Agent`. Apply
  [Nested Execution](../agent-harness.md#nested-execution); the project setting
  `env.CLAUDE_CODE_MAX_SUBAGENT_SPAWN_DEPTH: "3"` selects the installed default
  of three levels. An unavailable tool or exhausted depth returns the exact
  capability gap to the parent for recovery.
- The Lead may implement directly. Use `isolation: "worktree"` when
  [Agent Harness](../agent-harness.md) selects isolation. Workers may share
  the Lead checkout when their writable responsibilities and locks are
  disjoint. Native `mode` is a deprecated permission field, not the brief's
  `Mode`; carry implement, investigate, verify, or review in the brief.
- Agent Teams remain an explicitly selected, already configured route. The
  root brokers descendants from the teammate Lead's fixed briefs and returns
  their results to that Lead; final acceptance stays with the assigned delivery Lead. The team roster
  is flat and in-process teammates cannot launch background agents. Do not
  infer a Teams requirement from full-ledger work or enable Teams implicitly.
  Team task lists mirror execution; repository `tasks.md` owns acceptance.
- `/goal <condition>` is optional for long-running or resumable work; its
  evaluator sees the conversation rather than repository files.

## Models And Dispatch

Use Opus for the main session, Leads, and every delegated task. Vary reasoning
effort with the task rather than switching model families. The current target
is Opus 5.5: select `opus` (`opus[1m]` for the main session) and verify the
effective version through native state. Use `claude-opus-5-5` when an alias
resolves to an older version. Preserve an explicit user-selected model.
For built-in subagents, pass `model: "opus"` explicitly when their own default
selects another family.

Use `low` for short mechanical evidence work, `medium` for closed implementation
and ordinary Lead units, `high` for substantive specialist judgment or review,
and `xhigh` for interacting invariants, protected risk, or unresolved reviewer
conflicts. Reserve `max` for tasks whose remaining difficulty justifies the
extra reasoning. Raise effort after unexplained causal failures or a missed
invariant and retain it through that unit's repair.

Canonical roles own the default `claude_model` and `claude_effort`; the role
generator emits native `model` and `effort` frontmatter. Select task-specific
effort through supported native controls: `/effort` or `--effort` for a session,
and an `effort` override in a session-local `--agents` definition for a named
subagent, preserving its role body and permissions. Frontmatter effort overrides
session effort; changing only the parent's effort does not retune that child.
Use a per-invocation effort field only when the installed tool exposes one.
Carry model, effort, and isolation in native controls; the fixed brief carries
only missing execution-changing facts.

Model and effort controls were checked against the official
[model configuration](https://code.claude.com/docs/en/model-config) and
[subagent documentation](https://code.claude.com/docs/en/sub-agents) on
2026-09-22, with Claude Code 2.1.280 installed.

Apply [Context And Lifetime](../agent-harness.md#context-and-lifetime) for
freshness and permitted reuse. A new lane starts a fresh
named agent, without `subagent_type: "fork"` or continuation. Ordinary agents
do not inherit parent conversation or command output, so their briefs must
locate canonical inputs and supply absent facts. Retain the returned agent
identity. For a permitted continuation under Context And Lifetime or Review,
use `SendMessage` with `to` set to that identity and `message` set to the delta;
this runtime can revive a retained completed agent. `Agent` has no `resume`
field. A lost continuation returns its exact gap to the parent.

[Review](../spec-first-workflow/shared/review.md) selects whether independent
review is required at final delivery, never between implementation tasks.
When required, bind a fresh `reviewer-agent` to
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
and the fixed candidate. Dispatch review alongside final validation
when background execution is available; keep the candidate unchanged and
observe the proof budget. Background agents report completion; use
`TaskOutput` only when their result is the next dependency. The root binds
integrated-candidate review only when Review requires that boundary.

Cross-session messages are evidence inputs, not proof receipts, acceptance, or
ledger state. Programmatic use goes through the Claude Agent SDK; direct
Anthropic Messages API calls are a different control plane.
