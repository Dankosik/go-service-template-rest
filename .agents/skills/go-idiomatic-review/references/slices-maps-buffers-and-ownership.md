# Slices, Maps, Buffers, And Ownership

## When To Load It
Load this reference when a Go review touches slice/map returns, `[]byte`, `bytes.Buffer`, `strings.Builder`, caches, headers, URL values, map iteration order, cloning, copying, append capacity, retaining sub-slices, or mutable data crossing a package boundary.

## Exa Source Links
- [Go Slices: usage and internals](https://go.dev/blog/slices-intro)
- [slices package](https://pkg.go.dev/slices)
- [maps package](https://pkg.go.dev/maps)
- [Go Code Review Comments: Declaring Empty Slices and Copying](https://go.dev/wiki/CodeReviewComments)
- [Go specification: range over maps](https://go.dev/ref/spec#For_statements)
- [net/http Header](https://pkg.go.dev/net/http#Header)
- [net/url Values](https://pkg.go.dev/net/url#Values)

## Review Cues
- A getter returns an internal map, slice, `[]byte`, header, or URL values map.
- A setter stores a caller-owned map/slice without documenting ownership transfer.
- A sub-slice of a large buffer is retained in a long-lived object.
- Code depends on map iteration order for stable output, signatures, tests, or hashing.
- Code mutates `http.Header` or `url.Values` by raw map access when methods preserve canonicalization or encoding contract.
- A clone is shallow but callers may assume deep isolation.

## Bad Review Examples
Bad review:

```text
Use slices.Clone because it is newer.
```

Why it is bad: the risk is ownership and aliasing. If aliasing is intended, cloning may be wasteful or contract-breaking.

Bad review:

```text
Map iteration order is random, sort it.
```

Why it is bad: sorting is needed only where stable order is observable or required for correctness.

Bad review:

```text
Don't expose maps.
```

Why it is bad: maps can be valid API types. The review must identify whether callers can mutate internal state or observe protected state after locks are released.

## Good Review Examples
Good finding:

```text
[high] [go-idiomatic-review] internal/config/config.go:58
Issue: Settings returns c.values directly.
Impact: Callers can mutate Config's internal map after validation and bypass invariants, including after locks are released.
Suggested fix: Return maps.Clone(c.values), or provide read-only accessor methods if mutation must stay internal.
Reference: https://pkg.go.dev/maps
```

Good finding:

```text
[high] [go-idiomatic-review] internal/token/token.go:43
Issue: Token.Bytes returns the internal []byte backing store.
Impact: A caller can modify the token after it has been validated or cached, changing later authorization or comparison behavior.
Suggested fix: Return slices.Clone(t.raw) or append([]byte(nil), t.raw...) depending on the supported Go version.
Reference: https://pkg.go.dev/slices
```

Good finding:

```text
[medium] [go-idiomatic-review] internal/sign/sign.go:77
Issue: The signature input ranges directly over a map when building canonical text.
Impact: Map iteration order is not specified, so identical logical input can produce different signatures across runs.
Suggested fix: Collect keys, sort them, and build the signature in deterministic key order.
Reference: https://go.dev/ref/spec#For_statements
```

Good finding:

```text
[medium] [go-idiomatic-review] internal/proxy/header.go:35
Issue: The code writes h["content-type"] directly instead of using Header.Set.
Impact: Direct map access bypasses canonicalization, so later Header.Get("Content-Type") can miss or duplicate values depending on key spelling.
Suggested fix: Use h.Set("Content-Type", value) unless non-canonical keys are explicitly required.
Reference: https://pkg.go.dev/net/http#Header
```

## Real Merge-Risk Impact
- External mutation can corrupt internal state, caches, validation results, or locked invariants.
- Shallow clones can still share pointer, slice, or map elements.
- Unspecified map order can break signatures, hashes, golden tests, and stable API output.
- Retaining a small sub-slice can keep a large backing array alive.
- Raw wrapper-map access can bypass canonicalization, encoding, or first-value/all-values semantics.

## Smallest Safe Correction
- Clone maps and slices at package boundaries when ownership isolation is required.
- Document ownership transfer when storing caller-provided mutable values is intentional.
- Use `slices.Clone`, `maps.Clone`, `copy`, or `append([]T(nil), s...)` according to the module's Go version.
- Use full-slice expressions or `slices.Clip` when capacity retention matters.
- Sort keys only where order is externally observable.
- Use `http.Header` and `url.Values` methods for their documented canonicalization and encoding behavior.

## Validation Ideas
- Add mutation-after-return tests: mutate the returned map/slice and assert internal state does not change.
- Add mutation-after-setter tests: mutate caller input after passing it and assert stored state is isolated when required.
- Add deterministic output tests for signatures or serialized maps.
- Add memory-retention tests only when the leak is material and measurable; otherwise mention the risk.

## Handoffs
- Hand off cache invalidation and DB ownership semantics to DB/cache review.
- Hand off concurrent mutation and lock safety to concurrency review.
- Hand off auth token mutation impact to security review.
- Hand off large-buffer retention performance proof to performance review.
