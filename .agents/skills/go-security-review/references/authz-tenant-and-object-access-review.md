# AuthZ, Tenant, And Object Access Review

## When To Load
Load this when changed Go code touches authentication middleware, caller identity, tenant propagation, account or organization scoping, object-by-ID lookups, admin checks, service-to-service identity, or access-control failure handling.

## Attacker Preconditions
- The attacker is unauthenticated and can reach a protected path, or is authenticated as a lower-privilege user, another tenant's user, or a service principal.
- The request includes an object ID, tenant ID, account ID, role, scope, user ID, header-derived identity, or route parameter that influences which object is accessed.
- The code path reads, modifies, deletes, exports, publishes, or bills against an object without a complete server-side authorization decision.

## Review Signals
- Authentication is treated as sufficient authorization.
- Object-level authorization is missing on `GET /{id}`, `PUT /{id}`, `DELETE /{id}`, export, webhook, or batch flows.
- Tenant ID is trusted from path, body, query, or header instead of being bound from verified identity and policy.
- A repository lookup loads by object ID alone, then the handler filters or checks tenant after reading sensitive data.
- Admin or service paths are default-allow, role string comparisons are ad hoc, or service/user identity is collapsed into one subject.
- Failure mode returns a useful object existence oracle when the policy requires not-found masking.

## Bad Finding Examples
- "This needs authz."
- "Tenant checks look weak."
- "Use RBAC here."

These do not separate authentication from authorization or show how the caller crosses the object boundary.

## Good Finding Examples
- "[high] [go-security-review] internal/infra/http/invoices.go:82 Axis: AuthZ And Tenant Isolation; `GetInvoice` verifies the JWT but loads `invoice_id` without binding it to the caller's tenant before returning the row. Any authenticated tenant user who can guess or obtain another invoice ID can read another tenant's invoice. Move the tenant/object authorization check into the repository query or a policy-checked service method that requires both subject and tenant."
- "[high] [go-security-review] internal/app/admin/users.go:58 Axis: AuthZ And Tenant Isolation; the new `X-Admin: true` shortcut trusts a client-controlled header as an admin decision. A caller can set the header and reach admin user export. Remove the header trust and derive admin capability only from verified identity claims or an internal, authenticated service context."

## Smallest Safe Correction
- Keep authn and authz as separate checks: verify identity first, then verify action on object.
- Bind tenant and subject from verified identity, not from user-controlled request fields.
- Prefer repository/service methods that take `{subject, tenant, objectID, action}` and enforce the predicate before returning sensitive data.
- Fail closed when policy input is missing, malformed, or ambiguous.
- Preserve service-principal and end-user subject separation in mixed flows.
- Add a design escalation instead of a local patch when the correct tenant policy is not knowable from code or spec.

## Validation Ideas
- Add negative tests for same role but different tenant, same tenant but wrong object owner, missing tenant, malformed tenant, and service identity acting without delegated user context.
- Add tests for not-found versus forbidden behavior only when the API contract specifies the distinction.
- Add table tests for every action on the resource, not just read.
- Run targeted package tests plus `make test` if auth middleware or shared policy helpers changed.

## Repo-Local Anchors
- Existing skills and specs in this repo consistently distinguish trust boundaries, authorization, tenant isolation, and object ownership as security design concerns.
- `api/openapi/service.yaml` and generated handlers are client-visible contract surfaces; auth error semantics should not drift casually during review.

## Exa Source Links
- OWASP Authorization Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html
- OWASP API1:2023 Broken Object Level Authorization: https://owasp.org/API-Security/editions/2023/en/0xa1-broken-object-level-authorization/
- OWASP API Security Top 10 2023: https://owasp.org/API-Security/editions/2023/en/0x11-t10/
