---
name: go-systematic-debugging
description: "First broken invariant. Use when a bug, flaky test, build failure, hang, deadlock, timeout, or regression has an unknown cause and needs diagnosis or authorized root-cause repair."
metadata:
  invocation: model
  kind: method
---

# Go Systematic Debugging

Debugging ends at the **first broken invariant**, not at the first plausible
explanation. Classify the request as `diagnosis_only` or `fix_authorized`.

The active workflow owns diagnostic execution. Inside ledger implementation,
stay within [Implementation](../../../docs/spec-first-workflow/phases/implementation.md#feedback-during-coding)'s
coding-feedback boundary; a blocker alone does not authorize runtime experiments.
During final validation, keep causal repair and its focused rerun together.

## Reliable Signal

Establish the smallest reliable signal available for the user's exact symptom.
Inspect code and form provisional hypotheses as needed to obtain or refine it.
For a local symptom, prefer an agent-runnable reproducer; record what was
actually run, its result, and any reproduction limit. For a live incident,
capture volatile evidence before restart, signal, rollout, or time can destroy it, and state which
side of every boundary the evidence covers.

Keep untested explanations provisional and name missing evidence before making
a causal claim.

## Causal Loop

Minimize further when it improves causal discrimination or materially reduces
rerun cost. Choose an experiment that distinguishes plausible causes; isolate
the variable or probe whose effect it measures. Track competing hypotheses
when several remain or a handoff needs their history, with the evidence that
supports, rejects, or blocks each.

Walk backward through affected boundaries until the earliest invariant whose
inputs were valid but whose output was wrong. A downstream victim is not the
root cause. Use [Integration Boundaries](../../../docs/architecture/integration.md)
when the far-side contract or runtime locator changes the experiment, and load
one matching [debugging reference](references/index.md) only when its stated
pressure does.

In `fix_authorized`, repair the earliest shared owner and confirm the fix against
the original symptom and focused regression proof at the correct seam. Reuse a
sufficient existing test; if confirmation cannot run, report that exact gap.
In `diagnosis_only`, make no repository edit.

Complete when one causal chain survives the signal and its smallest available
falsifier, or when the exact next experiment and missing capability are named.
Remove temporary instrumentation before completion. Use [production
diagnosis](../../../docs/universal-disciplines/production-diagnosis/SKILL.md)
for a running multi-service symptom.
