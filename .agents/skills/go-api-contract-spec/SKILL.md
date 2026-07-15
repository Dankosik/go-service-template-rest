---
name: go-api-contract-spec
description: "Use when client-visible REST behavior needs definition or revision before coding; Own resource models, HTTP semantics, representations, errors, pagination, idempotency, async behavior, and compatibility; Skip when the primary decision is chi topology, system boundaries, data architecture, or implementation."
---

# Go API Contract Spec

Load the [shared specialist contract](../specialist-contract.md), then apply this wire-contract boundary.

## Outcome And Boundary

Define what clients send, observe, retry, reconcile, and migrate. Own resources and URI semantics; endpoint methods and statuses; representations and errors; collection behavior; idempotency, preconditions, and retry visibility; async operations, bulk results, uploads, and webhooks; consistency and freshness disclosure; compatibility; and OpenAPI/generated-contract authority.

Do not choose chi router composition, middleware placement, service topology, physical data design, package/file ownership, worker mechanics, or distributed orchestration. Inspect the client problem, consumers and trust boundary, current public contract and generator authority, compatibility commitments, accepted architecture/data/security policy, consistency expectations, and resource owner. Treat missing wire-visible facts as assumptions or blockers, never handler guesses.

## Contract Core

1. Model collection, item, sub-resource, or operation resources from client-visible lifecycle, not handlers or rows; separate accepted, persisted, observable, and terminal moments when they differ.
2. Make methods, success and negative statuses, `Location`, validators, preconditions, content negotiation, and retry behavior agree across matrices, schemas, and examples.
3. Define required/read-only/write-only fields; omitted, `null`, and empty shapes; identifiers, time, exact values, enum evolution, normalization, unknown fields, and limits deterministically.
4. Use the established error profile or default to sanitized `application/problem+json`; distinguish correction, auth/concealment, conflict/precondition, overload/dependency, and accepted-later-failed outcomes.
5. Classify retry safety. For effectful writes, define idempotency-key scope, TTL, replay, same-key/different-payload behavior, timeout recovery, and optimistic concurrency where writes race.
6. Use `202` only with durable completion and recovery visibility. Define operation/bulk lifecycle, webhook signing/dedup/reordering, authoritative reads, projection lag, freshness, and reconciliation behavior when triggered.
7. Preserve established versioning or default the major version to the URI. Classify breaking/non-breaking changes, coexistence, deprecation/sunset, generated-source impact, and removal proof.

Default to JSON over HTTP, resource-oriented contracts, cursor pagination for mutable or large collections, and the smallest surface that solves the caller problem; established approved contracts and concrete client needs override defaults.

## Symptom-Driven References

State the expected behavior change before loading; use examples as rubrics, not copy-ready decisions.

| Pressure | Load | Required effect |
| --- | --- | --- |
| Resource/operation shape, URI ownership, lifecycle, fields, nullability, time, money, or enums | [resource-representations-and-lifecycle.md](references/resource-representations-and-lifecycle.md) | Model the observable resource instead of mirroring implementation. |
| Method, `201`/`202`/`204`, `Location`, validators, or negotiation | [http-method-status-semantics.md](references/http-method-status-semantics.md) | Select standards-consistent method/status behavior. |
| Problem Details, validation, auth/concealment, field errors, sanitization, or status mapping | [problem-details-errors.md](references/problem-details-errors.md) | Produce one stable caller-actionable error profile. |
| Pagination, filters, sorting, sparse fields, counts, links, or multi-item results | [pagination-filtering-sorting.md](references/pagination-filtering-sorting.md) | Define deterministic bounded collection behavior. |
| Idempotency keys, timeout recovery, validators, or `409`/`412`/`428` | [idempotency-preconditions-retries.md](references/idempotency-preconditions-retries.md) | Make acceptance, concurrency, and replay safe. |
| Long-running work, bulk, uploads, callbacks, or webhooks | [async-operations-and-webhooks.md](references/async-operations-and-webhooks.md) | Give clients durable completion, failure, dedup, and recovery semantics. |
| Decoding, limits, actor/tenant binding, correlation, rate limits, authoritative reads, or freshness | [boundary-validation-and-freshness.md](references/boundary-validation-and-freshness.md) | Specify boundary and reconciliation behavior without choosing middleware/cache mechanics. |
| Status, error, enum, pagination, nullability, version, coexistence, or sunset changes | [compatibility-and-versioning.md](references/compatibility-and-versioning.md) | Classify migration impact and removal conditions. |

## Return And Stop

Return the governing contract/source, affected consumers, resource and URI ownership, method/status/request/response/error matrix, triggered collection/async/webhook/freshness rules, retry/precondition semantics, compatibility class, generated outputs, acceptance proof, assumptions, and reopen conditions. Decide whether dense detail needs a focused `design/contracts/` artifact, but do not claim artifacts or generated files were propagated.

Stop on unresolved audience/resource ownership, trust or identity/tenant source, business lifecycle, consistency, retry policy, compatibility window, durable completion, or data truth; name its owner. Reject success-with-error payloads, fake `202`, retry-unsafe effects without recovery, generated-output authority drift, and wire changes mislabeled as implementation detail.
