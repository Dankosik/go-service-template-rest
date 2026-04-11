# Standard-Library-First Modern Go Review

## Behavior Change Thesis
When loaded for helper-reinvention symptoms, this file makes the model check effective Go version and semantic deltas before choosing standard-library replacement or preserving a wrapper instead of likely mistake "stdlib is always better" or outdated loop-variable folklore.

## When To Load
Load when a Go review touches helper packages that duplicate builtins or the standard library, compatibility shims, generic slice/map helpers, sorting/comparison helpers, URL/header wrappers, string/byte manipulation, custom error traversal, or loop-variable capture claims.

## Decision Rubric
- Identify the effective Go version from `go.mod`, build tags, or file constraints before recommending newer builtins or packages.
- Replace local helpers when they exactly duplicate supported builtins or standard-library behavior and add no compatibility, ownership, normalization, or domain contract.
- Keep local helpers when they intentionally carry policy: deep copy, nil/empty normalization, canonicalization, redaction, validation, compatibility with older supported versions, or domain naming.
- Check shallow/deep semantics before replacing clone helpers with `slices.Clone` or `maps.Clone`.
- Check error-tree semantics before preserving custom error traversal; `errors.Is` and `errors.As` cover standard wrapping and joined errors.
- Check loop-variable capture claims against effective Go version and declaration shape; in Go 1.22+ files or packages, variables declared by the loop get per-iteration instances, but preexisting variables assigned inside the loop can still have the old capture hazard.
- Treat wrapper removal as a public API/design question when the helper is exported or has broad callers.
- When the stdlib is almost enough, name the remaining semantic gap and decide whether it is real or accidental.

## Imitate
```text
[medium] [go-idiomatic-review] internal/slices/clone.go:14
Issue: cloneStrings duplicates slices.Clone exactly now that the module declares go 1.21.
Impact: The local helper adds another copy contract to review and can drift from the standard library's nil-preserving behavior.
Suggested fix: Use slices.Clone at call sites, or keep the helper only if it intentionally changes nil/empty behavior and document that policy.
Reference: Go 1.21 slices package availability
```

Copy the version-and-delta proof: the finding says why the helper no longer carries unique behavior.

```text
[medium] [go-idiomatic-review] internal/errors/contains.go:22
Issue: containsErr walks Unwrap manually but misses joined errors.
Impact: Callers can fail to recognize an error returned through errors.Join, while errors.Is already handles the standard error tree contract.
Suggested fix: Replace containsErr(err, target) with errors.Is(err, target).
Reference: errors.Is traversal contract
```

Copy the semantic mismatch: this is not just shorter code; the custom helper is observably incomplete.

```text
[low] [go-idiomatic-review] internal/math/min.go:8
Issue: minInt duplicates the predeclared min builtin while the package is built with Go 1.21 or newer.
Impact: The helper adds unnecessary local surface and makes reviewers verify behavior the toolchain already provides.
Suggested fix: Use min(a, b) directly unless this package still supports an older Go version through build tags.
Reference: Go 1.21 builtin availability
```

Copy the compatibility caveat: the fix is conditional on supported toolchain.

```text
No finding: cloneHeaders intentionally deep-copies []string values inside http.Header and preserves canonical keys; replacing it with maps.Clone would make only a shallow copy.
Reference: http.Header ownership semantics
```

Copy the non-finding: stdlib-first review still preserves local helpers that encode real policy.

## Reject
```text
Replace this with slices.Clone; stdlib is always better.
```

Reject because `slices.Clone` is shallow and preserves nilness. The local helper may intentionally deep-copy elements or normalize empty values.

```text
This loop variable capture is broken.
```

Reject until you check effective Go version and whether the captured variable still has the old bug shape.

```text
Delete this wrapper around url.Values.
```

Reject when the wrapper enforces encoding, normalization, redaction, validation, or domain policy.

## Agent Traps
- Do not recommend a package or builtin that the module cannot use under its declared Go version.
- Do not remove exported helpers as "cleanup" without routing public API compatibility.
- Do not miss local policy hidden in tests: nil preservation, deep copy, sorting, redaction, or validation often distinguishes wrappers from stdlib.
- Do not keep stale compatibility shims just because they already exist; once the toolchain baseline moves, they can become misleading.
- Do not use this file for boundary aliasing itself; load the ownership reference when the defect is caller mutation.

## Validation Shape
- Add tests that prove nil preservation, shallow/deep copy, ordering, redaction, or validation before replacing a helper.
- Run focused package tests after replacing helper call sites.
- Use `go vet` with a modern toolchain when standard-library symbol version mismatches are plausible.
- For exported helper removal, require API/design review or compatibility tooling rather than only `go test`.

## Handoffs
- Hand off performance claims about helper replacement to performance review.
- Hand off API compatibility of removing exported helpers to design/API review.
- Hand off security-sensitive standard-library wrapper behavior, such as URL or header redaction, to security review.
