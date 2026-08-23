# Keep authorization feature-owned until a policy authority exists

status: ready

Problem: Authentication identifies a caller, but the template has no real
permission model, protected business resource, tenant authority, or policy
backend. Adding a generic authorization capability now would make the template
promise behavior that no current adopter can define or prove.

## Scope and non-goals

This specification decides whether the current template adds an optional
Authorization capability. It accepts no runtime behavior delta: there is no
`AUTHZ` selector, dependency, configuration, policy client, cache, action
registry, middleware, use-case port, test kit, probe, readiness rule, or
telemetry surface.

It does not select OPA, OpenFGA, Cedar, local RBAC, or another PDP. It does not
define application roles, actions, resources, tenant membership, relationship
tuples, policies, entity attributes, decision administration, revocation,
audit retention, or deployment topology. Those are product and organization
authority decisions, not safe template defaults.

Technical Design is not triggered because there is no accepted mechanism,
runtime, data flow, package, or lifecycle behavior to realize. Test Design is
not triggered because the accepted outcome adds no behavior and has direct
repository-level absence and ownership falsifiers.

## Behavior and contract delta

### Current disposition

- Template initialization continues to expose no `AUTHZ` option. Selected and
  unselected profiles keep their current startup, readiness, request,
  telemetry, shutdown, and generated-source behavior.
- `AUTHN=oidc-jwt` continues to prove only the `reqctx.Principal` documented by
  `docs/authentication.md`. A scope, role, group, organization, or tenant claim
  is not automatically a permission decision.
- A business effect that requires authorization remains blocked until its
  feature defines a verified principal, explicit action, resource and tenant
  scope, decision authority, failure behavior, and enforcement point.
- Feature authorization, when accepted later, must execute at the
  effect-owning use-case boundary. HTTP or gRPC interception may reject early
  but cannot be the only enforcement because jobs, messages, and internal
  callers can reach the same effect.
- Missing or ambiguous identity, action, resource, tenant, policy input, or
  decision denies the effect. Dependency failure cannot become allow.
- Permission denial and policy-dependency failure remain distinct outcomes;
  neither may expose raw policy, entity, credential, subject, tenant, resource,
  or dependency text.

### Decision-flip test

All of the following are required before a template-level runtime capability is
specified:

| Required authority | Current result |
| --- | --- |
| A named adopter with protected business actions and resources | Absent; the production template exposes only health probes. |
| Verified principal and mandatory tenant derivation | Absent beyond issuer, subject, and client identity. |
| An accepted permission model and one decision authority | Absent; RBAC, ABAC, ReBAC, and existing PDP semantics are materially different. |
| Authoritative policy/relationship/entity write path and lifecycle | Absent. |
| Consistency, revocation, concurrency, replay, and retry contract | Absent. |
| HTTP, gRPC, job, message, and internal effect-entry inventory | Absent because no protected effect is named. |
| Deny/unavailable behavior and latency/availability budgets | Absent. |
| Concrete backend, deployment, trust, credentials, support owner, and version | Absent. |

The common vocabulary `principal, action, resource, context` and the existing
gRPC policy slots do not satisfy this table. They describe shapes and hooks,
not product meaning or authority.

## Invariants and edge cases

- No route, handler, use case, job, consumer, or internal caller may treat an
  authenticated principal as authorized by default.
- Tenant scope comes from a verified and accepted authority, never solely from
  a caller-supplied path, header, body, message, or job field.
- Authorization covers the object and caller-controlled state changes, not only
  route or method access. List and batch operations cannot authorize one item
  and leak another.
- An absent registered action or unsupported resource type denies rather than
  falling back to a wildcard.
- A cache cannot extend the accepted policy or relationship revocation window,
  and a stale or missing cache entry cannot grant access.
- A denied or failed check causes no protected effect. Retries cannot repeat a
  mutation merely because the decision path was unavailable or ambiguous.
- Health endpoints remain governed by their current public contract; this
  specification does not change their exposure.

## Rejected abstractions

The current template adds none of the following:

- `Authorizer`, `Policy`, `Role string`, generic permission maps, or context
  values carrying unverified roles or tenants;
- a universal adapter covering OPA, OpenFGA, Cedar, and local RBAC;
- route-name-to-permission conventions or automatic middleware authorization;
- generic action/resource registries without a real feature vocabulary;
- decision caching without an accepted consistency and revocation contract;
- a policy administration API, relationship store, policy bundle pipeline, or
  audit store; or
- permissive local fallback when a PDP is unavailable.

These are rejected current behavior, not implementation tasks deferred to a
later phase.

## Success criteria and proof expectations

This specification passes while all of the following remain true:

1. Dependency manifests, initializer selectors and generation oracles,
   configuration, bootstrap, contracts, docs, and runtime contain no generic
   Authorization capability or backend dependency.
2. OIDC authentication remains identity proof only and cannot silently grant
   roles, tenant access, or resource permission.
3. Transport policy hooks remain composition seams rather than the sole owner
   of a protected business effect.
4. No generic authorization abstraction exists without a named caller and
   implementation.

Proof is bounded to the current repository baseline and
`specs/authorization/research/synthesis.md`. It establishes current absence and
ownership constraints; it does not establish a future backend's correctness,
policy quality, tenant isolation, revocation latency, scale, or production
availability.

## Risks, assumptions, and reopen conditions

- Repository evidence comes from a dirty local tree based on
  `8967a4ac06d4fce0515703b15ffa5db35e5378ae`, with local `main` six commits
  behind `origin/main`. Refresh the affected surfaces before relying on this
  disposition after synchronization or overlapping edits.
- No adopter, organization PDP, authorization model, provider entitlement,
  policy repository, production topology, workload, or service objective was
  supplied. No external system was mutated.
- Similar check APIs across products do not make their data authority,
  consistency, revocation, lifecycle, or operational contracts compatible.

Reopen Specification only when one named adopter supplies every authority in
the decision-flip table. A template selector additionally requires either a
platform-owned backend contract supported for generated services or two real
adopters that share the same exact model, backend, trust route, lifecycle,
failure semantics, and support owner.

The maximum starting hypothesis after reopen is one backend-specific profile
with one decision model and fail-closed effect-level enforcement. A
multi-backend interface, generic role model, or decision cache remains out of
scope unless separate adopter evidence proves it necessary.
