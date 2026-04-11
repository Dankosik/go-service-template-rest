# AuthZ, Tenant, And Object Access Review

## Behavior Change Thesis
When loaded for symptom "caller identity reaches a resource or action decision," this file makes the model separate authentication, authorization, tenant binding, and object ownership instead of likely mistake treating login, role strings, or object IDs as sufficient access control.

## When To Load
Load this when changed Go code touches authentication middleware, caller identity, tenant propagation, account or organization scoping, object-by-ID lookups, admin checks, service-to-service identity, or access-control failure handling.

If the defect is mainly JWT signature/claim verification, reset-token generation, or password hashing, load `token-and-credential-flow-review.md`. If verified identity is present but the object/action decision is wrong, load this file.

## Decision Rubric
- Keep authentication and authorization as separate findings when both are broken.
- Bind tenant and subject from verified identity and policy, not from path, body, query, or arbitrary headers.
- Require object-level checks on `GET /{id}`, update, delete, export, webhook, batch, and billing flows.
- Prefer repository or service methods that enforce `{subject, tenant, objectID, action}` before returning sensitive data.
- Treat object-ID-only loads followed by late filtering as a likely exposure unless data returned before filtering is impossible.
- Preserve service-principal and end-user subject separation; default-allow admin or service paths are merge-risk.
- Name not-found versus forbidden behavior only when the API contract or approved policy defines masking.

## Imitate
```text
[high] [go-security-review] internal/infra/http/invoices.go:82
Issue: Axis: AuthZ And Tenant Isolation; `GetInvoice` verifies the JWT but loads `invoice_id` without binding it to the caller's tenant before returning the row.
Impact: Any authenticated tenant user who can guess or obtain another invoice ID can read another tenant's invoice.
Suggested fix: Move the tenant/object authorization check into the repository query or a policy-checked service method that requires both subject and tenant.
Reference: tenant isolation and object access contract.
```

Copy this shape when identity exists but object access is not scoped before data returns.

```text
[high] [go-security-review] internal/app/admin/users.go:58
Issue: Axis: AuthZ And Tenant Isolation; the new `X-Admin: true` shortcut trusts a client-controlled header as an admin decision.
Impact: A caller can set the header and reach admin user export.
Suggested fix: Remove the header trust and derive admin capability only from verified identity claims or an internal authenticated service context.
Reference: admin authorization boundary.
```

Copy this shape when the access-control input is attacker-controlled, even if the code labels it "internal."

## Reject
```text
Issue: This needs authz.
```

Reject because it does not say which action, object, subject, tenant, or asset is exposed.

```text
Suggested fix: Check the role string before the repository call.
```

Reject when role alone cannot prove object or tenant permission.

## Agent Traps
- Do not collapse "bad token validation" and "missing object authorization" into one vague auth finding; they have different fixes and tests.
- Do not assume `admin`, `internal`, or `service` strings imply trusted authority without a verified issuer or gateway contract.
- Do not require a broad RBAC redesign when the local safe fix is to pass subject and tenant into an existing lookup.
- Do not call `404` masking required unless the contract or surrounding code consistently uses it.

## Validation Shape
- Add negative tests for same role but different tenant, same tenant but wrong object owner, missing tenant, malformed tenant, and service identity without delegated user context.
- Add action-matrix tests for read, update, delete, export, and batch behavior when those actions are touched.
- Add not-found versus forbidden tests only when the contract specifies the distinction.

## Repo-Local Anchors
- `api/openapi/service.yaml` and generated handlers are client-visible contract surfaces; auth error semantics should not drift casually during review.
- This repo's skills distinguish trust boundaries, authorization, tenant isolation, and object ownership as separate security decisions.
