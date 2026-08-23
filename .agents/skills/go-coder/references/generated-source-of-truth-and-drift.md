# Generated Source Of Truth And Drift

## Load When

Load this when the change touches an OpenAPI operation or schema, a SQL query or
migration, a `.proto` service, a generator config, or a drift-check failure.

## Decide

The phase already owns the policy — change the canonical source first, regenerate,
and remove superseded generated output. What it cannot tell you is which file is
canonical here and which command turns it into the artifact:

| Canonical source | Generated output | Regenerate | Prove |
| --- | --- | --- | --- |
| `api/openapi/service.yaml`, `internal/openapi/oapi-codegen.yaml` | `internal/openapi/openapi.gen.go` | `make openapi-generate` | `make openapi-check` |
| `internal/infra/postgres/queries/*.sql`, `migrations/*.sql`, `internal/infra/postgres/sqlc.yaml` | `internal/infra/postgres/sqlcgen/*` | `make sqlc-generate` | `make sqlc-check` |
| `api/proto/**`, `buf.yaml`, `buf.gen.yaml` | `internal/gen/proto/**` | `make proto-generate` | `make proto-check` |

- A `*-check` target snapshots the current output, **runs the generator in place**,
  and diffs. A failing drift check has therefore already rewritten your working
  tree: the fix is to review the change it made and keep it, not to run the
  generator again and not to revert it.
- `openapi-generate` is `go generate` over the OpenAPI packages, so the command
  that actually rebuilds the bindings is the `//go:generate` line in
  `internal/openapi/doc.go`. Generation options change there, not in the Makefile.
- `sqlc-generate` prints "no sqlc query sources" and exits successfully when
  `internal/infra/postgres/queries` holds no `.sql` file, so a clean run is not
  evidence that the source changed. It also sits inside the
  `profile:database-postgres` block and is stripped from a service generated
  without Postgres; `sqlc-check` deliberately stays outside that block, because
  generated output left behind without its sources is drift a profile-less
  service can still have. Confirm the target exists before naming it as proof.
- `oapi-codegen.yaml` sets `strict-server: true`, so a handler receives an
  already-decoded typed `…RequestObject`. Body decoding, unknown-field rejection,
  and required-field checks belong to the spec and its validator: that class of
  behavior change edits `service.yaml`, not Go.
- The proto row is empty in the shipped template: `api/proto/` must not exist
  before the first owned `.proto`, and `make proto-generate` reports "gRPC
  capability disabled" without the profile-owned Buf targets. The worked layout lives in
  `examples/grpc-reference-service/`, which carries its own `buf.yaml`.
- Keep regeneration scoped to the source you changed. Generated churn with no
  source change is a separate finding to report, not diff to absorb.

## Reject

Reject `make lint-all` as evidence about generated code. `.golangci.yml` excludes
`internal/openapi/openapi.gen.go` and `internal/infra/postgres/sqlcgen/**` by
path, and `exclusions.generated: strict` drops every file carrying the canonical
generated header. Linters never run on these files; only the drift check
constrains them.

## Prove

- Run the single `*-check` for the surface you regenerated before any broader gate.
- Inspect `git diff` over the generated path and keep only hunks that trace back
  to the source change you made.
- When a generated symbol disappears, prove in the same diff that its
  hand-written callers were migrated or removed.
