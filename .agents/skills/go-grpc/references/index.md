# Reference Selector

Each row names a pressure where this repository's own wiring overrides the
obvious gRPC answer. State the expected behavior change before loading.

Server composition, interceptor order, streaming cardinalities and concurrency,
client channel construction and propagation, health and drain, transport limits,
and telemetry have no reference here. [docs/grpc.md](../../../../docs/grpc.md) owns
them with the current code, and `internal/infra/grpc` plus
`internal/infra/grpcclient` are the authorities it describes. Adding a reference
back requires a decision it would change.

| Pressure | Load | Required effect |
| --- | --- | --- |
| A new or changed RPC failure path, a status a caller says is wrong, or a claim about the code and detail the client observes | [status-and-error-mapping.md](status-and-error-mapping.md) | Publish the failure through the problem catalog and `DomainErrors()` instead of a handler `status.Error` that collapses to `INTERNAL`. |
| A `.proto` added, changed, moved, or removed, or a compatibility claim resting on the protobuf gates | [proto-contract-gates.md](proto-contract-gates.md) | Name what `proto-check` and `proto-breaking` actually compared, and hand-argue the rest. |
