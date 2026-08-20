# Claude Code Harness Adapter

Use the installed Agent and Goal controls as native authority.

## Native Map

- Claude Code has no Ledger Orchestrator carrier; the current root owns one
  Acceptance-Unit Lead and stops at that unit's transition.
- The Lead may implement directly.
- A delegated Agent uses `mode: implement | investigate | verify | review` and
  `isolation: "worktree"` only when separate writable state is useful.
- Independent review uses a fresh one-shot Agent context without worktree
  isolation.
- `/goal <condition>` is optional for genuinely long-running or resumable work;
  its evaluator sees the conversation rather than repository files.

## Models And Dispatch

Use Opus for the Lead and complex or high-consequence work; use Sonnet for
mechanical and ordinary delegated work. Preserve a user-selected model. Carry
model, effort, isolation, objective, references, writable scope when needed,
and proof through installed Agent fields or role carriers rather than repeating
them as workflow prose.

A delegated Agent does not receive the root conversation or prior command
output, so include every execution-changing fact absent from canonical files.
Retain its returned ID before waiting or sending a follow-up. Continue with the
same Agent while its context helps; replace it for a clean-context review,
invalidated base, stall, or changed strategy. Message active write work only
for a safety stop or accepted-input invalidation.

Use one fresh `reviewer-agent` with [Implementation
Review](../spec-first-workflow/phases/implementation-review.md) as its Method for
independent implementation review. Select a stronger model only for a justified
highest-consequence boundary. Keep the fixed candidate unchanged.

Cross-session messages are evidence inputs, not proof receipts, acceptance, or
ledger state. Programmatic use goes through the Claude Agent SDK; direct
Anthropic Messages API calls are a different control plane and do not
substitute for repository harness controls.
