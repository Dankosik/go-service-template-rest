# Slices, Maps, Buffers, And Ownership

## Behavior Change Thesis
When loaded for mutable aggregate symptoms, this file makes the model review aliasing, mutation authority, and observable order instead of likely mistake "just use `slices.Clone`", "never expose maps", or "map order is random so sort everything."

## When To Load
Load when a Go review touches slice/map returns, `[]byte`, `bytes.Buffer`, `strings.Builder`, caches, headers, URL values, map iteration order, cloning, copying, append capacity, retained sub-slices, or mutable data crossing a package boundary.

## Decision Rubric
- Treat map, slice, header, URL values, and `[]byte` values as aliasing surfaces; copying the header is not ownership isolation.
- Clone on boundary crossing when callers must not mutate internal state or when the callee must not retain caller-owned mutable state.
- Do not clone when aliasing or ownership transfer is the documented contract; document the transfer instead.
- Check whether a clone is shallow. `maps.Clone` and `slices.Clone` do not deep-copy nested maps, slices, pointers, or `http.Header` values.
- Sort map keys only when order is externally observable or correctness-relevant: signatures, hashes, stable serialized output, or golden tests.
- Use `http.Header` and `url.Values` methods when canonicalization, first/all value semantics, or encoding matters; raw map access is fine only when those semantics are intentionally bypassed.
- Watch for retained sub-slices of large buffers in long-lived objects. Use a clone, full-slice expression, or `slices.Clip` when capacity retention matters.
- Separate ownership leaks from concurrency risk. If the main harm is mutation after lock release, this file can frame it; deep race analysis belongs to concurrency review.

## Imitate
```text
[high] [go-idiomatic-review] internal/config/config.go:58
Issue: Settings returns c.values directly.
Impact: Callers can mutate Config's internal map after validation and bypass invariants, including after locks are released.
Suggested fix: Return maps.Clone(c.values), or provide read-only accessor methods if mutation must stay internal.
Reference: mutable map ownership contract
```

Copy the mutation-authority proof: the bug is caller access to internal state, not the mere existence of a map return.

```text
[high] [go-idiomatic-review] internal/token/token.go:43
Issue: Token.Bytes returns the internal []byte backing store.
Impact: A caller can modify the token after it has been validated or cached, changing later authorization or comparison behavior.
Suggested fix: Return slices.Clone(t.raw) or append([]byte(nil), t.raw...) depending on the supported Go version.
Reference: []byte boundary ownership
```

Copy the boundary-specific fix: choose the clone primitive according to Go version and contract.

```text
[medium] [go-idiomatic-review] internal/sign/sign.go:77
Issue: The signature input ranges directly over a map when building canonical text.
Impact: Map iteration order is not specified, so identical logical input can produce different signatures across runs.
Suggested fix: Collect keys, sort them, and build the signature in deterministic key order.
Reference: map iteration order contract
```

Copy the order test: sort only because canonical text is observable.

```text
[medium] [go-idiomatic-review] internal/proxy/header.go:35
Issue: The code writes h["content-type"] directly instead of using Header.Set.
Impact: Direct map access bypasses canonicalization, so later Header.Get("Content-Type") can miss or duplicate values depending on key spelling.
Suggested fix: Use h.Set("Content-Type", value) unless non-canonical keys are explicitly required.
Reference: http.Header method contract
```

Copy the wrapper-method reasoning: the standard-library type exposes methods because raw map mutation can skip semantics.

## Reject
```text
Use slices.Clone because it is newer.
```

Reject because the risk is ownership and aliasing. If aliasing is intended, cloning may be wasteful or contract-breaking.

```text
Map iteration order is random, sort it.
```

Reject unless stable order is observable or required for correctness.

```text
Don't expose maps.
```

Reject because maps can be valid API types. The finding must identify whether callers can mutate internal state or observe protected state after locks are released.

## Agent Traps
- Do not assume `maps.Clone(http.Header)` is a deep copy; header values are `[]string` and still alias.
- Do not forget setter ownership: storing caller-provided slices/maps can be as risky as returning internal ones.
- Do not conflate nil-vs-empty with ownership. Load the nil reference when encoding or absence semantics are the real issue.
- Do not make a performance claim about retained buffers unless the object is long-lived enough for retention to matter.
- Do not use this reference to replace local helpers with stdlib only because they exist; load the stdlib-first reference for reinvention questions.

## Validation Shape
- Add mutation-after-return tests: mutate the returned map/slice and assert internal state does not change.
- Add mutation-after-setter tests: mutate caller input after passing it and assert stored state is isolated when required.
- Add deterministic-output tests for signatures, hashes, or serialized maps.
- Add memory-retention proof only when the leak is material and measurable; otherwise report the risk without pretending it was benchmarked.

## Handoffs
- Hand off cache invalidation and DB ownership semantics to DB/cache review.
- Hand off concurrent mutation and lock safety to concurrency review.
- Hand off auth token mutation impact to security review.
- Hand off large-buffer retention performance proof to performance review.
