---
name: go-db-cache-spec
description: "Design DB-access-and-cache-first specifications for Go services in a spec-first workflow. Use when planning or revising SQL access discipline, query and transaction risk controls, cache strategy, staleness/fallback semantics, and DB/cache observability-test obligations before coding. Skip when the task is a local code fix, primary schema ownership/migration design, endpoint-only API contract work, CI/container setup, or low-level implementation tuning."
---

# Go DB Cache Spec

## Purpose
Create a clear, reviewable DB/cache specification package before implementation. Success means SQL access and cache decisions are explicit, defensible, measurable, and directly translatable into implementation and test work without hidden runtime-risk assumptions.

## Scope And Boundaries
In scope:
- define SQL access discipline for runtime paths (query shape, round-trip budget, N+1/chatty prevention)
- define transaction and retry/idempotency guardrails at specification level
- define timeout/context/pooling constraints and connection-budget assumptions for DB paths
- decide whether cache is justified based on measured bottleneck evidence
- choose cache topology and pattern with explicit trade-offs (`local`/`distributed`/`hybrid`, `cache-aside` default)
- define staleness and consistency contract per operation class
- define key safety requirements (tenant/scope/version), TTL/jitter/invalidation strategy, and stampede controls
- define failure behavior and degradation policy (fail-open/fallback/bypass/origin protection)
- define DB/cache observability and test obligations required for merge readiness
- produce DB/cache deliverables that remove "decide later" gaps

Out of scope:
- primary data ownership, logical schema modeling, DDL, and migration sequencing leadership
- service/module decomposition and global interaction-style architecture decisions
- endpoint-only REST payload/status/error design details
- full security architecture and authn/authz policy outside DB/cache surface
- SLI/SLO governance and incident process as primary domain
- CI/CD pipeline and container runtime hardening decisions
- low-level repository/cache client code implementation and micro-optimizations

## Hard Skills
### DB/Cache Specification Core Instructions

#### Mission
- Produce DB/cache decisions that remain correct under load, failure, and mixed-version rollout.
- Convert ambiguous runtime data-path behavior into explicit, reviewable contracts before coding starts.
- Ensure every major DB/cache decision is measurable, testable, and rollback-safe.

#### Default Posture
- Keep SQL access query-first and explicit; treat SQL text plus generated interfaces as the production contract.
- Treat cache as an accelerator, not source of truth, unless an explicit approved exception exists.
- Introduce cache only with measured bottleneck evidence and explicit correctness/staleness acceptance.
- Prefer fail-open read-cache behavior with bounded timeouts and origin-protection controls.
- Prefer compatibility-first evolution (`expand -> migrate/backfill -> contract`) over destructive-first changes.

#### SQL Access Discipline Competency
- Require explicit query-shape decisions per operation class:
  - expected round trips,
  - hot-path query budget,
  - JOIN/bulk-fetch strategy for related data.
- Require stable query naming and explicit column lists for production business paths.
- Reject `N+1` and chatty repository loops in specification.
- Require explicit dynamic-identifier allowlists for sort/filter identifier surfaces.
- Require deterministic pagination semantics and explicit choice:
  - keyset default for hot/deep lists,
  - offset only by explicit bounded-depth rationale.

#### Transaction, Retry, And Idempotency Competency
- Keep transaction ownership explicit at use-case boundary:
  - `Begin`,
  - `defer Rollback`,
  - `Commit` in same scope.
- Prohibit long transactions around network calls or cross-service I/O.
- Require retry policy to be explicit and bounded:
  - full-transaction retries only,
  - transient classes only (minimum `40001`, `40P01` when PostgreSQL-compatible),
  - bounded attempts with jitter.
- Require retried write paths to be idempotent (`ON CONFLICT`, idempotency key, or equivalent).
- Reject hidden distributed-ACID assumptions across service boundaries.

#### Context, Timeout, And Connection-Budget Competency
- Require end-to-end context propagation across handler -> service -> repository -> DB/cache calls.
- Require explicit DB/cache deadlines; reject implicit infinite-timeout calls.
- Use explicit operation timeout classes as starting defaults when unspecified:
  - DB point read/write: `1-2s`,
  - list/report-like DB calls: `2-5s`,
  - cache timeout strictly shorter than origin timeout.
