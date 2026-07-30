# Streaming

Use this reference when an RPC stream can change concurrency, cancellation,
termination, or retained application state.

## Decision

- Name the exact unary, server-streaming, client-streaming, or bidirectional
  contract, including request/response ordering, EOF, half-close, and terminal
  status.
- For one stream, allow one reader and one writer concurrently. Serialize reads
  with reads and writes with writes.
- Bound application-retained items, bytes, queue depth, and work independently
  of per-message limits and HTTP/2 flow control. Choose `RESOURCE_EXHAUSTED` or
  another accepted terminal result at the application boundary.
- Propagate stream context cancellation to feature work and upstream RPCs.
  Long-lived handlers must observe cancellation and release owned work; grpc-go
  cannot interrupt arbitrary application code.
- State duration, idle, and replay policy only when the RPC contract needs
  them. A universal stream timeout or retry is not a transport default.

## Review

Report unbounded aggregation or queues, assumptions that flow control bounds
total memory, concurrent reads or concurrent writes, ignored stream context,
goroutines with no join/unblock path, missing half-close handling, a
client-stream response sent before valid EOF, or tests that execute bidi as
strict request/response pairs and never exercise concurrency.

## Proof

Exercise every affected cardinality with generated clients. For streams, prove
ordered messages, EOF/half-close, one concurrent reader plus one writer, active
cancellation, feature-work release, each item/byte bound, terminal status, and
race/liveness behavior. Use a parked cooperative handler to make cancellation
observable rather than relying on sleeps.

Vendor authority:
[Go generated streaming APIs and concurrency](https://grpc.io/docs/languages/go/generated-code/),
[Flow control](https://grpc.io/docs/guides/flow-control/), and
[Cancellation](https://grpc.io/docs/guides/cancellation/).
