# Grok Build Harness Adapter

Use installed Grok primary-session and `spawn_subagent` controls as native
authority. Structured flags and spawn fields outrank public prose.

Grok has two session classes. A primary session can spawn one hop. A spawned
child cannot spawn. Keep the semantic tree; change only the carrier.

## Native Map

- The ordinary user-facing Grok primary session is the Orchestrator carrier.
  A user prompt in that session is the launch. Do not require
  `--agent orchestrator`, a prepared CLI prompt, or a second process for
  routing. `--agent orchestrator` remains an optional explicit bind.
- When a persisted Implementation ledger is ready, this session binds as
  LEDGER_ORCHESTRATOR: it routes ready units and owner-held recovery. It does
  not implement, review, or call `spawn_subagent` for unit work.
- Each Acceptance-Unit Lead is a new primary session, not a subagent. This
  Orchestrator creates it with `grok --agent acceptance-unit-lead
  --always-approve --output-format json`, a unit locator through
  `--prompt-file` or `-p`, `--model` and `--effort` when they differ from
  this session, and `--worktree` only when isolated candidate state is
  useful. Retain `sessionId`. Continue with `--resume`.
- That Lead applies Implementation's direct-versus-delegate rule. Delegated
  implement, investigate, verify, and review lanes use `spawn_subagent` with
  `worker-agent`, `specialist-agent`, `evidence-agent`, `reviewer-agent`, or
  `adjudicator-agent`. Installed spawn fields are `prompt`, `description`,
  `subagent_type`, `background`, `isolation`, `resume_from`, and `cwd`. Do
  not pass `capability_mode`. A child's tools come from its agent type and
  `.grok/roles/<name>.toml` `default_capability_mode`. Isolation is
  `worktree` only when collision or candidate-state risk justifies it.
- Spawned children do not receive `ask_user_question`. Lane and Lead agents
  pin `permission_mode: bypassPermissions` so tool calls do not wait for
  approval. Read-only versus mutable is the role capability default, not
  plan mode. Do not wait on a child question.
- Independent review uses one fresh `reviewer-agent` with no `resume_from`
  and no worktree isolation.
- Spawned children cannot take descendants. The Lead fans out any
  independently writable subset itself.
- `/goal` is optional on this session only for a genuinely long-running
  ledger. A Rhai workflow is not the Ledger Orchestrator. Direct Work stays
  on this session and is not ledger orchestration.

## Models And Dispatch

Use `grok-4.6` or `grok-build` with `high` effort for this Orchestrator
session and `xhigh` on each Lead `--effort`. Child lanes inherit the Lead
model unless a role pins a stronger one. Child effort is each role's
`reasoning_effort` default. Spawn has no `model` or `effort` field. Preserve
a user-selected model. Do not encode a model name only in prompt text.

Retain every Lead `sessionId` and child `subagent_id` before waiting. Never
wait on a lane that returned no identity. Dispatch independent ready lanes
before waiting, within current capacity. Concurrent mutable lanes require
disjoint files, resources, interfaces, and assumptions. Consume and
integrate results serially under the Lead.

A Lead prompt is the unit locator plus the [Subagent Brief
Template](../subagent-brief-template.md) for any delegated lane. A missing
identity, or `spawn_subagent` from a child, is a carrier failure, not a
completed lane.

## Review And Recovery

An independent implementation review uses a fresh `reviewer-agent` with
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
as its Method. Raise the reviewer role model pin only for a justified
highest-consequence boundary. Keep the fixed candidate unchanged.

If implementation invalidates an upstream decision, the Lead repairs the
smallest owner when it can, or the Orchestrator opens a fresh Lead for that
phase, waits for its canonical transition, and resumes the same unit. Add no
scheduler, journal, or recovery database.
