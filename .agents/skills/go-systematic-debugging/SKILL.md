---
name: go-systematic-debugging
description: "Root cause: Use when a Go bug, flaky test, build failure, hang, deadlock, timeout incident, or production regression needs causal diagnosis. Own the first broken invariant and, only when authorized, the smallest causal fix and replayed proof; Skip when feature implementation, unresolved policy design, broad review, or claim-only verification is primary."
---

# Go Systematic Debugging

Use [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) for authorized repair. Classify `diagnosis_only` or `fix_authorized`, preserve volatile evidence, reproduce with controls, and trace to the first broken invariant. Match the observed symptom to [the reference selector](references/index.md) and load one reference only when it can change the next discriminating experiment or repair owner. Test the highest-information hypothesis one at a time; continue until one causal chain is supported by reproduction or current incident evidence and survives the smallest available falsification, or a named blocker prevents the next experiment. In `fix_authorized`, repair only the earliest owner after causal proof and replay the signal. Route accepted feature work to `go-coder`; hand timeout, retry, degradation, or overload policy to `go-reliability`. Return evidence, rejected hypotheses, the supported root cause or explicit unproven state, results, and the next experiment or blocker. Missing causal proof keeps diagnosis open and cannot justify repair.
