---
name: go-structural-quality-review
description: "Run an unusually strict read-only Go maintainability review for structural simplification, abstraction cost, spaghetti growth, mixed-responsibility files, and missed deletion. Use when the user asks for thermonuclear/harsh review or a working diff appears overbuilt; skip ordinary idiom checks, local readability cleanup, and architecture conformance as the primary lane."
---

# Go Structural Quality Review

## Outcome

Find merge-relevant, behavior-preserving structural simplifications that delete concepts, branches, wrappers, modes, helper layers, file sprawl, or stale paths instead of polishing them.

## Review Method

1. Stay read-only. Read the diff, governing task artifacts, touched callers, owning packages, existing helpers, and directly affected tests.
2. Ask what concept can disappear, whether ownership became more cohesive, and whether a direct stdlib or repository path replaces custom machinery.
3. Rank only concrete structural risks with tight `file:line` evidence and the smallest safe correction.
4. Escalate corrections that change approved behavior, public contracts, data, security, reliability, concurrency, or architecture.

## Flag

- more concepts, indirection, hidden state, or ownership spread for the same behavior;
- one-off branches, flags, nullable modes, callbacks, or option bags bolted into busy flows;
- pass-through wrappers, managers, factories, adapters, producer-owned interfaces, or speculative extension points;
- `common`, `util`, `shared`, bootstrap, transport, generated-adjacent, or infrastructure surfaces absorbing owner-specific policy;
- large hand-written files whose growth mixes responsibilities or abstraction levels; generated files are exempt from size findings;
- refactors that move or rename complexity without reducing the mental model;
- custom helpers already covered by current Go stdlib or a canonical repository owner;
- replaced code, tests, fixtures, configs, docs, skills, agents, generated outputs, or mirrors left active without owner, reason, proof, and exit condition.

## Prefer

- deletion over rearrangement;
- logic in the package/file that already owns the concept;
- direct control flow when a helper hides only nearby facts;
- a seam-named same-package helper only when it owns stable policy, cleanup, or a sharp contract;
- responsibility-based file splits rather than arbitrary line-count splits;
- explicit policy flows over boolean or mode-driven helper APIs;
- removal or refactor of the old surface in the same accepted replacement.

## Boundaries And Handoffs

- Hand off architecture ownership, dependency direction, and source-of-truth drift to `go-design-review`.
- Hand off local cognitive complexity, predicates, naming, and helper economics to `go-language-simplifier-review`.
- Hand off Go semantics, error/context/nil/receiver/resource contracts, and stdlib correctness to `go-idiomatic-review`.
- Hand off test proof and specialist security, reliability, DB/cache, concurrency, performance, observability, routing, distributed, or domain concerns to their matching review skills.

## Output And Approval Bar

Use the [shared review finding envelope](../../../docs/subagent-contract.md#shared-review-finding-envelope). Each finding names the structural defect, future-change/review/regression/operability risk, smallest behavior-preserving correction, and handoff if required. Do not approve while a clear structural regression, owner leak, unjustified abstraction/file sprawl, or executable replaced path remains unexplained. Do not emit taste-only nits.
