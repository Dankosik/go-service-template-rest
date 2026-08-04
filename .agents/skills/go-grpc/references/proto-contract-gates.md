# Proto Contract Gates

## When To Load

Load this when a `.proto` is added, changed, moved, or removed, or when a claim
depends on what the protobuf gates actually proved.

## Behavior Change Thesis

Without this file, a green protobuf gate is read as wire-compatibility proof.
`make proto-check` is `scripts/proto.sh check`: format-check, lint, and
generated-drift. It does not compare against a base, and `ci-local` inherits
that scope. The breaking comparison is a separate target that runs only from
`make pr-check`, which supplies `BASE_REF`. Worse in this repository:
`breaking_proto()` compares `api/proto` only, that directory does not exist
here, and the one checked-in schema lives under
`examples/grpc-reference-service/api/proto`. `BASE_REF=origin/main make
proto-breaking` prints `no owned protobuf contract; breaking check not
applicable` and exits 0 — while format, lint, and drift *do* cover that example
module through `module_roots()`. The reference schema has no breaking-change
gate at all.

## Decision Rubric

- For a schema outside root `api/proto`, the compatibility argument is made by
  hand: name the retired field numbers and names you reserved, the enum zero
  value, and which decoders still read the old shape. No gate will contradict a
  wrong answer.
- Once a service owns `api/proto`, `proto-breaking` applies buf's `FILE`
  category against `.git#ref=<BASE_REF>,subdir=api/proto`. A first publication
  reports `not applicable` — that result describes the base, so it does not
  carry forward as evidence for the next change.
- `check_schema_policy()` runs inside `make proto-lint`, ahead of buf, and is
  stricter than buf's own rules: Edition 2023 must carry
  `option features.(pb.go).api_level = API_OPAQUE;`, Edition 2024 is rejected
  until the cross-language contract decision is reopened, and proto2/proto3 is
  accepted only when a readable `BASE_REF` holds legacy syntax **at the same
  path**. Renaming or relocating a legacy file makes it a new contract and is
  rejected, and linting any legacy file without `BASE_REF` fails outright.
- Generated Go under `internal/gen/proto` is derived. `proto-drift` regenerates
  into place and diffs, so schema and generated output land in one commit or the
  gate fails.
- [docs/grpc.md](../../../../docs/grpc.md) owns how to author and register a
  service — schema layout, the Opaque API, bootstrap wiring, client
  construction, runtime interceptor order, health, drain, limits, and the
  focused-proof command set. Read it there rather than reconstructing it.

## Reject

- Reporting `breaking check not applicable` as compatibility proof. It means the
  comparison never ran.

## Validation Shape

`make proto-check` proves format, lint, schema policy, and generated-drift for
every module `module_roots()` finds. `BASE_REF=origin/main make proto-breaking`
proves wire compatibility only for a root-owned `api/proto`. Anything else needs
the hand argument above stated with the change.
