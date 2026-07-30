# Operability And Proof

Use this reference when transport limits, telemetry, or proof scope can make a
gRPC production claim true or false.

## Decision

- Give connection, concurrent-RPC, per-connection-stream, received metadata,
  receive-message, send-message, and application-aggregate bounds explicit
  owners. Derive operational values from accepted workloads; a finite safety
  ceiling is not a throughput claim.
- Use grpc-go/OpenTelemetry `StatsHandler` for protocol telemetry when selected.
  Keep repository interceptors focused on final sanitized status, correlation,
  admission, and bounded service/method identity.
- Match proof to the boundary:
  1. configuration/unit proof covers validation and option construction;
  2. `bufconn` covers handler/interceptor behavior without TCP, TLS, or HTTP/2;
  3. loopback TCP covers real transport, credentials, metadata, messages, and
     stream behavior;
  4. production-process proof covers configuration, listener wiring, health,
     admission, and signal shutdown;
  5. deployed-route proof covers only the exact ingress, proxy, TLS, and trailer
     path exercised.
- Keep public native-gRPC support outside the claim until unary and streaming
  status/trailer behavior passes through the actual deployed route.

## Review

Report zero-value or unlimited bounds accepted accidentally, limit tests that
only validate configuration, `bufconn` presented as listener/TLS proof,
telemetry wiring without recording-provider assertions, peer-controlled metric
attributes, logs using pre-sanitization status, process tests that never call a
user RPC, or deployment guidance presented as live ingress evidence.

## Proof

For each changed limit, cross it over a real transport and assert the exact
failure boundary plus handler-entry oracle. Record unary and streaming spans
and metrics with bounded attributes. Exercise request correlation, final
status, health/log policy, process startup/shutdown, and TLS hostname mismatch.
Run deployed proof only when the completion claim includes that route.

Vendor authority:
[gRPC status codes](https://grpc.io/docs/guides/status-codes/),
[OpenTelemetry grpc-go instrumentation](https://pkg.go.dev/go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc),
[Health checking](https://grpc.io/docs/guides/health-checking/), and
[Go generated streaming behavior](https://grpc.io/docs/languages/go/generated-code/).
