---
name: cache-engineering
description: "Freshness-first cache engineering. Use whenever deciding whether caching should exist; designing or changing request, in-process, distributed/Redis, HTTP, or CDN cache contracts; diagnosing stale or cross-scope data, invalidation races, stampedes, hot keys, eviction, outages, or cache-driven latency/load; or proving cache correctness and measured value. Route PostgreSQL bottleneck attribution to postgres-performance and data ownership/schema invariants to postgres-schema-design."
---

# Freshness-Oriented Cache Engineering

Treat a cache as a bounded copy with an explicit **freshness** contract:

`authority -> key -> fill -> serve -> invalidate/expire -> degrade/recover -> prove`

A cache earns its complexity only when comparable evidence shows lower user-visible latency or origin load without breaking correctness, isolation, or failure behavior.

**Global completion criterion:** for every cached value being designed or materially changed, authority, complete key scope, freshness contract, fill owner, invalidation/expiry path, concurrent-fill behavior, degraded-mode policy, observable outcome, and executable falsifier are defined in one canonical contract row.

Preserve supplied latency targets, freshness limits, isolation rules, and safety boundaries as fixed constraints. Missing evidence yields a gap or measurement plan while the target stays unchanged.

## Authority and scope

- For review, diagnosis, or design, inspect code, configuration, telemetry, and supplied artifacts read-only; report evidence and gaps.
- For build or fix requests, make in-scope local changes and run safe checks.
- Treat production flushes, bulk invalidation, eviction-policy changes, cluster resize, traffic cutover, and any load test as separate actions requiring explicit authorization. Keep an authorized action's target and bounds exact, then verify it with fresh readback.

Hand off database bottleneck attribution and SQL/index/capacity work to `postgres-performance`. Hand off data ownership, relational identity, and schema invariants to `postgres-schema-design`. CDN deployment and platform administration are outside the primary scope; provide the effective cache contract, required headers/rules, key and purge dimensions, rollout checks, and rollback conditions to the operator.

Ask only when a missing answer changes correctness, isolation, or authority. Otherwise proceed with explicit assumptions.

## Choose the branch

Start at the earliest unresolved link in the chain; visible later steps are not mandatory narration.

- **Decide/design:** establish Step 1 evidence. Finish with a no-cache verdict or bounded measurement plan when justification is absent; otherwise continue into the contract and design gates.
- **Diagnose:** pin the observed symptom and walk from authority to the first violated link before proposing downstream changes.
- **Build/fix:** define or amend the canonical contract, change the earliest shared ownership point, and run safe local falsifiers.
- **Prove:** pin the candidate and baseline, then run comparable correctness and measured-effect checks.

Apply completion criteria at the current work layer only where they can change the selected branch's outcome. Review or diagnosis reports observed evidence, the executable falsifier, and missing evidence; design specifies the falsifier and expected result; build or fix runs safe local checks. A narrow fix amends the affected contract fields without reopening an accepted cache decision or inventing unrelated dimensions. Request-local work does not need distributed-cache ceremony. Production-only proof stays an explicit gap until separately authorized.

## 1. Prove the cache deserves to exist

Pin the environment, workload, time window, data shape, and user-visible operation. Record before any change:

- the latency target and protected correctness/isolation invariants;
- p50/p95/p99 latency, throughput, and error rate;
- origin request rate, latency, concurrency, and limiting resource;
- repetition, warmup, cache state, and stopping rule for a comparable replay.

Count repeated work that the proposed cache can actually remove. If PostgreSQL is suspected but the origin cost is unattributed, invoke `postgres-performance` first. If the measured avoidable work cannot pay for cache complexity, finish with a no-cache verdict.

Do not subtract independently reported percentile values as if they came from the same trace, and do not call a percentile an upper bound or maximum. When only independent percentiles are available, conclude only what the evidence supports and require joint trace evidence or a measured counterfactual for removable tail latency.

