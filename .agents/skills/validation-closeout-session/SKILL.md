---
name: validation-closeout-session
description: "Match fresh evidence to the implementation claim and report complete, partial, or blocked without overstating proof."
---

# Validation Closeout Session

Use inside implementation or for an explicitly requested validation pass.
[Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md)
owns Worker acceptance and correction, final integrated review, and phase
closeout; this skill owns claim-to-evidence matching and the resulting status.

Bind each claim to changed files, accepted task/spec, and the smallest fresh proof that covers it. Include relevant tests, build/lint/race/integration checks, generated or migration drift checks, smoke proof, and targeted negative searches.

Skipped, cached, failing, unavailable, Worker output alone, or too-narrow
evidence cannot prove the full claim. Record the narrower proven result and
remaining owner.

Return `complete` only when the phase owner's acceptance, integrated-review,
completion, and terminal-proof conditions all pass. Otherwise return `partial`
or `blocked` with exact evidence, the narrow proven boundary, and the reopen
target.

Do not create a next-session prompt for implementation-internal validation or
closeout. Stop at evidence only when the user explicitly requested a standalone
validation boundary; an independent review of completed implementation remains
a separate read-only request governed by the phase owner.
