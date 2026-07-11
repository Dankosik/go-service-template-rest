# System / Integration Design

Choose the runtime mechanism needed to satisfy the accepted behavior. Use a separate design artifact only when the mechanism is not already obvious from the spec and repository.

## Read When

- Implementation would otherwise decide contracts, source of truth, data flow, failure behavior, cross-service interaction, or rollout.
- Public API/events, persisted data, caches, security boundaries, concurrency/lifecycle, or deployment behavior changes.
- An existing design needs repair after review or new evidence.

## Inputs

- Ready spec and dispositioned accepted risks/downstream proof obligations.
- `docs/repo-architecture.md` when repository boundaries matter.
- Current provider contracts, OpenAPI/event/schema sources, generated-source owners, and relevant runtime code.
- Research that can change the mechanism.

## Outputs

Use `design/overview.md` or one focused file. Split contracts, data, sequence, or rollout only when that creates a useful review/ownership boundary.

The design answers the applicable questions:

- Which component owns the source of truth, and which surfaces are derived?
- What is the happy-path sequence and each material failure/partial-work branch?
- What are timeout, cancellation, retry/no-retry, idempotency, cleanup, recovery, and degraded-mode rules?
- What contract, schema, cache, consistency, retention, or mixed-version behavior changes?
- What security, tenant, secret, abuse, observability, and cardinality boundaries matter?
- What rollout/rollback order and proof are required?
- Which viable simpler/established alternatives were rejected, and why?

## Contract Rule

When a caller-visible API, generated contract, event, or material shared interface changes, decide the semantics before implementation:

- audience/owner and trust boundary;
- request/message, response/result, error/status, validation, and limits;
- retry, idempotency, concurrency, async, freshness, and compatibility behavior where relevant;
- canonical source and generated outputs;
- proof and migration/deprecation consequences.

When canonicalization, hashing, signing, or verification depends on a data shape, close it at byte level: exact schema, field order, requiredness/nullability, bounds, exact bytes covered by canonicalization, digest, or signature, and at least one deterministic non-secret golden vector. When keyed signing or verification applies, also define public trust-material lookup and rotation. A metamodel or prose field list is insufficient. Environment-owned keys or trust data follow the router's [implementation-input closure](../../spec-first-workflow.md#implementation-input-closure); do not persist production secrets or private keys in repository artifacts.

Design against an external platform or service requires current official contract evidence. When integration shape or operational fit is non-obvious, also consume credible real implementations or engineering writeups for proven patterns and failure modes. Do not infer current external behavior from model memory. `design/contracts/` is optional context; it never replaces the runtime OpenAPI/event/proto source.

## Fan-Out And Review

At phase entry, identify the materially affected domains: domain behavior, contract, data, security, reliability/distributed flow, observability/performance, and delivery/rollout. Resolve each affected domain locally with its matching skill when the reasoning is sequential or tightly coupled, or delegate one concrete bounded question to the matching specialist subagent with that skill when separate context, parallel evidence, or independence improves the result. Do not run unaffected lenses, and do not turn the number of affected domains into a required lane count.

Use specialist lanes only for live independent forks that can change the mechanism. Examples: sync vs async, source A vs B, fail-closed vs degraded, or expand/backfill/contract vs one-step migration.

For structured or orchestrated work, run [Technical Design Review](technical-design-review.md) after the system and Go-ownership decisions are complete. The owning root handles repair and fresh re-review in the same root session. Direct work uses independent design review only when the user or risk requires it.

## Stop Rule

Continue to Go ownership when implementation can proceed without inventing runtime behavior. Continue to test design or planning only after Go ownership is complete and the required technical-design review has returned `PASS`. Reopen specification or research when the missing fact changes accepted behavior, ownership policy, or proof feasibility.
