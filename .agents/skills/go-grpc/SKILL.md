---
name: go-grpc
description: "gRPC typed status paths. Use when grpc-go composition, interceptors, status mapping, streaming, credentials, Protobuf compatibility, limits, health, or shutdown changes an RPC."
metadata:
  invocation: model
  kind: method
---

# Go gRPC

Every RPC is a **typed status path**: the client observes one status, details,
and metadata, and that mapping is policy.

`proto -> registration/interceptors -> deadlines/limits -> status -> stream lifecycle -> health/shutdown -> proof`

For a delegated Decision or Review, or when the active artifact requires its
result interface, load the
[shared specialist contract](../../contracts/specialist-contract.md).
For every changed RPC, build `RPCPath{method, proto, registration,
interceptors, deadline, limits, status, details, metadata, stream_end,
health_shutdown, proof}` from canonical `.proto` through the terminal client
observable. Schema evolution uses Buf gates rather than hand edits; deadlines
propagate; each stream names its end and half-close semantics.

Use [Native gRPC](../../../docs/grpc.md) to select the matching architecture
leaf. Load the [reference selector](references/index.md) for a changed
failure/status path or a `.proto` compatibility claim.

For a **Decision**, disposition every affected unary, streaming,
standard-service, client, lifecycle, and generated-contract path. For
**Review**, account for every affected RPC path and proof boundary. Complete
only when no alternate registration, unmapped failure, or unowned stream end
remains and the proof fails for the wrong status or lifecycle.
