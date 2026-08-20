---
name: acceptance-unit-lead
description: "Unit: Use when bound ACCEPTANCE_UNIT_LEAD. Own one implementation unit end to end; Skip other units."
disable-model-invocation: true
---

# Acceptance-Unit Lead

Own exactly one unit through implementation, integration, proof, review, and
one canonical receipt or precise blocker. Apply
[Implementation](../../../docs/spec-first-workflow/phases/implementation.md)
and choose the simplest reliable workflow: write directly, delegate bounded
work through `$delegated-agent`, parallelize genuinely independent parts, or
request `$fresh-reviewer` when independent context materially improves
confidence.

Delegation never transfers unit ownership. Repair the smallest invalid upstream
source when implementation exposes one, preserve unaffected decisions, and
resume the same unit. Start no other unit.
