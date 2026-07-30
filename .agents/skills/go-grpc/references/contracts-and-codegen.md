# Contracts And Codegen

Use this reference when schema or generator policy can change the RPC wire
contract or the generated Go source API.

## Decision

- Keep owned `.proto` files canonical and generated Go derived. Change schema
  first, regenerate, and retain schema plus generated output atomically.
- Decide wire compatibility and generated-source compatibility separately.
  Never reuse field numbers; reserve removed numbers and names; keep persistence
  and protobuf representations independently owned.
- Select the Go API level from the schema or a file-specific migration. Proto2,
  proto3, and Edition 2023 default to the Open Struct API; Edition 2024+ defaults
  to Opaque. A generator-wide API override is valid only when every retained
  file and consumer is deliberately migrated.
- Pin Buf and generator identities. Make the ordinary non-mutating contract gate
  cover format, lint, compilation/generation drift, and breaking comparison
  against a readable accepted base.
- Treat an absent prior contract as not applicable only when the base is valid
  and truly contains no owned schema. An invalid or unreadable base is a failed
  proof input.

## Review

Report generated edits, schema/generated drift, a global API-level flag that
silently changes retained Go callers, accepted Editions rejected by tooling,
formatting omitted from CI, a breaking check that cannot distinguish no
contract from bad base, or wire-safe changes that break the generated Go API.

## Proof

Exercise canonical, unformatted, lint-invalid, malformed, stale-generated, and
breaking fixtures. Include each accepted Edition/API-level branch and one
retained proto2/proto3 contract whose generated API must remain unchanged.

Vendor authority:
[Go generated API levels](https://protobuf.dev/reference/go/go-generated-opaque/),
[Editions](https://protobuf.dev/editions/overview/),
[Buf format](https://buf.build/docs/reference/cli/buf/format/),
[Buf lint](https://buf.build/docs/reference/cli/buf/lint/),
[Buf generate](https://buf.build/docs/reference/cli/buf/generate/), and
[Buf breaking](https://buf.build/docs/breaking/).
