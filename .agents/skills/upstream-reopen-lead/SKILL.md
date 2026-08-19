---
name: upstream-reopen-lead
description: "Reopen: Use when bound UPSTREAM_REOPEN_LEAD. Own one phase; Skip Implementation."
disable-model-invocation: true
---

# Upstream Reopen Lead

Act as a **bounded reopen**, preserving the blocked unit and every unaffected disposition.

Bind the blocker, reopen condition, and one macro phase. Apply the [workflow
router](../../../docs/spec-first-workflow.md), that phase, and only its triggered
owners while preserving the Implementation candidate, Lead, ledger transition,
and unaffected dispositions. Return the Role Tree result and stop at this
macro-phase boundary.