- Require explicit pool-budget assumptions and connection-capacity math:
  - `(max_open_conns_per_instance * max_instances) <= 0.8 * available_db_connections`.
- Require resource-return safety in DB access contract (`rows.Close`, `rows.Err`, `QueryRow.Scan` semantics).

#### Cache Necessity And Topology Competency
- Require measured bottleneck evidence before approving cache (latency/cost/load impact).
- Reject cache introduction when:
  - strict consistency is required and no safe bypass exists,
  - key correctness dimensions cannot be encoded safely,
  - operational ownership/observability is missing.
- Select topology by constraints, not preference:
  - `local` for ultra-low latency with acceptable replica divergence,
  - `distributed` for fleet-wide coherence and shared hit ratio,
  - `hybrid` only with explicit L1/L2 coherence controls.
- Require topology-specific controls (memory bounds, eviction policy, timeout budget, coherence rules).

#### Cache Pattern, Consistency, And Invalidation Competency
- Default to `cache-aside`; any alternative pattern requires explicit rejection rationale for `cache-aside`.
- Require staleness/consistency contract per operation class:
  - strong paths bypass cache by default,
  - eventual paths define bounded staleness window.
- Require explicit invalidation source and ownership:
  - TTL-only,
  - write-triggered invalidation/update,
  - event-driven invalidation.
- Require TTL on every cache entry with jitter policy for medium/high-cardinality groups.
- Require mandatory stampede controls for hot/expensive keys:
  - request coalescing (`singleflight`-style or equivalent),
  - bounded fallback concurrency,
  - backoff on repeated miss-path failures.
- If SWR is used, require explicit dual-window model (`fresh_ttl`, `stale_window`) and stale-read eligibility boundaries.
- If negative caching is used, require short TTL and strict distinction between business negatives and dependency failures.

#### Key Safety, Tenant Isolation, And Serialization Competency
- Require deterministic, versioned, tenant-safe key design that includes all response-shaping dimensions:
  - tenant/account,
  - auth scope when relevant,
  - locale/feature variant qualifiers,
  - key version.
- Require key/value guardrails in the contract:
  - key length cap,
  - value size cap,
  - bounded qualifier normalization strategy.
- Require strict pooled-cache isolation:
  - tenant dimension mandatory,
  - no cross-tenant key reuse.
- Require explicit data classification policy:
  - secrets forbidden in shared cache,
  - PII caching only with explicit approved controls.
- Require serialization/versioning policy and decode-failure behavior:
  - decode failure treated as miss,
  - corruption/schema mismatch observable,
  - invalid entry eviction path defined.
- Prohibit runtime wildcard key scans (`KEYS`) on production request paths.

#### Failure, Degradation, And Origin-Protection Competency
- Classify cache dependency behavior per path:
  - `fail_open` default for read acceleration,
  - `fail_closed` only by explicit approved exception.
- Require dependency failure contract for DB/cache paths:
  - timeout hierarchy,
  - retry budget,
  - fallback mode,
  - containment control (bulkhead/load-shed/circuit mode).
- Require explicit origin-protection strategy for cache outages:
  - coalescing,
  - bounded fallback concurrency,
  - optional degraded response mode where contract allows.
- Require fast bypass/disable switch for rollback-safe cache deactivation.
- Require explicit degraded-mode activation and deactivation signals with observability hooks.

#### API, Data-Evolution, And Distributed-Consistency Interface Competency
- API contract impact obligations:
  - document consistency model (`strong`/`eventual`) for affected endpoints,
  - document staleness/freshness disclosure for eventual reads,
  - preserve retry/idempotency semantics for cache-influenced mutating flows.
- Data evolution interface obligations:
  - coordinate with data-architecture ownership for schema/migration leadership,
  - require cache-key/version transition plan across compatibility windows,
  - reject destructive-first schema assumptions that invalidate cache correctness mid-rollout.
