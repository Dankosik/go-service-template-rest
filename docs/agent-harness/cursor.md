# Cursor Harness Adapter

Use installed Task, skill, Goal, and worktree controls as native authority.
Callable Task fields outrank public prose.

## Native Map

- This session is Cursor. Load only this adapter. Sibling bootstrap files
  select other harnesses; do not follow their adapter choice.
- `/orchestrator` binds the current session as Ledger Orchestrator. Cursor
  reads `.agents/skills` directly.
- Dispatch every mutually independent ready unit before waiting, within
  current capacity. Spawn one fresh `acceptance-unit-lead` per ready unit
  through Task. That Lead owns proof, review, and the canonical transition;
  this session only routes. Land each Accepted candidate onto the current
  checkout serially from the ledger receipt.
- Bind that teammate to `acceptance-unit-lead`. Do not substitute
  `generalPurpose`, `explore`, `shell`, or generic worker semantics.
- Full-ledger work requires Task plus returned agent identities. If Task
  cannot spawn `acceptance-unit-lead` or returns no identity, report that
  exact carrier gap before dispatch.
- The Lead may implement directly. Delegated implement, investigate, verify,
  or review uses Task with `worker-agent`, `specialist-agent`,
  `evidence-agent`, `reviewer-agent`, or `adjudicator-agent`.
- The main session and its direct Task children may spawn one descendant hop.
  A grandchild cannot spawn. The Lead fans out any independently writable
  subset itself.
- Isolated candidate state uses a Git worktree, or Task `environment:
  "cloud"` when a separate VM and branch are useful. Isolation is not a
  ceremony for every edit.
- Independent review uses one fresh `reviewer-agent` with no `resume` and no
  worktree isolation. `readonly: true` is the role default.
- `/goal` is optional and rolling out; use it only for genuinely long-running
  or resumable work.

## Models And Dispatch

Project agents default to `inherit`. Use the strongest configured model for
the Lead, adjudicator, or complex, weak-oracle, protected-domain, or
high-consequence work. Use inherit or a faster configured model for
mechanical lanes. Preserve a user-selected model. Carry model through the
Task `model` field; do not encode a model name only in prompt text.

Pass the [delegation interface](../agent-harness.md#delegation-interface)
through Task fields: `subagent_type`, `prompt`, `description`, `model`,
`resume`, `run_in_background` only when the parent must continue, and
`environment` when cloud isolation is required. Retain every returned agent
ID before waiting. Never wait on a lane that returned no identity. Dispatch
independent ready lanes in one parent message before waiting, within current
capacity. Concurrent mutable lanes require disjoint files, resources,
interfaces, and assumptions. Consume and integrate results serially under
the Lead.

Continue the same agent while its context helps; use a fresh agent for
independent review, an invalidated base, a stall, or a changed strategy. A
missing identity is a carrier failure, not a completed lane.

## Review And Recovery

An independent implementation review uses a fresh `reviewer-agent` with
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
as its Method. When Review requires integrated-candidate review, this
session binds one fresh `reviewer-agent` to that boundary and still does not
accept units. Raise its Task `model` field only for a justified
highest-consequence boundary. Keep the fixed candidate unchanged.

If implementation invalidates an upstream decision, the Lead repairs the
smallest owner when it can, or this session opens a fresh
`acceptance-unit-lead` for that phase, waits for its canonical transition,
and resumes the same unit. Add no scheduler, journal, or recovery database.
