# Inter-Service Protocol Selection

## Behavior Change Thesis
When loaded for a new synchronous service-to-service contract, this file makes the model default to native gRPC and then price its Railway exposure, instead of the likely mistake "this repository already generates OpenAPI clients, so add a REST endpoint" — which leaves `internal/infra/grpcclient`'s bounded connections, correlation policies, and `make proto-breaking` unused while hand-rolling a second client, or ships a gRPC listener `railway.toml` never publishes.

## When To Load
Load when a new call crosses a service boundary synchronously, a consumer class changes, or an existing contract is proposed for migration between REST and gRPC. A queue, event, or durable workflow is a different decision.

## Decision Rubric
- Classify consumers from current consumers, repositories, and the deployment path — never from a service name or a private network. A contract is strictly internal only when every caller and callee is an organization-controlled workload, no browser or third party shares it, and every affected consumer can own a generated client.
- Default a new strictly-internal synchronous contract to native gRPC when the profile is present in the checkout. `internal/infra/grpc` owns server policy and lifecycle, `internal/infra/grpcclient` owns bounded shared connections, `api/proto/` is the contract source of truth, and `BASE_REF=<ref> make proto-breaking` is the compatibility gate.
- The gRPC profile is optional and stripped at initialization. If `internal/infra/grpcclient` is absent, choosing gRPC adopts the whole profile — protobuf tooling, generation drift checks, CI profile jobs — not just a listener. Price that against one more resource on a REST contract that already has `make openapi-breaking`.
- Railway publishes what `railway.toml` declares, and it declares the REST listener and `/health/ready` only. A native gRPC contract runs over Railway private networking with the neighbor's internal DNS and explicit port; a public gRPC endpoint needs end-to-end HTTP/2 trailers revalidated on the current platform, or TCP Proxy with application TLS and hostname verification. [docs/grpc.md](../../../../docs/grpc.md) owns that boundary; enabling the runtime proves no reachability.
- Keep public, browser, mobile, third-party, and webhook contracts on REST with `api/openapi/service.yaml` as the source of truth. Existing REST does not become a migration task because gRPC is the internal default.
- Select the outbound correlation policy per neighbor at its trust boundary: `PropagationNone`, `PropagationTraceContext`, or `PropagationTrustedService` in `internal/infra/grpcclient/propagation.go`. `PropagationNone` is the zero value and emits no remote correlation, so a trusted neighbor left at the default loses its trace join silently.
- Record the neighbor in the System Neighbors table in [repo-architecture.md](../../../../docs/repo-architecture.md): its contract source, how to read its runtime evidence, and the field joining it to this service's `X-Request-ID`. That table is repository-owned and template sync never mirrors it, so a new crossing that skips it leaves no inventory for the next diagnosis.
- After selection, `go-grpc` owns RPC schema, status, interceptor, and streaming contracts; `go-api-contract` owns REST representation and OpenAPI authority. Business behavior stays outside both transports.

## Reject
- Exposing both transports for one behavior without two current consumer classes. Two transports are two representations of the same state drifting under separate breaking-change gates.
- Treating a shared generated client as the boundary contract. The contract is the proto or the OpenAPI document; a client is derived from it and cannot carry a guarantee the contract does not state.

## Validation Shape
Name the consumer classification and its evidence, the selected protocol, the live alternative that lost and the reason that rejects it, the correlation policy chosen per neighbor, and the condition that reopens the choice — a change of consumer class, compatibility obligation, or deployment path. gRPC selection proves itself with `make proto-check` and `BASE_REF=<ref> make proto-breaking`; REST with `make openapi-check`.
