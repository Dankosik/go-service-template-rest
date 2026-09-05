# Grok Build Harness Adapter

Use installed Grok primary-session and `spawn_subagent` controls as native
authority. Structured flags and spawn fields outrank public prose.

The default subagent depth is one. Native configuration exposes
`subagents.max_depth` and `GROK_SUBAGENTS_MAX_DEPTH`; do not describe that default
as a fixed product ceiling. This adapter retains primary-session Leads until
effective depth and child spawn controls establish a deeper native route.

## Native Map

- The ordinary user-facing Grok primary session is the root carrier. A user
  prompt in that session is the launch. Do not require
  `--agent orchestrator`, a prepared CLI prompt, or a second process.
  `--agent orchestrator` remains an optional explicit bind.
- Bind this session as LEDGER_ORCHESTRATOR when
  [Implementation](../spec-first-workflow/phases/implementation.md#carrier)
  selects that carrier. It
  fills the independent ready frontier and does not implement, review, or
  call `spawn_subagent` for unit work. Dispatch every ready unit before
  waiting, within current capacity. Land only `Accepted` candidates serially from
  the
  [Acceptance Result](../spec-first-workflow/interfaces/acceptance-result-v1.md)
  and record the Lead-owned verdict without re-adjudicating it.
- Bind this session as ACCEPTANCE_UNIT_LEAD when Implementation selects the
  root-local Lead. Direct Work stays on this session and is not ledger
  orchestration.
- Assign each Acceptance-Unit Lead a primary session. Reuse a completed related
  Lead only under the shared ledger contract. Create a new one with `grok --agent acceptance-unit-lead
  --always-approve --output-format json`, a unit locator through
  `--prompt-file` or `-p`, `--model` and `--effort` when they differ from
  this session. When [Agent Harness](../agent-harness.md) selects isolation,
  prepare a Git worktree through repository lifecycle tools and pass its
  absolute path with `--cwd`. Headless `-p` does not create a worktree from
  `--worktree`. Retain `sessionId`; continue with `--resume`, refresh current
  checkout/packet inputs, and do not use `--restore-code` as routine continuation.
- That Lead applies Implementation's execution topology. Delegated
  implement, investigate, verify, and review lanes use `spawn_subagent` with
  `worker-agent`, `specialist-agent`, `evidence-agent`, `reviewer-agent`, or
  `adjudicator-agent`. Installed spawn fields are `prompt`, `description`,
  `subagent_type`, `background`, `isolation`, `resume_from`, `cwd`, and optional
  `model` when the installed schema permits explicit user-requested selection. Do
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
- At the effective depth limit, the Lead fans out additional worker subsets.
  Enable child delegation only after native controls and effective depth are
  established, applying shared Nested Execution. Do not infer enablement from
  a config key or public source for a different build.
- `/goal` is optional on this session only for a genuinely long-running
  ledger. A Rhai workflow is not the Ledger Orchestrator.

## Models And Dispatch

Use `grok-4.6` or `grok-build` with `high` effort for this Orchestrator
session. Lead `--effort` follows [Agent Harness](../agent-harness.md)
capability routing. Child lanes inherit the Lead
model unless a role pins a stronger one. Child effort is each role's
`reasoning_effort` default. Spawn has no `effort` field. Its optional `model`
override is ignored with `resume_from` and requires the schema's explicit user
selection authority; ordinary dispatch uses inheritance or role pins. Preserve
a user-selected model. Do not encode a model name only in prompt text.

Retain every Lead `sessionId` and child `subagent_id` before waiting. Never
wait on a lane that returned no identity. Dispatch independent ready lanes
before waiting, within current capacity. Concurrent mutable units and lanes
require disjoint packet mutable owners and exclusive locks. Consume and
integrate results serially under the Lead.

A Lead prompt is the unit locator plus the [Subagent Brief
Template](../subagent-brief-template.md) for any delegated lane. A missing
identity or rejected required spawn is a carrier failure, not a
completed lane.

Apply shared [Context And Lifetime](../agent-harness.md#context-and-lifetime).
New execution/evidence tasks and initial independent review omit `resume_from`.
For same-brief repair or a permitted review recheck, resume only a completed
source from the same parent and agent type. This continues its transcript,
model, and cwd; it may return a new child ID, which replaces the prior locator.
After a parent replacement, start a fresh bounded child if that lineage is no
longer resumable. Native background spawning may overlap fixed-candidate review
with non-mutating checks; deliver results without duplicating them.
Track primary Leads and their subtrees explicitly: stopping one primary does
not establish termination of sibling sessions or their descendants.

## Review And Recovery

Start a required independent implementation review with a fresh `reviewer-agent` and
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
as its Method; shared Review owns continuation. When shared Review requires integrated-candidate review, this
session binds one fresh `reviewer-agent` to that boundary and still does not
accept units. Raise the reviewer role model pin only for a justified
highest-consequence boundary. Keep the fixed candidate unchanged.

If implementation invalidates an upstream decision, the Lead repairs the
smallest owner when it can, or the Orchestrator dispatches a fresh actor governed
by the reopened phase, consumes its Transition Result, and resumes the same
implementation unit. Add no scheduler, journal, or recovery database.
