# Aliasing And Ownership

## Load When
Load when a Go review touches a slice, map, `[]byte`, or `http.Header` crossing a package boundary, a clone or copy helper, a struct copied by value, map iteration feeding serialized output, or a retained sub-slice of a large buffer.

## Decide
- Copying a map, slice, or header copies the header, not the data. Returning or storing one hands over mutation authority; the repository's boundary convention is `slices.Clone` on the way out, and a setter that keeps caller-owned state is the same defect pointed inward.
- Clone depth is the trap, and no configured linter checks it. `maps.Clone` on an `http.Header` still aliases every value slice, so a write through the copy is visible in the original; `Header.Clone()` copies them. Verify the depth the contract needs before calling a clone sufficient.
- `slices.Clip` and `s[:len(s):len(s)]` bound future appends and keep the same backing array. Releasing a large array for collection requires a copy.
- Sort map keys only where order reaches an observer: signatures, hashes, stable serialized output, golden files. Elsewhere the sort is cost without a contract.
- `modernize` proposes stdlib replacements for local helpers. Keep the helper where it carries policy the stdlib call drops — deep copy, redaction, canonicalization, nil normalization — and name that policy rather than the version.
- A value copy also copies copy-sensitive state. `govet`'s `copylocks` covers `sync` types; nothing covers `strings.Builder`, which panics at run time when a non-zero builder is used through its copy.

## Inspect
`func (c *Config) Settings() map[string]string { return c.values }` — the finding is caller mutation authority over validated state, not the existence of a map return.

## Reject
- "Map iteration is random, sort it" — sort where the order reaches an observer, not everywhere.
- "Do not expose maps" — a map is a fine API type; the question is whether it aliases state the type protects.

## Reopen
- Show the mutation path: who holds the second reference, and when they can write through it.
- Treat aliasing intended as ownership transfer as a documentation finding rather than a clone finding.
- Leave concurrent access, cache invalidation, and retention benchmarks to their own lanes.

## Prove
- Mutate the returned value and assert internal state is unchanged; mutate the caller's input after a setter and assert stored state is isolated.
- Assert deterministic bytes for signatures, hashes, and serialized maps.
- Prove clone depth through a nested value, not through the top-level length.
