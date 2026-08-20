# Claude Code Read-Only Lanes

## Read When

Read immediately before dispatching or waiting on a Claude Code research,
challenge, or review Agent. Apply the shared [Read-Only
Carrier](../shared/read-only-carrier.md).

Use one fresh background `Agent` context per independently checkable question,
without worktree isolation. Bind `READ_ONLY_SPECIALIST` or
`ACCEPTANCE_REVIEWER`; a triggered implementation review uses a fresh one-shot
`task-acceptance-agent`, with a critical reviewer reserved for a justified
highest-consequence boundary.

Retain the returned Agent ID and address native wait or result retrieval only
through that identity. No returned identity means no lane; a wait with none is
a capability failure. Harness task lists never replace repository `tasks.md` or
its receipts, and a new candidate or unit receives a new review Agent.
