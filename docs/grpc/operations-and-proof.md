# Operations And Proof

Enable the capability during initialization:

```bash
make template-init \
  MODULE=github.com/acme/widgets \
  CODEOWNER=@acme/platform \
  DATABASE=none \
  GRPC=enabled
```

`GRPC=none` removes runtime/client packages, protobuf tooling, generated and
example surfaces, configuration, tests, and profile-owned dependencies.

The standard health service starts `NOT_SERVING`, becomes `SERVING` after
startup admission, follows subsequent cached-readiness transitions, and enters
terminal `NOT_SERVING` before drain. `Health/Check` is admission-exempt;
`Health/Watch` uses a separate finite budget. The OIDC profile leaves only
`Check` public.

Server and client protocol traces and metrics come from `otelgrpc` stats
handlers. Server telemetry accepts only registered business method names;
peer-controlled unknown paths and routine health polling are filtered. Raw
messages, metadata values, and handler error text are not telemetry fields.

Shutdown marks readiness and gRPC health unavailable, waits the configured
propagation delay, and drains HTTP and gRPC concurrently under the shared
process budget. gRPC tries `GracefulStop` and uses `Stop` when the budget
expires. Handlers must release work when their RPC context is canceled.

Focused proof:

```bash
go test -vet=off \
  ./internal/infra/grpc \
  ./internal/infra/grpcclient \
  ./cmd/service/internal/bootstrap \
  ./examples/grpc-reference-service/...

go test -vet=off -race \
  ./internal/infra/grpc \
  ./internal/infra/grpcclient \
  ./cmd/service/internal/bootstrap

make proto-check
```

The integration-tag process test proves real TCP/TLS health and coordinated
SIGTERM. It does not prove public ingress or production capacity.
