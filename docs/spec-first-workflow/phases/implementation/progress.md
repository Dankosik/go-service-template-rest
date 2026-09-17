# Implementation Progress

Before a potentially long check or resource wait, choose a bounded observation
checkpoint and the result or progress signal expected by then. Use the prior
comparable duration, tool timeout, and current environment when available;
without a baseline, begin with an early checkpoint and adjust from observation.
Keep the command/native locator and checkpoint in existing execution state.
This is an executor-owned monitoring choice, not a new user deadline or permit.

Use native yielding or background execution so the owner can inspect the run
at that checkpoint. Inspect actual stage, logs, process/resource state, or lock
owner; a live process or repeated elapsed-time message alone is not progress.
Continue useful work with a new observation checkpoint when evidence supports
it. If the expected result is absent, diagnose the wait or stalled stage before
another long wait or rerun. Preserve safe cleanup and required proof; exceeding
a checkpoint neither grants acceptance nor makes cancelling an effect safe.

In the existing status, name implemented subresults, the next concrete result,
and any current delay: implementation, product repair, test repair, environment,
resource wait, usage limit, or external dependency. Distinguish implementation
from verified behavior. No extra report or ledger level is required.

Use an explicitly accepted task deadline or execution budget to expose likely
overruns and reconsider a stalled approach. Keep its value in the existing
task-local artifact; do not infer one when none was accepted. A target is not
measured speed or permission to omit accepted scope or final proof. Report
measured waiting intervals only when existing logs establish their start and end.
