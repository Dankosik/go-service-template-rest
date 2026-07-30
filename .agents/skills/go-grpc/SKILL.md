---
name: go-grpc
description: "gRPC transport: Use for grpc-go composition, interceptors/status, streaming, credentials, Protobuf/Buf, limits, shutdown, or review. Own its path; Skip REST, policy, business meaning, or Go."
---

# Go gRPC

Load the [shared specialist contract](../specialist-contract.md). Reconstruct
the affected RPC slice from canonical `.proto` and generated API through
registration, interceptor and handler execution, status and metadata returned
to the client, channel behavior, health/lifecycle, telemetry, and proof. Treat
grpc-go and standard gRPC services as protocol authorities rather than ordinary
application handlers.

## Choose The Branch

- **Decision** — select when a gRPC transport contract is absent or changing.
  Load the [reference selector](references/index.md) for one result-changing
  pressure. Complete when shared Decision dispositions cover every
  unary, streaming, standard-service, client, lifecycle, and generated-contract
  path with its forced consequence and focused proof.
- **Review** — select when changed gRPC code, configuration, schema tooling, or
  wiring must conform to accepted policy. Load the selector for the
  violated contract. Complete when the shared finding envelope accounts for
  every affected RPC path and proof boundary; missing policy returns to the
  gRPC Decision branch.

Load one reference by default and another only for an independent pressure.
Hand business-visible RPC meaning to the owning specification/domain method,
trust policy to `go-security`, resilience policy to
`go-reliability`, telemetry policy to `go-observability`, concurrency mechanics
to `go-concurrency`, listener/ingress topology to `go-system-architecture`,
profiles and delivery gates to `go-delivery-platform`, and
implementation to `go-coder`.
