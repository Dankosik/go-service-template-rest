# Stdlib-First Modern Go

## When To Load
Load this when an implementation choice could use a builtin, standard-library package, or current Go language/library feature instead of a custom helper or dependency. Check the target repository's `go.mod` before using version-specific APIs.

## Good/Bad Examples

Bad: custom helpers that only rename stdlib behavior.

```go
func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}

if containsString(roles, "admin") {
	allow()
}
```

Good: use the stdlib when it expresses the same contract.

```go
if slices.Contains(roles, "admin") {
	allow()
}
```

Bad: a helper that hides zero-value policy.

```go
func chooseString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

displayName := chooseString(strings.TrimSpace(input.Name), "anonymous")
```

Good: use `cmp.Or` only when the zero value really means "use the fallback."

```go
displayName := cmp.Or(strings.TrimSpace(input.Name), "anonymous")
```

Bad: custom sorting glue when the order is simple and local.

```go
sort.Slice(users, func(i, j int) bool {
	if users[i].Team != users[j].Team {
		return users[i].Team < users[j].Team
	}
	return users[i].Name < users[j].Name
})
```

Good: make the comparison read as the policy.

```go
slices.SortFunc(users, func(a, b User) int {
	return cmp.Or(
		strings.Compare(a.Team, b.Team),
		strings.Compare(a.Name, b.Name),
	)
})
```

Bad: string or direct type checks for wrapped errors.

```go
if err == fs.ErrNotExist {
	return nil
}
```

Good: inspect the error tree. In Go 1.26+, `errors.AsType` can replace many pointer-target `errors.As` calls.

```go
if errors.Is(err, fs.ErrNotExist) {
	return nil
}

if pathErr, ok := errors.AsType[*fs.PathError](err); ok {
	return fmt.Errorf("read %s: %w", pathErr.Path, err)
}
```

## Common False Simplifications
- Replacing a domain helper with `cmp.Or` when zero is a meaningful user choice, such as `limit=0` meaning "no limit."
- Replacing a policy helper with `slices.ContainsFunc` or `maps.Clone` when the helper also normalizes case, preserves nil/empty semantics, or owns authorization meaning.
- Treating `slices.Clone`, `maps.Clone`, or `bytes.Clone` as deep copies. They clone the top-level container; element values are assigned.
- Using a newly released API just because the local toolchain supports it. The module's declared `go` version controls compatibility expectations.
- Pulling in a third-party dependency for sorting, cloning, comparison, simple set membership, error joining, path handling, URL parsing, context propagation, or test temp files before checking stdlib.

## Validation Or Test Patterns
- Run `go list -m -f '{{.GoVersion}}'` or inspect `go.mod` before modernizing code.
- Add regression tests around semantic gaps that justified custom code: nil versus empty, case folding, stable ordering, wrapped error identity, or zero-value fallback policy.
- Prefer targeted tests that would fail if a custom helper was replaced with a superficially similar stdlib call.
- When applying broad modernization, run the affected package tests first, then the repository's normal full verification command.

## Source Links Gathered Through Exa
- [Go release history](https://go.dev/doc/devel/release.html)
- [Go 1.26 release notes](https://go.dev/doc/go1.26)
- [slices package](https://pkg.go.dev/slices)
- [maps package](https://pkg.go.dev/maps)
- [cmp package](https://pkg.go.dev/cmp)
- [errors package](https://pkg.go.dev/errors)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Effective Go](https://go.dev/doc/effective_go) - useful for core idioms, but it says it is not actively updated and does not cover major later additions such as generics, modules, or newer libraries.
