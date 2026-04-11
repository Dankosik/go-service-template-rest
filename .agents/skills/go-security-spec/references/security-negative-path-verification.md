# Security Negative-Path Verification Examples

## When To Load
Load this when turning security requirements into validation obligations before coding, when a spec needs negative-path tests, when auth matrices are involved, or when abuse, tenant-crossing, JWT tampering, injection, SSRF, secret leakage, or resource exhaustion needs proof.

## Selected Controls
- Make every security decision produce at least one positive-path and one negative-path proof obligation.
- Use authorization matrices for role/function/data combinations when access rules are nontrivial.
- Test authentication failures separately from authorization failures: missing credentials, malformed credentials, invalid signature, wrong issuer/audience, expired token, insufficient scope, wrong tenant, and wrong object.
- Test BOLA with two accounts or tenants, multiple HTTP methods, object IDs in path/query/header/body, and bulk/list endpoints.
- Test property-level authorization by adding forbidden mutable fields and checking response schemas for sensitive fields.
- Test SSRF and outbound access with loopback, private, link-local, metadata, userinfo, redirect, non-HTTP scheme, large response, slow response, and invalid media type cases.
- Test abuse resistance with large bodies, high concurrency, batching, expensive operations, repeated auth attempts, and provider-cost triggers.
- Tie code/config changes to repo gates when applicable: unit/integration tests, `make go-security`, `make secrets-scan`, and contract checks for OpenAPI behavior.

## Rejected Controls
- Reject "covered by integration tests" unless the tests include negative security cases.
- Reject relying only on scanners for authorization, tenant isolation, business-flow abuse, or privacy rules.
- Reject testing only one HTTP method when the resource supports multiple methods or method tampering is plausible.
- Reject proof that only checks status code when sensitive data, side effects, cache writes, telemetry leakage, or audit events are the real risk.
- Reject manual-only verification for stable authorization matrices when the matrix can be encoded in deterministic tests.

## Fail-Closed Examples
- If auth context creation fails, tests assert no handler/service/repository side effect ran.
- If authorization policy panics or returns unknown, tests assert deny and sanitized error behavior.
- If cache, secret, or policy dependency fails, tests assert the service uses the documented fail-closed path.
- If request parsing rejects unknown fields or oversized bodies, tests assert no partially decoded state is used.
- If outbound URL validation rejects a target, tests assert no dial happened.

## Testable Requirements
- For each protected endpoint: missing auth returns `401`; valid auth with wrong role/scope/tenant/object returns `403` or approved concealment; no side effect occurs.
- For each tenant-scoped repository call: tests prove tenant condition is present and cross-tenant IDs do not return rows.
- For each mutable API request: forbidden fields are rejected or ignored only by explicit contract, and response fields are least-privilege.
- For each JWT-bearing flow: altered payload, `alg: none`, wrong audience, wrong issuer, expired token, future `nbf`, unknown key ID, and sensitive payload checks are covered.
- For each SSRF-capable feature: disallowed destinations and redirect targets are rejected before dial; allowed targets have timeouts, response-size limits, and media-type checks.
- For each abuse-sensitive flow: tests or benchmark-style probes cover rate/concurrency/body/batch limits and expected `429`, `413`, or safe degradation behavior.
- For secrets and privacy: tests prove secret-like config rejection, redaction of secret-like parse errors, no secrets in logs, and no sensitive data in URL/query strings.

## Repo-Local Anchors
- `internal/infra/http/router_test.go` includes fail-closed CORS preflight, security header, request framing, request ID, and body-limit tests.
- `internal/config/config_test.go` includes secret policy and raw-secret redaction tests.
- `Makefile` provides `go-security`, `secrets-scan`, `openapi-check`, `test`, and `test-race` proof commands.
- `scripts/ci/required-guardrails-check.sh` tracks required guardrails including security policy and CI checks.

## Exa Source Links
- OWASP Authorization Testing Automation Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Testing_Automation_Cheat_Sheet.html
- OWASP WSTG API Broken Object Level Authorization: https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/12-API_Testing/02-API_Broken_Object_Level_Authorization
- OWASP WSTG Testing JSON Web Tokens: https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/06-Session_Management_Testing/10-Testing_JSON_Web_Tokens
- OWASP WSTG Testing for Bypassing Authorization Schema: https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/05-Authorization_Testing/02-Testing_for_Bypassing_Authorization_Schema
- OWASP API1:2023 Broken Object Level Authorization: https://owasp.org/API-Security/editions/2023/en/0xa1-broken-object-level-authorization/
- OWASP API4:2023 Unrestricted Resource Consumption: https://owasp.org/API-Security/editions/2023/en/0xa4-unrestricted-resource-consumption/
- OWASP API7:2023 Server Side Request Forgery: https://owasp.org/API-Security/editions/2023/en/0xa7-server-side-request-forgery/
