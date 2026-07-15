---
name: validation-closeout-session
description: "Match fresh evidence to the implementation claim and report complete, partial, or blocked without overstating proof."
---

# Validation Closeout Session

Use inside implementation or for an explicitly requested validation pass. Follow [implementation/validation/closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md).

Bind each claim to changed files, accepted task/spec, and the smallest fresh proof that covers it. Include relevant tests, build/lint/race/integration checks, generated or migration drift checks, smoke proof, and targeted negative searches.

Skipped, cached, failing, unavailable, worker output alone, or too-narrow evidence cannot prove the full claim. Record the narrower proven result and remaining owner.

Inspect each external Worker's integrated diff and proof before updating progress. Apply matching review skills and affected specialist lenses locally; do not launch a built-in subagent lane. Accept a direct outcome or task only when its criteria and proof pass; otherwise resume the same Worker session with concrete gaps and do not author the repair. Re-inspect its correction, and start a fresh Worker/session for the next ledger task only after acceptance. Return complete only when the root has reviewed the final integrated diff and the accepted completion condition and terminal proof pass; otherwise report partial or blocked with exact evidence and reopen target.

Validation, implementation-owned Worker repair, root re-inspection, and closeout remain internal to the active implementation request and do not create a next-session prompt. An explicitly requested independent review of completed implementation is a separate read-only boundary after the active implementation/validation/closeout macro phase. The same authorized root session may continue into that boundary, but the review never becomes an implementation acceptance or closeout gate. Stop at evidence only when the user explicitly requested a standalone validation boundary.
