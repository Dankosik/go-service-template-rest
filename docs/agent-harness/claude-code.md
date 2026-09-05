# Claude Code Harness Adapter

Use the installed Agent and Goal controls as native authority. The ordinary
nested route is supported by Claude Code 2.1.227; recheck native controls after
runtime changes.

## Native Map

- `/orchestrator` binds the current session as Ledger Orchestrator. Dispatch
  mutually independent ready units through `Agent` with
  `subagent_type: "acceptance-unit-lead"`, within current capacity. The native
  [Lead carrier](../../.claude/agents/acceptance-unit-lead.md) loads the existing
  role skill and owns its fixed unit through proof, required review, and its
  acceptance verdict. The root lands only `Accepted` candidates serially from
  [Acceptance Result](../spec-first-workflow/interfaces/acceptance-result-v1.md)
  and records the verdict without re-adjudicating or implementing the unit.
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
  their results to that Lead; acceptance stays with the Lead. The team roster
  is flat and in-process teammates cannot launch background agents. Do not
  infer a Teams requirement from full-ledger work or enable Teams implicitly.
  Team task lists mirror execution; repository `tasks.md` owns acceptance.
- `/goal <condition>` is optional for long-running or resumable work; its
  evaluator sees the conversation rather than repository files.

## Models And Dispatch

Use Sonnet for a closed, strongly owned Lead unit and ordinary delegated work;
use Opus when uncertainty, protected risk, or high consequence requires it.
Preserve a user-selected model. Carry supported model, effort, and isolation
controls through native fields or the role carrier; the fixed brief carries
only the missing execution-changing facts.

Apply [Context And Lifetime](../agent-harness.md#context-and-lifetime) for
freshness and ledger-controlled Lead reuse. A new small task starts a fresh
named agent, without `subagent_type: "fork"` or continuation. Ordinary agents
do not inherit parent conversation or command output, so their briefs must
locate canonical inputs and supply absent facts. Retain the returned agent
identity. For a valid same-brief correction, including a review correction,
use `SendMessage` with `to` set to that identity and `message` set to the delta;
this runtime can revive a retained completed agent. `Agent` has no `resume`
field. A lost continuation returns its exact gap to the parent.

[Review](../spec-first-workflow/shared/review.md) selects whether independent
review is required. When required, bind a fresh `reviewer-agent` to
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
and the fixed candidate. Dispatch review before independent focused proof
when background execution is available; keep the candidate unchanged and
observe the proof budget. Background agents report completion; use
`TaskOutput` only when their result is the next dependency. The root binds
integrated-candidate review only when Review requires that boundary.

Cross-session messages are evidence inputs, not proof receipts, acceptance, or
ledger state. Programmatic use goes through the Claude Agent SDK; direct
Anthropic Messages API calls are a different control plane.
