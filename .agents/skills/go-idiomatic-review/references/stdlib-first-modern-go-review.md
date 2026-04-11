# Standard-Library-First Modern Go Review

## When To Load It
Load this reference when a Go review touches helper packages that duplicate builtins or the standard library, compatibility shims, generic slice/map helpers, sorting/comparison helpers, URL/header wrappers, string/byte manipulation, custom error traversal, or loop-variable folklore.

## Exa Source Links
- [Go 1.21 release notes](https://go.dev/doc/go1.21)
- [Go 1.22 release notes](https://go.dev/doc/go1.22)
- [Go 1.23 release notes](https://go.dev/doc/go1.23)
- [slices package](https://pkg.go.dev/slices)
- [maps package](https://pkg.go.dev/maps)
- [errors package](https://pkg.go.dev/errors)
- [strings package](https://pkg.go.dev/strings)
- [net/http Header](https://pkg.go.dev/net/http#Header)
- [net/url Values](https://pkg.go.dev/net/url#Values)
- [Fixing For Loops in Go 1.22](https://go.dev/blog/loopvar-preview)

## Review Cues
- A repository helper does what `min`, `max`, `clear`, `slices`, `maps`, `cmp`, `errors`, `strings`, `bytes`, `net/http`, or `net/url` already does for the declared Go version.
- A compatibility helper remains after `go.mod` has moved to a version that includes the stdlib feature.
- A review comment repeats pre-Go-1.22 loop-variable capture warnings without checking the module/file version.
- A helper almost matches stdlib behavior but quietly differs in nilness, ordering, deep/shallow copy, canonicalization, or error exposure.
- The helper carries domain meaning, and replacing it with stdlib would erase policy.

## Bad Review Examples
Bad review:

```text
Replace this with slices.Clone; stdlib is always better.
```

Why it is bad: `slices.Clone` is shallow and preserves nilness. If the local helper deep-copies elements or normalizes nil to empty, it carries extra contract.

Bad review:

```text
This loop variable capture is broken.
```

Why it is bad: Go 1.22 changed loop variable semantics for modules/files using the new version. The finding must check the effective Go version.

Bad review:

```text
Delete this wrapper around url.Values.
```

Why it is bad: wrappers can be justified when they enforce encoding, normalization, redaction, or domain validation policy.

## Good Review Examples
Good finding:

```text
[medium] [go-idiomatic-review] internal/slices/clone.go:14
Issue: cloneStrings duplicates slices.Clone exactly now that the module declares go 1.21.
Impact: The local helper adds another copy contract to review and can drift from the stdlib's nil-preserving behavior.
Suggested fix: Use slices.Clone at call sites, or keep the helper only if it intentionally changes nil/empty behavior and document that policy.
Reference: https://pkg.go.dev/slices
```

Good finding:

```text
[medium] [go-idiomatic-review] internal/errors/contains.go:22
Issue: containsErr walks Unwrap manually but misses joined errors.
Impact: Callers can fail to recognize an error returned through errors.Join, while errors.Is already handles the standard error tree contract.
Suggested fix: Replace containsErr(err, target) with errors.Is(err, target).
Reference: https://pkg.go.dev/errors
```

Good finding:

```text
[low] [go-idiomatic-review] internal/math/min.go:8
Issue: minInt duplicates the predeclared min builtin while the package is built with Go 1.21 or newer.
Impact: The helper adds unnecessary local surface and makes reviewers verify behavior the toolchain already provides.
Suggested fix: Use min(a, b) directly unless this package still supports an older Go version through build tags.
Reference: https://go.dev/doc/go1.21
```

Good non-finding:

```text
No finding: cloneHeaders intentionally deep-copies []string values inside http.Header and preserves canonical keys; replacing it with maps.Clone would make only a shallow copy.
Reference: https://pkg.go.dev/net/http#Header
```

## Real Merge-Risk Impact
- Reinvented helpers can miss newer stdlib semantics, such as joined error traversal or nil preservation.
- Compatibility shims can become misleading once the module version moves forward.
- Premature stdlib replacement can erase local policy around normalization, ownership, or API compatibility.
- Outdated folklore can create noisy review findings and unnecessary churn.

## Smallest Safe Correction
- Check the effective Go version before recommending a builtin or stdlib replacement.
- Replace exact duplicates with stdlib helpers when behavior matches.
- Keep local helpers that encode domain policy, compatibility, nil/empty normalization, ownership isolation, or deep-copy behavior.
- Document the semantic delta when a helper intentionally differs from stdlib.
- For Go 1.23 or newer, consider `slices.Sorted(maps.Keys(m))` for deterministic sorted keys when it matches version and behavior.

## Validation Ideas
- Add tests that prove nil preservation, shallow/deep copy, or ordering before replacing a helper.
- Run `go test ./...` after replacing helper call sites.
- Use `go vet` with Go 1.23 or newer to catch stdlib symbol version mismatches through the `stdversion` analyzer.
- Add compile-time version/build-tag checks only when the repo already uses that pattern.

## Handoffs
- Hand off performance claims about helper replacement to performance review.
- Hand off API compatibility of removing exported helpers to design/API review.
- Hand off security-sensitive stdlib wrapper behavior, such as URL or header redaction, to security review.
