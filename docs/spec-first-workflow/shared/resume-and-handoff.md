# Resume And Macro-Phase Handoff

Resume from durable authority and hand off only across a real actor or
macro-phase boundary.

## Read When

- Resuming after compaction, interruption, or a different session.
- The current actor or session cannot continue and another owner must act.
- An implementation session stops at an accepted unit boundary with later
  implementation work remaining.
- A ready macro phase has reached its user-started boundary.

## Resume

Apply the [Artifact Model Resume Order](artifact-model.md#resume-order). If the
authoritative sources disagree, reopen the narrowest owner instead of merging
the conflict from chat.

When completed coordination is larger than the live decision state, refresh
the ledger's compact `Active wave` block and continue from that artifact in a
fresh root context when the harness supports it. Carry no transcript replay;
retain only accepted inputs, unit and candidate identities, proof receipts,
open causal class, and next action.

## Implementation Entry And Continuation Handoff

When the current actor or session hands a ready unit to a different session that
will enter or continue Implementation — after Planning movement, a persisted
acceptance-unit transition, or an explicitly requested different-session resume
— return a standalone `Next Session Prompt` without waiting for the user to ask.
Inspect the authoritative ledger and current checkout to select only the next
ready acceptance unit and determine whether its current paths and fixed
dependencies satisfy [Shared-Checkout Implementation Lane
eligibility](../phases/implementation-worker-execution.md#shared-checkout-implementation-lanes).
Record the accepted ledger revision or receipt and relevant pre-existing dirty
paths plus their owner as the handoff basis. Do not rewrite `tasks.md` merely to
record an execution-carrier choice.

When eligible, emit a copy-pastable prompt in this shape:

```text
Implement <acceptance unit> only.

Execution decision:
- Carrier: <count> parallel shared-checkout implementation lanes.
- Handoff basis: <accepted ledger revision or receipt; relevant pre-existing dirty paths and owner, or none>.
- Independence basis: <fixed contract and why the write/proof slices are independent>.
- Root-reserved surfaces: <exact shared, integration, formatting, aggregate-proof, review, and receipt surfaces>.
- Excluded work: <dependent units and the receipt that keeps them blocked>.

Preflight: inspect the current ledger revision, owned paths, fixed dependencies, and relevant dirt. If current evidence still supports this exact lane map, dispatch every lane immediately. Otherwise recompute the serial or shared-lane carrier before editing.

Global constraints: preserve all existing and unrelated dirt. Workers may read repository sources but write only their owned paths. They must not edit the task ledger, commit, rebase, stash, deploy, mutate Git state, run broad or Docker gates, or cross another writer's ownership.

Lane <name>:
- Outcome: <lane-specific postcondition>.
- Writes only: <exact paths>.
- Focused proof: <commands and expected observable>.
- Stop: <cross-owner file, shared resource, unrecorded choice, or scope expansion>.

Repeat that block for each lane. Use one active writer per file. If a stop condition occurs, the lane returns the issue without crossing ownership.

Worker return: `DONE` or `BLOCKED`; changed paths; focused commands and results; unresolved issue; `provisional edits: present|none`. A blocked lane leaves partial edits in its owned paths; neither root nor a lane resets, checks out, or stashes them.

Root execution: while lanes run, edit only root-reserved paths. Wait for every lane before fan-in. Return a lane-local correction to the same lane. After all writers stop, verify ownership and scope, preserve and disposition provisional edits, reconcile and format the combined change, run focused checks, then execute <exact broad or Docker proof> serially. Complete review and the <unit> receipt. Do not start or accept <dependent later units> in this session.
```

When the next unit is coupled or has only one useful write slice, emit the
ordinary serial continuation prompt instead and state the concrete coupling;
do not manufacture empty lanes or parallelize dependent units.

## Macro-Phase Handoff

Treat handoff as a chain of custody. At a ready macro-phase boundary, return the
phase result and a short standalone `Next Session Prompt`, then stop. Emit that
prompt only when the current macro phase is complete, every triggered review has
a movement-allowing disposition, and a different macro phase is the next owner.
When that next owner is Implementation, use the Implementation Entry And
Continuation Handoff above instead of the generic prompt below.
An incomplete or blocked phase, same-phase reopen, or context rollover preserves
or reports resume state without a user-visible prompt unless the user explicitly
asks for one or the Implementation Entry And Continuation Handoff above applies.
The result tells the user
what was completed, the decisions and authority that now hold, the movement
evidence, and any open proof or risk. The prompt carries only the facts and
evidence-backed direction needed to start the target macro phase from durable
sources without replaying chat. Give the receiver the strongest current
continuation hypothesis, why it deserves attention, and the constraint or
evidence that could overturn it. This steers the next phase without turning a
candidate direction into accepted authority: the receiver tests it first, then
keeps, revises, or rejects it from current evidence and records why. When the
completed phase supports no candidate solution, carry its most discriminating
next decision question and evidence basis instead of inventing one.

```text
Continue with <target macro phase> only. <Prior macro phase> is ready: <movement evidence>. Start from <minimal authoritative paths or external sources>. Use <current candidate direction> as the leading hypothesis because <decision-relevant evidence>; test it first against <named constraints or falsifier>, and keep, revise, or reject it from current evidence rather than treating it as settled. First, <one concrete action>. Preserve <critical authority boundary>; stop or reopen at <exact condition and owner>. Do not enter the following macro phase in this session.
```

A risk-triggered research-synthesis challenge and triggered specification,
technical-design, test-design, and task-readiness reviews are internal
checkpoints. The root launches the applicable read-only lane and continues the
review, repair, and focused re-review loop in the same session when possible.
They do not create a macro-phase handoff. When current-phase evidence invalidates
a narrow upstream artifact, the root suspends the active phase, closes that
upstream repair and its triggered fresh review, and resumes the active phase in
the same session.

An explicitly requested standalone review remains read-only: return its complete
result and stop at that review boundary. It gains no repair, implementation, or
workflow-handoff authority unless the user separately grants it.

An explicitly requested review of completed implementation may run inside
implementation when the request makes it an acceptance condition; otherwise it
begins after Implementation / Validation / Closeout and never retroactively
becomes an acceptance or closeout gate.

## Stop Rule

The receiver can continue from the named sources without reconstructing chat,
and the phase result plus prompt preserve the target owner, authority boundary,
proof obligation, evidence-backed leading hypothesis or decision question and
its falsifier, next executable action, and exact stop or reopen condition. The
prompt neither discards the prior phase's decision implications nor presents a
candidate direction as closed. The current session does not begin the target
macro phase.
