# Exported API And Interface Shape

## Behavior Change Thesis
When loaded for exported-surface symptoms, this file makes the model review consumer-owned abstraction, compatibility, and public contracts instead of likely mistake "small interfaces are good", "return concrete types always", or "add a doc comment."

## When To Load
Load when a Go review touches exported names, package names, doc comments, constructors, interface definitions, return types, option structs, public signatures, compatibility promises, package globals, `init` side effects, or public zero-value behavior.

## Decision Rubric
- Ask whether the exported shape is a public contract, not just whether it compiles or looks idiomatic.
- Prefer concrete return types from producer packages unless the exported interface is the intended behavior boundary.
- Prefer consumer-defined interfaces when the consumer needs substitution over a narrow subset of behavior.
- Treat breaking public signatures in v1+ modules as high risk unless an approved versioning/API decision exists.
- Prefer additive APIs, option structs, or new methods when they preserve existing callers and make future expansion likely.
- Review docs for behavior that callers must know: nil/zero-value usability, ownership, concurrency promises, error inspection, and required initialization. Do not raise doc findings that only restate the name.
- Treat package-level mutable variables and `init` hooks as public behavior when they change process-wide state or test order.
- Avoid package names that hide responsibility (`common`, `util`, `types`, `interfaces`) only when the name makes ownership or imports harder to reason about.

## Imitate
```text
[medium] [go-idiomatic-review] internal/email/sender.go:17
Issue: The producer package exports Sender as an interface with the same methods as its only concrete type, only to support tests.
Impact: Returning the interface freezes the producer-owned abstraction and makes future concrete methods harder to add without another exported seam.
Suggested fix: Return *SMTPClient from New and let consumer packages define the narrow interface they need.
Reference: consumer-owned interface rule
```

Copy the boundary test: the problem is who owns the abstraction, not the number of methods.

```text
[high] [go-idiomatic-review] pkg/client/client.go:44
Issue: The exported Do signature now requires context.Context, replacing the old Do(req Request) API.
Impact: Existing callers of the public module will fail to compile even though a compatible additive path exists.
Suggested fix: Add DoContext(ctx, req) and keep Do(req) delegating through the existing contract, or route the breaking change through an approved API/version decision.
Reference: Go module compatibility contract
```

Copy the compatibility framing: exported breakage is stronger than local style preference.

```text
[medium] [go-idiomatic-review] pkg/cache/cache.go:12
Issue: Cache is exported but its doc does not state whether the zero value is usable or whether callers own returned maps.
Impact: Callers can reasonably use var c Cache or mutate returned state and get panics or invariant corruption that are not visible from the API.
Suggested fix: Document the zero-value and ownership contract, and adjust the implementation if the intended contract is zero-value usable.
Reference: exported doc behavior contract
```

Copy the doc standard: the missing comment matters because it hides a callable contract.

## Reject
```text
Interfaces should be small.
```

Reject because small is not sufficient. The interface must represent a useful consumer-facing behavior boundary.

```text
Return a concrete type because Go says so.
```

Reject because concrete returns are a default, not a law. Some APIs intentionally hide implementation or expose a stable behavior contract.

```text
Add a doc comment because exported things need docs.
```

Reject unless the missing doc hides behavior, compatibility, ownership, zero-value, or error semantics callers need.

## Agent Traps
- Do not use this file to redesign package boundaries from taste; prove exported caller impact or hand off to design/architecture review.
- Do not break public APIs in the suggested fix unless the finding explicitly routes through an approved compatibility decision.
- Do not recommend an option struct if the zero value cannot preserve old behavior.
- Do not treat `internal/` vs exported as only naming hygiene; it is a dependency and compatibility surface.
- Do not duplicate nil/ownership findings here when the risk is purely runtime-local; load the narrower nil or ownership reference instead.

## Validation Shape
- Compile representative callers or add package tests/examples when exported usage changes.
- Add zero-value tests for exported types when the contract says they are usable or harmless.
- Add ownership tests for exported getters/setters that return or accept mutable values.
- Use API compatibility tooling or explicit public API review when module-facing signatures changed.

## Handoffs
- Hand off public API contract and versioning decisions to API/design review.
- Hand off package boundary and ownership drift to architecture/design review.
- Hand off externally visible response/status behavior to API or chi review.
- Hand off security-sensitive docs or global state to security review.
