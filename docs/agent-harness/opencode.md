# OpenCode Harness Adapter

Use installed OpenCode 1 agent, Task, skill, and command controls as native
authority. Callable Task fields and agent frontmatter outrank public prose.
This adapter is for `opencode`. `opencode2` is a different schema; do not mix
those control planes.

## Native Map

- This session is OpenCode. Load only this adapter. Sibling bootstrap files
  select other harnesses; do not follow their adapter choice. Default `build`
  remains Direct Work and is not ledger orchestration.
- `/orchestrator` binds the current session as Ledger Orchestrator. OpenCode
  reads `.agents/skills` through the `skill` tool. Do not require a second
  process for that bind.
- Dispatch every mutually independent ready unit before waiting, within
  current capacity. Spawn one fresh `acceptance-unit-lead` per ready unit
  through Task. That Lead owns proof, review, and the canonical transition;
  this session only routes. Land each Accepted candidate onto the current
  checkout serially from the ledger receipt.
- Bind that teammate to `acceptance-unit-lead`. Do not substitute `general`,
  `explore`, `scout`, or generic worker semantics.
- Full-ledger work requires Task, returned session identities, and
  `subagent_depth` of at least 2 so a Lead can spawn child lanes. If Task
  cannot spawn `acceptance-unit-lead`, returns no identity, or a Lead cannot
  spawn because depth is 1, report that exact carrier gap before dispatch.
- The Lead may implement directly. Delegated implement, investigate, verify,
  or review uses Task with `worker-agent`, `specialist-agent`,
  `evidence-agent`, `reviewer-agent`, or `adjudicator-agent`.
- A primary session and a Task-spawned Lead may spawn one descendant hop.
  A grandchild cannot spawn. The Lead fans out any independently writable
  subset itself.
- OpenCode Task has no isolation field. Use a Git worktree, or `opencode
  --agent acceptance-unit-lead --dir <worktree>`, only when collision or
  candidate-state risk justifies it. Isolation is not a ceremony for every
  edit. Do not require an experimental workspace flag or a community plugin.
- Independent review uses one fresh `reviewer-agent` with no `task_id` and
  no worktree isolation. `edit: deny` is the role default. Generated lanes
  set `hidden: true`; invoke them through Task, not `@`.
- There is no native Goal. Do not invent one. Background Task requires
  `OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS`; foreground is the default.
- OpenCode ignores `disable-model-invocation`. `build` and `plan` deny
  `user/workflow` and `role/carrier` skill names. Slash commands, `/orchestrator`,
  and bound agent files still reach them.
- Do not commit OpenCode runtime files. `.opencode/.gitignore` excludes
  `node_modules/`, `package.json`, and `package-lock.json`.

## Models And Dispatch

Connect xAI with `/connect` (SuperGrok or API key). Use `xai/grok-4.6` with
variant `high` for this Orchestrator session and `xhigh` on each Lead. OpenCode
applies `variant` only when the agent pins a model, so child lanes pin
`xai/grok-4.6` plus the role's `grok_effort` variant instead of inheriting the
parent's effort. Preserve a user-selected model on the primary session. Carry
model and variant through agent frontmatter; Task has no model or variant
field. Do not encode a model name only in prompt text. Do not invent a variant
the installed model does not list.

Pass the [delegation interface](../agent-harness.md#delegation-interface)
through Task fields: `subagent_type`, `prompt`, `description`, and `task_id`
only to resume the same child session. Retain every returned Task session id
before waiting. Never wait on a lane that returned no identity. Dispatch
independent ready lanes in one parent message before waiting, within current
capacity. Concurrent mutable lanes require disjoint files, resources,
interfaces, and assumptions. Consume and integrate results serially under
the Lead.

Continue the same agent with `task_id` while its context helps; use a fresh
agent for independent review, an invalidated base, a stall, or a changed
strategy. A missing identity is a carrier failure, not a completed lane.
Spawned children do not receive `question`. The Orchestrator primary may use
`question` for an `AGENTS.md` user-owned decision. Do not wait on a child
question.

## Review And Recovery

An independent implementation review uses a fresh `reviewer-agent` with
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
as its Method. When Review requires integrated-candidate review, this
session binds one fresh `reviewer-agent` to that boundary and still does not
accept units. Raise its agent model pin only for a justified
highest-consequence boundary. Keep the fixed candidate unchanged.

If implementation invalidates an upstream decision, the Lead repairs the
smallest owner when it can, or this session opens a fresh
`acceptance-unit-lead` for that phase, waits for its canonical transition,
and resumes the same unit. Add no scheduler, journal, or recovery database.
