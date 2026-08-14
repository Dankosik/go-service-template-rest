# Keep Redis and Valkey out until measured adopter evidence earns them

status: ready

Completion: The current repository continues to satisfy the direct closure
checks below, PostgreSQL and feature/deployment ownership remain unchanged, and
no accepted decision-flip evidence exists; no implementation change is required.

Blocked stop: none. This ledger has no executable implementation unit.
Implementation must not start from it; a failed closure check or complete
decision-flip evidence reopens Specification.

## Obligation reconciliation

| Accepted obligation | Planning disposition |
| --- | --- |
| Add no Redis or Valkey dependency, profile, selector, configuration, runtime client, probe, lifecycle/readiness behavior, telemetry surface, or operational obligation; selected and unselected template behavior stay unchanged. | Proved no implementation: the capability-surface and package-path checks below find no product surface at `40e6d212799ae8677b675339929c559246536181` plus the inspected dirty worktree. The unrelated dependency and bootstrap dirt remains outside this outcome. |
| Preserve PostgreSQL as persisted-state authority and keep every future cached value derived and disposable. | Proved no implementation: [the ready behavior contract](spec.md#behavior-and-contract-delta) already fixes this unchanged authority, [Repository Architecture](../../docs/repo-architecture.md#stable-component-boundaries) keeps feature behavior in `internal/<feature>` and PostgreSQL integration in `internal/infra/postgres`, and no cache adapter or write path exists. |
| Keep authority revision, full key and tenant/policy scope, freshness, TTL, invalidation, read-after-write behavior, serialization and mixed-version handling, negative entries, fill/coalescing, degradation/origin protection, privacy, and semantic telemetry with the independently adopting feature. | Proved no implementation: this is an unchanged ownership constraint, not a template mechanism. The generic-abstraction check below finds no competing repository-wide owner. |
| Keep future product/version/topology/provider/operator selection, endpoint and TLS trust, ACLs/credentials, server memory/eviction, failover, and recovery with deployment. | Scope exit under [Scope and non-goals](spec.md#scope-and-non-goals): no product or deployment contract is selected. Deployment and the reopened Technical Design own these decisions only after Specification accepts an adopter. |
| Reject every generic cache abstraction and default named by the specification. | Proved no implementation and scope exit: the generic-abstraction check finds no competing named cache owner. `Cache[K,V]`, `Get/Set/Delete` backends, `GetOrLoad`, repository/read-through/write-through wrappers, global TTL/jitter/stale-on-error/negative-cache/fail-open defaults, key/namespace/tenant/serializer registries, invalidation buses or Pub/Sub correctness layers, cross-store transaction wrappers, distributed locks or distributed `singleflight`, automatic client-side caching, interchangeable product/topology/provider configuration, automatic readiness criticality, hit-rate success rules, and authoritative/durable/session/lock/write-behind cache roles remain rejected non-goals, not deferred tasks. |
| Preserve the current evidence boundary and exact reopen bar. | Proved no implementation now and recorded below: current-tree absence and ownership are established; performance, provider compatibility, production topology, and future cache correctness are not. |

## Direct closure proof

Evidence snapshot: 2026-08-12, HEAD
`40e6d212799ae8677b675339929c559246536181` plus the current dirty worktree.

- Product surface absence: `rg -n -i '\b(redis|valkey)\b|go-redis|rueidis|redigo' go.mod go.sum env scripts/init-module.sh scripts/ci/template-init-check.sh .github/workflows/ci.yml internal cmd` returns no matches. This falsifies a dependency, profile/generation oracle, configuration, bootstrap/runtime client, probe, or health/readiness integration.
- Package absence: `rg --files internal cmd | rg -i '(^|/)(cache|redis|valkey)(/|\.|$)'` returns no matches.
- Generic abstraction absence: `rg -n -g '*.go' 'Cache\[|type[[:space:]]+[[:alnum:]_]*Cache[[:alnum:]_]*|GetOrLoad|ReadThrough|WriteThrough|StaleOnError|NegativeCache|CacheHit|CacheMiss' internal cmd` returns no matches. `codegraph query Cache -p . -l 50` reaches only health-owned readiness caching and generated OpenAPI decoding, not a generic value-cache owner.
- Decision-flip absence: `rg -l -i '\b(redis|valkey)\b|go-redis|rueidis|redigo' specs -g 'spec.md'` returns only this specification. Other research mentions are rejected candidate evidence, not accepted adopters.

The proof establishes current absence and ownership only. No representative
workload, performance target, provider entitlement, live topology, benchmark,
provider mutation, compatibility certification, or future implementation proof
was supplied or run. Refresh the closure checks if `go.mod`, configuration,
bootstrap, initializer/profile, generation-oracle, health/readiness, or CI
surfaces change. Refresh product/client/provider claims only after a product is
selected and before Technical Design approval and release.

## Scope exits and reopen condition

- Cached-value semantics, feature adapters, and every rejected generic abstraction are outside the accepted no-pack outcome. A named feature owns any future proposal; similarity between client calls does not reopen it.
- Product/topology/provider operations and certification are outside the accepted no-pack outcome. Deployment owns any future proposal; current absence is not provider or production proof.
- The lifecycle-only pack described in [Risks, assumptions, and reopen conditions](spec.md#risks-assumptions-and-reopen-conditions) is a ceiling for a future specification, not an executable task.

Reopen Specification only when one named feature and cached value supply a
representative cache-disabled baseline, an accepted latency or origin-capacity
target that baseline misses, proof that PostgreSQL/query/computation
optimization, request coalescing, and a feature-owned local adapter cannot meet
the target, the complete feature-owned cache semantics listed above, and one
exact product/version/topology/operator/provider/TLS/auth/recovery contract. A
shared lifecycle-only pack additionally requires either one required template
profile with a named adopter and support owner whose production proof earns the
support burden, or two real features with the same exact product, topology,
credentials, lifecycle, resource bounds, and transport telemetry.

Readiness dry run: there is no acceptance unit or planned wave to execute. The
four direct closure checks above returned their expected observations, so no
implementation owner, writable surface, resource, dependency, handoff, or proof
command remains to schedule.
