# Payload Serialization And Streaming Budgets

## Behavior Change Thesis
When loaded for symptom "payload shape, JSON work, response size, streaming, flushing, or large-body behavior dominates," this file makes the model bound representation and streaming semantics instead of likely mistake optimizing serialization around unbounded payloads or choosing streaming without partial-output semantics.

## When To Load
Load when the performance spec must decide payload size limits, JSON encoding/decoding budgets, response body size telemetry, streaming versus buffered responses, flush behavior, or large request/response shape.

## Decision Rubric
- Choose smaller representation when unneeded fields, nested resources, or unbounded arrays dominate latency and bytes.
- Choose pagination or chunking when the API can expose bounded slices without a streaming contract.
- Choose streaming response only when clients can consume incremental data and the contract defines ordering, partial failure, flush cadence, timeout, and cancellation behavior.
- Choose buffered response when all-or-nothing semantics, small payloads, or error handling make streaming unnecessary.
- Choose precomputed or delayed JSON segments only when correctness and invalidation rules are explicit.
- Choose alternative encoding only when API compatibility, content negotiation, and client migration are part of the spec.

## Imitate
- Report endpoint rejects unbounded `include=items`, selects cursor pagination with `page_size <= 200`, response body `<= 256 KiB` for the default shape, and a separate export operation for larger results. Copy the bound-default-plus-export split.
- JSON-lines export defines row order, first-byte latency target, per-row encoding budget, flush cadence, cancellation behavior, and failure behavior after partial output. Copy the partial-output contract.
- Decoder-heavy ingest uses request body size buckets and streaming decode only after max document size, unknown-field policy, and validation failure behavior are defined. Copy the validation-before-streaming rule.

## Reject
- Switching to streaming to "make it faster" while clients require all-or-nothing JSON array semantics.
- Optimizing JSON encoding while optional nested collections are included by default and no byte-size ceiling exists.
- Relying on `Flush` as a guaranteed client-observed boundary through every proxy and protocol path.
- Adding a custom binary response without API compatibility, content negotiation, and client migration policy.

## Agent Traps
- Benchmarking tiny payloads while production payloads are large, skewed, or field-rich.
- Treating serialization as the bottleneck before bounding response shape and pagination.
- Forgetting client cancellation and timeout behavior for large responses.
- Assuming streaming makes memory safe without defining buffering, backpressure, and partial failure.

## Validation Shape
Use proof obligations that preserve payload buckets and client-visible behavior:

```bash
go test -run='^$' -bench='BenchmarkEncodeReport/(page50|page200|export10k)$' -benchmem -count=20 ./internal/reports > encode.txt
benchstat encode.txt
go test -run='^$' -bench='BenchmarkDecodeIngest/(small|large|invalid)$' -benchmem -count=20 ./internal/ingest > decode.txt
benchstat decode.txt
go test -run='^$' -bench='BenchmarkStreamExport/export10k$' -trace trace.out ./internal/reports
go tool trace trace.out
```

For runtime validation, require metrics for request duration, request body size, response body size, timeout/cancellation count, encode/decode failures, and client-visible partial-stream failure count when streaming is used.
