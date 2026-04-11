# API-Facing Security Semantics Examples

## When To Load
Load this when the security requirements touch REST/OpenAPI behavior, status codes, CORS, method policy, request size/media limits, problem responses, rate limits, retry/idempotency, management endpoints, or browser-facing headers.

## Selected Controls
- Document security requirements in the API contract or task-local API design: security scheme, per-operation security, protected/public endpoint distinction, supported methods, media types, request limits, and problem responses.
- Use `401` for missing or invalid authentication and `403` for authenticated callers without permission.
- Use `405` for disallowed methods with an `Allow` header when route semantics require it.
- Use `413` for configured request size limits, `415` for unsupported request media types, `406` when response negotiation is intentionally constrained, and `429` for rate or abuse limits.
- Use `503` plus retry guidance only when temporary unavailability is safe to disclose and clients may retry safely.
- Require idempotency-key semantics for retry-unsafe create or mutate operations: key scope, caller/tenant binding, TTL, conflict response, replay behavior, and storage boundary.
- For browser-callable APIs, define CORS allowlist policy and security headers. If CORS is unsupported, reject preflight explicitly.

## Rejected Controls
- Reject returning `200` with error bodies for security denials.
- Reject `404` as a blanket authz hiding strategy unless the resource-existence disclosure policy is explicit and consistently applied.
- Reject permissive CORS defaults such as wildcard origins on credentialed or sensitive APIs.
- Reject leaking tokens, API keys, passwords, or secrets in URLs.
- Reject undocumented retry semantics for non-idempotent operations.
- Reject raw stack traces, SQL errors, service names, internal hosts, or panic values in client-visible problem details.

## Fail-Closed Examples
- Missing authentication on a protected operation returns `401` and no handler side effect.
- Authenticated caller without object, function, property, or tenant permission returns `403` before repository mutation.
- CORS preflight for an unsupported cross-origin browser flow returns a denial response rather than implicitly enabling browser access.
- Method not in the operation allowlist returns `405`; it does not fall through to another handler or side effect.
- Rate limit exceeded returns `429` with bounded headers, not a best-effort background action.

## Testable Requirements
- Contract tests or handler tests prove `401`, `403`, `405`, `413`, `415`, `429`, and `503` behavior where those semantics are in scope.
- Tests prove problem responses use `application/problem+json` or the repo-approved format and omit sensitive implementation detail.
- CORS tests prove unsupported preflight fails closed and supported origins/methods/headers are exactly allowlisted.
- Idempotency tests prove same key/same payload replay is stable, same key/different payload conflicts, and keys are scoped to caller and tenant.
- Request limit tests prove oversized body rejection occurs before expensive parsing or side effects.

## Repo-Local Anchors
- `api/openapi/service.yaml` is the REST contract source of truth and currently defines shared problem responses and a `bearerAuth` component.
- `internal/infra/http/router.go` explicitly handles `NotFound`, `MethodNotAllowed`, `OPTIONS`, CORS preflight rejection, and request body limits.
- `internal/infra/http/router_test.go` already includes fail-closed CORS preflight and body-limit assertions that can guide future tests.

## Exa Source Links
- OWASP REST Security Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/REST_Security_Cheat_Sheet.html
- OWASP API4:2023 Unrestricted Resource Consumption: https://owasp.org/API-Security/editions/2023/en/0xa4-unrestricted-resource-consumption/
- OWASP API8:2023 Security Misconfiguration: https://owasp.org/API-Security/editions/2023/en/0xa8-security-misconfiguration/
- OWASP Error Handling Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Error_Handling_Cheat_Sheet.html
- Go `net/http` documentation: https://pkg.go.dev/net/http
