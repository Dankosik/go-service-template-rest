# Contract And Generation

Load for protobuf schema, generated API, compatibility, or Buf.

## Add a service

### 1. Define the contract

Create the canonical schema under `api/proto/<organization>/<service>/v1`.
New schemas use Edition 2023 and select the Go Opaque API in the schema:

```proto
edition = "2023";

package acme.orders.v1;

import "google/protobuf/go_features.proto";

option features.(pb.go).api_level = API_OPAQUE;
option go_package = "github.com/acme/orders/internal/gen/proto/acme/orders/v1;ordersv1";

service OrdersService {
  // GetOrder returns one order by identifier.
  rpc GetOrder(GetOrderRequest) returns (GetOrderResponse);
  // WatchOrders streams matching order updates.
  rpc WatchOrders(WatchOrdersRequest) returns (stream WatchOrdersResponse);
  // ImportOrders imports a client-provided order sequence.
  rpc ImportOrders(stream ImportOrdersRequest) returns (ImportOrdersResponse);
  // SyncOrders synchronizes orders bidirectionally.
  rpc SyncOrders(stream SyncOrdersRequest) returns (stream SyncOrdersResponse);
}
```

The `.proto` file is the public authority. Generated Go under
`internal/gen/proto` is derived and must not be edited by hand. Keep messages
independent from database rows, never reuse field numbers, reserve deleted
numbers and names, and start enums with an `UNSPECIFIED` zero value.
The generator configuration does not override the Go API level. The current
policy accepts new Edition 2023 schemas only when they select `API_OPAQUE` and
rejects Edition 2024 until the cross-language contract decision is reopened.
Retained proto2/proto3 contracts keep their existing generated API until their
owner migrates them deliberately, but lint accepts one only when a readable
`BASE_REF` contains legacy syntax at the same path. A renamed or newly added
proto2/proto3 file is therefore a new contract and is rejected.

### 2. Generate and verify

```bash
make proto-format
make proto-format-check
make proto-generate
BASE_REF=origin/main make proto-check
BASE_REF=origin/main make proto-breaking
```

The pinned Buf v2 runner compiles without a local `protoc` installation or Buf
account. Official `protoc-gen-go` and `protoc-gen-go-grpc` plugins come from
the pinned tools module. `proto-check` performs a non-mutating format check,
runs `STANDARD` plus public-contract `COMMENTS` lint, and fails when generated
Go has drifted. `proto-breaking` requires a readable Git base, reports a first
publication as not applicable, and applies conservative `FILE` compatibility
rules. `BASE_REF` is optional for an Edition-only repository and required to
prove that any proto2/proto3 path is retained rather than newly introduced.

Commit schema and generated Go together. Add `buf.lock` only after
`buf.yaml` gains an external dependency; with no dependencies there is nothing
to lock.

### Editor feedback and manual calls

Buf includes an LSP server, so editor diagnostics, navigation, completion, and
formatting use the same pinned binary and repository policy as CI. Configure
the editor's LSP client to start this command from the repository root:

```bash
bash ./scripts/run-buf.sh lsp serve
```

Reflection stays disabled by default. For a local plaintext smoke call, pass
the checked-in schema to `buf curl` instead:

```bash
bash ./scripts/run-buf.sh curl \
  --schema . \
  --protocol grpc \
  --http2-prior-knowledge \
  --data '{"id":"order-123"}' \
  http://127.0.0.1:9091/acme.orders.v1.OrdersService/GetOrder
```

Use the deployment's TLS endpoint and trust configuration instead of
`--http2-prior-knowledge` outside an explicitly allowed plaintext boundary.
Generated clients and the real TCP integration suite remain the CI oracle;
`buf curl` is for development and operational smoke checks.

## Upstream references

- [Go Opaque API](https://protobuf.dev/reference/go/opaque-faq/) and
  [Protobuf compatibility practices](https://protobuf.dev/best-practices/dos-donts/)
- [Buf generation](https://buf.build/docs/generate/),
  [formatting](https://buf.build/docs/format/),
  [lint rules](https://buf.build/docs/lint/rules/),
  [editor/LSP integration](https://buf.build/docs/cli/editors-lsp/),
  [`buf curl`](https://buf.build/docs/curl/),
  [breaking rules](https://buf.build/docs/breaking/rules/), and
  [CLI installation guidance](https://buf.build/docs/cli/installation/)
