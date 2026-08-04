# Reference Selector

Each row names a pressure where this repository or a current specification
overrides the obvious answer. State the expected behavior change before loading.

Resource shape, method choice, status selection, pagination mechanics, and
request validation ordering have no reference here: decide them against the
current contract in `api/openapi/service.yaml` and the router that serves it.
Adding a reference back requires a decision it would change.

| Pressure | Load | Required effect |
| --- | --- | --- |
| A new, moved, or removed error response; a domain failure needing a client answer; a `Problem` member that does not exist yet | [error-contract-profile.md](error-contract-profile.md) | Answer from the `internal/problem` catalog and the closed `Problem` schema instead of an invented problem type. |
| A change to a published operation: status, error `code`, enum, default, nullability, pagination behavior, consistency, deprecation, or removal | [compatibility-and-versioning.md](compatibility-and-versioning.md) | Hand-classify what the schema differ cannot see, instead of reading a green gate as proof. |
| A retryable `POST`/`PATCH`, a timeout hiding whether a mutation happened, or duplicate work worth a key | [idempotency-and-replay.md](idempotency-and-replay.md) | Publish key scope, TTL, mismatch answer, and reservation rule as one clause. |
| `202 Accepted`, work outliving its request, or events this service publishes | [async-operation-contracts.md](async-operation-contracts.md) | Place acceptance at a durable commit and give clients one recovery path. |
