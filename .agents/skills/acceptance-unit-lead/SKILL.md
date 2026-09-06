---
name: acceptance-unit-lead
description: "Unit: Use when bound ACCEPTANCE_UNIT_LEAD for an implementation unit or final delivery validation. Own the assigned boundary; Skip sibling scheduling."
metadata:
  invocation: role
  kind: carrier
disable-model-invocation: true
---

# Acceptance-Unit Lead

Own one assigned implementation unit or the final delivery boundary. Apply
[Implementation](../../../docs/spec-first-workflow/phases/implementation.md).
Do not schedule sibling tasks.

Implement directly or use useful parallel subtask lanes, integrate their code,
join writers, and return Implemented immediately. Run no task-transition checks
or reviews. When separately assigned the final delivery boundary, consolidate
all required proof and any final review under Implementation.

During orchestrated execution, return one [Acceptance Result
V1](../../../docs/spec-first-workflow/interfaces/acceptance-result-v1.md) and
do not write canonical ledger state. A root-local Lead may write its own fixed
unit result.

For a persisted ledger, load the current task packet and consumed outputs
without absorbing adjacent work. Reopen only the smallest invalid owner and
resume the same unit.
