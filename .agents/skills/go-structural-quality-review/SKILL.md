---
name: go-structural-quality-review
description: "Use when the user explicitly requests a harsh whole-diff review or a change appears structurally overbuilt across files; Own abstraction cost, spaghetti growth, mixed responsibilities, speculative flexibility, and missed deletion; Skip when ordinary Go semantics, local readability, or architecture conformance is the primary lane."
---

# Go Structural Quality Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Outcome

Find merge-relevant, behavior-preserving structural simplifications that delete concepts, branches, wrappers, modes, helper layers, file sprawl, or stale paths instead of polishing them.

Use only for an explicitly harsh whole-diff review or clear cross-file overbuild. Stay read-only; read the governing artifacts, complete diff, touched callers, owning packages, existing helpers, and affected tests, then rank concrete risks by merge impact. Generated files are exempt from size findings.

## Structural Defects

- more concepts, indirection, hidden state, or ownership spread for the same behavior;
- one-off branches, flags, nullable modes, callbacks, or option bags bolted into busy flows;
- pass-through wrappers, managers, factories, adapters, producer-owned interfaces, or speculative extension points;
- `common`, `util`, `shared`, bootstrap, transport, generated-adjacent, or infrastructure surfaces absorbing owner-specific policy;
- large hand-written files whose growth mixes responsibilities or abstraction levels; generated files are exempt from size findings;
- refactors that move or rename complexity without reducing the mental model;
- custom helpers already covered by current Go stdlib or a canonical repository owner;
- replaced code, tests, fixtures, configs, docs, skills, agents, generated outputs, or mirrors left active without owner, reason, proof, and exit condition.

Ask what concept can disappear and whether ownership becomes more cohesive. Prefer:

- deletion over rearrangement;
- logic in the package/file that already owns the concept;
- direct control flow when a helper hides only nearby facts;
- a seam-named same-package helper only when it owns stable policy, cleanup, or a sharp contract;
- responsibility-based file splits rather than arbitrary line-count splits;
- explicit policy flows over boolean or mode-driven helper APIs;
- removal or refactor of the old surface in the same accepted replacement.

## Findings And Stop

Each finding names a concrete cross-file structural defect at `file:line` and its future-change, review, regression, or operability risk. Do not emit taste-only nits or approve while a clear structural regression, owner leak, unjustified abstraction/file sprawl, or executable replaced path remains unexplained.

Hand off package/file ownership, dependency direction, and source-of-truth seams to `go-implementation-ownership-review`; local control-flow, predicate, naming, and helper clarity to `go-language-simplifier-review`; Go/stdlib contracts to `go-idiomatic-review`; and test or specialist correctness to the matching review. Escalate when correction changes accepted behavior, public contracts, data, security, reliability, concurrency, or architecture; use `go-implementation-ownership-spec` when responsibility or placement is unset, otherwise the owning behavior specification.