**Completion criterion:** one reproducible baseline names the user-visible target, quantified origin cost, representative workload, environment, and invariants, or the result is a bounded measurement plan with no performance claim.

## 2. Define authority and the cached-value contract

For every cached value being designed or materially changed, write one canonical row. A narrow diagnosis or fix amends the affected fields and states correctness-relevant unknowns instead of manufacturing a full new contract. Later steps verify or amend the row instead of restating its decisions:

| Field | Required decision |
| --- | --- |
| Authority | Source of truth, authoritative version/revision, and owner of create/update/delete |
| Value | Exact positive, negative, or derived representation; serializer/schema version |
| Key | Namespace and version, resource identity, canonical inputs, tenant, authorization/policy, representation, and every other response-varying dimension |
| Freshness | Fresh lifetime, maximum permitted age, stale-while-refresh/error windows, read-your-writes expectation, and clock/age source |
| Fill | Fill owner, origin deadline, admission rule, and publish condition |
| Invalidate/expire | Trigger for create/update/delete, dependent variants, delivery/retry/reconciliation path, and TTL backstop |
| Concurrency | Duplicate-fill scope, generation/version fence, and waiter behavior |
| Failure | Timeout budget, fail-open/fail-closed decision, stale fallback, origin protection, and recovery |
| Proof | Observable outcome and executable test that would falsify the contract |

Common invariants live here:

- **Authority:** use the authoritative revision as proof of truth; a cache hit supplies only a reusable copy. Compare or derive from that revision when ordering matters.
- **Key:** equal keys must mean interchangeable values for the requesting principal. Canonicalize equivalent inputs before keying; use an explicit namespace/schema version. Keep raw tokens, credentials, and sensitive identifiers out of key material and logs. When output depends on current authorization, field policy, or localization, prefer caching a tenant-scoped authority-independent value and reapply current policy after retrieval; cache final response variants only when their policy-version key and invalidation contract are reliable.
- **Freshness:** classify every candidate or serve decision as `fresh`, `allowed-stale`, `forbidden-stale`, or `unknown-age`. TTL bounds reuse only when every serve path enforces it. Define age from origin generation or validation, not merely local insertion; route unknown age through the contract's explicit failure policy. Negative results are distinct from errors and need their own TTL and create-time invalidation.
- **Failure:** a miss, timeout, eviction, corruption, and partition are different states. Each has a bounded serve/fallback decision, and fallback demand stays within origin capacity.

**Completion criterion:** every cached value has a complete contract row; an omitted dimension is demonstrated irrelevant rather than left implicit.

## 3. Choose the narrowest layer and strategy

Choose the narrowest sharing boundary that meets the measured need. Combine layers only when each removes separately measured work:

| Layer | Choose when | Cost to accept |
| --- | --- | --- |
| Request | Equivalent work repeats inside one request/job | No reuse across requests |
| In-process | Process-local reuse is enough and per-instance cold fill is affordable | Per-process divergence, memory, and deploy cold starts |
| Distributed | Cross-instance reuse or coordination repays network and service dependency | Remote latency, ambiguous outcomes, eviction, partitions |
| HTTP/CDN | Responses are safely reusable by the intended private/shared audience | Protocol/vendor key, header, purge, and propagation semantics |

After selecting the layer, read only the references for behavior being designed, changed, or diagnosed:

- Read [references/request-process.md](references/request-process.md) when reuse or fill coordination is confined to one request or process.
- Read [references/distributed.md](references/distributed.md) when a remote shared cache such as Redis is proposed or already involved.
- Read [references/http-cdn.md](references/http-cdn.md) when browsers, proxies, gateways, or CDNs can store or revalidate the response.

For a multi-layer design, read each selected layer's reference.

Use cache-aside when application-owned miss handling matches the proven need. Choose read-through only when centralizing fills materially reduces duplicated policy. Choose write-through only when its cross-system success, retry, and rollback semantics are explicit. Use refresh-ahead or stale-while-revalidate only when the latency target cannot wait for fill and bounded staleness is acceptable. Choose write-behind for an explicit loss/durability budget with authoritative reconciliation, rather than as a general latency shortcut.

