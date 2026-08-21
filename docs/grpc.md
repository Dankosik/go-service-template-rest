# Native gRPC

The `GRPC=enabled` initialization profile adds an optional native gRPC-Go
client/server capability beside the existing REST API. It does not replace
OpenAPI, add a gateway, or create a second business layer.

```text
REST :8080 ─────┐
                ├─> internal/<feature> ─> dependencies
gRPC :9091 ─────┘

metrics :9090 (private diagnostics)
```

REST and gRPC use separate listeners in one process. They share startup
admission, dependency readiness, request IDs, telemetry, shutdown budgeting,
and feature services. The native gRPC path keeps unary, server-streaming,
client-streaming, and bidirectional-streaming APIs available without routing
them through `net/http`.

## Select one leaf

| Changed pressure | Load |
| --- | --- |
| Protobuf schema, generated API, compatibility, or Buf | [Contract And Generation](grpc/contract-and-generation.md) |
| Registration, status mapping, client, or streaming behavior | [Runtime And Streaming](grpc/runtime-and-streaming.md) |
| Plaintext, TLS, mTLS, trust, or certificate rotation | [Transport Security](grpc/transport-security.md) |
| Profile initialization, health, readiness, drain, telemetry, deployment, or proof | [Operations And Proof](grpc/operations-and-proof.md) |

Load another leaf only for an independent changed pressure.
