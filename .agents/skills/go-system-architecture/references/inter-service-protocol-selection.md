# Inter-Service Protocol Selection

Use this decision only after the interaction has been classified as a synchronous
service boundary. A queue, event, or durable workflow is a different architecture
decision, not a REST-versus-gRPC choice.

## Default

Apply this order:

1. Honor an explicit accepted protocol or compatibility requirement.
2. For a new strictly internal service-to-service contract, choose native gRPC
   when the user expressed no protocol priority and no current constraint below
   defeats it.
3. For a public, browser, mobile, third-party, webhook, or human-oriented HTTP
   contract, choose REST with an
   [OpenAPI HTTP API description](https://spec.openapis.org/oas/latest.html)
   unless stronger current evidence requires another protocol.

A contract is strictly internal only when every known caller and callee is an
organization-controlled service workload, no external or browser-facing consumer
shares the contract, the deployment path preserves native gRPC semantics, and
generated client ownership described by
[gRPC core concepts](https://grpc.io/docs/what-is-grpc/core-concepts/) can close
for every affected service. Derive that classification from current consumers,
repositories, and deployment evidence; do not infer it from a service name or
private network alone.

## Constraints That Defeat The Internal Default

Keep or choose REST/OpenAPI when an accepted compatibility requirement depends
on it; the platform cannot preserve the
[gRPC over HTTP/2 protocol](https://github.com/grpc/grpc/blob/master/doc/PROTOCOL-HTTP2.md),
including trailers and status semantics; HTTP caching, intermediary
compatibility, or direct human tooling is material; or an affected consumer
cannot own generated gRPC clients. Existing REST/OpenAPI does not become a
migration task merely because gRPC is the default for new internal contracts.

Do not add both transports for hypothetical reuse. Dual exposure requires
evidence of distinct current consumer classes, one canonical behavior owner, and
proof that transport-specific representations and failures cannot drift.

## Decision Closure

A complete decision records the consumer classification and evidence, selected
protocol, dominant reason the live alternative lost, affected contract authority,
required proof, and reopen condition. Reopen when the consumer class,
compatibility obligation, or deployment path changes.

After selection, load `go-grpc` for the RPC schema, status, interceptor, health,
streaming, and lifecycle contract, or `go-api-contract` for REST representation,
HTTP semantics, and OpenAPI authority. Keep business behavior owned outside both
transports.
