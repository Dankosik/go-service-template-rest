# State Machine And Transition Rules

## Decide
- Model a lifecycle only where allowed behavior differs by state. A command whose legality never depends on current state belongs in the invariant register instead.
- Per transition: trigger, guard, allowed next state, **forbidden next states**, and violation outcome. The forbidden set is the part a reader cannot reconstruct from the allowed set, because an unlisted move reads as unspecified rather than rejected.
- Name terminal states and say what may reopen them — replay, reconciliation, support action, or nothing. A terminal state with no stated reopen policy is the one an admin path quietly reopens later.
- Say what a repeat, a duplicate event, a timeout, and a stuck state do, where those can change the outcome. `idempotency-replay-and-async-domain-rules.md` owns the sameness rule they depend on.
- Keep domain states separate from implementation status. A state earns its place by changing what is legal next, not by naming a queue, table, or handler step.
- A transition guard that a concurrent writer can lose is enforced where the write is serialized, not in the caller — see the enforcement-point criterion in `invariant-register-patterns.md`.

## Reject
```text
States: creating, saving, logging, returning
```
Failure: these are implementation steps. No state grants a different permission or outcome, so the model has no rejecting power.

```text
Retry later.
```
Failure: an unnamed state hiding an ambiguous external outcome. Name pending, ambiguous-commit, reconciliation, manual intervention, or reject.

## Prove
Proof should cover one allowed transition, one forbidden transition rejected, the terminal or reopen case when one exists, and the duplicate or stuck-state case when replay can reach it.
