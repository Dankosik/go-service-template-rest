---
name: auth-access-control
description: "Principal-first authentication and access control. Use when designing, building, auditing, or diagnosing login and credential flows, OAuth2/OIDC, sessions, JWTs and refresh rotation, API keys, service-to-service identity, RBAC/ABAC/ReBAC permission models, tenant isolation, revocation, or access bugs such as privilege escalation, IDOR, and confused-deputy paths. Route permission storage schema to postgres-schema-design, decision/session cache freshness to cache-engineering, and the external IdP HTTP boundary to external-api-integration."
---

# Principal-First Auth and Access Control

Tie every protected effect to a verified **principal** and an explicit permission decision:

`principal -> credential -> session/token -> permission model -> enforcement -> propagation -> revocation -> prove`

Authentication proves who is acting. Authorization decides what that principal may do to this resource now. Keep the two contracts separate, and make deny the default whenever identity, tenant, scope, or context is missing or ambiguous.

## Choose the branch

Run only the branch the request needs. Record missing evidence as a gap instead of broadening the task.

- **Audit or diagnose:** trace real requests from credential to effect. Find the first link where identity is assumed instead of verified or a decision consumes client-controlled input, report it with its escalation path, and propose the smallest fix plus a runnable deny-path proof. Keep code and production unchanged.
- **Design or plan:** define principals, credential and session/token lifecycles, the permission model, enforcement points, and revocation windows to the fidelity requested; label artifacts proposed.
- **Build or fix:** change the smallest enforcement or lifecycle surface and prove it with a test that fails on the old behavior through the deny path, not only the allow path.
- **Operate:** production key rotation, forced logout, credential or token revocation, and permission backfills require explicit authorization for the exact action and targets. Preflight, execute only that action, verify with fresh readback.

Hand permission-table schema and constraints to `postgres-schema-design`; session or decision cache freshness to `cache-engineering`; the IdP or token-provider HTTP boundary to `external-api-integration`; races between an authorization check and its effect to `concurrency-control`; cross-service trust topology tradeoffs to `distributed-system-design`. This skill retains principals, credentials, session/token lifecycle, permission semantics, enforcement placement, identity propagation, and revocation.

**Complete when:** the branch, authority boundary, requested artifact, and excluded concerns are explicit.

## Map principals and entry points

List every principal type the system accepts: end users, admin/operator users, API clients, service accounts, anonymous. For each, name its identifier, credential, issuer, lifecycle owner, and tenant membership. Model delegation explicitly: an impersonating admin or an on-behalf-of service carries both identities end to end, never only the target's.

List every entry point — public endpoints, internal endpoints, admin surfaces, background jobs, consumers, scheduled tasks, debug and ops ports — and the principal types allowed to reach each. Background work runs as a named principal: either a service principal with its own narrow permissions or a durably captured user context; it never inherits ambient superuser access. "Public, by decision" is a valid classification, but it must be written down.

**Complete when:** every entry point names its accepted principal types and the identity evidence each carries; no caller category is unclassified.

## Contract credentials

For each credential type — passwords, OAuth2/OIDC flows, API keys, service identity — define issue, verify, rotate, revoke, and recover, plus abuse limits. Use standard primitives for the mechanics (memory-hard password hashes, code-with-PKCE for humans, client credentials for services, hashed scoped expiring API keys, workload identity instead of shared secrets); this skill owns the lifecycle contract, not the cryptography.

The sharp edges live off the happy path: account recovery is an authentication path with the same rigor as login — it is the most attacked bypass; guessing and enumeration get rate limits plus identical response and timing for unknown user and wrong password; changing MFA or a recovery channel re-authenticates first.

**Complete when:** each credential type has all five lifecycle paths and its abuse limits; no credential is stored recoverably.

## Choose session or token from the revocation window

Decide first: after logout, credential compromise, role downgrade, or tenant removal, how long may old access keep working? That bound — the **revocation window** — selects the mechanism:

- **Server-side session** — immediate revocation, a lookup per request; the default for interactive apps.
- **Self-contained token (JWT)** — stateless verification, but revocation waits for expiry unless a deny event is checked; fit for short-lived access tokens.
- **Hybrid** — short access token plus rotating refresh token: the access TTL is the revocation window; rotation with reuse detection revokes the family when a stolen refresh token is replayed.

