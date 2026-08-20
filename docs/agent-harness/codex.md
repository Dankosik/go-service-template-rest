# Codex Harness Adapter

Use the installed Codex schemas as native authority; callable fields outrank
public prose.

## Native Map

- A saved-project task using `$orchestrator` can own a persisted ledger.
- A Local Acceptance-Unit Lead applies Implementation's direct-versus-delegate
  rule and uses shared `worker-agent` lanes for delegated mutable work.
- Use a Worktree task when isolated candidate state prevents collisions or
  makes handoff useful; isolation is not a correctness ceremony for every edit.
- When the callable collaboration schema permits it, a child may delegate a
  strict subset of its brief to descendants without broadening writable scope
  or authority.
- Use a fresh project subagent with no inherited turns for independent review.
- A Goal is optional and thread-local; use one only for a genuinely long-running
  or resumable stage.

## Models

Use Sol with `xhigh` reasoning effort for the Acceptance-Unit Lead. `ultra` is
a Codex workflow mode, not a reasoning-effort value, and must not be sent in
the effort field. Use Luna at low effort for closed mechanical work, Terra at
balanced effort for ordinary delegated work or review, and Sol for complex,
cross-cutting, protected-domain, or high-consequence reasoning. Preserve a
user-named model. If a structured field is unsupported or rejected, retain the
native evidence and use the effective configured value; do not encode a model
name only in prompt text.

## Dispatch And Coordination

Pass the [delegation interface](../agent-harness.md#delegation-interface)
through installed structured fields where available. Retain every returned
`threadId`, `hostId`, task or operation identity, worktree identity, and wait
cursor. Never wait on a lane that returned no identity. Dispatch all independent
ready lanes before waiting, within current capacity; capacity is a ceiling, not
a fan-out target. Concurrent mutable lanes require disjoint files, resources,
interfaces, and assumptions. Consume and integrate results serially under the
Lead.

Send only the changed dependency, question, status, or identity: use a direct
message for an active agent and a follow-up to resume an idle one. Send sibling
dependencies to the affected sibling and inform the parent when they change a
shared assumption or acceptance state. Keep same-task corrections with the same
agent while its context remains useful; do not resend the full brief.

When initial task creation cannot carry the selected model or effort, bootstrap
with role and scope only, require exact `READY_FOR_DISPATCH`, then send one
technical follow-up using the supported structured fields. Do not repeat an
ambiguously delivered dispatch. Use a fresh agent when a clean context or
changed strategy is more reliable.

For isolated work, validate the actual worktree and base before accepting its
bytes. A Worktree Lead returns exact `HANDOFF_READY` with a fixed candidate;
the Local Lead integrates and validates it before any ledger transition.
Handoff is routing evidence, not acceptance.

## Review And Recovery

An independent implementation review uses a fresh `reviewer-agent` with
Implementation Review as its Method. Raise its model/effort fields for a
justified highest-consequence boundary. A changed candidate receives a fresh
review when the trigger still applies.

Reconcile unknown create or handoff state from native task state, the canonical
ledger, and Git candidate identity. Zero or multiple exact matches remain an
unknown outcome; do not redispatch or repeat Handoff blindly. If implementation
invalidates an upstream decision, the Lead repairs the smallest owner when it
can, or the Orchestrator opens a fresh task for that phase and resumes the same
unit. Add no scheduler, journal, or recovery database.

Archive a child only after its result and candidate are safe and no continuation
needs its identity.
