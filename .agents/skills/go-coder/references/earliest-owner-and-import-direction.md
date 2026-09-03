# Earliest Owner And Import Direction

## Load When

Load this when the change adds a file, moves code between packages, introduces a
helper or interface, or when the obvious edit site would need a new import.

## Decide

`depguard` in `.golangci.yml` fixes the direction. A feature package under
`internal/**` — anything outside `cmd/`, `internal/config`, `internal/infra`,
`internal/observability`, and `internal/openapi` — may not import:

| Denied from a feature package | Where that work belongs |
| --- | --- |
| `internal/openapi` | map generated types in `internal/infra/http` |
| `internal/infra` | wire the concrete adapter in bootstrap |
| `internal/config` | pass resolved values in from the composition root |
| `cmd/**` | composition depends on features, never the reverse |
| `github.com/jackc/pgx/v5` | `internal/infra/postgres` |

The rule matches `**/internal/**/*.go`, so it also binds `examples/*/internal/**`
and `_test.go` files: a test that reaches for an adapter is the same violation.
Narrower rules sit on top — `internal/config` may import only
`internal/observability/otelconfig`, `internal/health` may import no other
`internal` package, and `internal/infra/http` may not call
`internal/infra/postgres`.

- The earliest valid owner is the package that can already see what the change
  needs. When the symptom surfaces in a package that would need a denied import
  to fix it, the defect is upstream: its caller is passing too little, and the
  change belongs where the value is already known. Adding the import is not the
  smaller diff — it is a larger change to the architecture wearing a one-line
  disguise.
- Look for an existing seam before adding one. `internal/infra/http.RouterConfig`
  already carries `DomainErrors []failure.Mapper` for classifying the errors a
  generated operation returns, plus `Authenticate`, `RateLimit`, and
  `RateLimitKey` for the policies a template cannot guess. A new value on an
  existing seam beats a new package.
- Extract a helper when repeated behavior is stable policy that one package
  owns; keep one-use logic inline where the state change stays visible.
- Define an interface at the consumer that needs substitution. `ireturn`,
  `iface`, and `interfacebloat` are enabled, so a returned interface or an
  unused, identical, or oversized one fails lint rather than review.

## Prove

- Run `make lint` after any cross-package move; `depguard` and `iface` fail on
  direction before tests reach the behavior.
- Check the imports of the package you moved into: a correct boundary adds no
  cycle and no unrelated dependency.
- Run the tests of the old call sites and of the package that now owns the code.