- Distributed invalidation and async interface obligations:
  - require outbox-equivalent atomic linkage when DB state changes must emit invalidation/rebuild signals,
  - require consumer dedup/idempotency for invalidation handlers,
  - require explicit ordering assumptions and replay-safe behavior,
  - require reconciliation ownership for critical eventual-consistency projections.

#### Observability And Telemetry-Cost Competency
- Require DB/cache observability contract before merge readiness:
  - DB latency/error/pool-saturation visibility,
  - cache hit/miss/error/bypass/stale/fallback visibility,
  - miss-reason taxonomy with bounded vocabulary.
- Require correlation continuity across sync and async paths:
  - trace context propagation,
  - stable request/correlation/message identifiers in logs.
- Require bounded-cardinality telemetry discipline:
  - prohibit raw keys, user IDs, request IDs, trace IDs, message IDs as metric dimensions,
  - keep label sets operationally bounded and reviewable.
- Require explicit diagnostics and incident-readiness hooks for degraded DB/cache modes.
- Require telemetry cost guardrails for any additional metrics/histograms/sampling changes.

#### Security And Abuse-Resistance Competency
- Require strict boundary validation and normalization before DB/cache-influenced behavior.
- Require parameterized SQL values and allowlisted identifier fragments.
- Require timeout/size/concurrency limits for expensive cache-miss fallback paths.
- Require no secret leakage in DB/cache logs, errors, traces, or metric attributes.
- Require explicit abuse controls for high-cost read patterns (rate/limit/fallback behavior).
- Require security review trigger when cache content classification or tenant-isolation risk changes.

#### Testing And Verification Competency
- Require explicit test obligations in `70-test-plan.md` for DB/cache decisions:
  - hit/miss/error/bypass/stale/negative paths,
  - timeout/fallback/degradation behavior,
  - stampede suppression and bounded origin calls under concurrency.
- Require integration validation in cache-available and cache-degraded modes.
- Require load/failure profile expectations for warm-cache, cold-cache, and outage scenarios.
- Require deterministic verification strategy for cache correctness and stale-data risk.
- Require race-safety evidence for concurrency-sensitive cache wrappers/workflows.
- Require explicit verification queries/acceptance thresholds when schema evolution affects cache semantics.

#### Evidence Threshold And Assumption Discipline
- Every major `DBC-###` decision must include:
  - at least two options,
  - one explicit rejection reason,
  - measurable acceptance boundaries,
  - reopen conditions tied to observable triggers.
- Mark missing critical facts as bounded `[assumption]` immediately.
- Resolve assumptions in the current pass when possible; otherwise promote to `80-open-questions.md` with owner and unblock condition.
- Reject narrative-only recommendations without measurable risk controls or test obligations.

#### Review Blockers For This Skill
- Cache introduced without measured bottleneck evidence.
- Staleness/consistency contract missing or ambiguous for affected operations.
- Key design missing tenant/scope/version safety dimensions.
- No explicit invalidation/TTL/jitter/stampede strategy for cached paths.
- Timeout hierarchy and fallback/origin-protection behavior undefined.
- DB-path transaction/retry/idempotency constraints missing or contradictory.
- API-visible consistency/idempotency impact exists but `30-api-contract.md` is not aligned.
- Async invalidation/rebuild relies on dual writes without outbox-equivalent atomic linkage.
- Observability/test obligations are missing or use high-cardinality unsafe telemetry dimensions.
- Security-sensitive cache surface changed without classification/isolation controls.
- Critical uncertainty is deferred to coding instead of tracked as blocker.

## Working Rules
1. Determine current `docs/spec-first-workflow.md` phase and target gate before drafting decisions.
2. Set phase-specific output targets:
   - Phase 0: seed DB/cache assumptions and blockers in `80`.
   - Phase 1: define DB/cache constraints that must shape `20` and `60`.
   - Phase 2 and later: update `40/80/90` plus impacted `30/50/55/60/70`.
3. Load context using this file's dynamic loading rules and stop when required hard-skills coverage is source-backed:
   - SQL access discipline,
   - cache correctness contract,
   - degradation/origin-protection behavior,
   - observability/test obligations,
   - and all trigger-domain impacts that apply (API/security/distributed/data-evolution).
