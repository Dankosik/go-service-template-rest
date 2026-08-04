# Gates And Policy Ownership

## Behavior Change Thesis

When loaded for idiom or cleanup pressure, this file keeps the diff on its
accepted criterion instead of absorbing mechanical modernization the gates
already own, and makes a stdlib swap the modernizer suggests get checked against
the policy the replaced code carried before it is accepted.

## When To Load

Load this when adjacent code looks modernizable, when a local helper looks
replaceable by `slices`, `maps`, or `cmp`, or when deciding how much of a
mechanical concern still needs proof in this change.

## Decision Rubric

Both modules pin `go 1.26.5`, so every version gate below 1.26 is already
satisfied: `slices`, `maps`, `min`/`max`, `clear`, `cmp.Or`, `t.Context`,
`sync.WaitGroup.Go`, `testing/synctest`, `errors.AsType`, and `t.ArtifactDir`
are all available without checking. `errors.AsType` is already the repository
idiom — see `internal/infra/http/router.go` and
`internal/infra/httpclient/client.go`.

These concerns are gate-owned. Leaving them to the gate is correct; hand-applying
them across files the criterion did not touch widens the diff without adding
proof:

| Concern | Owner |
| --- | --- |
| stdlib modernization (`slices.Contains`, `maps.Clone`, `min`/`max`, `any`, `t.Context`) | `modernize` linter, and `make modernize-check` (`go fix -diff ./...`) |
| redundant loop-variable copies | `copyloopvar` |
| `errors.Is`/`errors.As` instead of `==` or a bare type assertion | `errorlint` |
| an error crossing a boundary unwrapped | `wrapcheck` |
| `rows.Close`, `rows.Err`, `resp.Body.Close` | `sqlclosecheck`, `rowserrcheck`, `bodyclose` |
| context stored in a struct, dropped, or fattened | `containedctx`, `contextcheck`, `fatcontext`, `noctx` |
| a value receiver copying a lock | `govet` copylocks, `recvcheck` |
| `t.Helper`, `t.TempDir`, `t.Context` in tests | `thelper`, `usetesting` |

- A gate suggestion is a default, not a decision. Accept a swap only when the
  replaced code carried no policy the stdlib call drops: `cmp.Or(input.Limit,
  defaultLimit)` is wrong wherever `0` means "no limit" rather than "unset", and
  `maps.Clone` and `slices.Clone` are shallow, so nested slices, maps, and
  pointers still alias every caller after the swap.
- Cleanup is scoped to what this change replaced: the superseded function, its
  wiring, its tests, its config keys and env defaults, its generated output, and
  the docs and comments that named it. Tidying anything else is a separate change.

## Validation Shape

- `make lint` is the mechanical gate; do not re-argue in the diff what it decides.
- `make modernize-check` reports modernizations as a diff without applying them.
- When a swap replaced a policy-carrying helper, add the test for the policy —
  nil versus empty, zero-value fallback, order, or shallow versus deep copy —
  before accepting it.
