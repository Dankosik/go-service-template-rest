# Transition

Use when a result may move or reopen a macro phase, or cross an actor/session
boundary.

Move only when the current owner is `ready`, every triggered decision has a
disposition, required review permits movement, and the next owner can act
without inventing meaning, mechanism, ownership, proof strategy, or authority.
An untriggered phase may be `skipped` without loading or reviewing it. Ordinary
inline movement needs no receipt; a durable macro-phase boundary returns
[Transition Result V1](../interfaces/transition-result-v1.md).

Carry only the current result, authoritative locators, movement evidence,
candidate identity when relevant, proof boundary, exact blocker/next action,
and reopen condition. Package a next-session prompt through [Prompt
Composition](../../prompt-composition.md). Same-phase repair or context rollover
reports resume state without pretending movement occurred.

For a persisted Implementation ledger, the [Planning Ledger
Contract](../phases/planning/ledger-contract.md#acceptance-transition) owns the
immediate `Accepted` or `Blocked` transition. A fixed inline unit creates no
synthetic ledger transition.

Reopen only the smallest owner invalidated by current evidence and preserve
unaffected decisions and proof. Stop at an explicit phase boundary, unavailable
required external input, new authority boundary, or required durable handoff.
That stop applies to the current phase actor; an explicit user stop still
bounds the whole request.

A mechanical locator or receipt refresh after an accepted repair does not
reopen unaffected decisions. Check the exact delta, update current identities,
and retain the earlier verdict only for its unchanged semantic scope. Changed
meaning, mechanism, oracle, risk, or dependency admission still requires its
owner and applicable review; a changed hash alone is not that decision.

## Cross-Phase Continuation

For an outcome spanning phases, the root coordinator retains the accepted
outcome and active phase identity. Dispatch one selected phase actor through
the [Agent Harness](../../agent-harness.md), wait for its reviewed Transition
Result, re-read the authoritative result, and open the next applicable owner.
A ready phase handoff continues the request without technical confirmation.
Persist coordination only under [Artifacts](artifacts.md); phase results remain
the decision authority.

The phase actor owns its decision, artifact, repair, and movement evidence. When
native nesting cannot support a needed specialist or independent reviewer, the
coordinator dispatches that bounded request on the actor's behalf and relays the
result. This request is not phase movement; only the phase actor returns its
reviewed Transition Result. The coordinator does not substitute its own review
or decision for the actor's.

At ready Planning, select [Implementation](../phases/implementation.md)'s carrier.
For a persisted ledger, the existing root may bind as `LEDGER_ORCHESTRATOR`
when its authorized native controls suffice; that role becomes the sole ledger
writer. For a fixed unit, dispatch its Acceptance-Unit Lead. Preserve the
separate phase, review, and acceptance owners without requiring a new
user-visible task. If no authorized carrier supports the required next action,
report the exact capability gap and retain the resumable result.

Keep one active continuation coordinator. If an authorized separate Orchestrator
takes over, transfer the current continuation state to it; the former coordinator
does not add a second routine polling or status-relay loop. Preserve a usable
result-delivery route and resume supervision for a terminal result, required
intervention, or explicit user request. Leads and reviewers retain their roles.

## Parent-Owned Recovery

A delegated blocker, missing proof, or technical-policy gap remains work for
the requesting parent, including a root-local Lead. Without a separate parent,
the current task owner retains that responsibility. Identify the dependent
action and smallest decision or evidence owner. Close the input directly when
the parent's role and active phase permit it; otherwise dispatch that owner
through an authorized carrier and consume its result. Keep required phase,
review, and acceptance boundaries intact.

When technical recommendations conflict, the responsible decision owner
compares evidence and constraints, obtains a discriminating probe or specialist
judgment when needed, and resolves each material objection against the accepted
drivers. Converge on an evidence-supported choice and reopen condition; do not
use a vote or unanimity as proof. When equivalent options survive, the decision
owner chooses the simplest one satisfying the constraints.

For uncertain recovery, name the next discriminating result and a bounded work
window in existing task state before another attempt. On that checkpoint or a
repeated result with no new evidence, compare actual progress with the intended
result. Continue while findings close or a concrete available probe can
distinguish remaining causes. Changing reviewer, owner, or reasoning effort
does not reset a no-progress recovery; reworded concerns, elapsed time, and
repeated status reports are not new evidence.

For the same unresolved finding without new evidence, use one focused
independent diagnosis when needed, not another whole-unit review. The
responsible decision owner resolves the finding against the accepted contract
and resulting evidence. Reuse that diagnosis until new evidence invalidates it;
another actor or higher effort alone is not a new recovery path. Confirmed
defects and missing mandatory proof still prevent acceptance.

When no available authorized probe or recovery can obtain the required evidence,
retain `Blocked` with the exact unverified claim, limitation, and input or
capability that would permit resumption. Do not redispatch the unchanged gap.
A work-window limit changes strategy; it neither waives proof nor creates a
new task boundary, user approval, or success claim. Retain useful partial work.

Continue independent authorized work, then resume the blocked unit from the
resolved input. A terminal child result does not complete the parent's outcome.
Escalate user-owned decisions or unavailable external inputs and authority only
through [Decision Ownership](../../../AGENTS.md#decision-ownership). If required
evidence or native capability remains unavailable after authorized recovery,
report the exact limitation and incomplete claim; do not invent success or ask
the user to choose a technical mechanism.
