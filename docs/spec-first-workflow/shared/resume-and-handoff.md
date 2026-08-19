# Resume And Macro-Phase Handoff

Resume from durable authority and hand off only across a real actor or
macro-phase boundary.

## Read When

- Resuming after compaction, interruption, or a different session.
- The current actor or session cannot continue and another owner must act.
- A ready macro phase has reached its user-started boundary.

Implementation entry, continuation, terminalization, and upstream-reopen return
use the conditional [Implementation Handoff](implementation-handoff.md) branch.

## Resume

Apply the [Artifact Model Resume Order](artifact-model.md#resume-order). If the
authoritative sources disagree, reopen the narrowest owner instead of merging
the conflict from chat.

When completed coordination is larger than the live decision state, refresh
from the canonical ledger, native task status, and Git candidate identities in
a fresh root context when the harness supports it. Carry no transcript replay
or duplicate task-lifecycle record.

## Open Decisions And Fog

An **open decision** is a precise question with a named effect, owner, and
blocked decision. Record it in `workflow-plan.md` only at a real handoff,
context rollover, or second-lane dispatch. A question blocking the next action
belongs in `status: blocked`; the unblocked listed questions form the current
**frontier**.

```markdown
- <decision-changing question> — owner: <agent, external owner, or user> — blocks: <decision, phase, task, or nothing> — route: <lane, phase owner, probe, or escalation>
```

**Fog** is a visible decision surface that cannot yet be phrased precisely. It
survives only with the evidence or open decision that will sharpen it:

```markdown
- <suspected area> — sharpens when: <decision or evidence>
```

Fog never appears in a readiness or completion claim and never defers a
decision triggered by the current phase. Resolve each entry into an open
decision or delete it when its sharpening condition changes. Work outside the
accepted outcome follows Planning's [scope-exit
record](../phases/planning.md#obligation-reconciliation).

## Macro-Phase Handoff

Treat handoff as chain of custody. Emit a short standalone `Next Session Prompt`
only when the current macro phase and every triggered review permit movement to
a different macro phase, then stop. Implementation entry uses [Implementation
Handoff](implementation-handoff.md#implementation-entry-and-continuation-handoff)
instead. An `UPSTREAM_REOPEN_LEAD` returns through that branch without a
user-visible prompt.

An incomplete or blocked phase, same-phase reopen, or context rollover reports
resume state without a prompt unless the user explicitly asks for one. The
phase result states what closed, current authority, movement evidence, and open
proof or risk. The prompt carries only the durable sources, strongest current
continuation hypothesis or decision question, its evidence and falsifier, first
action, and exact stop or reopen condition.

```text
Continue with <target macro phase> only. <Prior macro phase> is ready: <movement evidence>. Start from <minimal authoritative paths or external sources>. Use <current candidate direction> as the leading hypothesis because <decision-relevant evidence>; test it first against <named constraints or falsifier>, and keep, revise, or reject it from current evidence rather than treating it as settled. First, <one concrete action>. Preserve <critical authority boundary>; stop or reopen at <exact condition and owner>. Do not enter the following macro phase in this session.
```

Internal research, specification, technical-design, test-design, and readiness
reviews remain checkpoints inside their owning macro phase. The root completes
their review, repair, and focused re-review loop and likewise closes the
smallest upstream repair before resuming the active phase.

An explicitly requested standalone review remains read-only and ends at that
review boundary. A requested review of completed implementation is an
Implementation acceptance condition only when the request makes it one;
otherwise it begins after closeout and does not retroactively become a gate.

## Stop Rule

The target session can continue from named sources without chat reconstruction.
The result and prompt preserve owner, authority, proof, next action, leading
hypothesis or decision question and falsifier, and the exact stop or reopen
condition without presenting a candidate as closed. The current session does
not begin the target macro phase.
