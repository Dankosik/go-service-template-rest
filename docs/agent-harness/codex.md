# Codex Harness Adapter

Use the installed Codex schemas as native authority; callable fields outrank
public prose.

## Native Map

- A saved-project task using `$orchestrator` can own a persisted ledger.
- Before Planning, the existing root can coordinate a full-outcome request
  through [Transition](../spec-first-workflow/shared/transition.md#cross-phase-continuation).
  Use fresh collaboration subagents for phase actors. At ready Planning, bind
  the current root as Ledger Orchestrator or dispatch a fixed-unit Lead as
  Implementation requires; preserve returned identities across that transition.
- Delegate internal work through subagents when their native controls preserve
  the selected topology. Creating a separate app task requires an explicit user
  request for that task; an implementation or phase-repair request alone does
  not authorize it. Check both native authorization and capability before
  selecting a carrier. If no authorized carrier preserves the required
  topology, report that exact gap before dispatch.
- A Local Acceptance-Unit Lead applies Implementation's execution topology
  and uses shared `worker-agent` lanes for delegated mutable work.
- A Worktree task is the isolation control when [Agent
  Harness](../agent-harness.md) selects isolation for a concurrent mutable
  Lead. Isolation is not a ceremony for sequential edits, cheap disjoint
  units, or read-only review. Workers inside one Lead may share that Lead's
  checkout when writable responsibility and exclusive locks are disjoint.
- Only the bound Lead fans out mutable Implementation execution lanes; those
  lanes do not spawn. Before Implementation, phase actors request specialist or
  reviewer work from the root coordinator through the Subagent Brief. The root
  dispatches and relays the result; the phase actor retains decision, repair,
  and transition ownership. This keeps review independent without requiring
  another native nesting level.
- Use a fresh project subagent with no inherited turns for independent review.
- A Goal is optional and thread-local. Create one only after an explicit user
  request or system/developer instruction; task duration is not authorization.

## Models

Use `gpt-6-astra` for every decision-owning role: the root coordinator,
Ledger Orchestrator, phase owner, Acceptance-Unit Lead, domain specialist,
independent reviewer, and adjudicator. Model capability does not merge
role responsibilities: the Orchestrator routes, the Lead accepts its unit,
and the fresh reviewer challenges a fixed candidate. Acceptance still requires
all mandatory proof and resolution of material review findings.

The installed Codex catalog supports `low`, `medium`, `high`, `xhigh`, `max`,
and `ultra` for Astra, with a native default of `medium`; this project chooses
`high`. The [Astra API model page](https://developers.openai.com/api/docs/models/gpt-6-astra)
lists the first five. Codex `ultra` adds automatic task delegation to maximum
reasoning; it is not an additional API reasoning level. Recheck the callable
schema when the harness changes.

Select Astra effort from the remaining judgment, not the role title alone:

| Effort | Use when |
| --- | --- |
| `low` | Mechanical retrieval or status readback with no new judgment; prefer a bounded helper when delegation is worthwhile. |
| `medium` | Coordination only applies closed ledger decisions or routes known results; no new decision, acceptance, or review verdict is needed. |
| `high` | Default for phase decisions, implementation ownership, acceptance, domain judgment, and independent review. |
| `xhigh` | Interacting cross-domain invariants, ambiguous recovery, weak proof, a material reviewer conflict, or a failed causal attempt requires deeper reasoning. |
| `max` | A concrete unresolved reasoning gap remains after `xhigh`, and additional depth justifies the time and usage within the accepted budget. |
| `ultra` | Maximum reasoning with automatic delegation, only when the installed harness preserves the accepted topology, role authority, and capacity. |

Raise effort before the affected decision or remaining repair. Missing facts,
authority, or a broken harness require recovery through their owner, not more
reasoning. Do not select `ultra` automatically for a critical role; use it only
when the installed harness semantics and accepted delegation topology justify
it. After a difficult unit closes, choose effort afresh for the next unit.

Project and subagent defaults use Astra so omitted model fields cannot assign
a decision-owning role to an execution model. Override those defaults through
native model and effort fields only for bounded execution or evidence work:
Luna at `low` for closed mechanical work, Terra at `medium` for ordinary
implementation and at `high` or `xhigh` for harder implementation, and Sol at
`high` for difficult code within an accepted contract. These models may reason
about implementation details and propose alternatives; Astra retains decision
and acceptance authority. Use `evidence-agent` for advisory research, not a
lower-model `specialist-agent` or `reviewer-agent` verdict.

Before delegating, Astra closes the expected result, accepted decisions,
writable boundary, focused proof, and conditions that return work to its owner
in the existing Subagent Brief. An executor that discovers a missing decision,
contract conflict, or inadequate proof returns evidence and a proposal to
Astra; it does not change semantics, weaken checks, add a fallback, or accept
the result. Astra resolves the gap through the smallest existing owner and
resumes the work. Routine local coding choices remain with the executor.
Astra may implement a coherent unit directly when delegation would cost more.

This Codex policy specializes Agent Harness's capability selection: raise
effort within Astra for decision-owning roles; model escalation applies only
to execution and evidence work. Resolve supported values from the callable
schema and verify the effective model before assigning decision authority.
If Astra is unavailable or its selection is rejected, retain the native
failure and stop the dependent decision or acceptance; never silently fall
back to Sol or Terra. Preserve an explicit user-selected model, but assigning
a non-Astra model decision authority requires an explicit exception to this
policy. A model name in prompt prose alone is not a native selection.

## Dispatch And Coordination

Pass the [delegation interface](../agent-harness.md#delegation-interface)
through installed structured fields where available. Retain every returned
`threadId`, `hostId`, task or operation identity, worktree identity, and wait
cursor. Never wait on a lane that returned no identity. Dispatch all independent
ready lanes before waiting, within current capacity; capacity is a ceiling, not
a fan-out target. Concurrent mutable units and lanes require disjoint packet
mutable owners and exclusive locks. Consume and integrate results serially
under the Lead. Land only `Accepted` candidates onto the shared checkout serially
from the
[Acceptance Result](../spec-first-workflow/interfaces/acceptance-result-v1.md)
and record the Lead-owned verdict without re-adjudicating it.

Send only the changed dependency, question, status, or identity: use a direct
message for an active agent and a follow-up to resume an idle one. Send sibling
dependencies to the affected sibling and inform the parent when they change a
shared assumption or acceptance state. Keep same-task corrections with the same
agent while its context remains useful; do not resend the full brief.

When authorized initial task creation cannot carry the selected model or effort, bootstrap
with role and scope only, require exact `READY_FOR_DISPATCH`, then send one
technical follow-up using the supported structured fields. Do not repeat an
ambiguously delivered dispatch. Use a fresh agent when a clean context or
changed strategy is more reliable.

For isolated work, validate the actual worktree and base before accepting its
bytes. For an `Accepted` isolated candidate, the Worktree Lead returns an
Acceptance Result with the fixed candidate and exact `HANDOFF_READY`.
The Orchestrator, or a root-local Lead, lands that
candidate serially and records the verdict without re-adjudicating it.
Handoff is routing evidence, not acceptance.

## Review And Recovery

An independent implementation review uses a fresh `reviewer-agent` with
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
as its Method. When Review requires integrated-candidate review, the
Orchestrator task binds one fresh `reviewer-agent` to that boundary and still
does not accept units. Raise its model/effort fields for a
justified highest-consequence boundary. Keep the fixed candidate unchanged.

Reconcile unknown create or handoff state from native task state, the canonical
ledger, and Git candidate identity. Zero or multiple exact matches remain an
unknown outcome; do not redispatch or repeat Handoff blindly. If implementation
invalidates an upstream decision, the Lead repairs the smallest owner when it
can, or the Orchestrator dispatches that phase to a fresh authorized carrier
selected through the Native Map. A phase actor may stop after its durable
handoff, but the Orchestrator keeps its identity, waits for that actor, re-reads
the transition, and resumes the same unit. It does not ask the user to confirm
technical routing within existing authority. Add no
scheduler, journal, or recovery database.

Archive a child only after its result and candidate are safe and no continuation
needs its identity.
