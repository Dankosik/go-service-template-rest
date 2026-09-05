# OpenCode Harness Adapter

Use installed OpenCode 1 agent, Task, skill, and command controls as native
authority. Callable Task fields and agent frontmatter outrank public prose.
This adapter is for `opencode`. `opencode2` is a different schema; do not mix
those control planes.

## Native Map

- This session is OpenCode. Load only this adapter. Sibling bootstrap files
  select other harnesses; do not follow their adapter choice. Default `build`
  remains Direct Work until the user asks to orchestrate a persisted ledger.
- A user request to orchestrate, or to run a ready Implementation ledger, is
  the launch. This session routes from `build` or `orchestrator`. Do not ask
  the user to Tab, `/agent`, `/orchestrator`, or start a second process.
  `/orchestrator` remains an optional tighter bind (`permission.task` allowlist).
  OpenCode reads `.agents/skills` through the `skill` tool; `build` may load
  `orchestrator`.
- Dispatch every mutually independent ready unit before waiting, within
  current capacity. Assign an `acceptance-unit-lead` through Task, reusing a
  completed related Lead only under the shared ledger contract. That Lead owns
  proof, any required review, and the acceptance verdict.
  This session lands only `Accepted` candidates and records verdicts through the
  [Ledger Contract's acceptance transition](../spec-first-workflow/phases/planning/ledger-contract.md#acceptance-transition)
  from the Lead's immutable Acceptance Result.
- Bind that teammate to `acceptance-unit-lead`. Task `subagent_type` is a free
  string. `.opencode/plugins/task-subagents.js` appends the project catalog to
  the Task blurb. Pass `acceptance-unit-lead` anyway if the blurb is stale. Do
  not substitute `general`, `explore`, `scout`, or generic worker semantics.
- Full-ledger work requires Task, returned session identities, and
  `subagent_depth` of at least 2 so a Lead can spawn child lanes. Report a
  carrier gap only after Task rejects the call, returns unknown agent, returns
  no identity, or a Lead cannot spawn because depth is 1.
- The Lead may implement directly. Delegated implement, investigate, verify,
  or review uses Task with `worker-agent`, `specialist-agent`,
  `evidence-agent`, `reviewer-agent`, or `adjudicator-agent`.
- `opencode.json` sets `subagent_depth: 3`, permitting root -> Lead -> worker
  -> worker. This is a portable configured ceiling, not a fixed product limit.
  Generated workers allow Task only to `worker-agent` and `evidence-agent`;
  review roles retain Task denial. Apply shared Nested Execution; at the depth
  limit return the subset to the nearest capable parent.
- OpenCode Task has no isolation field. Use a Git worktree, or `opencode
  --agent acceptance-unit-lead --dir <worktree>`, only when collision or
  candidate-state risk justifies it. Isolation is not a ceremony for every
  edit. Do not require an experimental workspace flag or a community plugin;
  `.opencode/plugins/task-subagents.js` is repository-owned Task catalog.
- Independent review uses one fresh `reviewer-agent` with no `task_id` and
  no worktree isolation. `edit: deny` is the role default. Generated lanes
  set `hidden: true`; invoke them through Task, not `@`.
- There is no native Goal. Do not invent one. Background Task requires
  `OPENCODE_EXPERIMENTAL_BACKGROUND_SUBAGENTS`; foreground is the default.
- OpenCode ignores `disable-model-invocation`. `build` and `plan` deny
  `user/workflow` skills. The `orchestrator` skill stays loadable on `build`.
  Other `role/carrier` entries are Task `subagent_type` values.
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
only for an allowed continuation. Use `background` only when its native feature
is enabled and overlap is useful; otherwise use supported foreground calls and
non-mutating shell checks without inventing a background setting. Retain every returned Task session id
before waiting. Never wait on a lane that returned no identity. Dispatch
independent ready lanes in one parent message before waiting, within current
capacity. Concurrent mutable lanes require disjoint files, resources,
interfaces, and assumptions. Consume and integrate results serially under
the Lead.

Apply shared [Context And Lifetime](../agent-harness.md#context-and-lifetime).
New execution/evidence tasks and initial independent review omit `task_id`.
Use a current returned `task_id` only for same-brief repair, shared Review's
permitted delta recheck, or an admitted Lead reassignment. A missing identity
is a carrier failure, not a completed lane.
Spawned children do not receive `question`. The Orchestrator primary may use
`question` for an `AGENTS.md` user-owned decision. Do not wait on a child
question.

## Review And Recovery

Start a required independent implementation review with a fresh `reviewer-agent` and
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
as its Method; shared Review owns continuation. When Review requires integrated-candidate review, this
session binds one fresh `reviewer-agent` to that boundary and still does not
accept units. Raise its agent model pin only for a justified
highest-consequence boundary. Keep the fixed candidate unchanged.

If implementation invalidates an upstream decision, the Lead repairs the
smallest owner when it can, or this session dispatches a fresh actor governed
by the reopened phase, consumes its Transition Result, and resumes the same
implementation unit. Add no scheduler, journal, or recovery database.
