# Helper Extraction Economics

Behavior Change Thesis: When loaded for helper extraction or inlining, this file makes the model judge whether the helper compresses stable policy at the call site instead of likely mistake of treating every wrapper as either free readability or useless indirection.

## When To Load
Load this when a diff extracts, inlines, moves, renames, or generalizes helpers, wrappers, interfaces, callbacks, option bags, or helper packages.

Use this as the primary reference for helper economics. If the helper mainly changes error identity, use `error-path-simplification.md`; if it mainly hides Go-semantic protection, use `go-semantic-stop-signs.md`.

## Decision Rubric
- Keep or recommend a helper when its name compresses stable package policy, defaulting, cleanup scope, stdlib quirks, error classification, alias isolation, or ownership.
- Flag a helper when it is single-use and only moves nearby assignments away from the decision that consumes them.
- Flag a helper when it uses positional booleans, raw modes, callbacks, or option blobs to merge callers with different semantics.
- Flag a helper package when it turns a package-owned rule into `util`, `common`, `shared`, `helpers`, or a similar bucket.
- Inline only when inlining preserves semantics and makes the local decision easier to read.

## Imitate
Finding shape to copy when a helper adds jumps without semantic compression:

```text
[medium] [go-language-simplifier-review] internal/app/users/create.go:34
Issue: The new `prepareInput` helper is used once and only moves three local validation assignments away from the decision that consumes them.
Impact: A reader now has to jump between functions to verify the create preconditions, but the helper name does not expose a stable policy beyond "do the next few lines."
Suggested fix: Inline the validation back into `CreateUser`, or rename and narrow the helper only if this is the package's stable user-normalization policy.
Reference: references/helper-extraction-economics.md
```

Copy the move: show the call-site cost and leave a path for a better policy-named helper if the policy is real.

Finding shape to copy when a generic helper merges callers through flags:

```text
[high] [go-language-simplifier-review] internal/infra/http/respond.go:72
Issue: `writeResult(w, result, true, false)` deduplicates response paths by pushing cache and retry policy into positional booleans.
Impact: The call site no longer says which response behavior is selected, and adding a new response class can route through an invalid flag combination.
Suggested fix: Split the helper into policy-named functions such as `writeCachedOK` and `writeRetryableProblem`, or keep distinct branches local.
Reference: references/helper-extraction-economics.md
```

Copy the move: name the hidden policies and the invalid-combination risk.

## Reject
Reject this wrapper when it only copies a method name:

```go
func load(ctx context.Context, id OrderID) (Order, error) {
	return repo.Load(ctx, id)
}
```

It creates a jump without owning policy.

Do not reject this thin helper:

```go
func cloneOrderForCaller(order Order) Order {
	order.Tags = slices.Clone(order.Tags)
	return order
}
```

It protects alias isolation. The helper is short, but it owns a contract a reader should not have to rediscover.

Reject flag-driven helper APIs like this:

```go
func writeOrder(w http.ResponseWriter, order Order, includePrivate bool, cached bool) {
	if cached {
		w.Header().Set("Cache-Control", "max-age=60")
	}
	encodeOrder(w, order, includePrivate)
}
```

Prefer policy-named helpers when behavior classes are stable:

```go
func writePublicCachedOrder(w http.ResponseWriter, order Order) {
	w.Header().Set("Cache-Control", "max-age=60")
	encodePublicOrder(w, order)
}

func writePrivateOrder(w http.ResponseWriter, order Order) {
	encodePrivateOrder(w, order)
}
```

## Agent Traps
- Do not inline helpers that exist to preserve alias isolation, cleanup ordering, error identity, lock scope, cancellation, or package-owned defaults.
- Do not create a new global helper bucket as the fix for repeated same-package policy.
- Do not flag a single-use helper when it names a lifecycle boundary or hides a sharp stdlib contract intentionally.
- Do not approve callback-heavy helpers just because they remove repeated `if err != nil` blocks.

## Validation Shape
When a helper changes branch selection or behavior modes, ask for proof that each old behavior class still maps to the same status, error identity, side effect, and cleanup path. When the issue is only local call-site clarity, validation can be a targeted compile/test command plus reviewer inspection of the corrected call sites.
