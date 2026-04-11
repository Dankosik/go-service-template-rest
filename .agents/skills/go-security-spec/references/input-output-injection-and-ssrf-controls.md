# Input, Output, Injection, And SSRF Controls

## Behavior Change Thesis
When loaded for untrusted input, interpreter, outbound network, or error-output requirements, this file makes the model choose strict parser, allowlist, SSRF dial, and sanitized response requirements instead of likely mistake: denylist validation, late validation, string-built queries, or raw internal error relay.

## When To Load
Load this when the spec touches JSON decoding, query parameters, file/path/URL fields, SQL or other interpreters, outbound HTTP, webhooks, response encoding, or client-visible error detail.

## Decision Rubric
- Validate before the first interpreter, network dial, persistence write, cache key construction, or side effect.
- For fixed-shape JSON mutation, require body size limit, media-type check, unknown-field policy, duplicate-key policy when security relevant, trailing-token rejection, numeric precision choice, and forbidden-field handling.
- Use allowlists for schemes, hosts, ports, HTTP methods, media types, enum values, sort fields, table/column identifiers, and pagination limits.
- For SQL, pass values through `database/sql` or generated-query parameters. Dynamic identifiers must come from code-owned allowlists, not user strings.
- For SSRF-prone behavior, define allowed origins, scheme and port policy, DNS/IP resolution policy, loopback/private/link-local/metadata blocking, redirect behavior, timeout, response size, media-type validation, and no raw internal response relay.
- Sanitize output and problem responses. Client-visible errors should not include stack traces, SQL errors, credentials, filesystem paths, internal hosts, or raw user-supplied error strings.

## Imitate
- "For mutable JSON requests, unknown fields and trailing tokens are rejected before domain-object construction; forbidden mutable fields use the property-authorization rule, not struct binding side effects." Copy the parse-before-domain boundary.
- "For user-selected sorting, only code-owned column aliases are accepted; unapproved values return validation failure rather than falling back to string interpolation." Copy the dynamic-identifier allowlist.
- "For user-provided callback URLs, reject loopback, link-local, private ranges, metadata endpoints, non-HTTP schemes, userinfo tricks, and redirects to disallowed targets before dialing." Copy the pre-dial SSRF policy.

## Reject
- "Block `localhost` and `127.0.0.1`." This denylist misses encoded hosts, DNS rebinding, link-local targets, private ranges, metadata endpoints, redirects, and IPv6 forms.
- "Validate in business logic after fetch." The unsafe interpreter or network side effect has already happened.
- "Use `fmt.Sprintf` for a trusted query fragment." Trust depends on a code-owned allowlist, not the absence of scary characters.
- "Return the upstream error to the client." This risks leaking internal addresses, credentials, query detail, or provider behavior.

## Agent Traps
- Do not conflate strict JSON syntax with semantic authorization. Parser strictness does not prove the caller may set the field.
- Do not treat generated handlers or request structs as an input-security control unless their rejection behavior is explicit.
- Do not claim SSRF is solved by timeout alone. Timeout limits blast radius; it does not authorize a target.

## Validation Shape
- Parser cases: unknown mutable fields, duplicate security-relevant keys, trailing tokens, invalid UTF-8, oversized bodies, unsupported media types, and overly large numbers.
- Injection cases: malicious SQL values remain parameters, and unapproved dynamic identifiers are rejected.
- SSRF cases: loopback, private, link-local, metadata, userinfo, encoded host, redirect, non-HTTP scheme, oversized response, slow response, and invalid media type fail before dial or persist.
- Output cases: internal errors produce stable sanitized problem responses while logs retain only bounded diagnostic fields.

## Repo-Local Anchors
- `internal/infra/http/middleware.go` includes `RequestFramingGuard`, `RequestBodyLimit`, and `SecurityHeaders`; new parsing requirements should preserve those transport protections.
- `internal/infra/http/problem.go` is the local problem-response surface. Security requirements should call for sanitized details when errors cross the API boundary.
- `internal/infra/postgres` and `sqlc` generated query paths are preferred over hand-built SQL for service-owned queries.
