---
name: go-grpc
description: "gRPC transport: Use for grpc-go composition, interceptors/status, streaming, credentials, Protobuf/Buf, limits, or shutdown. Own its path; Skip REST, policy, business meaning, or Go."
---

# Go gRPC

Every RPC is a **typed status path**: whatever happens inside, the client observes exactly one status with its details and metadata — and that mapping is policy, not accident.

`proto contract -> registration and interceptor order -> deadlines and limits -> status mapping -> streaming lifecycle -> channel and health -> shutdown -> proof`

Treat grpc-go and standard gRPC services as protocol authorities rather than ordinary application handlers. The canonical `.proto` and its generated API are the source of truth, and schema evolution passes through Buf gates rather than hand edits. Deadlines propagate or the deepest hop invents its own budget; every stream names who ends it and what a half-closed peer means; interceptor order carries the same semantic weight as middleware order.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct the affected RPC slice from canonical `.proto` and generated API through registration, interceptor and handler execution, status and metadata returned to the client, channel behavior, health/lifecycle, telemetry, and proof.

## Choose The Branch

- **Decision** — select when a gRPC transport contract is absent or changing. Complete when shared Decision dispositions cover every unary, streaming, standard-service, client, lifecycle, and generated-contract path with its forced consequence and focused proof.
- **Review** — select when changed gRPC code, configuration, schema tooling, or wiring must conform to accepted policy. Complete when the shared finding envelope accounts for every affected RPC path and proof boundary.

Decide against this repository's own gRPC authority: [docs/grpc.md](../../../docs/grpc.md) states the enabled-capability surface, schema and Opaque-API policy, bootstrap registration seam, interceptor order, health and drain behavior, client construction and propagation tiers, transport limits, telemetry scope, the Railway boundary, and the focused-proof commands — with `internal/infra/grpc` and `internal/infra/grpcclient` as the code it describes. Load the [reference selector](references/index.md) when an RPC failure path or caller-observed status changes, or a `.proto` change rests its compatibility claim on the protobuf gates.

Hand business-visible RPC meaning to the owning specification/domain method, trust policy to `go-security`, resilience policy to `go-reliability`, telemetry policy to `go-observability`, concurrency mechanics to `go-concurrency`, listener/ingress topology to `go-system-architecture`, profiles and delivery gates to `go-delivery-platform`, and implementation to `go-coder`.
