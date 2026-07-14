---
name: validation-closeout-session
description: "Match fresh evidence to the implementation claim and report complete, partial, or blocked without overstating proof."
---

# Validation Closeout Session

Use inside implementation or for an explicitly requested validation pass. Follow [implementation/validation/closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md).

Bind each claim to changed files, accepted task/spec, and the smallest fresh proof that covers it. Include relevant tests, build/lint/race/integration checks, generated or migration drift checks, smoke proof, and targeted negative searches.

Skipped, cached, failing, unavailable, worker output alone, or too-narrow evidence cannot prove the full claim. Record the narrower proven result and remaining owner.

For ledger work, inspect each worker's integrated task diff and proof before updating progress. Accept the task and start the next worker only when its criteria and proof pass; otherwise return concrete gaps to the same worker without advancing. Small direct work returns complete after root diff inspection and bounded proof. For every path, return complete only when the accepted completion condition and terminal proof pass, plus any concretely triggered independent review has reached convergence; otherwise report partial or blocked with exact evidence and reopen target.

Validation, implementation-owned repair, re-review, and closeout remain internal to the active implementation request and do not create a next-session prompt. Stop at findings or evidence only when the user explicitly requested a standalone review or validation boundary.