**Completion criterion:** the selected layer and strategy meet measured sharing, latency, consistency, isolation, and failure needs, and no narrower layer does; every additional layer removes separately measured work.

## 4. Complete the key and representation

Trace every producer, caller, and invalidator before editing a shared key builder. Compose keys from the contract, including tenant and authorization/policy scope before resource identity. Canonicalize order-insensitive sets, query defaults, casing, locale, currency, feature/policy versions, and representation format where they affect output. Bound high-cardinality dimensions.

Test both directions of the key contract: equivalent inputs must share a key, while any evidenced response-varying input must produce a distinct key and value.

Version the key namespace for incompatible meaning or serialization. Define value-size and decode limits. Compare compression only when measured network/memory savings exceed CPU and tail-latency cost. Reject oversized or corrupt values safely and count them.

Map each mutation to the exact positive, negative, aggregate, and variant keys it supersedes. If that mapping is not reliable, prefer authority revisions/generations over key enumeration.

**Completion criterion:** an executable collision and decode falsifier makes interchangeable requests share a key while distinct tenant/auth/representation requests do not; old or malformed encodings fail safely; every mutation reaches all dependent variants.

## 5. Bound fill and serve behavior

On miss, admit origin work through a declared concurrency and queue budget. Collapse equivalent concurrent fills at the narrowest sufficient scope. Give the shared fill its own bounded deadline; caller cancellation bounds that caller's wait rather than silently defining every waiter's result.

Publish only a value whose authority revision or generation is still current. A lock or lease suppresses duplicate work; the revision or generation fence authorizes publication. Bound waiter count and wait time, and define whether errors are shared, retried, or returned. Cache a not-found result only after an authoritative not-found response; treat timeouts and dependency failures as errors.

Use TTL jitter to spread synchronized expiry; authority revision or generation still determines ordering. Serve stale only inside the contract's maximum age and only to audiences for which that value remains safe. Record value age and freshness state on every serve path.

**Completion criterion:** a concurrent miss has one bounded fill policy and an executable stale-publication falsifier; positive/negative/error/timeout outcomes and fresh/allowed-stale/forbidden-stale/unknown-age states are independently distinguishable in checks and telemetry.

## 6. Invalidate or expire after writes

For cache-aside, commit the authoritative mutation before invalidating or superseding its cached copies. Authority remains committed when cache invalidation times out; treat the invalidation result as ambiguous. Retry only idempotent or generation-guarded operations and reconcile ambiguous outcomes.

Protect against this race explicitly: a reader starts an old fill, a writer commits and invalidates, then the reader publishes the old value. Use an authority revision/generation in the key or value and reject stale publication. TTL is only a bounded backstop. When missed invalidation can exceed the staleness budget, use a durable delivery path such as an outbox/change stream plus lag monitoring and reconciliation.

Create must supersede matching negative entries. Update and delete must cover every representation and aggregate named in the contract. For write-through/read-through systems, name the atomic boundary and what a timeout means for each store. Model each store as a separate outcome unless one documented atomic boundary includes both.

**Completion criterion:** create/update/delete and ambiguous-failure timelines end in a state allowed by the freshness contract, including an in-flight stale fill; retry and repair converge from each unresolved state.

## 7. Degrade and recover

Budget cache connect/command deadlines inside the request deadline. Protect the origin with bounded fallback concurrency, queue length, retry count, backoff/jitter, and a shed/circuit decision. Fail open to origin only when origin headroom and data semantics allow it. Fail closed, bypass shared reuse, or use an explicitly safe stale value when authorization, privacy, or correctness would otherwise be violated.

Model hot-key and partition skew, mass expiry/eviction, and a cold fleet. Track cache and origin saturation together. Recovery ramps demand within origin capacity; prewarming is justified by measured cold-start risk and uses the same key/freshness contract. Treat a broad flush as a separately authorized invalidation action that increases cold demand.

