# Authentication, Authorization, And Tenant Isolation Examples

## When To Load
Load this when a flow needs an identity model, bearer/JWT requirements, caller versus subject rules, tenant binding, object-level authorization, property-level authorization, function-level authorization, admin boundaries, or authorization negative tests.

## Selected Controls
- Define one `AuthContext` contract with caller, subject, tenant, scopes/roles/attributes, authentication method, issuer, audience, expiry, and assurance requirements.
- Authenticate at the request boundary and enforce authorization in the trusted service layer before side effects. Repository/data access must also enforce tenant scoping where persisted records are tenant-owned.
- Require object-level checks for every endpoint that accepts object identifiers from path, query, header, or body.
- Require property-level checks for response fields and mutable request fields. Return or update only fields the caller may see or change.
- Require function-level checks for privileged methods regardless of URL shape. Admin-looking paths are not the only admin functions.
- For JWT or OAuth-style access tokens, require explicit issuer, audience, expiry/not-before, signature algorithm, key-source, and token type rules. Do not trust the token header to select verification policy.

## Rejected Controls
- Reject using API keys as user authentication. API keys may identify clients or plans, not end-user authority.
- Reject comparing only `subject_id == path_id` as a BOLA solution when ownership, relationship, role, tenant, or delegated access can differ.
- Reject client-filtered responses or mass assignment into internal objects as a property-level authorization control.
- Reject unsigned request headers such as `X-User-Id`, `X-Tenant-Id`, or role headers as trusted identity unless they are authenticated by an explicit upstream contract.
- Reject broad RBAC-only policy when the domain requires object, tenant, relationship, or environment attributes.

## Fail-Closed Examples
- Missing or invalid token maps to `401` for protected endpoints; authenticated but unauthorized access maps to `403`.
- Unknown tenant, conflicting tenant, tenant omitted on tenant-scoped data, or tenant from an untrusted source denies before read/write.
- Auth policy lookup failure, stale key set without approved cache semantics, or malformed token denies the request.
- Sensitive action requiring step-up reauthentication denies when the assurance level is missing or stale.

## Testable Requirements
- Given account A and account B with the same role in different tenants, account B cannot read, update, delete, bulk list, or infer account A objects.
- Given an admin-only function under a non-admin-looking path, a non-admin caller receives `403` and no side effect.
- Given a payload that includes a sensitive or internal property, the service rejects the change or ignores it only when the API contract explicitly allows ignore semantics.
- Given a JWT with `alg: none`, wrong issuer, wrong audience, expired `exp`, future `nbf`, unknown `kid`, altered claims, or missing signature, authentication fails.
- Given a valid user token but wrong tenant binding, object access fails before repository mutation.

## Repo-Local Anchors
- `api/openapi/service.yaml` includes a `bearerAuth` JWT scheme but the current global security array is empty; new protected operations should make security requirements explicit in the contract or in task-local API design.
- `internal/infra/http` currently provides transport middleware, not a full auth context. Any new auth model should define where it is created and where service-layer authorization occurs.

## Exa Source Links
- OWASP Authentication Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Authentication_Cheat_Sheet.html
- OWASP Authorization Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html
- OWASP ASVS V4 Access Control: https://github.com/OWASP/ASVS/blob/master/4.0/en/0x12-V4-Access-Control.md
- OWASP API1:2023 Broken Object Level Authorization: https://owasp.org/API-Security/editions/2023/en/0xa1-broken-object-level-authorization/
- OWASP API2:2023 Broken Authentication: https://owasp.org/API-Security/editions/2023/en/0xa2-broken-authentication/
- OWASP API3:2023 Broken Object Property Level Authorization: https://owasp.org/API-Security/editions/2023/en/0xa3-broken-object-property-level-authorization/
- OWASP API5:2023 Broken Function Level Authorization: https://owasp.org/API-Security/editions/2023/en/0xa5-broken-function-level-authorization/
- OWASP REST Security Cheat Sheet for JWT and access control: https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html
- OWASP OAuth2 Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/OAuth2_Cheat_Sheet.html
- OWASP WSTG Testing JSON Web Tokens: https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/06-Session_Management_Testing/10-Testing_JSON_Web_Tokens
