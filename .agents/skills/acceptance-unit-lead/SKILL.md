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
[Implementation](../../../docs/spec-first-workflow/phases/implementation.md),
integrate any bounded delegation, run its proof and required review, then write
one canonical accepted result or precise blocker. For a persisted ledger, load
the current task packet and consumed outputs without absorbing adjacent work.
Reopen only the smallest invalid owner and resume the same unit. Start no other
unit.