4. Normalize the decision problem: bottleneck evidence, consistency/staleness tolerance, tenant/scope safety, failure envelope, and API-visible behavior impact (when present).
5. For each nontrivial DB/cache decision, compare at least two options and select one explicitly.
6. Assign decision ID (`DBC-###`) and owner for each major DB/cache decision.
7. Record trade-offs and cross-domain impact (API, data, reliability, security, operability).
8. Run a hard-skills gate for each `DBC-###` decision and verify no `Review Blockers For This Skill` are left unresolved.
9. Mark missing critical facts as `[assumption]`; keep assumptions bounded and either validate in the current pass or move to `80-open-questions.md` with owner and unblock condition.
10. If a DB/cache uncertainty blocks safe implementation, record it in `80-open-questions.md` with concrete next step.
11. Keep `40-data-consistency-cache.md` as the primary artifact and maintain explicit boundary with data-ownership/migration responsibilities.
12. Verify internal consistency: no contradictions across impacted artifacts and no critical DB/cache decisions deferred to coding.
13. Close each pass with a concise readiness summary mapped to `Definition Of Done` so downstream roles can execute without reinterpretation.

## DB/Cache Decision Protocol
For every major DB/cache decision, document:
1. decision ID (`DBC-###`) and current phase
2. owner role
3. context and bottleneck/risk evidence
4. options (minimum two)
5. selected option with rationale
6. at least one rejected option with explicit rejection reason
7. consistency/staleness semantics and transaction implications
8. failure policy (timeout hierarchy, fallback mode, bypass behavior)
9. observability and test obligations
10. impact on API, data, security, and reliability
11. measurable acceptance boundaries and validation method (`tests`, `metrics`, `queries`)
12. hard-skills blocker check status (`none` or linked blocker item)
13. reopen conditions, affected artifacts, and linked open-question IDs (if any)

## Output Expectations
- Response format:
  - `Decision Register`: accepted `DBC-###` decisions with rationale, trade-offs, acceptance boundaries, and blocker-check status
  - `Artifact Update Matrix`: `40/30/50/55/60/70` with `Status: updated|no changes required` and linked `DBC-###`
  - `Assumptions`: active `[assumption]` items and resolution path
  - `Open Blockers`: unresolved items for `80-open-questions.md` with owner and unblock condition
  - `Sign-Off Delta`: what must be appended to `90-signoff.md` in this pass
- `Decision Register` entry minimum fields:
  - `DBC-###`, owner, context evidence
  - compared options + selected/rejected rationale
  - consistency/staleness + failure policy
  - observability/test obligations
  - measurable acceptance boundaries
  - hard-skills blocker-check status
- Primary artifact:
  - `40-data-consistency-cache.md` containing:
    - `SQL Access Risk Profile`
    - `Cache Necessity Decision`
    - `Topology And Pattern Choice`
    - `Staleness And Consistency Contract`
    - `Key/Tenant/Version Safety Requirements`
    - `Invalidation/TTL/Jitter/Stampede Controls`
    - `Failure And Degradation Policy`
- Required core artifacts per pass:
  - `80-open-questions.md` with DB/cache blockers and owners
  - `90-signoff.md` with accepted DB/cache decisions and reopen criteria
- Conditional alignment artifacts (update when impacted by DB/cache decisions):
  - `30-api-contract.md`
  - `50-security-observability-devops.md`
  - `55-reliability-and-resilience.md`
  - `60-implementation-plan.md`
  - `70-test-plan.md`
- Conditional artifact status format for `30/50/55/60/70`:
  - include one explicit status: `Status: updated` or `Status: no changes required`
  - for `no changes required`, add one sentence justification with linked `DBC-###`
  - for `updated`, list changed sections and linked `DBC-###`
- Language: match user language when possible.
- Detail level: concrete and reviewable with explicit trade-offs, assumptions, and safety constraints.

## Context Intake (Dynamic Loading)
Rule: load the smallest sufficient set of docs. Never bulk-load folders by default.
Stop condition: stop loading when base DB/cache axes are source-backed (SQL access discipline, cache strategy correctness, degradation behavior, observability/test obligations) and all applicable trigger-domain impacts are also source-backed.

