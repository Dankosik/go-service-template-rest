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

## Signal Gate

For a local symptom, establish one agent-runnable command that has already been
run and catches the user's exact symptom. Make it deterministic and fast, or
record and raise its reproduction rate. For a live incident, capture volatile
evidence before restart, signal, rollout, or time can destroy it, and state which
side of every boundary the evidence covers.

No hypothesis work starts until this signal exists or the exact evidence
blocker is named.

## Causal Loop

Minimize the reproducer until every remaining input, dependency, step, and
timing condition is load-bearing. Build ranked falsifiable hypotheses; each
states the observation that would distinguish it. Change one variable or add
one targeted probe at a time, recording each hypothesis as rejected, supported,
or blocked.

Walk backward through affected boundaries until the earliest invariant whose
inputs were valid but whose output was wrong. A downstream victim is not the
root cause. Use [Integration Boundaries](../../../docs/architecture/integration.md)
when the far-side contract or runtime locator changes the experiment, and load
one matching [debugging reference](references/index.md) only when its stated
pressure does.

In `fix_authorized`, repair the earliest shared owner, turn the minimized
reproducer into deterministic regression proof at the correct seam, and replay
the original unminimized signal. In `diagnosis_only`, make no repository edit.

Complete when one causal chain survives the signal and its smallest available
falsifier, or when the exact next experiment and missing capability are named.
Remove temporary instrumentation before completion. Use [production
diagnosis](../../../docs/universal-disciplines/production-diagnosis/SKILL.md)
for a running multi-service symptom.
