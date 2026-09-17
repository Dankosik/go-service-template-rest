# Validation Repair

Use after a failed check or bounded behavioral feedback.
[Implementation](../implementation.md) owns execution timing and final acceptance.

Classify failure as product, test/oracle, or environment. For an optional check's
unavailable or broken environment, record its material gap and stop that path;
do not provision or repair infrastructure to finish ordinary development.
Required checks and concrete in-scope defects retain a repair owner, including
defects exposed by optional checks. Assign diagnosis, source or fixture repair,
execution-input preparation, and the smallest discriminating rerun to the same
executor, with the current candidate, diagnostics, runner, writable scope,
and existing resource authority.

The executor continues this loop without a coordinator handoff between edits,
input preparation, and reruns. Return the repaired result with its evidence,
or a concrete blocker requiring another owner's decision. Retire superseded
code-only repair briefs when entering this stage.

Preserve required candidate identities, validation locks, and review
independence. Final acceptance stays with the delivery owner; publication
and production effects retain their applicable authority and gates.

When the cause is still uncertain, select the next run for the explanations it
can distinguish. If failure occurred before the intended behavior, check the
nearest broken setup precondition first. After repairing it, return to the
original scenario; successful setup does not establish product behavior.

Repair the defect class at its shared source: compare a broken fixture with
the complete required shape, trace retained callers before deleting a helper,
and retained references before deleting durable data. Gather related failures
in that affected scope instead of repairing only the first reported line.
If another attempt yields the same failure without new discriminating evidence,
apply [Parent-Owned Recovery](../../shared/transition.md#parent-owned-recovery)
before rerunning; do not repeat the whole pipeline or increase timeouts blindly.

After an aggregate failure, continue its pending plan under the Evidence
Contract's scoped-reuse rules. A focused repair does not automatically schedule
the whole aggregate again; retain every not-yet-run or newly affected claim.

Use existing validation locks; do not hold them while editing or waiting.
Within established scope and heavy-run authority, lock availability schedules
retries without a fresh CPU permit or time-window negotiation. Coordinate anew
only when authority, resource scope, or budget changes. Never run heavy checks
concurrently or bypass effect authority.