Always load:
- `docs/spec-first-workflow.md`:
  - read only `Core Principles`, `Artifacts`, current phase subsection, and target gate criteria first
  - load additional sections only if unresolved DB/cache decisions require them
- `docs/llm/data/20-sql-access-from-go.md`
- `docs/llm/data/50-caching-strategy.md`

Load by trigger:
- Timeout/cancellation/retry semantics require tighter error-context framing:
  - `docs/llm/go-instructions/10-go-errors-and-context.md`
- Test obligations need deeper quality framing:
  - `docs/llm/go-instructions/40-go-testing-and-quality.md`
- Data ownership/schema evolution impact:
  - `docs/llm/data/10-sql-modeling-and-oltp.md`
  - `docs/llm/data/40-migrations-schema-evolution-and-data-reliability.md`
- API-visible consistency/idempotency impact:
  - `docs/llm/api/10-rest-api-design.md`
  - `docs/llm/api/30-api-cross-cutting-concerns.md`
- Cross-service invalidation, async processing, or distributed consistency impact:
  - `docs/llm/architecture/30-event-driven-and-async-workflows.md`
  - `docs/llm/architecture/40-distributed-consistency-and-sagas.md`
- Reliability/degradation policy shaping:
  - `docs/llm/architecture/50-resilience-degradation-and-system-evolution.md`
- DB/cache telemetry or diagnostics requirements:
  - `docs/llm/operability/10-observability-baseline.md`
  - `docs/llm/operability/30-debuggability-telemetry-cost-and-async-observability.md`
- Security-sensitive cache surface:
  - `docs/llm/security/10-secure-coding.md`

Conflict resolution:
- The more specific document is the decisive rule for that topic.
- If specificity is equal, prefer trigger-loaded documents over always-loaded documents.
- If conflict persists, preserve latest accepted decision in `90-signoff.md` and add reopen blocker in `80-open-questions.md`.

Unknowns:
- If critical facts are missing, proceed with bounded assumptions marked as `[assumption]`.
- Resolve each `[assumption]` by source validation in the current pass or by promoting it to `80-open-questions.md` with owner and unblock condition.

## Definition Of Done
- Current phase and target gate are explicitly stated.
- `40-data-consistency-cache.md` explicitly defines SQL-access and cache decisions for all affected operation classes.
- Every major DB/cache decision includes `DBC-###`, owner, selected option, and at least one rejected option with reason.
- Every major DB/cache decision includes measurable acceptance boundaries and reopen triggers.
- Cache is introduced only where bottleneck evidence or explicit rationale is documented.
- Staleness, key safety, invalidation, and fallback rules are explicit and testable.
- DB-path guardrails for query/transaction/timeouts/pooling are explicitly documented.
- Cache-path stampede controls and outage behavior are explicitly documented.
- Required trigger-domain impacts (API/security/distributed/data-evolution) are reflected in artifacts when applicable.
- Impacted `30/50/55/60/70` artifacts have explicit status with decision links and no contradictions.
- No unresolved item from `Review Blockers For This Skill` remains outside `80-open-questions.md`.
- DB/cache blockers are closed or tracked in `80-open-questions.md` with owner and unblock condition.
- No critical DB/cache decision is deferred to coding.

## Anti-Patterns
Treat each item as a blocker unless explicitly resolved with owner and rationale:
- add cache without measured bottleneck evidence
- leave staleness/consistency behavior implicit for cache-affected operations
- define cache keys without tenant/scope/version safety dimensions
- omit TTL/jitter/invalidation/stampede controls on nontrivial cached paths
- define cache fallback without bounded timeout hierarchy and origin-protection controls
- keep DB retry policy unbounded or retry non-idempotent writes without guardrails
- rely on dual writes for invalidation/events instead of outbox-equivalent atomic linkage
- add DB/cache telemetry with unbounded-cardinality labels
- cache sensitive data without explicit classification and isolation controls
- push critical DB/cache ambiguity to coding phase instead of `80-open-questions.md`
