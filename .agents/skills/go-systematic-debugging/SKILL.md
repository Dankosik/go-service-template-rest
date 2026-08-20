---
name: go-systematic-debugging
description: "Root cause: Use for bugs, flaky tests, builds, hangs, deadlocks, timeouts, or regressions. Own diagnosis and authorized repair; Skip features."
---

# Go Systematic Debugging

Debugging ends at the **first broken invariant**, not the first plausible story.

`reproduce or capture -> boundary map -> hypotheses -> discriminating experiment -> first broken invariant -> authorized repair -> replay`

Classify `diagnosis_only` or `fix_authorized`. For a local symptom, establish one
agent-runnable command that fails on the reported behavior before ranking causes;
make it deterministic or record its failure rate. For a live incident, capture
volatile evidence before reproduction can stale it.

Map every service, client, job, and managed dependency on the failing path and
mark which side of each hop the evidence covers. Use the System Neighbors table
in [Repository Architecture](../../../docs/repo-architecture.md) for the far
side's contract and runtime locator. One-sided evidence leaves an open
hypothesis. Load one matching [debugging reference](references/index.md) only
when it changes the next experiment.

Test the highest-information hypothesis and record its observation as rejected,
supported, or blocked. Continue until one causal chain survives reproduction or
current incident evidence and the smallest available falsifier. In
`fix_authorized`, apply [Implementation](../../../docs/spec-first-workflow/phases/implementation.md),
only when its structured or conditional boundary applies; direct repair follows
the root [Direct Work](../../../AGENTS.md#direct-work) contract. Repair the
earliest shared owner and replay the signal. Otherwise return the unproven state
and next experiment.

Use [production diagnosis](../../../docs/universal-disciplines/production-diagnosis/SKILL.md)
for a running multi-service symptom. Hand policy, semantics, deterministic
waiting, performance, and completion proof to their matching skills.
