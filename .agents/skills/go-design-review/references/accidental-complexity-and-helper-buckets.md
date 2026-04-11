# Accidental Complexity And Helper Buckets

## When To Load
Load this when a diff adds wrappers, generalized helpers, broad `common` or `util` packages, producer-side interfaces, option bags, manager types, or indirection layers that might obscure ownership.

Use approved repo intent first. External Go review guidance is useful here because package names and interface placement often reveal whether an abstraction has a real consumer.

## Concrete Review Examples
Finding example: a new `internal/common` package collects unrelated normalization, HTTP response, config, and SQL helpers.

```text
[high] [go-design-review] internal/common/helpers.go:1
Issue: The new helper bucket groups unrelated transport, config, and persistence policy under one ownership-neutral package.
Impact: Future changes can route policy through `common` instead of the actual owner, making boundary drift look like reuse.
Suggested fix: Keep each helper in the owning package or extract a seam-named package only for one cohesive policy with real consumers.
Reference: Go review guidance discouraging meaningless package names like `util`, `common`, `misc`, `api`, `types`, and `interfaces`.
```

Finding example: a producer package defines an interface only so tests can mock it.

```text
[medium] [go-design-review] internal/infra/postgres/repository.go:19
Issue: The adapter defines a producer-owned interface before any consumer has a design need for it.
Impact: The abstraction freezes an implementor-shaped API and adds ceremony without clarifying app ownership.
Suggested fix: Return the concrete repository from the adapter; define a small consumer-owned interface in app/domain only if the use case needs inversion.
Reference: Go review guidance on interfaces belonging in the package that uses them.
```

Finding example: a generic `PipelineManager` wraps two direct calls and a boolean flag for future stages.

```text
[low] [go-design-review] internal/app/reports/pipeline.go:27
Issue: The wrapper layer introduces a manager and future-stage flag but does not remove current duplication or isolate a real seam.
Impact: Readers must follow indirection to understand a simple sequence, and later stages can accumulate hidden policy in the wrapper.
Suggested fix: Keep the direct call sequence local until a second real workflow or approved design seam exists.
Reference: task `plan.md` if it calls for a direct implementation path; otherwise approved app ownership in `docs/repo-architecture.md`.
```

Finding example: three files repeat identical classification rules for one local package policy.

```text
[medium] [go-design-review] internal/infra/http/problems.go:66
Issue: Problem classification policy is duplicated across handlers instead of owned once by the HTTP adapter.
Impact: Adding a new error class will require synchronized edits and can silently diverge by endpoint.
Suggested fix: Extract a seam-named same-package function such as `classifyProblem` and call it from each handler.
Reference: HTTP adapter ownership in `docs/repo-architecture.md`.
```

## Non-Findings To Avoid
- Do not penalize every helper. A small same-package helper with a clear name can be the smallest safe correction.
- Do not force a new package when a same-package function keeps ownership clearer.
- Do not call an abstraction accidental if it is required by an approved design artifact or a real second implementation.
- Do not make a readability preference a design finding unless it affects future change risk, blast radius, or ownership clarity.

## Smallest Safe Correction
- Inline speculative wrappers when they do not protect a real seam.
- Rename or relocate helpers to the owning package instead of a generic bucket.
- Replace producer-owned interfaces with concrete returns, or move narrow interfaces to the consuming package when needed.
- Extract only stable duplicated policy, and give it a seam name tied to the owning package.

## Escalation Rules
- Escalate to `go-design-spec` when the abstraction is trying to express a real but undocumented design seam.
- Hand off to `go-language-simplifier-review` when the issue is mainly control-flow or helper readability after ownership is clear.
- Hand off to `go-performance-review` when the abstraction affects hot-path allocation, serialization, batching, or contention.
- Hand off to `go-qa-review` when the abstraction is acceptable but makes test proof too weak or unclear.

## Exa Source Links
- [Go Code Review Comments - Package Names and Interfaces](https://go.dev/wiki/CodeReviewComments)
- [Organizing a Go module - The Go Programming Language](https://go.dev/doc/modules/layout)
- [arc42 Section 9 - Architecture Decisions](https://docs.arc42.org/section-9/)
- [Architecture Decision Record - Martin Fowler](https://martinfowler.com/bliki/ArchitectureDecisionRecord.html)
