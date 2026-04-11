# Go-Semantic Stop Signs

Behavior Change Thesis: When loaded for simplification that touches Go-semantic protective code, this file makes the model stop before flagging ownership, lifetime, nil, method-set, or stdlib wrapper guards as clutter instead of likely mistake of recommending a behavior-changing cleanup.

## When To Load
Load this when a cleanup removes or inlines code around slice/map cloning, nil versus empty behavior, zero-value usability, receiver or method-set shape, `defer cancel`, close/unlock/rollback ordering, or standard-library wrapper contracts such as `http.Header`, `url.Values`, and similar types.

Use this as a stop-sign reference. It helps decide whether simplification review should avoid a false positive or hand off to `go-idiomatic-review`; it is not a replacement for deep Go-semantic review.

## Decision Rubric
- Non-finding: code that looks ceremonial but preserves alias isolation, nil/empty external behavior, zero-value safety, method-set expectations, or lifetime cleanup.
- Finding-worthy: a "cleanup" removes the protective step and thereby exposes mutable state, changes observable nil/empty behavior, drops required cancellation/close/unlock/rollback, or erases a wrapper's contract.
- Handoff-worthy: the risk depends on subtle Go method sets, interface satisfaction, comparability, `errors.Join` trees, `defer` and named returns, or standard-library contract details beyond the simplification lane.
- Smallest safe fix: restore the protective line or helper and name why it exists; do not broaden the refactor.
- Clone helpers such as `slices.Clone` and `maps.Clone` are shallow; treat them as protecting the copied container only unless nested reference fields are separately proven.

## Imitate
Finding shape to copy when clone-before-store is removed:

```text
[high] [go-language-simplifier-review] internal/app/orders/cache.go:41
Issue: The cleanup inlines `storeOrder` and drops the `slices.Clone(order.Tags)` step that isolated cached state from caller-owned slices.
Impact: The shorter code now lets a caller mutate tags after saving and change the cached order, so a later reader can miss an aliasing bug that the old helper prevented.
Suggested fix: Restore the clone-before-store step or keep a policy-named helper such as `cloneOrderForCache`; hand off deeper aliasing review to `go-idiomatic-review` if more shared fields are involved.
Reference: references/go-semantic-stop-signs.md
```

Copy the move: identify the protective semantic contract, the mutation or lifetime path it blocks, and the smallest restoration.

Finding shape to copy when cleanup removes cancellation:

```text
[medium] [go-language-simplifier-review] internal/infra/http/export.go:76
Issue: The refactor removes `defer cancel()` after creating the request-scoped timeout context because the parent request will eventually finish.
Impact: The simplified path can keep timer resources alive longer than needed and makes timeout ownership less explicit for future cleanup changes.
Suggested fix: Restore `defer cancel()` next to `context.WithTimeout`, or hand off to `go-reliability-review` if timeout ownership changed across the call chain.
Reference: references/go-semantic-stop-signs.md
```

Copy the move: show why the "obvious cleanup" was protecting lifetime ownership.

## Reject
Reject "inline this tiny helper" when the helper owns alias isolation:

```go
func (s *Store) Save(order Order) {
	s.current = order
}
```

Prefer preserving the protective copy:

```go
func cloneOrderForStore(order Order) Order {
	order.Tags = slices.Clone(order.Tags)
	return order
}

func (s *Store) Save(order Order) {
	s.current = cloneOrderForStore(order)
}
```

Reject "empty slices are cleaner than nil" when nil is caller-visible:

```go
func Tags() []string {
	return []string{}
}
```

Do not change this unless the contract says nil and empty are equivalent:

```go
func Tags() []string {
	return nil
}
```

Reject cleanup that removes close or rollback order without proving the replacement owns the same lifetime:

```go
rows, _ := db.QueryContext(ctx, q)
return scan(rows)
```

Prefer explicit ownership at the acquisition site:

```go
rows, err := db.QueryContext(ctx, q)
if err != nil {
	return nil, err
}
defer rows.Close()
return scan(rows)
```

## Agent Traps
- Do not call clone/copy helpers "dead wrappers" until alias ownership is clear.
- Do not overstate `slices.Clone` or `maps.Clone` as deep-copy protection.
- Do not normalize nil and empty values unless the API contract says they are interchangeable.
- Do not remove `defer cancel`, `Close`, `Unlock`, or rollback code as cleanup without proving another owner closes the same lifetime.
- Do not turn `Rows.Close` into a blanket close-error finding; normal exhaustion can close rows implicitly, while early returns and write/autocommit paths still need explicit ownership and error policy.
- Do not simplify receiver shape or exported method names without considering interface satisfaction and zero-value behavior.
- Do not hide stdlib wrapper types behind generic maps if their methods or canonicalization rules are part of the contract.

## Validation Shape
Prefer proof that exposes the protected behavior: mutation after save does not affect stored state, nil/empty behavior remains caller-visible as before, timeout contexts release promptly, resources close on all paths, and exported method/interface checks still compile. If proving this requires deep Go semantics, record a `go-idiomatic-review` handoff rather than over-owning the finding.