Validate audience, issuer, expiry, and signature on every hop; select keys by `kid` from a rotated key set with overlap; allow bounded clock skew. Claims are plaintext to the holder — never secrets or private data. Every claim consumed downstream (roles, tenant, scopes) is a cached fact whose staleness bound is the token TTL; a decision that cannot tolerate that staleness re-reads the source at decision time.

| Event | What stops old access | Maximum delay |
| --- | --- | --- |
| Logout | | |
| Credential compromise | | |
| Role or scope downgrade | | |
| Tenant/member removal | | |

Fill every row. "Waits for expiry" is acceptable only when the stated delay is explicitly accepted for that event.

**Complete when:** the table is filled, the mechanism matches the tightest required window, and refresh rotation with reuse detection exists wherever refresh tokens do.

## Model permissions from rules

Write the access rules first as sentences: who may do what to which resource in which state. Then pick the smallest model that expresses them:

- **RBAC** for small, stable role-to-permission bundles;
- **ABAC** when decisions hinge on attributes and context (state, amount, time, ownership);
- **ReBAC** when access follows relationships — ownership, sharing, folder or org hierarchies.

Express every check as data: `decide(principal, action, resource, context)` with named inputs and an enumerable permission registry. Scattered `if role == "admin"` string checks are a defect: they cannot be audited, enumerated, or revoked. Default deny: an action without an explicit rule is denied, including newly added endpoints.

Tenant isolation is a partition, not a permission: derive tenant scope from the verified principal and apply it in every query, mutation, cache key (`cache-engineering`), and background job. Client-supplied tenant or user IDs never select the scope — they may only narrow within it. Defense in depth at the data layer (for example row-level security) covers the handler that forgot the filter; hand its schema to `postgres-schema-design`.

**Complete when:** every action–resource pair resolves to a named rule with defined inputs; no decision consumes client-controlled identity; tenant scope is derived and mandatory.

## Enforce at the effect

Place one enforcement point per boundary and put the decision as close to the effect as possible. Authorize the object, not only the route: any resource ID in a request is a claim to verify against the principal's scope (IDOR). Reject fields the caller may not set instead of mass-assigning. Enforcement lives in the layer that owns the data, so internal callers, jobs, and consumers pass the same check — an "internal" endpoint reachable without verification is an unauthenticated entry point.

A permission check followed later by the effect is a check-then-act race: the grant can be revoked mid-flight. When that gap is material, re-check inside the effect transaction or route the mechanism to `concurrency-control`. A cached decision is a downstream-consumed claim like any other: it carries the same staleness bound.

**Complete when:** each entry point names its enforcement point, decision inputs, and object-level check, and nothing reachable bypasses them.

## Propagate identity across services

Distinguish the service acting as itself from the service acting on behalf of a user, and make each hop verify what it receives. Identity headers set by an edge are trustworthy only inside a boundary unauthenticated parties cannot reach; otherwise pass verifiable tokens. Audience-restrict tokens so a service cannot replay a caller's token against a third service (confused deputy); use token exchange for delegation instead of forwarding. Async work re-establishes identity from durable context — job and message payloads carry principal context, not live tokens that expire mid-queue (`durable-background-jobs`, `reliable-messaging`).

**Complete when:** every hop names the identity it verifies and what prevents replay or forwarding beyond its audience.

## Prove the contract

Prove deny paths; allow-path tests barely constrain the design. Minimum falsifiers, where applicable:

- cross-tenant read and write attempts against every resource type fail, including via direct object IDs and via cache hits;
- a revoked or logged-out session/token stops working within the stated window — measured, not assumed;
- refresh-token reuse is detected and revokes the family;
- a role or scope downgrade propagates within its stated bound;
- an endpoint with no explicit rule is unreachable (default deny);
- horizontal (peer) and vertical (user to admin) escalation probes fail at the enforcement point, not only in the UI;
- background jobs and internal endpoints reject unauthenticated or out-of-scope callers.

Run what the local environment allows; label the rest proposed with setup and exact assertions.

**Complete when:** every contract row has an executable deny-path falsifier, and revocation windows are demonstrated rather than declared.

## Report

Lead with the verdict and the highest-severity gap first: unauthenticated reachability, then escalation, then isolation, then revocation latency, then hygiene. Separate verified facts from inference; label each artifact proposed, implemented locally, tested, or verified. For a diagnosis, prefer `first broken link -> escalation path -> smallest fix -> deny-path proof` over restating the whole lifecycle.
