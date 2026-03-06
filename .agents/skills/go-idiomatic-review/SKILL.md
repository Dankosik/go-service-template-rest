---
name: go-idiomatic-review
description: "Review Go code changes for idiomatic correctness, error and context handling, boundary discipline, naming clarity, and maintainability with real merge-risk impact."
---

# Go Idiomatic Review

## Purpose
Protect changed Go code from language-level mistakes and non-idiomatic patterns that increase correctness, operability, or maintenance risk.

## Scope
- review control flow, error handling, and context propagation
- review package boundaries, exported surface, and composition-root discipline
- review interface usage, pointer or value choices, and zero-value friendliness
- review naming, docs, and public-surface clarity
- review whether the validation path matches the changed risk surface

## Boundaries
Do not:
- turn idiomatic review into architecture redesign or deep specialist review
- block on taste-only comments with no correctness or maintainability impact
- take primary ownership of business rules, DB/cache contracts, concurrency correctness, or security depth
- confuse “shorter code” with clearer or safer code

## Core Defaults
- Correctness comes before style.
- Prefer explicit, readable, toolchain-compatible code over clever abstraction.
- Errors should remain explicit contract values, not logs-only side effects.
- Request-scoped work should preserve request context and cancellation semantics.
- Prefer concrete types, minimal exports, and the smallest safe idiomatic correction.

## Expertise

### Control Flow And Readability
- Prefer guard clauses and early returns for the happy path.
- Flag unnecessary nesting, mixed abstraction levels, and functions with multiple unrelated responsibilities.
- Prefer explicit behavior over helper indirection that hides what the code actually does.
- Treat confusing control flow as a maintenance risk when it makes failure behavior hard to follow.

### Error Handling
- Require errors to be returned or handled explicitly, not swallowed behind logs.
- Require operation context in errors when that is needed for diagnosis.
- Use `%w` when callers need to inspect causes; use `errors.Is` and `errors.As` rather than string matching.
- Reject panic for normal error handling.
- Keep error strings lowercase and punctuation-free unless an external contract requires otherwise.

### Context Propagation
- Require `ctx context.Context` first where request scope, cancellation, or deadlines matter.
- Flag storing contexts in structs or passing nil context.
- Require `cancel()` on derived contexts.
- Reject replacing request context with `context.Background()` in request flows.
- Preserve `context.Canceled` and `context.DeadlineExceeded` semantics.

### Package And Boundary Discipline
- Keep package responsibilities focused and import direction clear.
- Flag junk-drawer packages and hidden wiring through globals or `init` side effects.
- Keep composition explicit at the composition root.
- Minimize exported surface and use `internal/` where privacy matters.
- Treat accidental public API growth as a contract risk, not just style noise.

### Types, Interfaces, And Zero Values
- Prefer concrete types unless consumer-side substitution really exists.
- Flag interface-per-struct and producer-owned “for mocking” interfaces without real need.
- Require pointer usage to be justified by mutation, identity, or copy cost.
- Prefer useful zero values when practical.
- Flag over-embedding, pointer-to-basic cargo-culting, and abstraction layers that add no value.

### Naming, Documentation, And Public Surface
- Enforce Go naming norms, consistent initialisms, and non-stuttering package APIs.
- Require boolean names that read as facts or questions.
- For exported changes, require documentation that explains behavior or constraints, not just restates the name.
- Treat unclear naming on critical paths as a maintainability defect.

### Validation Path
- Suggest only the minimal command set that honestly validates the changed risk surface.
- Expect race checks for concurrency-sensitive touched paths and security checks for security-sensitive ones when relevant.
- Do not claim readiness without a clear verification path.

### Cross-Domain Handoffs
- Hand off deep race, deadlock, and shutdown analysis to `go-concurrency-review`.
- Hand off public API semantic depth to `go-design-review` or the contract owner.
- Hand off coverage completeness to `go-qa-review`.
- Hand off profiling and hot-path evidence questions to `go-performance-review`.
- Hand off threat-depth analysis to `go-security-review`.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the concrete Go rule or idiomatic defect
- impact on correctness, diagnosability, or maintenance
- the smallest safe correction
- a validation command when useful
- whether the issue is local code drift or needs design escalation

Severity is merge-risk based:
- `critical`: confirmed idiomatic defect with direct correctness or operational risk
- `high`: strong evidence of meaningful maintainability or correctness risk
- `medium`: bounded but important idiomatic weakness
- `low`: local cleanup that improves clarity or consistency

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

Use this format for each finding:

```text
[severity] [go-idiomatic-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

## Escalate When
Escalate when:
- safe correction changes the public API or the approved package or boundary model (`go-design-spec`)
- the issue reveals a missing contract for public behavior or transport semantics (`api-contract-designer-spec` or `go-chi-spec`)
- local cleanup is blocked by a broader design mistake (`go-design-spec` or `go-architect-spec`)
