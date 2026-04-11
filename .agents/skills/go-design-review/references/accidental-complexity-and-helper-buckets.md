# Accidental Complexity And Helper Buckets

## Behavior Change Thesis
When loaded for symptom "the diff adds helpers, wrappers, interfaces, managers, or option bags," this file makes the model distinguish ownership-protecting seams from speculative indirection instead of reflexively praising abstraction or banning helpers.

## When To Load
Load this when a diff adds broad `common` or `util` packages, generalized helpers, producer-side interfaces, option bags, manager types, wrappers, or indirection layers that may obscure ownership.

Prefer `source-of-truth-seam-drift.md` when the main problem is duplicated canonical policy; use this file when the main question is whether the abstraction itself is justified.

## Decision Rubric
- Flag owner-neutral helper buckets when they collect unrelated transport, config, data, or domain policy.
- Flag producer-owned interfaces when they freeze implementor shape before a consumer needs inversion.
- Flag wrapper layers that add naming, flags, or future extension points without reducing current duplication, boundary risk, or change blast radius.
- Accept small same-package helpers when they name a real seam and keep ownership local.
- Accept approved abstractions and real second implementations; review whether the code honors the approved seam rather than relitigating it.

## Imitate
```text
[high] [go-design-review] internal/common/helpers.go:1
Issue: The new helper bucket groups unrelated transport, config, and persistence policy under one ownership-neutral package.
Impact: Future changes can route policy through `common` instead of the actual owner, making boundary drift look like reuse.
Suggested fix: Keep each helper in the owning package or extract a seam-named package only for one cohesive policy with real consumers.
Reference: owning packages in `docs/repo-architecture.md` or task `design/component-map.md`.
```

Copy this shape when the abstraction erases ownership.

```text
[medium] [go-design-review] internal/infra/postgres/repository.go:19
Issue: The adapter defines a producer-owned interface before any consumer has a design need for it.
Impact: The abstraction freezes an implementor-shaped API and adds ceremony without clarifying app ownership.
Suggested fix: Return the concrete repository from the adapter; define a small consumer-owned interface in app/domain only if the use case needs inversion.
Reference: app/repository ownership in task design or `docs/repo-architecture.md`.
```

Copy this shape when the review needs to correct interface placement without demanding an interface by default.

```text
[low] [go-design-review] internal/app/reports/pipeline.go:27
Issue: The wrapper layer introduces a manager and future-stage flag but does not remove current duplication or isolate a real seam.
Impact: Readers must follow indirection to understand a simple sequence, and later stages can accumulate hidden policy in the wrapper.
Suggested fix: Keep the direct call sequence local until a second real workflow or approved design seam exists.
Reference: task `plan.md` if it calls for a direct implementation path.
```

Copy this shape for speculative future-proofing with concrete comprehension or ownership cost.

## Reject
```text
[medium] [go-design-review] internal/infra/http/problems.go:66
Issue: Helper functions are bad.
Suggested fix: Inline it.
```

Reject because helpers are acceptable when they name an owned stable seam.

```text
[low] [go-design-review] internal/app/reports/pipeline.go:27
Issue: This is too complex.
```

Reject because "too complex" is not merge-risk unless it names the hidden owner, blast radius, or future-change trap.

## Agent Traps
- Do not make readability taste a design finding unless it affects future change risk, blast radius, or ownership clarity.
- Do not force a new package when a same-package function keeps ownership clearer.
- Do not use this broad smell reference as the default when a narrower boundary, dependency, or source-of-truth reference matches.

## Validation Shape
Look for the consumers the abstraction claims to serve. If there is one caller, one implementation, no approved seam, and no risk reduction, the safer review move is to inline or relocate. If there are multiple real consumers or an approved seam, validate owner names and call direction instead.
