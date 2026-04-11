# Stdlib-First Modern Go

## Behavior Change Thesis
When loaded for custom-helper, dependency, or older-idiom pressure, this file makes the model choose module-compatible Go language and standard-library facilities instead of writing wrapper helpers or dependencies that duplicate current Go.

## When To Load
Load this when an implementation choice could use a builtin, standard-library package, or current Go API instead of custom code or a dependency.

## Decision Rubric
- Check the module's `go` directive before using a version-specific API.
- Version-gate newer APIs and builtins that agents commonly over-apply: `bytes.Clone` is Go 1.20+, `min`, `max`, `clear`, `slices`, and `maps` are Go 1.21+, `cmp.Or` is Go 1.22+, `sync.WaitGroup.Go` and `testing/synctest` are Go 1.25+, and `errors.AsType` and `testing.T.ArtifactDir` are Go 1.26+.
- Prefer the builtin or stdlib call when it expresses the same caller-visible contract.
- Keep custom code when it owns a semantic the stdlib call does not: domain naming, normalization, nil/empty shape, error identity, bounds, ordering, or authorization meaning.
- Avoid third-party dependencies for sorting, cloning, set membership, comparisons, simple path/URL/string work, test temp files, or error inspection until the stdlib gap is concrete.
- Modernize touched local code only when the diff still tells one story and the behavior stays proved.

## Imitate
Use the standard package when it states the contract directly.

```go
if slices.Contains(roles, "admin") {
	allow()
}
```

Use `cmp.Or` only when the zero value really means fallback.

```go
displayName := cmp.Or(strings.TrimSpace(input.Name), "anonymous")
```

On Go 1.26+, prefer `errors.AsType` when matching a typed error that implements `error`; keep `errors.As` for older modules or non-error interface targets.

```go
if pathErr, ok := errors.AsType[*fs.PathError](err); ok {
	return fmt.Errorf("read %s: %w", pathErr.Path, err)
}
```

## Reject
Reject helpers that only rename stdlib behavior.

```go
func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
```

Reject stdlib replacements that erase policy.

```go
limit := cmp.Or(input.Limit, defaultLimit)
```

This is wrong when `0` means "no limit" rather than "missing limit."

Reject clone substitutions that pretend to deep-copy nested mutable values.

```go
snap := maps.Clone(settings)
```

This only clones the top-level map. Nested slices, maps, pointers, or buffers still alias.

## Agent Traps
- Treating the local Go toolchain as permission to use APIs newer than the module's declared `go` version.
- Replacing a policy helper with `slices.Contains`, `slices.ContainsFunc`, `maps.Clone`, or `cmp.Or` when the helper also normalizes, preserves nilness, or owns domain meaning.
- Adding a helper to avoid one import.
- Pulling in a dependency for behavior already covered by `slices`, `maps`, `cmp`, `errors`, `io`, `net/http`, `path/filepath`, `strings`, `testing`, or `time`.
- Collapsing a readable local conditional into a clever stdlib expression that hides the contract.

## Validation Shape
- Verify the module Go version with `go list -m -f '{{.GoVersion}}'` or `go.mod`.
- Add a regression test around any semantic gap that justified keeping custom code.
- When replacing custom code, test the contract the helper used to protect: nil versus empty, zero-value fallback, stable order, case folding, wrapped error identity, or shallow versus deep copy.
