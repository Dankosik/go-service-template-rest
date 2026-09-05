# Untrusted Input And Interpreters

## Load When

Load this when caller- or provider-influenced data reaches a decoder, a SQL
statement, a template, a subprocess, or a filesystem path.

## Decide

- Where a duplicate member could change a security decision, decode with a
  strict decoder instead of `encoding/json`. `internal/infra/oidcjwt/decode.go`
  routes every header, claim, and JWKS decode through go-jose's `json` package
  for exactly that reason: it errors on a duplicate key, and
  `token_test.go` carries the duplicate-header and duplicate-payload cases.
  `encoding/json/v2` ships in the toolchain but is not enabled for this module,
  so v1 semantics are what apply. Reject unknown members explicitly where
  ignoring one would silently drop caller intent — and treat strict decoding as
  a parse rule, never as evidence the caller may set the field.
- SQL values bind as placeholders; identifiers cannot. A sort column, table, or
  operator chosen by a caller maps through a code-owned allowlist. Queries live
  in `internal/infra/postgres/queries` and generate into `sqlcgen` with
  `make sqlc-check` failing on drift, so a hand-built statement beside them is
  the thing that needs justifying.
- Confine an untrusted filename with `os.Root` or `os.OpenInRoot` (Go 1.24+),
  which resolve inside the root and refuse to leave it. `filepath.Clean`,
  `Join`, `IsLocal`, and `Localize` are lexical checks worth running before the
  open, and none of them is confinement.
- Prefer a Go API to a subprocess. Where one is genuinely required, hardcode the
  executable, pass operands as separate `exec.CommandContext` arguments, and let
  the request context bound it.
- Validate at the boundary that owns the value, before the write, dial, enqueue,
  or spawn it feeds.

## Reject

- Sanitizing inside business logic: by then the interpreter or the side effect
  has already run, and the boundary that should have decided admissibility is
  still ambiguous.
- `text/template` for anything a browser renders: it applies no contextual
  escaping. `html/template` does, and `template.HTML` opts back out of it.

## Prove

Table cases at the owning boundary for accepted, rejected, boundary-length, and
duplicate inputs, asserting that rejection happens before the side effect rather
than that a status was returned. `make gosec` and `make govulncheck` run
`govulncheck`, which find pattern and dependency classes; neither shows an
allowlist is complete.
