---
name: go-grpc
description: "gRPC transport: Use for grpc-go, interceptors/status, streaming, credentials, Protobuf/Buf, limits, or shutdown. Own RPC behavior; Skip REST, policy, and domain meaning."
---

# Go gRPC

Every RPC is a **typed status path**: the client observes one status, details,
and metadata, and that mapping is policy.

`proto -> registration/interceptors -> deadlines/limits -> status -> stream lifecycle -> health/shutdown -> proof`

Load the [shared specialist contract](../specialist-contract.md). Reconstruct the
affected slice from canonical `.proto` and generated API through registration,
interceptors, handler, client-visible status and metadata, channel behavior,
health, shutdown, telemetry, and proof. Schema evolution uses Buf gates rather
than hand edits; deadlines propagate; each stream names its end and half-close
semantics.

Use [gRPC authority](../../../docs/grpc.md) for enabled capabilities, Opaque API,
registration, interceptor order, health/drain, client propagation, limits,
telemetry, deployment boundary, and proof commands. Load the [reference
selector](references/index.md) for a changed failure/status path or a `.proto`
compatibility claim.

For a **Decision**, cover every affected unary, streaming, standard-service,
client, lifecycle, and generated-contract path. For **Review**, account for
every affected RPC path and proof boundary in the shared finding envelope.

Hand business meaning, security, reliability, observability, concurrency,
topology, delivery, and implementation to their matching owners.
