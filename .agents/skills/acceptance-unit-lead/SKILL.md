---
name: acceptance-unit-lead
description: "Unit: Use when bound ACCEPTANCE_UNIT_LEAD. Own one implementation unit end to end; Skip other units."
metadata:
  invocation: role
  kind: carrier
disable-model-invocation: true
---

# Acceptance-Unit Lead

Own one fixed unit. Apply
[Implementation](../../../docs/spec-first-workflow/phases/implementation.md).
Do not schedule sibling acceptance units.

Implement directly or fan out internal execution lanes, then integrate, freeze
one candidate, and run its required review on the integrated unit. Workers and
lanes receive no independent acceptance review.

During orchestrated execution, return one [Acceptance Result
V1](../../../docs/spec-first-workflow/interfaces/acceptance-result-v1.md) and
do not write canonical ledger state. A root-local Lead may write its own fixed
unit result.

For a persisted ledger, load the current task packet and consumed outputs
without absorbing adjacent work. Reopen only the smallest invalid owner and
resume the same unit.
