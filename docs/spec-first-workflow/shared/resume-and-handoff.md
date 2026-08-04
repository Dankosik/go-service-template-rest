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

Treat handoff as a chain of custody. Emit a prompt only when the current actor
or session cannot continue; otherwise return the phase result. Use every field
in this fixed interface; write `none` only for an absent open proof or risk:

```text
Macro phase: <next owning macro phase>
Outcome: <accepted result the receiver continues from>
Accepted sources: <minimal authoritative paths or external sources>
Closed decisions and authority: <non-obvious constraints and ownership boundary>
Movement evidence: <why the prior phase may move>
Open proof or risk: <obligation, owner, and checkpoint; or none>
Next executable action: <one concrete first action>
Stop/reopen: <exact condition and owner>
```

A risk-triggered research-synthesis challenge and triggered specification,
technical-design, test-design, and task-readiness reviews are internal
checkpoints. The root launches the applicable read-only lane and continues the
review, repair, and focused re-review loop in the same session when possible.
They do not create a macro-phase handoff.

An explicitly requested standalone review remains read-only: return its complete
result and stop at that review boundary. It gains no repair, implementation, or
workflow-handoff authority unless the user separately grants it.

An explicitly requested review of completed implementation may run inside
implementation when the request makes it an acceptance condition; otherwise it
begins after Implementation / Validation / Closeout and never retroactively
becomes an acceptance or closeout gate.

## Stop Rule

The receiver can continue from the named sources without reconstructing chat,
and the handoff preserves the next owner, authority boundary, proof obligation,
next executable action, and exact stop or reopen condition. When the current
session can safely finish, continue instead of creating a handoff.
