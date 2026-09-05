# Cursor Harness Adapter

Use installed Task, skill, Goal, and worktree controls as native authority.
Callable Task fields outrank public prose.

## Native Map

- This session is Cursor. Load only this adapter. Sibling bootstrap files
  select other harnesses; do not follow their adapter choice.
- `/orchestrator` binds the current session as Ledger Orchestrator. Cursor
  reads `.agents/skills` directly.
- Dispatch every mutually independent ready unit before waiting, within
  current capacity. Assign an `acceptance-unit-lead` through Task, reusing a
  completed related Lead only under the shared ledger contract. That Lead owns
  proof, any required review, and the acceptance verdict;
  this session lands only `Accepted` candidates serially from the
  [Acceptance Result](../spec-first-workflow/interfaces/acceptance-result-v1.md)
  and records that verdict without re-adjudicating it.
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
- When [Agent Harness](../agent-harness.md) selects isolation, use a Git
  worktree, or Task `environment: "cloud"` when a separate VM and branch are
  useful. Workers inside one Lead may share the Lead checkout when writable
  responsibility and exclusive locks are disjoint. Isolation is not a
  ceremony for sequential work, cheap disjoint units, or bounded read-only
  review.
- Independent review uses one fresh `reviewer-agent` with no `resume` and no
  worktree isolation. `readonly: true` is the role default.
- `/goal` is optional and rolling out; use it only for genuinely long-running
  or resumable work.

## Models And Dispatch

Project agents default to `inherit`. Use inherit or a faster configured model
for a closed, strongly owned Lead unit; use the strongest configured model
when remaining uncertainty, protected-risk surface, or high consequence
requires it, including the adjudicator. Use inherit or a faster configured
model for mechanical lanes. Preserve a user-selected model. Carry model through the
Task `model` field; do not encode a model name only in prompt text.

Pass the [delegation interface](../agent-harness.md#delegation-interface)
through Task fields: `subagent_type`, `prompt`, `description`, `model`,
`resume`, `run_in_background` only when the parent must continue, and
`environment` when cloud isolation is required. Retain every returned agent
ID before waiting. Never wait on a lane that returned no identity. Dispatch
independent ready lanes in one parent message before waiting, within current
capacity. Concurrent mutable units and lanes require disjoint packet
mutable owners and exclusive locks. Consume and integrate results serially
under the Lead.

Apply shared [Context And Lifetime](../agent-harness.md#context-and-lifetime).
New execution/evidence tasks and initial independent review omit `resume`;
do not fork the parent's conversation. Resume a returned agent ID only for
same-brief repair, shared Review's permitted delta recheck, or an admitted Lead
reassignment. A missing identity is a carrier failure, not a completed lane.
Shared Nested Execution applies within Cursor's two-level limit; additional
worker subsets return to the Lead. Use native background Task when review and
non-mutating checks should overlap; acceptance still waits for required results.

## Review And Recovery

Start a required independent implementation review with a fresh `reviewer-agent` and
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
as its Method; shared Review owns continuation. When Review requires integrated-candidate review, this
session binds one fresh `reviewer-agent` to that boundary and still does not
accept units. Raise its Task `model` field only for a justified
highest-consequence boundary. Keep the fixed candidate unchanged.

If implementation invalidates an upstream decision, the Lead repairs the
smallest owner when it can, or this session dispatches a fresh actor governed
by the reopened phase, consumes its Transition Result, and resumes the same
implementation unit. Add no scheduler, journal, or recovery database.
