# Resume And Macro-Phase Handoff

Resume from durable authority and hand off only across a real actor or
macro-phase boundary.

## Read When

- Resuming after compaction, interruption, or a different session.
- The current actor or session cannot continue and another owner must act.
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

## Macro-Phase Handoff

Treat handoff as a chain of custody. At a ready macro-phase boundary, return the
phase result and a short standalone `Next Session Prompt`, then stop. Emit that
prompt only when the current macro phase is complete, every triggered review has
a movement-allowing disposition, and a different macro phase is the next owner.
An incomplete or blocked phase, same-phase reopen, or context rollover preserves
or reports resume state without a user-visible prompt unless the user explicitly
asks for one. The result tells the user
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
