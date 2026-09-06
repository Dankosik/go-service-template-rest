# Codex Harness Adapter

Use the installed Codex schemas and tool instructions as native authority;
field availability alone does not authorize its use.

## Native Map

- The current root task using `$orchestrator` owns the persisted ledger.
  Use collaboration subagents for all internal execution and review, including
  mutable work and cross-repository work. Do not create separate Codex App chats
  or use App task handoff as an execution carrier or fallback.
- Before Planning, the existing root can coordinate a full-outcome request
  through [Transition](../spec-first-workflow/shared/transition.md#cross-phase-continuation).
  Use fresh collaboration subagents for phase actors. At ready Planning, bind
  the current root as Ledger Orchestrator or dispatch a fixed-unit Lead as
  Implementation requires; preserve returned identities across that transition.
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
- Phase actors and Leads may spawn their own specialists and fresh reviewers.
  Preserve macro-phase focus and review independence. The parent remains
  responsible for consuming results; the root need not relay every child call.
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

Use `gpt-6-astra` for every decision-owning role: the root coordinator,
Ledger Orchestrator, phase owner, Acceptance-Unit Lead, domain specialist,
independent reviewer, and adjudicator. Model capability does not merge
role responsibilities: the Orchestrator routes, the delivery Lead accepts,
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

Project and subagent defaults use Astra; inheritance and explicit overrides
still require effective-model verification. Override those defaults through
native model and effort fields only for bounded execution or evidence work:
`gpt-5.6-luna` at `low` for closed mechanical work, `gpt-5.6-terra` at `medium`
for ordinary implementation and at `high` or `xhigh` for harder implementation
within an accepted contract. These models may reason about implementation
details and propose alternatives; Astra retains decision and acceptance
authority. Use `evidence-agent` for advisory research, not a lower-model
`specialist-agent` or `reviewer-agent` verdict. Do not select `gpt-5.6-sol`
for any role, execution, evidence work, escalation, or fallback.

Before delegating, Astra supplies accepted behavior, the expected outcome,
writable scope, and genuine product/architecture reopen conditions in the
existing Subagent Brief. Routine coding choices, test cases, fixtures,
assertions, and test commands belong to the executor as it writes the task.
No prior test plan, named check, or passing proof is needed to delegate.
A contract conflict or missing product decision returns to Astra with the
available evidence and a proposal; the executor cannot change accepted behavior,
add an unaccepted fallback, or grant final acceptance. Missing test cases and
commands are implementation work, not an upstream gap.

Return Implemented and continue the ledger without checks or review. At final
validation after assembly, Astra assesses the combined code and evidence;
shared Review selects any final independent reviewer. Terra or Luna can repair
code and test failures then, with only invalidated checks rerun under the
delivery owner. Return unresolved judgment or stalled diagnosis to Astra.
A missed invariant raises Astra's effort under Models; no extra per-task gate
is created. Astra may implement directly when delegation would cost more.

This Codex policy specializes Agent Harness's capability selection: raise
effort within Astra for decision-owning roles; execution and evidence work may
escalate from Luna to Terra, with unresolved judgment returning to Astra.
Resolve supported values from the callable schema and verify the effective
model before assigning decision authority.
If Astra is unavailable or its selection is rejected, retain the native
failure and stop the dependent decision or acceptance; never silently fall
back to an execution model. Preserve an explicit user-selected model, but assigning
a non-Astra model decision authority requires an explicit exception to this
policy. A model name in prompt prose alone is not a native selection.

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
returned no identity. Dispatch all independent
ready lanes before waiting, within current capacity; capacity is a ceiling, not
a fan-out target. Concurrent mutable units and lanes require disjoint packet
mutable owners and exclusive locks. Consume and integrate results serially
under the Lead. Integrate `Implemented` candidates serially into the local
development tree from the
[Acceptance Result](../spec-first-workflow/interfaces/acceptance-result-v1.md),
then immediately unlock dependent implementation. Do not run task checks or
reviews. Assign one general-purpose Lead the final delivery boundary after
assembly; it owns consolidated proof and acceptance. The Orchestrator records
that final result without repeating it.

Send only the changed dependency, question, status, or identity: use a direct
message for an active agent and a follow-up to resume an idle one. Send sibling
dependencies to the affected sibling and inform the parent when they change a
shared assumption or acceptance state. Apply shared Context And Lifetime before
reusing an identity; send only the delta for permitted corrections or evidence
follow-ups.
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
The Orchestrator, or a root-local Lead, lands that
candidate serially and records the verdict without re-adjudicating it.
Handoff is routing evidence, not acceptance.

## Review And Recovery

Only at final delivery, start a required independent implementation review
with a fresh `reviewer-agent` and
[Implementation Review](../spec-first-workflow/phases/implementation-review.md)
as its Method; shared Review owns same-reviewer rechecks. When Review requires
integrated-candidate review, the
Orchestrator task binds one fresh `reviewer-agent` to that boundary and still
does not accept units. Raise its model/effort fields for a
justified highest-consequence boundary. Keep the fixed candidate unchanged.

Reconcile unknown dispatch or result state from the native subagent tree, the
canonical ledger, and Git candidate identity. Zero or multiple exact matches
remain an unknown outcome; do not redispatch blindly. If implementation
invalidates an upstream decision, the Lead repairs the smallest owner when it
can, or the Orchestrator dispatches that phase to a fresh authorized carrier
selected through the Native Map. A phase actor may stop after its durable
handoff, but the Orchestrator keeps its identity, waits for that actor, re-reads
the transition, and resumes the same unit. It does not ask the user to confirm
technical routing within existing authority. Add no
scheduler, journal, or recovery database.

Do not equate a parent interrupt with subtree termination. Confirm descendant
state through native controls before cleanup or writable-scope reassignment.
After a session interruption, resume from the ledger and verified checkout;
reuse an agent only while its native identity remains available, otherwise
spawn a replacement with the remaining brief and evidence.
