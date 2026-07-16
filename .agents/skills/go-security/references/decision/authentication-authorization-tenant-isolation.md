# Authentication, Authorization, And Tenant Isolation

## Behavior Change Thesis
When loaded for identity or access-control requirements, this file makes the model choose caller/subject/tenant-bound object, property, and function rules instead of likely mistake: role-only authorization, trusted identity headers, or `subject_id == path_id` shortcuts.

## When To Load
Load this when a flow needs authentication requirements, JWT or bearer-token rules, caller versus subject separation, tenant binding, object-level authorization, property-level authorization, function-level authorization, admin boundaries, or authorization negative tests.

## Decision Rubric
- Define one auth context contract: caller, subject, tenant, scopes or roles, relevant attributes, authentication method, issuer, audience, expiry, assurance level, and source of truth.
- Authenticate at the request boundary, then authorize in trusted service behavior before side effects. Tenant-owned persistence still needs tenant scoping at the data-access boundary.
- Require object-level checks for identifiers from path, query, header, body, async payload, or derived lookup.
- Require property-level checks for both mutable request fields and response fields. Reject or explicitly ignore forbidden mutable fields according to the API contract.
- Require function-level checks for privileged actions regardless of URL shape. Admin behavior is a capability, not a path prefix.
- For JWT or OAuth-style tokens, require issuer, audience, expiry/not-before, signature algorithm allowlist, key-source, token type, and key-rotation failure behavior. Treat header parameters such as `alg`, `kid`, `jku`, and `x5u` as untrusted lookup inputs; never let the token header choose verification policy or an unapproved remote key source.

## Imitate
- "For `GET /accounts/{account_id}`, require the caller's tenant and relationship to `account_id` before the repository read; same-role callers in other tenants receive `403` or an approved concealment response with no data disclosure." Copy the object plus tenant rule before data access.
- "For update payloads, `role`, `tenant_id`, and internal status fields are rejected unless the caller has the specific property-level permission." Copy the property-specific rule instead of relying on domain-object binding.
- "For service-to-service admin actions, the service credential must carry the target audience and admin scope; end-user JWTs are not propagated through async payloads." Copy caller/subject and credential-purpose separation.

## Reject
- "Check `subject_id == path_id`." This misses delegated access, tenant ownership, relationships, and admin support roles.
- "API key means user is authenticated." API keys can identify clients or plans, not end-user authority unless a separate user binding exists.
- "Use RBAC." RBAC is insufficient when object, property, tenant, relationship, or environment attributes decide access.
- "Trust `X-User-Id`, `X-Tenant-Id`, or role headers from internal callers." These are data until bound to an authenticated upstream contract.

## Agent Traps
- Do not merge authentication failure and authorization failure. Protected endpoints should usually distinguish `401` for missing/invalid authentication from `403` for authenticated-but-denied access unless concealment is explicitly chosen.
- Do not authorize only at the route boundary when service methods or repositories can be reached from async workers or other handlers.
- Do not call tenant isolation "covered by roles"; tenant is a scope dimension, not a role name.

## Validation Shape
- Authorization matrix: caller role/scope -> tenant -> object relation -> mutable fields -> response fields -> allowed action -> denial response.
- JWT negative cases: `alg: none`, wrong issuer, wrong audience, expired `exp`, future `nbf`, unknown or injection-shaped `kid`, unapproved `jku`/`x5u`, altered claims, missing signature, wrong token type, and stale key-set behavior.
- Tenant-crossing checks with two accounts or tenants, including read, update, delete, bulk/list, and inference paths.

## Repo-Local Anchors
- `api/openapi/service.yaml` includes a `bearerAuth` JWT scheme but the current global security array is empty; new protected operations should make security requirements explicit in the contract or task-local API design.
- `internal/infra/http` currently provides transport middleware, not a full auth context. Any new auth model should define where it is created and where service-layer authorization occurs.
