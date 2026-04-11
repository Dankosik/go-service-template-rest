# HTTP Security And Generated Error Contract

## Operation Security Decision Marker

Every OpenAPI operation must carry an explicit security decision.

Preferred extension shape:

```yaml
x-security-decision:
  exposure: public
  rationale: sample endpoint for template smoke checks
```

Allowed exposure values for this task:

- `public`: intentionally public business or sample endpoint.
- `operational-private-required`: operational endpoint that may exist in the contract but needs private network, internal listener, or future real auth before internet exposure.
- `protected`: real auth is required; must use a real OpenAPI security scheme and 401/403 Problem responses.
- `blocked`: endpoint must not be implemented until a security spec exists.

Current expected decisions:

- `/api/v1/ping`: public sample endpoint.
- `/health/live`: public or platform-facing system endpoint; liveness must remain process-only.
- `/health/ready`: platform-facing system endpoint; readiness may expose only generic ready/not-ready state.
- `/metrics`: operational-private-required.

Do not add `protected` without a real auth design.

## Contract Test Rules

The OpenAPI contract test should fail when:

- an operation has no `x-security-decision`,
- an operation is `protected` but has no real security requirement,
- an operation is `protected` but does not define 401 and 403 Problem responses,
- an operation declares `security: []` without an explicit public/operational decision,
- `/metrics` is marked as ordinary public business API.

## Generated Chi Error Contract

Generated chi wrapper errors must use the same external response policy as strict request errors:

- client detail is generic malformed-request text,
- response content type is `application/problem+json`,
- response status is 400,
- request id is included when available,
- logs include sanitized error class and request id,
- logs do not include raw attacker-controlled parse details.

If the current OpenAPI contract has no path/query/body shape that triggers generated wrapper parse errors naturally, test the local options/error-handler helper directly.

## Browser Endpoint Guidance

Docs must state that CORS fail-closed is not CSRF protection.

Future browser-callable endpoints need a separate decision covering:

- allowed origins,
- allowed headers and methods,
- credential mode,
- cookie `Secure`, `HttpOnly`, `SameSite`, Path, and Domain attributes,
- CSRF control such as origin policy, token policy, or `http.CrossOriginProtection`,
- negative tests.

No browser runtime security should be implemented in this follow-up.
