# HTTP and CDN Cache Mechanics

This file owns HTTP and CDN storage, revalidation, and purge mechanics. The common cache contract remains canonical in `SKILL.md`.

## Start with HTTP semantics

RFC 9111 is the current HTTP caching standard and obsoletes RFC 7234. A response is fresh while its calculated age is within its freshness lifetime. For shared caches, explicit freshness prefers `s-maxage`, then `max-age`, then `Expires`. Avoid heuristic freshness for correctness-sensitive data.

Directive decisions:

| Directive | Meaning that changes design |
| --- | --- |
| `no-store` | Prohibits intentional storage/reuse |
| `no-cache` | Allows storage but requires successful validation before reuse |
| `private` | Allows private storage but excludes shared caches |
| `max-age` | Fresh lifetime for caches generally |
| `s-maxage` | Shared-cache freshness and revalidation semantics |
| `must-revalidate` | Forbids stale reuse when validation fails |
| `stale-while-revalidate` | May serve bounded stale while asynchronous validation runs |
| `stale-if-error` | May serve bounded stale for qualifying origin/network errors |

RFC 5861 defines the stale extensions, but they are optional behavior. Worst permitted age includes the fresh lifetime plus the relevant stale window. Sparse traffic may not trigger refresh. Verify every intermediary's implementation and precedence; do not combine revalidation requirements and stale behavior without an observed effective result.

Use `ETag`/`If-None-Match` or, with weaker time precision, `Last-Modified`/`If-Modified-Since` to revalidate. A 304 saves response bytes but still reaches the origin, so measure origin work. Emit and observe `Age` plus cache/vendor status; RFC 9111 removed the obsolete `Warning` header used by RFC 5861 examples.

## Prove the effective key and audience

The HTTP base key is method plus target URI. `Vary` adds selected request-header dimensions. Enumerate every response-varying input, including normalized query, encoding, language, tenant, authorization/policy, locale, currency, device, and experiment state as applicable. High-cardinality personalized variants often belong in a private cache or should bypass caching.

A request `Authorization` header does not become a per-principal cache key automatically. Shared reuse requires an enabling directive, but permission to store is not isolation. Prefer `private` or `no-store`, or cache an authority-independent representation and apply authorization after retrieval. `Set-Cookie` is not a standards-level privacy barrier; send explicit directives.

Define which statuses are cacheable. Negative responses such as 404 can be cached, so give them explicit freshness and create-time supersession rather than relying on heuristic defaults. Keep an authoritative not-found distinct from an origin error.

For an immutable content-addressed URL, verify that the served bytes match the digest/version encoded by the URL, that no conflicting headers defeat the intended policy, and that repeated CDN requests show cache status/Age plus the expected bounded origin fills. If negotiation or transformation changes bytes, include that representation in the version/key contract.

## Revalidation, collapse, and invalidation

An intermediary may collapse equivalent misses into one origin request. Treat that as scoped single-flight: one slow or failed fill can delay many callers, and behavior can be per location rather than global.

Unsafe requests are forwarded to origin. A successful unsafe response invalidates the target URI under RFC rules, but this does not cover related resources, every CDN location, custom variants, application aggregates, or purge propagation. Name them in the contract. Use validators or versioned URLs/keys when purge latency cannot meet the freshness budget.

Purge outcome can be ambiguous and is not atomic with the authoritative write. Retry/reconcile by exact key/tag/version, retain bounded stale behavior, and verify propagation.

## Apply a production CDN model

Cloudflare is one concrete model, not a universal default:

- effective cacheability depends on plan, Origin Cache Control, Cache Rules, response headers/status, and request method;
- the default cache key contains the full URL/query plus additional request headers, while custom keys can accidentally merge private variants or shard reusable content;
- current `Vary`, stale-while-revalidate, stale-if-error, and `Set-Cookie` behavior depends on the serving path and configuration;
- tiered cache funnels lower-tier misses through upper tiers, reducing origin fan-out, but topology changes and cold upper tiers can spike misses;
- purge with custom keys must supply the same key dimensions; purge-all can amplify origin load.

Before handoff, record effective zone/rules/plan, observed response headers, computed key dimensions, cache status, purge selector and propagation check, origin shield/tiering, and rollback. Platform administration remains with the CDN operator.

Observe cache status and age alongside user latency and origin load. For Cloudflare, distinguish `HIT`, `MISS`, `EXPIRED`, `STALE`, `BYPASS`, `REVALIDATED`, and `UPDATING`; an absent/reset `Age` can reflect misses, revalidation, purge, eviction, or tier behavior. Test two tenants/principals, every `Vary` variant, validator flows, negative-then-create, simultaneous expiry, stale on each error class, purge ambiguity, cold locations, and origin failure.

## Primary sources

Checked 2026-08-02:

- [RFC 9111: HTTP Caching](https://www.rfc-editor.org/rfc/rfc9111.html)
- [RFC 5861: stale-while-revalidate and stale-if-error](https://www.rfc-editor.org/rfc/rfc5861.html) and the [IANA HTTP Cache Directive Registry](https://www.iana.org/assignments/http-cache-directives/http-cache-directives.xhtml)
- Cloudflare [default cache behavior](https://developers.cloudflare.com/cache/concepts/default-cache-behavior/), [Origin Cache Control](https://developers.cloudflare.com/cache/concepts/cache-control/), [revalidation](https://developers.cloudflare.com/cache/concepts/revalidation/), and [`Vary`](https://developers.cloudflare.com/cache/concepts/vary/)
- Cloudflare [cache keys](https://developers.cloudflare.com/cache/how-to/cache-keys/), [Tiered Cache](https://developers.cloudflare.com/cache/how-to/tiered-cache/), [cache responses](https://developers.cloudflare.com/cache/concepts/cache-responses/), and [purge by URL](https://developers.cloudflare.com/cache/how-to/purge-cache/purge-by-single-file/)
