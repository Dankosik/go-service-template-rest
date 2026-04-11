# Nil, Zero Value, And Typed Nil

## When To Load It
Load this reference when a Go review touches nil interfaces, typed-nil errors, nil maps, nil slices, nil channels, constructors, zero-value usability, optional no-op implementations, empty vs absent collections, JSON-visible nil behavior, or public contracts around missing values.

## Exa Source Links
- [Go FAQ: Why is my nil error value not equal to nil?](https://go.dev/doc/faq#nil_error)
- [Go Code Review Comments: Declaring Empty Slices](https://go.dev/wiki/CodeReviewComments)
- [Go specification](https://go.dev/ref/spec)
- [Keeping Your Modules Compatible](https://go.dev/blog/module-compatibility)
- [Effective Go: new and make](https://go.dev/doc/effective_go#allocation_new), with the official caveat that Effective Go is not actively updated.

## Review Cues
- A function returns an interface type while holding a nil concrete pointer.
- A constructor exists only to initialize maps/slices that could be made zero-value safe.
- A method writes to a nil map or sends/receives on a nil channel.
- A public API changes from nil slice to empty slice, or the reverse, where JSON or caller checks make that observable.
- An optional dependency returns a typed-nil implementation instead of a real nil, no-op implementation, or explicit `(value, ok)`.
- A zero value panics before any documented initialization requirement.

## Bad Review Examples
Bad review:

```text
Nil slices and empty slices are the same, so this is fine.
```

Why it is bad: they are often interchangeable for length and range, but they differ under JSON encoding and explicit nil checks.

Bad review:

```text
Always prefer nil slices.
```

Why it is bad: nil is a good default, but a public encoding contract may require `[]` rather than `null`.

Bad review:

```text
This nil interface behavior is confusing; use reflect to check nil.
```

Why it is bad: reflection usually papers over a bad API contract. Prefer returning a real nil, a concrete type, a no-op implementation, or an explicit presence result.

## Good Review Examples
Good finding:

```text
[critical] [go-idiomatic-review] internal/notifier/factory.go:41
Issue: NewNotifier returns a typed-nil *EmailNotifier as the Notifier interface when email is disabled.
Impact: Callers see notifier != nil and then panic when invoking methods on a nil receiver path that was supposed to mean "disabled".
Suggested fix: Return a real nil interface, return a no-op Notifier, or change the signature to (*EmailNotifier, bool) if absence is part of the contract.
Reference: https://go.dev/doc/faq#nil_error
```

Good finding:

```text
[high] [go-idiomatic-review] internal/tags/tags.go:26
Issue: Add writes to t.values without ensuring the map is initialized, so the zero value of Tags panics.
Impact: The exported type looks usable as var t Tags, but the first Add call can panic in ordinary caller code.
Suggested fix: Lazily initialize the map in Add or make the constructor requirement explicit and keep the type unexported.
Reference: https://go.dev/blog/module-compatibility
```

Good finding:

```text
[medium] [go-idiomatic-review] internal/api/response.go:63
Issue: The response changed omitted items from []string{} to nil without updating the JSON contract.
Impact: Existing clients can observe `null` instead of `[]`, even though len/range behavior in Go tests remains the same.
Suggested fix: Preserve the previous non-nil empty slice at the serialization boundary, or record the API contract change for the API lane.
Reference: https://go.dev/wiki/CodeReviewComments
```

## Real Merge-Risk Impact
- Typed-nil interfaces can make disabled or absent dependencies look present and panic later.
- Nil map writes panic at runtime.
- Nil channels block forever if exposed as an operational channel.
- Zero-value-hostile exported types force hidden construction order onto callers.
- Nil-vs-empty changes can break JSON clients, equality checks, or compatibility expectations.

## Smallest Safe Correction
- Return a real nil interface by returning `nil` directly, not a nil concrete pointer stored in an interface.
- Prefer concrete return types when the producer does not need to own an interface contract.
- Lazily initialize maps in mutating methods when zero-value usability is intended.
- Use no-op implementations for optional behavior when method calls should remain safe.
- Preserve nil-vs-empty behavior at public serialization boundaries.
- Document any constructor requirement when the zero value cannot be useful or harmless.

## Validation Ideas
- Add a test that compares the returned interface directly to nil.
- Add zero-value tests for exported structs and their first mutating method.
- Add JSON golden tests for nil-vs-empty public response fields.
- Add panic-focused tests for nil map and nil receiver paths only when they prove an expected public contract.

## Handoffs
- Hand off JSON/API-visible absent vs empty semantics to API review.
- Hand off optional dependency policy to architecture/design review.
- Hand off nil channel and goroutine blocking to concurrency review.
