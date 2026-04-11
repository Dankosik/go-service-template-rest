# Helper Extraction Economics

## When To Load
Load this when a diff extracts, inlines, moves, renames, or generalizes helpers, wrappers, interfaces, callbacks, option bags, or helper packages.

The review question is whether the helper reduces reasoning load at the call site and preserves ownership, not whether the original function got shorter.

## Review Lens
- A helper earns its keep when its name compresses a stable policy or protects ownership, cleanup, stdlib quirks, error classification, or defaulting.
- A helper is suspect when it is single-use, pass-through, mode-driven, callback-heavy, or named `util`, `common`, `shared`, `helpers`, or similar.
- Inline a helper only when doing so preserves semantics and makes the local decision easier to read.
- Keep small helpers when they protect clone-before-store, error inspection, cancellation, cleanup order, lock scope, or a package-owned policy.

## Real Finding Examples
Finding example: a helper adds jumps without semantic compression.

```text
[medium] [go-language-simplifier-review] internal/app/users/create.go:34
Issue: The new `prepareInput` helper is used once and only moves three local validation assignments away from the decision that consumes them.
Impact: A reader now has to jump between functions to verify the create preconditions, but the helper name does not expose a stable policy beyond "do the next few lines."
Suggested fix: Inline the validation back into `CreateUser`, or rename and narrow the helper only if this is the package's stable user-normalization policy.
Reference: references/helper-extraction-economics.md
```

Finding example: a generic helper merged callers through booleans.

```text
[high] [go-language-simplifier-review] internal/infra/http/respond.go:72
Issue: `writeResult(w, result, true, false)` deduplicates response paths by pushing cache and retry policy into positional booleans.
Impact: The call site no longer says which response behavior is selected, and adding a new response class can route through an invalid flag combination.
Suggested fix: Split the helper into policy-named functions such as `writeCachedOK` and `writeRetryableProblem`, or keep distinct branches local.
Reference: references/helper-extraction-economics.md
```

## Non-Findings To Avoid
- Do not flag a small helper that isolates stable package policy, even if it is only a few lines.
- Do not inline helpers that exist to preserve alias isolation, cleanup ordering, or error identity.
- Do not require a new helper just because two branches have the same shape while their semantics differ.
- Do not turn a same-package helper recommendation into a new global utility package.

## Bad And Good Simplifications
Bad: a helper forces the reader to decode flags.

```go
func writeOrder(w http.ResponseWriter, order Order, includePrivate bool, cached bool) {
	if cached {
		w.Header().Set("Cache-Control", "max-age=60")
	}
	encodeOrder(w, order, includePrivate)
}
```

Good: the names expose policy at the call site.

```go
func writePublicCachedOrder(w http.ResponseWriter, order Order) {
	w.Header().Set("Cache-Control", "max-age=60")
	encodePublicOrder(w, order)
}

func writePrivateOrder(w http.ResponseWriter, order Order) {
	encodePrivateOrder(w, order)
}
```

Bad: a dead wrapper copies a method name without adding meaning.

```go
func load(ctx context.Context, id OrderID) (Order, error) {
	return repo.Load(ctx, id)
}
```

Good: a thin helper can still be worth keeping when it owns policy.

```go
func cloneOrderForCaller(order Order) Order {
	order.Tags = slices.Clone(order.Tags)
	return order
}
```

## Escalation Guidance
- Escalate to `go-design-review` if the helper move crosses package ownership or creates a new abstraction boundary.
- Escalate to `go-idiomatic-review` if the helper replaces a standard-library or builtin operation without preserving extra semantics.
- Escalate to `go-concurrency-review` if a helper hides lock, channel, goroutine, or lifecycle ownership.
- Escalate to `go-qa-review` when a helper is acceptable but tests no longer reveal which branch or invariant failed.

## Source Anchors
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments): package names, interfaces, error flow, and line-length guidance are useful for helper calibration.
- [Package names](https://go.dev/blog/package-names): avoid meaningless package names and use package names to focus ownership.
- [Organizing Go code](https://go.dev/blog/organizing-go-code): avoid grab-bag packages, but do not over-split into package design overhead.
- Repository pattern: `go-coder/references/helper-extraction-and-package-ownership.md`.
