# Nil, Zero Value, And Typed Nil

## Behavior Change Thesis
When loaded for nil or zero-value symptoms, this file makes the model review observable runtime/API contracts instead of likely mistake "nil and empty are basically the same", "always prefer nil", or "use reflection to check typed nil."

## When To Load
Load when a Go review touches nil interfaces, typed-nil errors, nil maps, nil slices, nil channels, constructors, zero-value usability, optional no-op implementations, empty vs absent collections, JSON-visible nil behavior, or public contracts around missing values.

## Decision Rubric
- For interface returns, check disabled/error/empty branches for a nil concrete pointer stored in an interface. The caller sees a non-nil interface.
- Treat nil map writes and nil channel operations as runtime defects when reachable through changed code.
- Prefer useful or harmless zero values for exported types when practical; otherwise require an explicit constructor contract.
- Review nil-vs-empty only when it is observable: JSON, equality, `nil` checks, compatibility, or documented API semantics. Do not raise it for plain `len`/`range` behavior alone.
- Prefer a real nil interface, a no-op implementation, or an explicit `(value, ok)` contract over typed-nil optional dependencies.
- Avoid reflection-based nil probes as the primary fix; they usually preserve a confusing API shape.
- Treat zero-value panics on exported types as stronger than local constructor tidiness because callers can reasonably write `var t T`.
- Hand off if the nil/empty choice is really an API payload policy or product semantics decision.

## Imitate
```text
[critical] [go-idiomatic-review] internal/notifier/factory.go:41
Issue: NewNotifier returns a typed-nil *EmailNotifier as the Notifier interface when email is disabled.
Impact: Callers see notifier != nil and then panic when invoking methods on a nil receiver path that was supposed to mean "disabled".
Suggested fix: Return a real nil interface, return a no-op Notifier, or change the signature to report absence explicitly.
Reference: typed-nil interface rule
```

Copy the caller-observable proof: a non-nil interface violates the disabled contract.

```text
[high] [go-idiomatic-review] internal/tags/tags.go:26
Issue: Add writes to t.values without ensuring the map is initialized, so the zero value of Tags panics.
Impact: The exported type looks usable as var t Tags, but the first Add call can panic in ordinary caller code.
Suggested fix: Lazily initialize the map in Add or make the constructor requirement explicit and keep the type unexported if zero-value use is not supported.
Reference: zero-value usability contract
```

Copy the exported zero-value reasoning: constructors are not automatically enough when the type is public.

```text
[medium] [go-idiomatic-review] internal/api/response.go:63
Issue: The response changed omitted items from []string{} to nil without updating the JSON contract.
Impact: Existing clients can observe null instead of [], even though len/range behavior in Go tests remains the same.
Suggested fix: Preserve the previous non-nil empty slice at the serialization boundary, or record the API contract change for the API lane.
Reference: nil-vs-empty encoding contract
```

Copy the "observable where" test: Go iteration equivalence is irrelevant if JSON clients see a different value.

## Reject
```text
Nil slices and empty slices are the same, so this is fine.
```

Reject when encoding, explicit nil checks, or compatibility make the distinction observable.

```text
Always prefer nil slices.
```

Reject because public response contracts may require `[]` rather than `null`.

```text
This nil interface behavior is confusing; use reflect to check nil.
```

Reject because reflection usually papers over a bad contract. Prefer a real nil, no-op implementation, concrete return type, or explicit presence result.

## Agent Traps
- Do not raise nil-vs-empty as a finding without an observable caller or serialization effect.
- Do not say `return (*T)(nil)` through an interface is nil; it is not nil to callers.
- Do not hide zero-value breakage behind "call the constructor" if the exported type invites direct use.
- Do not recommend no-op implementations when the caller must distinguish disabled from enabled behavior.
- Do not use this file for mutable aliasing unless nilness or zero-value behavior is the actual defect.

## Validation Shape
- Add a test comparing the returned interface directly to nil on the disabled/absent path.
- Add zero-value tests for exported structs and their first mutating method.
- Add JSON golden tests for nil-vs-empty public response fields.
- Add panic-focused tests only when they prove an expected public contract; otherwise fix the reachable panic directly.

## Handoffs
- Hand off JSON/API-visible absent vs empty semantics to API review.
- Hand off optional dependency policy to architecture/design review.
- Hand off nil channel and goroutine blocking to concurrency review.
