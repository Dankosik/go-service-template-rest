---
name: go-db-cache-review
description: "Use when changed Go executes SQL or reads, writes, invalidates, or falls back around a cache; Own query discipline, transaction execution, DB resource safety, cache isolation, freshness, serialization, and origin protection; Skip when the primary defect is schema architecture, business policy, or broad concurrency or reliability policy."
---

# Go DB Cache Review

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Target And Boundary

Review changed SQL and cache paths for local correctness, isolation, cancellation, resource, freshness, fallback, and origin-protection defects. Treat cache as an accelerator rather than truth unless an accepted contract says otherwise.

Do not redesign schema or data ownership, invent business acceptance or API consistency, stretch local review into distributed coordination, or own broad concurrency/reliability policy. Escalate when the smallest safe correction needs one of those decisions.

## DB And Cache Defect Invariants

1. SQL values use bind arguments and dynamic identifiers use allowlists. Flag changed N+1 loops, repeated reads, round-trip amplification, or hidden full scans only with a concrete path and behavior-preserving local batch/query correction. Preserve `sql.ErrNoRows`, scan errors, and duplicate-row detection semantics.
2. DB/cache calls preserve caller context and an already owned operation budget. `Rows`, iteration errors, statements, reserved connections, cursors, and transactions have explicit close/error/end paths; `QueryRowContext` errors are handled at `Scan`.
3. Dependent DB operations that must commit together use one explicit transaction with rollback on every non-commit path. Keep external network/cache work outside the transaction, treat `Commit` failure as unknown outcome rather than success, and never infer rollback from a failed commit.
4. Retry only approved transient classes around the whole transaction boundary. Partial-statement retry is unsafe; retried writes require the accepted idempotency protection. Do not invent retry, outbox, saga, or public idempotency policy.
5. Queries and cache keys carry every tenant, auth, locale, feature, version, pagination, and other correctness dimension. Keys are deterministic and collision-safe; aliased payloads, payload versioning, marshal/decode errors, corrupt entries, and zero values cannot silently cross scopes or become successful data.
6. Every cache path has an accepted freshness owner: exact post-commit invalidation/update, TTL, or deliberate hybrid. Preserve TTL on overwrite, keep stale windows explicit, and cache only authoritative misses—not transient dependency failures—as negative truth.
7. Distinguish cache miss from cache failure. Hot misses and outage fallback are bounded against origin load; coalescing keys match cache isolation dimensions, process-local suppression is not presented as distributed protection, and expiring locks use safe ownership/release semantics. Degraded behavior remains observable and contract-aligned.

## Symptom-Driven References

Choose the reference whose examples change the local finding.

| Symptom | Load | Distinction preserved |
| --- | --- | --- |
| Dynamic SQL, binding, query loops, one-row/result handling, or cursor cleanup is primary. | [sql-query-and-resource-safety-review.md](references/sql-query-and-resource-safety-review.md) | Bind/allowlist/batch/close locally; do not switch drivers or redesign schema. |
| Transaction scope, split writes, retries, isolation, commit handling, or cache work around commit changed. | [transaction-boundary-review.md](references/transaction-boundary-review.md) | Restore atomic DB scope and post-commit cache order; do not invent distributed policy. |
| Caller context, timeout, cancel, statement/connection ownership, rows, or transaction cleanup is primary. | [context-timeout-and-rows-cleanup.md](references/context-timeout-and-rows-cleanup.md) | Preserve caller-derived cancellation and explicit cleanup; do not invent budgets. |
| Key dimensions, deterministic material, tenant/auth scope, payload version, or corrupt decode changed. | [cache-key-isolation-and-serialization.md](references/cache-key-isolation-and-serialization.md) | Restore isolation and safe serialization rather than hashing incomplete keys. |
| Write invalidation, TTL, negative caching, stale serving, cache-aside freshness, or overwrite changed. | [invalidation-ttl-and-staleness-review.md](references/invalidation-ttl-and-staleness-review.md) | Name exact freshness ownership; avoid TTL handwaving, wildcard deletes, and cached transient failures. |
| Hot misses, cache outage, `singleflight`, locks, stale fallback, or origin load changed. | [stampede-fallback-and-origin-protection.md](references/stampede-fallback-and-origin-protection.md) | Require bounded fallback and correctly scoped protection without overstating local coordination. |

## Findings, Evidence, And Escalation

Each finding names the concrete DB/cache defect, accepted data/cache contract, correctness/isolation/availability impact, smallest safe correction, and focused hit/miss/stale/error/invalidation or integration proof. For concurrency-sensitive wrappers request race evidence or state the gap; for DB semantics prefer a realistic integration path over unit-only confidence.

`critical` means a confirmed correctness, isolation, or freshness-contract breach that makes merge unsafe; `high` means strong evidence of a significant DB/cache mismatch. Cross-tenant cache exposure also requires a forced security handoff, without dropping the local key/query defect.

Stop and hand off changed data ownership or transaction/cache policy to `go-db-cache-spec` or `go-data-architecture-spec`; business acceptance to `go-domain-invariant-spec`; API-visible staleness, idempotency, or errors to `go-api-contract-spec`; fallback/retry/overload policy to `go-reliability-spec`; distributed locks/recovery to `go-distributed-spec`; new tenant or sensitive-data policy to `go-security-spec`; and package responsibility drift to `go-implementation-ownership-spec`. For accepted supporting contracts, hand benchmark/hot-path proof to `go-performance-review`, lock/goroutine lifecycle to `go-concurrency-review`, resilience enforcement to `go-reliability-review`, and tenant/sensitive-data enforcement to `go-security-review` without duplicating their findings.
