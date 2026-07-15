---
name: go-language-simplifier-review
description: "Use when changed Go is behaviorally correct and locally owned but control flow, predicates, naming, or helper shape is hard to understand or change; Own behavior-preserving cognitive simplification; Skip when Go semantics, package ownership, or explicit harsh whole-diff structure is the primary issue."
---

# Go Language Simplifier Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Target And Invariants

Review only behaviorally correct, locally owned Go for cognitive cost. Simpler means less reasoning, not fewer lines; prefer the smallest local change, and accept duplication when extraction would flatten policy, state transitions, side-effect order, ownership, cleanup, error identity, or caller-visible behavior.

- Prefer guard clauses and one abstraction level when they preserve audit/cleanup/rollback order and which error wins; flag nested success pyramids, sentinel state, temporal coupling, and delayed interpretation.
- Make decisions readable at call sites. Flag compound negatives, boolean clusters, raw modes, same-typed arguments, and `map[string]any` option decoding when they hide intent.
- Keep helpers that name stable policy, ownership, defaulting, cleanup, stdlib quirks, or error normalization. Flag pass-through wrappers, vague `util/common/shared` buckets, and mode-heavy extraction that only relocates complexity.
- Flag stable same-package policy repeated across files when one seam-named local owner would prevent drift; do not turn this into package-placement review.
- Preserve distinct failure classes, inspectability, status/cancellation mapping, and cleanup/audit precedence. A semantic change is not a simplification finding: route it to `go-idiomatic-review` or the owning behavior review.
- Use small Go-native patterns only when they shorten the reader's path; reject class-shaped ceremony locally, while explicit harsh whole-diff overbuild belongs to `go-structural-quality-review`.
- Raise naming and test-readability findings only when they materially affect safe change or diagnosis; green tests alone do not prove reasoning safety.
- Treat an unexplained executable/importable/generated/validated old path as reasoning debt after replacement unless accepted artifacts retain it with owner, reason, proof, and exit condition.

Do not recommend removing clone/copy isolation, observable nil/empty behavior, receiver or method-set protections, zero-value guarantees, must-not-copy state, cleanup/lifetime ordering, or stdlib wrapper contracts as ceremony.

Prefer official Go documentation, Go Code Review Comments, module/package guidance, and repository-local patterns over generic clean-code advice; use Effective Go as core-idiom guidance rather than current authority for modules, generics, or newer stdlib behavior.

## Symptom-Driven References

Use broad false-simplification triage only when no narrower selector owns the pressure.

| Pressure | Load |
| --- | --- |
| Broad cleanup, DRY, deduplication, or readability claim spans several local axes. | [false-simplification-patterns.md](references/false-simplification-patterns.md) |
| Helpers, wrappers, interfaces, callbacks, option bags, or helper buckets changed. | [helper-extraction-economics.md](references/helper-extraction-economics.md) |
| Stable same-package policy is repeated, drifting, or moved away from its local owner. | [source-of-truth-extraction.md](references/source-of-truth-extraction.md) |
| Branches, sentinels, named returns, defer, cleanup, rollback, audit, or phase order changed. | [control-flow-and-temporal-coupling.md](references/control-flow-and-temporal-coupling.md) |
| Predicates, negatives, flags, modes, same-typed args, or option decoding obscure a decision. | [predicate-condition-and-mode-clarity.md](references/predicate-condition-and-mode-clarity.md) |
| Error handling was deduplicated, normalized, mapped, logged, joined, or reordered. | [error-path-simplification.md](references/error-path-simplification.md) |
| Tables, helpers, assertions, fixtures, or terse failures obscure test proof intent. | [test-readability-and-proof-shape.md](references/test-readability-and-proof-shape.md) |
| Names or vocabulary obscure role, phase, ownership, or policy with merge risk. | [naming-and-intent-exposure.md](references/naming-and-intent-exposure.md) |
| Cleanup touches alias isolation, nil/empty, receivers, zero values, lifetime, cleanup, or stdlib contracts. | [go-semantic-stop-signs.md](references/go-semantic-stop-signs.md) |

## Findings And Stop

Each finding names the local clarity defect, branch-misread/change/maintenance risk, and whether policy is under-extracted, a vague helper is over-extracted, a local pattern adds complexity, or stale legacy remains. Name a loaded reference only when it shaped judgment. `critical` means critical behavior is too obscured for safe change; `high` means material hidden state, false simplification, API opacity, or an active replaced path.

Hand off package/dependency/source-of-truth ownership to `go-implementation-ownership-review`, specialist correctness to its review owner, and proof completeness to `go-test-review`. Stop and escalate to `go-implementation-ownership-spec`, `go-system-architecture-spec`, `go-api-contract-spec`, `go-chi-spec`, or the owning behavior spec when a safe simplification would define or change public, transport, design, domain, security, reliability, or data behavior.
