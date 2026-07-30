# Reference Selector

State which gRPC-specific decision or violated contract the selected reference
changes.

| Pressure | Load | Required effect |
| --- | --- | --- |
| `.proto` authority, Editions/API level, generated Go, Buf format/lint/generate/breaking, or source compatibility | [contracts-and-codegen.md](contracts-and-codegen.md) | Preserve one canonical schema owner and prove both wire and generated-source compatibility. |
| Unary/stream interceptors, status provenance, auth policy, recovery, metadata, health semantics, or registration | [server-protocol.md](server-protocol.md) | Separate protocol-, policy-, handler-, and unexpected-error ownership on every affected server path. |
| Target/resolver behavior, credentials, `grpc.NewClient`, connection ownership, retries, `WaitForReady`, or TLS identity | [client-channel.md](client-channel.md) | Make channel construction, connectivity, replay, and security ownership explicit. |
| Cardinalities, EOF/half-close, cancellation, concurrent read/write, flow control, or aggregate memory | [streaming.md](streaming.md) | Bound application work and prove the exact stream concurrency and termination contract. |
| Startup, readiness, standard health, admission, drain, `GracefulStop`, `Stop`, or shutdown budget | [lifecycle-and-health.md](lifecycle-and-health.md) | Define one observable lifecycle whose stop API returns within its accepted budget. |
| Message/metadata/connection limits, OpenTelemetry, cardinality, `bufconn`, loopback TCP, process, or ingress proof | [operability-and-proof.md](operability-and-proof.md) | Match each production claim to a boundary-level falsifier of equal scope. |