**Completion criterion:** every applicable outage, timeout, eviction, hot-key, cold-start, and recovery scenario has a bounded response, origin-load ceiling, observable trip signal, and exit condition.

## 8. Implement and migrate safely

For a build/fix request, make the smallest change at the shared ownership point and add the smallest runnable test that breaks on the relevant race or isolation failure. Keep serializer/key versions compatible across rolling deploys, or introduce a new namespace and explicit read/write transition. Declare cold-fill demand, canary/ramp, rollback behavior, and cleanup timing before deploy.

For a key fix, exercise the actual cache path: prove equivalent canonical inputs reuse one entry and prove a response-varying input cannot retrieve another input's value. Key-string inequality alone is not enough.

For a cache isolation incident, migration proof covers same-ID tenants, distinct roles/policies/locales, policy revocation, create/update/delete, mixed and cold fleets, canary isolation signals, cache outage, and rollback to bypass/no-cache. If final-response caching remains as a fallback design, enumerate its tenant, authorization/policy-version, locale, representation, and serializer key dimensions even when the preferred design caches authority-independent data.

**Completion criterion:** local code and tests implement the contract; rollout, cold start, migration, rollback, and later cleanup preserve mixed-version correctness without requiring an unauthorized production action.

## 9. Prove the outcome

Run the same representative before/after workload with controlled data, concurrency, duration/repetitions, warmup, and cache state. Report distributions and comparable deltas for:

- end-to-end p50/p95/p99, throughput, and errors;
- origin QPS, concurrency, latency, and database/CPU/I/O load as relevant;
- hit, miss, fill, coalesced-waiter, stale, negative, eviction, timeout, retry, and decode-error rates;
- cache operation latency, value age, memory/capacity, hot-key/partition skew, and recovery time.

Select tests that can falsify the contract. Include, when applicable: same hot key at expiry, many keys, slow/erroring fill, leader cancellation, update during fill, negative-then-create, tenant/auth separation, cache timeout/outage, eviction pressure, cold deploy, and rollback. Predeclare guardrails and stop conditions for any authorized load test.

Call a performance improvement only when fresh comparable measurements meet the user-visible target and origin-load goal while correctness, freshness, isolation, and failure tests pass. Use hit ratio only to explain those measured outcomes.

**Completion criterion:** retained evidence proves or falsifies the target under the same workload; missing production evidence remains an explicit gap, not an inferred win.

## Report

Lead with the verdict and artifact state: proposed, implemented locally, tested, deployed, or verified live.

For a mutable shared cache, multi-tenant/auth boundary, outage, migration, or production-readiness decision, use the full decision interface:

- baseline/target or observed symptom;
- canonical contract and first violated link;
- design or fix, including degraded mode and rollout/rollback;
- executable correctness, freshness, isolation, failure, and recovery falsifiers;
- comparable measured effects when actually proven;
- actions performed, authority boundaries, and remaining gaps.

For request-local memoization, immutable content-addressed assets, a no-cache verdict, or a narrow fix, include only what carries information for that branch:

- the relevant baseline/target or observed symptom;
- affected contract decisions and cause or design;
- executable falsifiers and observed results;
- measured effects when actually proven;
- actions performed, authority boundaries, and remaining gaps.

For a no-cache result, name the authority only for the measured lookup, retain the measured reason or bounded measurement plan, note the added correctness/isolation/failure surfaces without designing them, include the comparable end-to-end and origin evidence required to reconsider the decision, preserve production-action boundaries, and omit speculative cache design. For a narrow fix, center the patch and test rather than reproducing the full lifecycle. Missing production evidence remains an explicit gap, not an inferred win.

Before finishing, check the affected contract rather than the prose: every relevant field has a decision, evidence-backed unknown, or falsifier; rollout claims include rollback proof; performance claims use comparable baseline/candidate evidence; key tests prove both equivalence and separation. Do not narrate irrelevant checks.
