---
name: go-systematic-debugging
description: "Root cause: Use for Go bugs, flaky tests, build failures, hangs, deadlocks, timeouts, or regressions. Own the first broken invariant and authorized proof/fix; Skip features, policy design, or broad review."
---

# Go Systematic Debugging

Debugging ends at the **first broken invariant**: the earliest point where reality diverges from the contract, reached through discriminating experiments — not at the first plausible story.

`reproduce or capture -> boundary map -> hypothesis set -> discriminating experiment -> first broken invariant -> authorized repair -> replayed signal`

Use [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md) for authorized repair, and classify `diagnosis_only` or `fix_authorized` first.

For a locally accessible symptom, establish and run one agent-runnable command that can fail on the exact reported behavior before ranking causal hypotheses; make it deterministic or record its failure rate, and minimize it until every retained input, caller, configuration, and step is load-bearing. Volatile live-incident evidence is the alternate entry point: capture it before local reproduction work can destroy or stale it.

Before ranking hypotheses, classify the symptom's boundary: name every service, client, job, and managed dependency on the failing path, and mark which side of each hop the current evidence comes from — a hop with evidence from only one side is an open hypothesis, not a cleared one. The far side is obtainable: the System Neighbors table in [`docs/repo-architecture.md`](../../../docs/repo-architecture.md) is this repository's owned inventory of what sits across each edge, and its columns name the neighbor's canonical contract source, its local checkout, and the query or command that reads its runtime evidence. A neighbor's client stub, a vendored copy, and this repository's expectations are not its contract. Reconstruct the affected invariant and hypothesis set from the symptom, volatile evidence, callers, state transitions, and sibling paths. Match the observed symptom to [the reference selector](references/index.md) and load one reference only when it can change the next discriminating experiment or repair owner. Test the highest-information hypothesis one at a time and disposition it as rejected, supported, or blocked, carrying the observation that produced that disposition — a disposition without its observation is a preference. Continue until one causal chain is supported by reproduction or current incident evidence and survives the smallest available falsification, or a named blocker prevents the next experiment. A chain terminating at a hop you did not inspect from both sides is unproven, not supported; call that hop a blocker only when its source and telemetry are genuinely unreachable.

In `fix_authorized`, the repair gate opens only after that diagnosis observable exists; repair only the earliest owner and replay the signal. Missing causal proof keeps diagnosis open and cannot justify repair. Return evidence, hypothesis dispositions, the supported root cause or explicit unproven state, results, and the next experiment or blocker.

Load [`production-diagnosis`](../../../docs/universal-disciplines/production-diagnosis/SKILL.md) when the symptom lives in a running multi-service system rather than in reproducible local code: it forces a quantified symptom contract and hop localization before any cause is entertained, separates the component that produces a degradation from the ones that merely inherit it, and owns recurrence guardrails once the cause is proven.

Hand off rather than absorbing: accepted feature work to `go-coder`, which also owns generated-artifact authority and the `*-generate` / `*-check` pair behind a build or drift failure; timeout, retry, degradation, and overload policy to `go-reliability`; the Go semantics a backtrace crosses — typed nil in an interface, `errors.Is`/`As`, context errors — to `go-idiomatic`; deterministic waiting and `testing/synctest` to `go-test-implementation`; which profile answers a measured delta to `docs/benchmarking.md` and `go-performance`; and matching a completion claim to proof of equal scope to `go-verification-before-completion`.
