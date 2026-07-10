---
name: api-contract-designer-spec
description: "Design API-contract-first specifications for Go services. Use when planning or revising client-visible REST API behavior before coding and you need explicit resource modeling, HTTP method/status semantics, request/response/error contracts, pagination/filter semantics, idempotency/retry/concurrency rules, async behavior, consistency disclosure, and compatibility-safe evolution. Skip when the task is local code implementation, service decomposition, chi router topology, SQL schema/migration design, or low-level observability/security operations tuning."
---

# API Contract Designer Spec

## Trigger And Scope

Use this skill before coding when a Go service needs a new or changed client-visible REST contract: resource/URI model, methods and statuses, representations and errors, collections, retries and preconditions, async/bulk/upload/webhook behavior, freshness, limits, or compatibility evolution.

Own wire-visible semantics, not chi topology, handler/repository code, SQL schema, service decomposition, worker runtime, distributed orchestration, or low-level security/observability operations. Hand those questions to their owners once the contract consequence is explicit.

## Approved Input And Decision Boundary

Inspect the client problem, affected callers and trust boundary, current public contract and OpenAPI/generator authority, compatibility commitments, consistency and retry expectations, owning resource/service, and any approved architecture/data/security decisions. Treat missing client-visible facts as labeled assumptions or blockers, never handler guesses.

When used in system/integration design, decide whether contract detail belongs in the compact design or needs a focused `design/contracts/` artifact; if required client or compatibility evidence is missing, return the blocker and reopen owner. Name the runtime source of truth and generated consumers. This skill recommends the contract decision and proof carrier but does not propagate artifacts or claim generated files were changed.

## Contract Invariants And Defaults

1. **The API is a compatibility contract.** Define what clients send, observe, retry, reconcile, and migrate; do not let implementation or OpenAPI generation invent semantics.
2. **Resources and lifecycle lead handler shape.** Prefer collection/item, sub-resource, or operation resources over action-RPC endpoints; expose accepted, persisted, observable, and terminal moments separately when they differ.
3. **HTTP semantics stay precise.** Methods, success statuses, `Location`, validators, preconditions, content negotiation, and negative statuses must agree across endpoint matrices, schemas, examples, and retry behavior.
4. **Representations are deterministic.** Define required/read-only/write-only fields, omitted versus `null` versus empty, identifiers, time, exact money/precision, enum evolution, normalization, unknown fields, and boundary limits.
5. **Errors are stable and sanitized.** Default to one `application/problem+json` profile unless an established API profile wins; distinguish caller correction, auth/concealment, conflict/precondition, overload/dependency failure, and accepted-later-failed work.
6. **Retries cannot duplicate business effects.** Classify every operation by retry safety; define idempotency key scope/TTL/replay/payload-mismatch behavior and optimistic concurrency where writes can race.
7. **Async and eventual behavior is honest.** Use `202` only with durable recovery/reporting responsibility; define operation or bulk result lifecycle, webhook dedup/reordering, authoritative read path, freshness, and timeout-recovery behavior.
8. **Evolution is explicit.** Preserve the existing versioning policy; otherwise default the major version to the URI. Classify breaking and non-breaking changes, coexistence, deprecation/sunset, and removal proof instead of hiding compatibility work.

Default to JSON over HTTP, resource-oriented contracts, cursor pagination for mutable or large collections, and the smallest surface that solves the stated caller problem. Defaults yield to an established approved contract or concrete client need.

## Symptom-Driven Reference Selector

Load at most one reference by default. Load more only when independent contract pressures exist. State the expected behavior change before loading; examples are rubrics, not copy-ready decisions.

| Symptom or decision pressure | Load | Behavior change |
| --- | --- | --- |
| Resource versus operation shape, URI ownership, representation fields, lifecycle states, nullability, timestamps, money, or enum behavior is unclear. | [resource-representations-and-lifecycle.md](references/resource-representations-and-lifecycle.md) | Model the client-visible resource and observable lifecycle instead of mirroring handlers or database rows. |
| `GET`/`POST`/`PUT`/`PATCH`/`DELETE`, `201`/`202`/`204`, `Location`, validators, or content-negotiation statuses are disputed. | [http-method-status-semantics.md](references/http-method-status-semantics.md) | Choose standards-consistent method/status semantics and make edge behavior explicit. |
| Problem Details, validation errors, auth/concealment, field errors, sanitized negative paths, or status mapping is unclear. | [problem-details-errors.md](references/problem-details-errors.md) | Produce one stable caller-actionable error profile instead of ad hoc or leaked failures. |
| Cursor/offset pagination, filters, sorting, sparse fields, counts, links, or multi-item collection results are changing. | [pagination-filtering-sorting.md](references/pagination-filtering-sorting.md) | Define deterministic bounded collection semantics instead of implementation-shaped querying. |
| Write retries, `Idempotency-Key`, timeout recovery, `If-Match`, `If-None-Match`, `409`/`412`/`428`, or replay results are live questions. | [idempotency-preconditions-retries.md](references/idempotency-preconditions-retries.md) | Make acceptance and replay safe instead of treating network retries as transport-only behavior. |
| Long-running work, `202`, operations, bulk work, uploads, callbacks, or webhooks are in scope. | [async-operations-and-webhooks.md](references/async-operations-and-webhooks.md) | Give clients a durable completion, failure, dedup, and recovery model instead of fake synchronous success. |
| Strict decoding, limits, tenant/actor binding, correlation, rate limits, authoritative reads, or freshness/read-after-write is unclear. | [boundary-validation-and-freshness.md](references/boundary-validation-and-freshness.md) | Specify the boundary pipeline and reconciliation path instead of leaving correctness to middleware or cache behavior. |
| Status/error/enum/pagination/nullability/default behavior changes, versioning, coexistence, deprecation, or sunset is involved. | [compatibility-and-versioning.md](references/compatibility-and-versioning.md) | Classify migration impact and removal conditions instead of calling a wire change “internal.” |

## Required Evidence And Deliverable

For each material contract decision, record the client problem, affected consumers, governing current contract/source, whether a real live fork exists, selected option, rejected viable options when applicable, observable request/success/error/retry/freshness behavior, compatibility class, generated-source impact, testable acceptance boundaries, assumptions, and reopen trigger.

Return a compact contract packet with:

- resource/operation and URI ownership;
- method/status/request/response/error matrix;
- retry, idempotency, precondition, and concurrency semantics;
- collection, async, webhook, and freshness rules only when triggered;
- compatibility/deprecation and source-of-truth/generated-output consequences;
- domain handoffs only where another owner must decide now; otherwise state the constraint or `no new decision required`.

## Success, Escalation, And Stop Conditions

Success means OpenAPI, implementation, tests, callers, and unavoidable migration can converge on one explicit target-state wire contract without inventing method, status, error, retry, lifecycle, freshness, or compatibility semantics.

Stop or escalate when client audience/resource ownership, trust boundary, consistency, retry policy, identity/tenant source, compatibility window, distributed completion, or data truth is unresolved; when several contract surfaces contradict each other; or when the smallest safe answer requires architecture, routing, security, data/cache, domain, or distributed decisions outside this lane. Reject success-with-error payloads, fake `202`, retry-unsafe writes without recovery semantics, generated-output authority drift, and compatibility changes mislabeled as implementation detail.
