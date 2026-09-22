# Codex Harness Adapter

Use the installed Codex schemas and tool instructions as native authority;
field availability alone does not authorize its use.

## Native Map

- The current root task using `$orchestrator` owns the persisted ledger.
  Use collaboration subagents for all internal execution and review, including
  mutable work and cross-repository work. Do not create separate Codex App chats
  or use App task handoff as an execution carrier or fallback.
- For cross-phase work selected by [Transition](../spec-first-workflow/shared/transition.md#cross-phase-continuation),
  map phase actors to fresh collaboration subagents and the continuation owner
  to the existing root.
- Subagents start in the parent's working context. Before working in another
  checkout, apply [Repository
  Boundaries](../spec-first-workflow/shared/repository-boundaries.md) and use the
  target checkout for commands and code navigation. A path in the brief or a
  command's working directory does not reload that project's native
  configuration, roles, or tools.
- For each ready ledger unit, spawn a general-purpose subagent with
  Implementation as its Method and that packet as its boundary. It is the
  Acceptance-Unit Lead; execution-only `worker-agent` and read-only roles cannot
  replace its acceptance authority. A root-local Lead remains available for
  Implementation's single-unit path.
- When [Agent Harness](../agent-harness.md) selects isolation, create or select
  a Git worktree with repository-native lifecycle tools and assign its absolute
  root to the subagent. A Git worktree does not require an App chat. Preserve
  disjoint writable owners and locks for shared checkouts. If required tools,
  configuration, or write access cannot be provided to the subagent, report
  that exact capability gap; do not bypass it with another control plane.
- Phase actors and Leads may call collaboration tools directly for descendants;
  the root need not relay child calls.
- Use a fresh project subagent with no inherited turns for independent review.
- A Goal is optional and thread-local. Create one only after an explicit user
  request or system/developer instruction; task duration is not authorization.

## Nested Execution

The portable runtime explicitly enables agents and Multi-Agent V2 in
`.agents/codex-project.toml`; `scripts/codex-agents-sync.sh` generates the project
config. V2 ignores the legacy V1 `agents.max_depth`, so do not add a workflow
depth cap there. The current ceiling permits 20 open subagents across the
session tree, excluding the root; it limits concurrency, not desired depth.

Apply shared [Nested Execution](../agent-harness.md#nested-execution) and
[Context And Lifetime](../agent-harness.md#context-and-lifetime). Codex permits
worker descendants within the session ceiling; execution/evidence lanes use
explicit `fork_turns: "none"`. At capacity, finish existing work or execute the
bounded work locally within current role authority; do not create App chats.

## Models

Before Implementation, preserve the user's native model and effort selection;
this repository does not choose a model for those phases. The project runtime
does not pin the root or subagent model or effort. Verify the effective native
selection when assigning a decision, and preserve an explicit user selection.
Pass that selection through native model and effort fields for fresh phase
actors when those fields are available.

During Implementation, unless the user selects another model for that task,
use `gpt-6-sol` for the root delivery owner, Ledger Orchestrator,
Acceptance-Unit Lead, domain specialist, independent reviewer,
and adjudicator. Sol owns task decisions, assembled validation, review verdicts,
and acceptance under their existing role boundaries. Use `gpt-6-luna` at
`medium` for bounded code-writing `worker-agent` lanes and at `low` for bounded
mechanical evidence work. Luna can implement and repair within an accepted
contract; return unresolved decisions and material review questions to Sol.
Sol may implement directly when delegation costs more than it saves.

Use Sol at `medium` for closed coordination and at `high` for task ownership,
acceptance, specialist judgment, and independent review. Raise it to `xhigh`
for interacting invariants, ambiguous recovery, weak proof, or a material
reviewer conflict. Use `max` only when a concrete reasoning gap remains after
`xhigh`. The [Sol](https://developers.openai.com/api/docs/models/gpt-6-sol)
and [Luna](https://developers.openai.com/api/docs/models/gpt-6-luna) model pages
list these effort levels. Recheck the callable schema when the harness changes.

Select the Implementation model and effort through native controls before the
dependent work. The root model remains controlled by the user's picker;
subagent briefs carry their own model and effort fields. If the root task has a
different effective model, assign implementation ownership to a Sol Lead
through the native subagent controls rather than claiming root-local Sol
acceptance. If a required selection is unavailable or rejected, retain the
native failure and stop only the dependent decision or acceptance. A model
name in prompt prose alone is not a native selection. For executor briefs,
apply [Agent Harness](../agent-harness.md#context-and-lifetime);
Implementation owns handoff, repair, and final-validation timing.

## Dispatch And Coordination

Choose model and effort for each brief under Models and pass the full brief
with the selected settings on the initial call when native authority permits:

- `collaboration.spawn_agent` accepts `model` and `reasoning_effort` with
  `fork_turns: "none"` or a bounded turn count. Full-history forks (`"all"`, also
  the default) inherit the parent's model and effort and reject overrides.
  Independent review, execution lanes, and evidence lanes use `"none"`.

Do not add a default-model bootstrap turn or use a follow-up to bypass initial
selection restrictions. Recheck this mapping when the installed tools change.

Pass the [delegation interface](../agent-harness.md#delegation-interface)
through installed structured fields where available. Retain each returned agent
identity/task path and assigned checkout. Use `collaboration.list_agents` to
inspect the tree, messages and follow-ups to steer it, and
`collaboration.wait_agent` for event-driven waiting. Never wait on a lane that
returned no identity. Use
[Implementation](../spec-first-workflow/phases/implementation.md) for ready-lane
dispatch, serial integration, and the assembled acceptance boundary;
[Agent Harness](../agent-harness.md#delegation-interface) owns capacity,
progress intervention, and writable-scope isolation.

Use
`collaboration.send_message` to steer an active agent; it does not start a turn.
For idle or completed work, use `collaboration.followup_task` within the permitted
reuse boundary. Before waiting for the requested result, establish that dispatch
started an active turn or already returned that result from native status or
events. Message delivery and a retained agent id do not establish execution.
If dispatch is uncertain, reconcile once before resuming or replacing the lane;
do not wait on an idle lane or resend blindly. Apply shared Context And Lifetime
before reusing an identity; send only the delta for permitted corrections or
evidence follow-ups.
For a sequential Lead reassignment admitted by the Planning Ledger Contract,
use a follow-up with the new packet and current candidate/input locators.
Re-evaluate model and effort for the new unit under Models; reuse the native
agent only while its settings and context remain suitable.

Subagent `send_message` and `followup_task` do not change model or effort. For a
required escalation, start a fresh agent with the selected settings and the
remaining brief, accepted state, candidate, and prior evidence. Before replacing
mutable work, stop the old lane and reconcile its edits so writable ownership
does not overlap. Do not repeat an ambiguously delivered dispatch. Use a fresh
agent when a clean context or changed strategy is more reliable.

For isolated work, validate the actual worktree and base before accepting its
bytes. For an `Implemented` isolated candidate, the Worktree Lead returns an
Acceptance Result with the fixed candidate and exact `HANDOFF_READY`.
The Orchestrator, or a root-local Lead, lands that candidate serially and
records the verdict without re-adjudicating it. Handoff is routing evidence,
not acceptance.

## Review And Recovery

When [Review](../spec-first-workflow/shared/review.md) selects an independent
review, map it to a fresh `reviewer-agent` with `fork_turns: "none"` and the
selected phase review method. Use native model/effort fields under Models.

Reconcile unknown dispatch or result state from the native subagent tree, the
canonical ledger, and Git candidate identity. Zero or multiple exact matches
remain an unknown outcome; do not redispatch blindly. For upstream reopen and
continuation, use
[Transition](../spec-first-workflow/shared/transition.md); keep returned agent
identities so native completion and the durable result can be reconciled.

Do not equate a parent interrupt with subtree termination. Confirm descendant
state through native controls before cleanup or writable-scope reassignment.
After a session interruption, resume from the ledger and verified checkout;
reuse an agent only while its native identity remains available, otherwise
spawn a replacement with the remaining brief and evidence.
