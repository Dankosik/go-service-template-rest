# Input, Output, Injection, And SSRF Control Examples

## When To Load
Load this when untrusted request fields, JSON decoding, query parameters, file/path/URL inputs, SQL or other interpreters, outbound HTTP, webhooks, response encoding, or sanitized error behavior are in scope.

## Selected Controls
- Validate at the earliest trusted boundary with both syntactic checks, such as type, length, range, enum, and format, and semantic checks, such as state and ownership.
- Use allowlists for structured values such as schemes, hosts, ports, methods, enum values, media types, sort fields, and pagination limits.
- For mutable JSON requests, require strict decoding where the API contract expects fixed fields: body size limit, content type check, unknown-field rejection, duplicate-key policy when security relevant, trailing-token rejection, and numeric precision choice.
- Use safe interpreter APIs. For SQL, pass parameters as `database/sql` arguments and keep dynamic identifiers from code-owned allowlists.
- For SSRF-prone features, require allowed remote origins, schemes, ports, redirect behavior, DNS/IP resolution policy, cloud metadata blocking, response size limit, media type validation, timeout, and no raw internal response relay.
- Sanitize output and error detail. Return stable problem responses without stack traces, SQL errors, credentials, filesystem paths, or internal addresses.

## Rejected Controls
- Reject denylist-only validation such as blocking `localhost`, `127.0.0.1`, or `' OR 1=1` while allowing arbitrary URLs or query fragments.
- Reject "validation later in business logic" when an interpreter or outbound network call happens first.
- Reject `fmt.Sprintf` or string concatenation for SQL values.
- Reject automatic binding of request bodies into domain or persistence objects when sensitive/internal fields exist.
- Reject following redirects from third-party APIs or user-provided URLs unless the redirect target is explicitly allowed.
- Reject logging raw request bodies, tokens, secrets, or user-supplied error strings without sanitization.

## Fail-Closed Examples
- Unknown JSON field in a mutable protected request returns a client error instead of silently changing internal state.
- Oversized body returns `413` and no handler side effect.
- Unsupported media type returns `415` and no decode attempt.
- User-provided URL with a disallowed scheme, host, port, private IP, loopback target, link-local target, metadata endpoint, or redirect target is rejected before dialing.
- SQL identifier not in the code-owned allowlist returns validation failure rather than falling back to raw string interpolation.

## Testable Requirements
- Given trailing JSON tokens, unknown mutable fields, duplicate security-relevant keys, invalid UTF-8, or overly large numbers, the endpoint rejects or handles according to a documented strictness policy.
- Given a malicious SQL value, the query uses parameters and does not change the query structure.
- Given an unapproved sort field, table name, or column name, the service rejects rather than interpolating it.
- Given URL payloads for loopback, link-local metadata, private network ranges, userinfo tricks, encoded hosts, redirects, and non-HTTP schemes, the outbound client refuses the request.
- Given an internal error, the API response is stable and sanitized while server-side logs retain only bounded diagnostic fields.

## Repo-Local Anchors
- `internal/infra/http/middleware.go` already includes `RequestFramingGuard`, `RequestBodyLimit`, and `SecurityHeaders`; new request parsing requirements should preserve those transport protections.
- `internal/infra/http/problem.go` is the local problem-response surface. Security requirements should call for sanitized details when errors cross the API boundary.
- `internal/infra/postgres` and `sqlc` generated query paths are preferred over hand-built SQL for service-owned queries.

## Exa Source Links
- OWASP Input Validation Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html
- OWASP Injection Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Injection_Prevention_Cheat_Sheet.html
- OWASP API7:2023 Server Side Request Forgery: https://owasp.org/API-Security/editions/2023/en/0xa7-server-side-request-forgery/
- OWASP Error Handling Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Error_Handling_Cheat_Sheet.html
- Go `encoding/json` security considerations: https://pkg.go.dev/encoding/json
- Go `net/http` documentation for request body limiting and client/server APIs: https://pkg.go.dev/net/http
- Go avoiding SQL injection risk: https://go.dev/doc/database/sql-injection
