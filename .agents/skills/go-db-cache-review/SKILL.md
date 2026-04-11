---
name: go-db-cache-review
description: "Review Go code changes for SQL access discipline, transaction boundaries, context and resource safety, cache key correctness, invalidation behavior, and stampede or fallback risk."
---

# Go DB Cache Review

## Purpose
Protect changed data-access and cache paths from consistency, isolation, timeout, invalidation, and origin-protection defects.

## Specialist Stance
- Review DB and cache code as correctness surfaces, not performance decorations.
- Prioritize transaction scope, context propagation, cursor/resource cleanup, cache key dimensions, and invalidation timing.
- Treat stale, aliased, cross-tenant, and fail-open cache behavior as merge risk when callers can observe it.
- Hand off schema ownership, migration strategy, API semantics, and broad reliability design when local DB/cache review cannot own the fix.

## Scope
- review SQL query discipline and request-path round-trip amplification
- review transaction boundaries and partial-side-effect risk
- review DB and cache context propagation, timeout use, and resource cleanup
- review cache key isolation, versioning, and serialization safety
- review invalidation, update, TTL, and staleness behavior
- review stampede suppression, fallback behavior, and origin protection
- review test and validation signals for DB/cache-sensitive behavior

## Lazy Reference Loading
Keep this `SKILL.md` as the decision guide. References are compact rubrics and example banks, not exhaustive checklists or documentation dumps. Load at most one reference by default; load multiple only when the diff clearly spans independent decision pressures, such as a transaction-boundary defect plus a cache-stampede defect.

Choose references by symptom and expected behavior change:

| Symptom in the diff | Load | Behavior change |
| --- | --- | --- |
| Query construction, dynamic identifiers, value binding, query loops, `QueryContext` vs `QueryRowContext`, or cursor cleanup in the query path | `references/sql-query-and-resource-safety-review.md` | Choose a local bind/allowlist/batch/close-check finding instead of generic SQL advice, driver switching, or schema redesign. |
| Transaction starts, moves, retries, split writes, isolation options, commit handling, or cache work around commit boundaries | `references/transaction-boundary-review.md` | Choose the atomic DB boundary and post-commit cache decision instead of stretching a transaction across Redis or inventing outbox/saga policy in review. |
| Dropped caller context, operation timeout, missing `cancel`, prepared statement or reserved connection lifecycle, or row/transaction cleanup as the primary symptom | `references/context-timeout-and-rows-cleanup.md` | Choose caller-derived cancellation and explicit cleanup instead of `context.Background`, arbitrary global budgets, or treating `QueryContext` alone as sufficient. |
| Cache key construction, tenant/auth/locale/feature scoping, deterministic key material, cached payload versioning, or corrupt decode behavior | `references/cache-key-isolation-and-serialization.md` | Choose complete key dimensions and safe decode/version handling instead of "just hash it" or silently treating corrupt/aliased values as misses. |
| Write-driven invalidation, TTL behavior, negative caching, stale serving, cache-aside freshness, or Redis `SET` overwrites | `references/invalidation-ttl-and-staleness-review.md` | Choose exact freshness ownership and TTL/negative-cache correction instead of TTL-only handwaving, wildcard deletes, or caching transient failures as truth. |
| Hot cache misses, cache outage fallback, local `singleflight`, Redis locks, stale fallback, or origin DB protection | `references/stampede-fallback-and-origin-protection.md` | Choose bounded miss/fallback behavior and correctly scoped coalescing/locks instead of unbounded origin fallback or pretending process-local coalescing is distributed protection. |

Do not load a reference just because it mentions a keyword; load it when its examples would change the finding you write. Escalate instead of solving here when the smallest safe correction changes schema ownership, API-visible consistency, tenant security policy, distributed locking policy, or broad reliability policy.

## Boundaries
Do not:
- turn DB/cache review into a broad architecture rewrite
- treat performance tuning as the primary task before correctness and consistency are explicit
- accept cache behavior without a freshness or correctness contract
- absorb primary ownership of security, concurrency, or reliability issues when DB/cache is only the symptom surface

