# Proto Contract Gates

## Load When

Load this when a `.proto` is added, changed, moved, or removed, or when a claim
depends on what the protobuf gates actually proved.

## Decide

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
- [Native gRPC](../../../../docs/grpc.md) selects how to author and register a
  service. Load the matching leaf rather than reconstructing schema layout,
  generation, bootstrap wiring, client construction, runtime behavior, or proof.

## Reject

- Reporting `breaking check not applicable` as compatibility proof. It means the
  comparison never ran.

## Prove

`make proto-check` proves format, lint, schema policy, and generated-drift for
every module `module_roots()` finds. `BASE_REF=origin/main make proto-breaking`
proves wire compatibility only for a root-owned `api/proto`. Anything else needs
the hand argument above stated with the change.
