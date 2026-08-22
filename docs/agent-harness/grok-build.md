# Grok Build Harness Adapter

Use installed Grok primary-session and `spawn_subagent` controls as native
authority. Structured flags and spawn fields outrank public prose.

Grok has two session classes. A primary session can spawn one hop. A spawned
child cannot spawn. Keep the semantic tree; change only the carrier.

## Native Map

- The ordinary user-facing Grok primary session is the root carrier. A user
  prompt in that session is the launch. Do not require
  `--agent orchestrator`, a prepared CLI prompt, or a second process.
  `--agent orchestrator` remains an optional explicit bind.
- Bind this session as LEDGER_ORCHESTRATOR when a persisted Implementation
  ledger has sibling units to schedule or owner-held recovery to route. It
  fills the independent ready frontier and does not implement, review, or
  call `spawn_subagent` for unit work. Dispatch every ready unit before
  waiting, within current capacity. Land each candidate serially from the
  [Acceptance Result](../spec-first-workflow/interfaces/acceptance-result-v1.md)
  and record the Lead-owned verdict without re-adjudicating it.
- Bind this session as ACCEPTANCE_UNIT_LEAD when exactly one fixed
  implementation unit is in scope. Direct Work stays on this session and is
  not ledger orchestration.
- Each sibling Acceptance-Unit Lead is a new primary session, not a
  subagent. Create it with `grok --agent acceptance-unit-lead
  --always-approve --output-format json`, a unit locator through
  `--prompt-file` or `-p`, `--model` and `--effort` when they differ from
  this session, and `--worktree` when [Agent
  Harness](../agent-harness.md) selects isolation for that Lead. Retain
  `sessionId`. Continue with `--resume`.
- That Lead applies Implementation's execution topology. Delegated
  implement, investigate, verify, and review lanes use `spawn_subagent` with
  `worker-agent`, `specialist-agent`, `evidence-agent`, `reviewer-agent`, or
  `adjudicator-agent`. Installed spawn fields are `prompt`, `description`,
  `subagent_type`, `background`, `isolation`, `resume_from`, and `cwd`. Do
  not pass `capability_mode`. A child's tools come from its agent type and
  `.grok/roles/<name>.toml` `default_capability_mode`. Workers inside one
  Lead may share that Lead's checkout when writable responsibility and
  exclusive locks are disjoint. Do not create worktrees for sequential work
  or bounded read-only review.
- Spawned children do not receive `ask_user_question`. Lane and Lead agents
  pin `permission_mode: bypassPermissions` so tool calls do not wait for
  approval. Read-only versus mutable is the role capability default, not
  plan mode. Do not wait on a child question.
- Independent review uses one fresh `reviewer-agent` with no `resume_from`
  and no worktree isolation.
- Spawned children cannot take descendants. The Lead fans out any
  independently writable subset itself.
- `/goal` is optional on this session only for a genuinely long-running
  ledger. A Rhai workflow is not the Ledger Orchestrator.

## Models And Dispatch

Use `grok-4.6` or `grok-build` with `high` effort for this Orchestrator
session. Lead `--effort` follows [Agent Harness](../agent-harness.md)
capability routing. Child lanes inherit the Lead
model unless a role pins a stronger one. Child effort is each role's
`reasoning_effort` default. Spawn has no `model` or `effort` field. Preserve
a user-selected model. Do not encode a model name only in prompt text.

Retain every Lead `sessionId` and child `subagent_id` before waiting. Never
wait on a lane that returned no identity. Dispatch independent ready lanes
before waiting, within current capacity. Concurrent mutable units and lanes
require disjoint packet mutable owners and exclusive locks. Consume and
integrate results serially under the Lead.

A Lead prompt is the unit locator plus the [Subagent Brief
Template](../subagent-brief-template.md) for any delegated lane. A missing
identity, or `spawn_subagent` from a child, is a carrier failure, not a
completed lane.

## Review And Recovery

An independent implementation review uses a fresh `reviewer-agent` with
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
as its Method. When the ledger requires integrated-candidate review, this
session binds one fresh `reviewer-agent` to that boundary and still does not
accept units. Raise the reviewer role model pin only for a justified
highest-consequence boundary. Keep the fixed candidate unchanged.

If implementation invalidates an upstream decision, the Lead repairs the
smallest owner when it can, or the Orchestrator opens a fresh Lead for that
phase, waits for its canonical transition, and resumes the same unit. Add no
scheduler, journal, or recovery database.