## Core Defaults
- Correctness comes before optimization.
- Treat cache as an accelerator, not the source of truth, unless an explicit contract says otherwise.
- Require propagated cancellation, explicit cleanup, and operation deadlines when the caller or package owns a budget.
- Require cache keys to encode every dimension needed for correctness and isolation.
- Prefer the smallest safe fix that restores consistency and predictable fallback behavior.

## Expertise

### Query Discipline
- Flag `N+1`, per-item query loops, avoidable round-trip amplification, and hidden full-scan risk in changed paths.
- Require parameterization for values and allowlisting for dynamic identifiers.
- Flag repeated identical reads in the same flow when they can be batched or cached safely.
- Treat hot-path query amplification as both correctness and operational risk when it can distort timeout behavior.

### Transaction Boundaries And Partial Side Effects
- Verify dependent read/write steps that must commit together stay in one explicit transaction boundary.
- Flag partial commit risk and transactions stretched across network or cross-service calls.
- Require retry to target the whole transaction block for approved transient classes only.
- Require idempotency protection when retried writes can duplicate effects.

### Context, Timeout, And Resource Safety
- Require request context propagation into DB and cache calls.
- Flag `context.Background()` in request paths unless ownership is explicit and safe.
- Require explicit time bounds for blocking calls on critical paths.
- Verify `rows.Close`, `rows.Err`, rollback discipline, and no leaked transactions or cursors.

### Cache Key Isolation And Serialization
- Require tenant, auth, locale, feature, or version dimensions when response shape depends on them.
- Flag cross-tenant or cross-scope key collisions as high-severity isolation defects.
- Require deterministic key construction and safe decode behavior on schema mismatch or corrupt cache entries.
- Reject wildcard key scans in request paths.

### Invalidation, TTL, And Staleness
- Verify every cached path has an explicit freshness owner: invalidation, update, TTL, or a deliberate hybrid.
- Flag TTL-only approaches when correctness requires write-driven invalidation.
- Verify negative-cache behavior does not convert transient dependency failure into business truth.
- Require stale windows and bypass behavior to remain explicit when contract-sensitive.

### Stampede, Degradation, And Origin Protection
- Require coalescing or equivalent suppression on hot miss paths where origin load matters.
- Flag cache outage behavior that can overwhelm the origin or DB.
- Verify fallback mode matches the intended read or write contract.
- Require degraded cache behavior to stay observable and bounded.

### Verification Signals
- Review whether changed behavior is testable across hit, miss, stale, error, and invalidation paths when relevant.
- For concurrency-sensitive cache wrappers, expect race evidence or an explicit evidence gap.
- For integration-sensitive DB behavior, expect a realistic validation path rather than unit-only confidence.

### Cross-Domain Handoffs
- Hand off benchmark and hot-path proof questions to `go-performance-review`.
- Hand off goroutine, lock, and `singleflight` lifecycle defects to `go-concurrency-review`.
- Hand off timeout, retry, fallback, and overload policy defects to `go-reliability-review`.
- Hand off tenant-isolation and sensitive-data defects to `go-security-review`.
- Hand off broader architectural drift to `go-design-review`.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the concrete DB or cache defect
- correctness, isolation, or availability impact
- the smallest safe correction
- a validation command when useful
- whether the issue is local code drift or needs design escalation

Severity is merge-risk based:
- `critical`: confirmed correctness, isolation, or stale-contract breach that makes merge unsafe
- `high`: strong evidence of significant DB/cache contract mismatch
- `medium`: bounded but meaningful DB/cache weakness
- `low`: local hardening or clarity improvement

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

Use this format for each finding:

```text
[severity] [go-db-cache-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

Use `Reference` for the relevant data contract, cache rule, or approved decision when one exists.

## Escalate When
Escalate when:
- safe correction changes data ownership, transaction strategy, or cache contract (`go-db-cache-spec` or `go-data-architect-spec`)
- API-visible staleness, idempotency, or error semantics must change (`api-contract-designer-spec`)
- the right fix requires a new fallback, retry, or overload policy (`go-reliability-spec`)
- tenant or sensitive-data handling needs a new security contract (`go-security-spec`)
- local repair exposes broader design drift (`go-design-spec`)
