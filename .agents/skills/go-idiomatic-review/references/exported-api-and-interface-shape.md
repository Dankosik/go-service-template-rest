# Exported API And Interface Shape

## When To Load It
Load this reference when a Go review touches exported names, package names, doc comments, constructors, interface definitions, return types, option structs, public method/function signatures, compatibility promises, package globals, `init` side effects, or public zero-value behavior.

## Exa Source Links
- [Go Code Review Comments: Interfaces, Package Names, Doc Comments, Pass Values](https://go.dev/wiki/CodeReviewComments)
- [Go Doc Comments](https://go.dev/doc/comment)
- [Package names](https://go.dev/blog/package-names)
- [Keeping Your Modules Compatible](https://go.dev/blog/module-compatibility)
- [Backward Compatibility, Go 1.21, and Go 2](https://go.dev/blog/compat)
- [Effective Go](https://go.dev/doc/effective_go), with the official caveat that Effective Go is not actively updated.

## Review Cues
- A producer package exports an interface that mirrors its only implementation.
- An exported function returns an interface for mocking rather than a real consumer-side behavior boundary.
- A public function signature changes instead of adding a compatible new function or method.
- Exported docs restate names but omit behavior, nil/zero-value constraints, ownership, or error contract.
- A package name is `api`, `types`, `interfaces`, `common`, `util`, or otherwise hides responsibility.
- A package-level mutable variable or `init` hook changes process-wide behavior implicitly.
- An exported struct gains a field whose zero value changes old behavior.

## Bad Review Examples
Bad review:

```text
Interfaces should be small.
```

Why it is bad: small is not enough. The question is whether the interface belongs to the consumer and encodes a real behavior boundary.

Bad review:

```text
Return a concrete type because Go says so.
```

Why it is bad: concrete returns are a default, not a law. Some APIs intentionally hide implementation or define an abstraction boundary.

Bad review:

```text
Add a doc comment because exported things need docs.
```

Why it is bad: a merge-risk review should explain what public behavior is ambiguous without the doc.

## Good Review Examples
Good finding:

```text
[medium] [go-idiomatic-review] internal/email/sender.go:17
Issue: The producer package exports Sender as an interface with the same methods as its only concrete type, only to support tests.
Impact: Returning the interface freezes the producer-owned abstraction and makes future concrete methods harder to add without another exported seam.
Suggested fix: Return *SMTPClient from New and let consumer packages define the narrow interface they need.
Reference: https://go.dev/wiki/CodeReviewComments
```

Good finding:

```text
[high] [go-idiomatic-review] pkg/client/client.go:44
Issue: The exported Do signature now requires context.Context, replacing the old Do(req Request) API.
Impact: Existing callers of the public module will fail to compile even though a compatible additive path exists.
Suggested fix: Add DoContext(ctx, req) and keep Do(req) delegating with an appropriate root context, or route the breaking change through an approved API/version decision.
Reference: https://go.dev/blog/module-compatibility
```

Good finding:

```text
[medium] [go-idiomatic-review] pkg/cache/cache.go:12
Issue: Cache is exported but its doc does not state whether the zero value is usable or whether callers own returned maps.
Impact: Callers can reasonably use var c Cache or mutate returned state and get panics or invariant corruption that are not visible from the API.
Suggested fix: Document the zero-value and ownership contract, and adjust the implementation if the intended contract is zero-value usable.
Reference: https://go.dev/doc/comment
```

## Real Merge-Risk Impact
- Public interface changes can break every external implementation.
- Producer-owned interfaces can freeze the wrong abstraction and push test seams into production API.
- Signature changes in v1+ modules can require a major version or an additive compatibility path.
- Weak docs can leave nil, zero-value, ownership, and error contracts ambiguous at the package boundary.
- Mutable globals and `init` side effects create hidden process-wide behavior and test coupling.

## Smallest Safe Correction
- Return concrete types from producer packages unless the interface is the intended public abstraction.
- Define interfaces in consumer packages when they describe what the consumer needs.
- Add new exported functions or methods for new parameters instead of breaking existing signatures.
- Use option structs when future expansion is likely and zero values can preserve old behavior.
- Write exported docs that state behavior, constraints, ownership, nil/zero-value semantics, and error contracts when those affect callers.
- Keep package names short, lower-case, and responsibility-revealing; avoid junk-drawer names.

## Validation Ideas
- Add compile-time assertions for intended public interface satisfaction only when they guard a real contract.
- Run package tests with examples or documentation tests when exported usage changed.
- Use API compatibility tooling or a deliberate public API review for module-facing changes.
- Add zero-value and ownership tests for exported types.

## Handoffs
- Hand off public API contract and versioning decisions to API/design review.
- Hand off package boundary and ownership drift to architecture/design review.
- Hand off externally visible response/status behavior to API or chi review.
- Hand off security-sensitive docs or global state to security review.
