# Boundary Validation And Freshness

## Behavior Change Thesis

When loaded for boundary or freshness ambiguity, this file makes the contract define one strict request pipeline and one authoritative reconciliation path instead of leaving limits, identity, lag, or cache behavior to middleware and implementation guesses.

## Boundary Pipeline

- Apply transport/media/size limits, strict decode, normalization, semantic validation, authorization/tenant binding, and business execution in explicit order.
- Reject malformed input, unknown fields, trailing data, unsupported media, oversize payloads, and immutable/read-only writes consistently when the established contract is strict.
- Derive tenant and actor identity from validated security context, not arbitrary request fields or untrusted headers.
- Define request/trace correlation, `429` and `Retry-After`, overload behavior, and any public upload/file-type limits that callers must handle.
- Keep admin, debug, override, or internal controls off general public endpoints unless they are deliberately part of the contract.

## Freshness And Reconciliation

- Declare read-after-write, monotonic-read, snapshot/live-pagination, and cache/projection freshness behavior only to the level actually guaranteed.
- Name the authoritative read path for timeout recovery and correctness-critical reconciliation. Derived views, caches, search indexes, and projections disclose lag and never silently become write truth.
- Expose `as_of`, version, `last_updated_at`, operation ID, or another reconciliation token when eventual reads and webhooks can race.
- State cursor expiry and recovery, projection unavailability fallback, and what a client should do when a webhook is ahead of a read model.
- Treat a consistency-model change as client-visible compatibility work.

## Evidence And Rejection Tests

Require the current boundary owner, identity source, limit policy, authoritative store/read, lag expectation, fallback, and proof carrier. Reject claimed read-after-write without an owning mechanism, cache-dependent correctness without a staleness contract, arbitrary caller callback targets without trust controls, and rate/limit semantics that exist only in middleware code.
