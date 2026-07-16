---
name: go-systematic-debugging
description: "Use when a Go bug, flaky test, build failure, hang, deadlock, timeout incident, or production regression needs a supported root cause; Own evidence-driven diagnosis and, only when authorized, the smallest causal fix and proof; Skip when the task is feature implementation, unresolved policy design, broad review, or claim-only verification."
---

# Go Systematic Debugging

Use [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) for authorized repair. Classify `diagnosis_only` or `fix_authorized`, preserve volatile evidence, reproduce with controls, and trace to the first broken invariant. Load [the reference selector](references/index.md) only when its symptom changes the discriminating experiment. Test one hypothesis; repair only the earliest owner and replay the signal. Route accepted feature work to `go-coder`; hand timeout, retry, degradation, or overload policy to `go-reliability`. Return evidence, rejected hypotheses, root cause, results, and next experiment; stop when causal proof is missing.
