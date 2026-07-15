# Reference Selector

State the expected behavior change before loading; use examples as rubrics, not copy-ready decisions.

| Pressure | Load | Required effect |
| --- | --- | --- |
| Resource/operation shape, URI ownership, lifecycle, fields, nullability, time, money, or enums | [resource-representations-and-lifecycle.md](resource-representations-and-lifecycle.md) | Model the observable resource instead of mirroring implementation. |
| Method, `201`/`202`/`204`, `Location`, validators, or negotiation | [http-method-status-semantics.md](http-method-status-semantics.md) | Select standards-consistent method/status behavior. |
| Problem Details, validation, auth/concealment, field errors, sanitization, or status mapping | [problem-details-errors.md](problem-details-errors.md) | Produce one stable caller-actionable error profile. |
| Pagination, filters, sorting, sparse fields, counts, links, or multi-item results | [pagination-filtering-sorting.md](pagination-filtering-sorting.md) | Define deterministic bounded collection behavior. |
| Idempotency keys, timeout recovery, validators, or `409`/`412`/`428` | [idempotency-preconditions-retries.md](idempotency-preconditions-retries.md) | Make acceptance, concurrency, and replay safe. |
| Long-running work, bulk, uploads, callbacks, or webhooks | [async-operations-and-webhooks.md](async-operations-and-webhooks.md) | Give clients durable completion, failure, dedup, and recovery semantics. |
| Decoding, limits, actor/tenant binding, correlation, rate limits, authoritative reads, or freshness | [boundary-validation-and-freshness.md](boundary-validation-and-freshness.md) | Specify boundary and reconciliation behavior without choosing middleware/cache mechanics. |
| Status, error, enum, pagination, nullability, version, coexistence, or sunset changes | [compatibility-and-versioning.md](compatibility-and-versioning.md) | Classify migration impact and removal conditions. |
