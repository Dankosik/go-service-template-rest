# Native gRPC

`GRPC=enabled` adds native grpc-go server and client support beside REST. It
uses a separate listener and the same feature services, readiness, telemetry,
and shutdown budget.

```text
REST :8080 ─┐
            ├─> internal/<feature>
gRPC :9091 ─┘
```

The feature author owns protobuf messages, generated service methods, domain
errors, and business stream/deadline behavior. The template owns transport
composition; a feature must not build interceptors, health, validation,
credentials, retry, load balancing, telemetry, limits, or shutdown.

| Task | Read |
| --- | --- |
| Schema, generation, compatibility | [Contract and generation](grpc/contract-and-generation.md) |
| Server implementation, registration, client use | [Runtime and streaming](grpc/runtime-and-streaming.md) |
| Plaintext, TLS, or mTLS | [Transport security](grpc/transport-security.md) |
| Health, readiness, lifecycle, profile, proof | [Operations and proof](grpc/operations-and-proof.md) |
