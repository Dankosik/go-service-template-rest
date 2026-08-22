# Claude Code Harness Adapter

Use the installed Agent and Goal controls as native authority.

## Native Map

- `/orchestrator` binds the current session as Ledger Orchestrator when Agent
  Team controls are callable. It uses the shared task list only for execution;
  repository `tasks.md` remains the acceptance ledger.
- Dispatch every mutually independent ready unit before waiting, within current
  capacity. One teammate owns each ready Acceptance Unit through proof, fresh
  review, and its acceptance verdict. The team lead lands each candidate
  serially from the
  [Acceptance Result](../spec-first-workflow/interfaces/acceptance-result-v1.md)
  and records that verdict without re-adjudicating or implementing the unit.
- Bind that teammate to the canonical `acceptance-unit-lead` carrier in its
  spawn brief; do not substitute generic worker semantics.
- Without Agent Team controls, ordinary single-unit work remains available, but
  full-ledger invocation returns the exact carrier gap before dispatch. A
  one-shot subagent cannot substitute because it cannot spawn descendants.
- Agent Teams are experimental and user-configured. Name the missing team
  capability or enablement condition; do not write user settings implicitly.
- The Lead may implement directly.
- Use `isolation: "worktree"` when [Agent Harness](../agent-harness.md)
  selects isolation for a concurrent mutable Lead. A delegated Agent uses
  `mode: implement | investigate | verify | review`. Workers inside one Lead
  may share the Lead checkout when writable responsibility and exclusive
  locks are disjoint. Do not create worktrees for sequential work, cheap
  disjoint units, or bounded read-only review.
- Independent review uses a fresh one-shot Agent context without worktree
  isolation.
- `/goal <condition>` is optional for genuinely long-running or resumable work;
  its evaluator sees the conversation rather than repository files.

## Models And Dispatch

Use Sonnet for a closed, strongly owned Lead unit; use Opus when remaining
uncertainty, protected-risk surface, or high consequence requires it. Use
Sonnet for mechanical and ordinary delegated work. Preserve a user-selected
model. Carry model, effort, isolation, objective, references, writable scope
when needed, and proof through installed Agent fields or role carriers rather
than repeating them as workflow prose.

A delegated Agent does not receive the root conversation or prior command
output, so include every execution-changing fact absent from canonical files.
Retain its returned ID before waiting or sending a follow-up. Continue with the
same Agent while its context helps; replace it for a clean-context review,
invalidated base, stall, or changed strategy. Message active write work only
for a safety stop or accepted-input invalidation.

For Agent Teams, retain the team, teammate, and shared-task identities. Route
review or worker creation from the Acceptance-Unit Lead's fixed brief; the team
lead may carry the spawn only when Claude exposes that control solely to the
lead, but ownership and the returned verdict remain with the unit Lead.

Use one fresh `reviewer-agent` with [Implementation
Review](../spec-first-workflow/phases/implementation-review.md) as its Method for
independent implementation review. When Review requires integrated-candidate
review, the team lead binds one fresh `reviewer-agent` to that boundary and
still does not accept units. Select a stronger model only for a justified
highest-consequence boundary. Keep the fixed candidate unchanged.

Cross-session messages are evidence inputs, not proof receipts, acceptance, or
ledger state. Programmatic use goes through the Claude Agent SDK; direct
Anthropic Messages API calls are a different control plane and do not
substitute for repository harness controls.
