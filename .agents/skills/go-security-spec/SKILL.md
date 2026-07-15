---
name: go-security-spec
description: "Use when trust boundaries, identities, access rules, tenant and object isolation, browser or token controls, threat defenses, abuse resistance, secure defaults, or security proof must be decided before coding; Own security policy; Skip when the task is enforcement review, API resource semantics, chi wiring, or implementation."
---

# Go Security Spec

Load the [shared specialist contract](../specialist-contract.md) for common selection, scope, evidence, reference, return, and handoff mechanics; apply the domain-specific rules below.

## Outcome And Boundary

Define testable policy for trust and threat assumptions; authn; authz; tenant/object/property/admin isolation; browser/session/CORS/CSRF; input and abuse; secrets/crypto; egress/SSRF; sensitive data; audit; dependencies/supply chain; and acceptance proof.

Keep authn, authz, tenant membership, and object ownership distinct. A valid credential proves identity claims, not permission, trusted tenant, or path-object ownership; never flatten them into one access check.

Do not redesign topology, API/schema, reliability/performance, observability, delivery, distributed coordination, or wiring. State forced security constraints only; block when unset neighboring policy prevents safety.

## Owned Contract

- Map external, partner, internal, async, admin, and third-party boundaries. Treat each as untrusted unless justified; name attacker path, asset/classification, enforcement owner, degraded behavior, and assumption.
- Authenticate caller/subject from accepted claims; independently authorize action, bind tenant from trusted identity/server state, and check object/property/admin scope authoritatively. Deny by default; headers, roles, or identity equality alone prove none of these.
- Define session fixation/rotation/expiry/revocation, cookie attributes, credentialed origin-specific CORS, ambient-authority CSRF, and safe redirects/errors. For tokens, define issuer, audience, expiry, replay, delegation, and least privilege.
- Bound and strictly parse input before effects; control injection/interpretation and sanitize output. Constrain egress scheme/destination/port, DNS, redirects, proxies, and connected IP; authenticate callbacks and block private/control-plane reachability.
- Classify, minimize, isolate, encrypt as required, retain/delete, and redact sensitive data. Keep secrets out of source, durable payloads, telemetry, and shared caches; require approved sources, scoped identity, rotation/revocation, and justified vetted crypto/dependencies.
- Across queues, callbacks, and service hops, require authenticity, step authorization, business idempotency, replay window, dedup, poison handling, and scoped credentials. Internal transport and durable bearer-token propagation are not trust.
- Gate abuse cheaply before allocation/effects by principal, tenant, operation, and cost; bound body, concurrency, queue, attempts, timeout, fan-out, and spend. Separate hostile-use controls from legitimate-capacity policy; preserve fail-closed invariants.
- Audit allow/deny/admin actions safely; record dependency, supply-chain, privilege, and runtime-hardening assumptions. Prove unauthenticated, unauthorized, cross-tenant, non-owner, injection, replay, secret, egress, and exhaustion denial with forbidden side effects; scanners are supplementary.

## Symptom-Driven References

Load the rubric whose symptom would materially change the security decision.

| Symptom | Load | Behavior Change |
| --- | --- | --- |
| Trust boundaries, attacker paths, or boundary ownership are implicit. | `references/trust-boundary-threat-modeling.md` | Choose named boundary, attacker-path, enforcement, and proof requirements instead of generic "use auth" or "validate input" advice. |
| Identity, authorization, tenant, object, property, or admin rules are in scope. | `references/authentication-authorization-tenant-isolation.md` | Choose caller/subject/tenant-bound access rules instead of role-only checks, untrusted headers, or `subject_id == path_id` shortcuts. |
| JSON/input parsing, SQL or interpreter use, outbound URLs, webhooks, or sanitized errors are in scope. | `references/input-output-injection-and-ssrf-controls.md` | Choose strict parser, allowlist, SSRF dial, and sanitized output requirements instead of denylist validation, late validation, or raw error relay. |
| REST/OpenAPI status codes, CORS, problem responses, method policy, request limits, retry/idempotency, or browser-visible headers are in scope. | `references/api-facing-security-semantics.md` | Choose contract-visible fail-closed semantics instead of ad hoc status codes, `200` error bodies, permissive CORS, or retry ambiguity. |
| Queues, workers, outbox/inbox, webhooks, callbacks, cross-service calls, third-party APIs, or retries are in scope. | `references/async-distributed-security.md` | Choose authenticity, replay, scoped credential, and step-authorization rules instead of trusting internal queues or propagating raw bearer tokens. |
| Sensitive data, privacy, cache keys, DB access, secrets, config source policy, logging, redaction, or telemetry fields are in scope. | `references/data-cache-and-secret-handling.md` | Choose classification, minimization, cache scoping, secret-source, and redaction requirements instead of shared caches, secret config, or log leakage. |
| Rate limits, expensive operations, bulk work, provider-cost triggers, repeated attempts, or resource exhaustion are in scope. | `references/resource-abuse-and-cost-controls.md` | Choose principal/tenant-scoped budgets and cheap pre-side-effect gates instead of one global rate limit or reliability-only overload wording. |
| Security decisions need proof obligations, test matrices, abuse-path checks, or scanner-vs-test separation. | `references/security-negative-path-verification.md` | Choose concrete negative-path and no-side-effect proof obligations instead of "covered by integration tests" or scanner-only confidence. |

Use repository API, transport, config, `SECURITY.md`, and CI authorities. Record absent identity, gateway/mesh, secret, cache/queue, third-party, or enforcement guarantees as assumptions or blockers.

## Return And Stop

Return the threat-control map, separate access decisions, data/secret/egress/abuse/audit/supply-chain policy, enforcement/fail behavior, negative proof and forbidden effects, forced consequences, assumptions, risks, and reopen conditions. Claims need an enforcement owner and falsifiable proof.

Block on ambiguous trust/identity; missing authn, authz, tenant, object/property/admin authority or enforcement; uncovered input/egress threats; unsafe retry/async authenticity or replay; incomplete secret/data lifecycle; unbounded abuse; material dependency/runtime assumptions; or denial proof that cannot exclude side effects.
