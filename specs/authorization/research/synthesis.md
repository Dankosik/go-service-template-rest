# Authorization needs a real policy authority before a template capability

status: ready

## Question and decision

Should the template add an `AUTHZ` runtime profile now, and can one generic
authorization contract safely cover OPA, OpenFGA, Cedar, or local RBAC?

This research supports the Authorization Specification. It decides only what
evidence exists and what it changes; the specification owns the behavior.

## Current repository evidence

The inspected working tree is based on
`8967a4ac06d4fce0515703b15ffa5db35e5378ae`. It is dirty and its local `main`
is six commits behind `origin/main`, so this is a current-tree baseline rather
than an upstream or immutable-candidate claim.

- `AUTHN=oidc-jwt` verifies inbound credentials and publishes
  `reqctx.Principal`. The verifier deliberately does not turn roles, scopes, or
  tenant claims into permissions (`docs/authentication.md`,
  `internal/reqctx/principal.go`).
- The native gRPC server exposes `UnaryPolicy` and `StreamPolicy` composition
  slots. They are transport hooks, not a permission model or an effect-level
  enforcement owner (`internal/infra/grpc/options.go`,
  `internal/infra/grpc/chain.go`).
- Feature packages own business use cases and effects. Repository guidance
  requires authorization near the effect so HTTP, gRPC, jobs, and message
  consumers cannot diverge (`docs/architecture/boundaries.md`,
  `docs/universal-disciplines/auth-access-control/references/authorization-enforcement.md`).
- The checked manifests, initializer, configuration, bootstrap, API, and docs
  contain no `AUTHZ` selector, OPA, OpenFGA, or Cedar runtime integration.
- The production template contract currently exposes only health probes. No
  adopter, protected business action, resource hierarchy, tenant authority, or
  permission administration flow is named.

## Current external contracts

Primary documentation was checked on 2026-08-22.

| Candidate | Authoritative behavior | Decision effect |
| --- | --- | --- |
| OPA | The [OPA REST API](https://www.openpolicyagent.org/docs/rest-api) evaluates policy/data documents. Its [external-data guidance](https://www.openpolicyagent.org/docs/external-data) says OPA is not the source of truth for policy or data and makes freshness, replication, and request-input choices part of the integration. | A service must already know which authoritative principal, resource, tenant, and state data to send and what freshness is acceptable. |
| OpenFGA | [OpenFGA modeling](https://openfga.dev/docs/modeling/getting-started) starts from domain object types and answers whether a user has a relation to an object. Its [design guidance](https://openfga.dev/docs/best-practices/modeling-design-principles) warns against an anything-shaped meta-model. | A relationship graph, tuple writers, model lifecycle, and consistency contract are product decisions, not transport plumbing. |
| Cedar | [Cedar authorization](https://docs.cedarpolicy.com/auth/authorization.html) evaluates principal, action, resource, context, policies, and entity data. Its [security guidance](https://docs.cedarpolicy.com/other/security.html) leaves correct policy and relevant input-data supply to the application. | A common request shape does not supply the missing entity schema, policy owner, data slice, or update/revocation semantics. |

These systems overlap at an allow/deny answer but do not share one authority
model. OPA can evaluate arbitrary contextual policy, OpenFGA is relationship
and tuple centered, and Cedar evaluates typed policy plus entity data. A local
RBAC table would add a fourth authority and lifecycle rather than remove this
choice.

## Counter-evidence and limits

- `principal, action, resource, context` is a useful vocabulary across several
  systems. It does not justify a template interface: there is no concrete
  caller, implementation, or stable result/failure contract to require one.
- gRPC policy slots prove that transport composition is possible. They do not
  prove safe authorization because an internal call, job, or consumer can skip
  them.
- Token scopes could support a narrow feature policy when the configured issuer
  actually proves them. The current verifier does not populate scopes, and a
  token claim would still not define tenant/object authority or revocation.
- No provider account, live PDP, policy repository, representative workload,
  latency target, availability target, or revocation target was supplied or
  tested. Product versions, hosted offerings, and operational limits remain
  unknown and drift-prone.

## Downstream implication

There is not enough authority to select a runtime backend or specify a complete
capability. The smallest safe result is to keep authorization feature-owned and
add no `AUTHZ` profile, generic `Authorizer`, action registry, role strings,
decision cache, policy test kit, configuration, or dependency.

Reopen the Specification when a named adopter supplies:

1. protected actions and resources, including create/list/batch behavior;
2. verified principal and tenant derivation;
3. permission semantics: RBAC, ABAC, ReBAC, or an existing organization PDP;
4. the authority and write path for roles, relationships, policies, and entity
   attributes;
5. consistency and revocation windows, including policy/data changes during an
   operation;
6. required behavior for deny, missing input, timeout, overload, dependency
   failure, replay, and retry;
7. every entry point that can own the effect; and
8. the selected backend/deployment owner, trust route, support boundary, and
   representative latency/availability target.

Refresh repository evidence if authentication, request identity, gRPC policy
composition, initializer profiles, or the first production feature changes.
Refresh external contracts only after a concrete backend becomes a candidate.
