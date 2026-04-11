# Browser Session, CORS, And CSRF Review

## Behavior Change Thesis
When loaded for symptom "browser-callable cookie or credentialed cross-origin behavior changed," this file makes the model review CSRF, CORS, and cookie attributes as browser-state controls instead of likely mistake treating CORS as authorization or relying on server authz alone.

## When To Load
Load this when changed Go code touches browser session cookies, login/logout handlers, cookie-authenticated state changes, CSRF checks, CORS middleware or headers, preflight behavior, admin routes callable from browsers, or security headers tied to browser credential handling.

If the primary issue is chi router method or preflight mechanics, hand off or pair with `go-chi-review`. If the issue is object permission after a valid request reaches the service, load the authz reference instead.

## Decision Rubric
- Treat cookie-authenticated unsafe methods as CSRF-exposed unless the route has a token, Go's `http.CrossOriginProtection`, same-site policy, origin check, or another explicit anti-CSRF control before side effects.
- Do not rely on CORS as server-side authorization; non-browser clients and same-site requests are outside CORS protection.
- Reject arbitrary reflected credentialed CORS. Treat literal wildcard-plus-credentials as fail-closed misconfiguration in standards-compliant browsers, not automatically as cross-origin data theft, unless the framework reflects the caller's origin.
- Require exact origin allowlists for credentialed browser flows; pattern matching needs a strong reason and tight tests.
- Keep session cookies `Secure`, `HttpOnly`, and appropriately `SameSite`; constrain `Path` and `Domain` instead of broadening them casually.
- Treat state-changing `GET` as a review finding, not a place to lean on `SameSite=Lax`.
- Keep authz and CSRF distinct: a route can authorize the user correctly and still allow cross-site request forgery.

## Imitate
```text
[high] [go-security-review] internal/infra/http/admin_session.go:73
Issue: Axis: Browser Session And CSRF; the cookie-authenticated admin delete handler accepts `POST` without a CSRF token, origin check, or same-site enforcement before deleting users.
Impact: A malicious site can cause an authenticated admin's browser to send the state-changing request with the session cookie attached.
Suggested fix: Enforce the repo-approved CSRF control before the side effect, or make the route use bearer auth that browsers do not attach implicitly.
Reference: browser session boundary.
```

Copy this shape when a valid logged-in browser can be driven by a third-party site.

```text
[high] [go-security-review] internal/infra/http/router.go:118
Issue: Axis: Browser Session And CORS; the new CORS handler reflects any `Origin` while allowing credentials.
Impact: Any attacker-controlled origin can read credentialed API responses in a victim's browser when the victim has a session cookie.
Suggested fix: Use an exact allowlist of trusted origins, send `Vary: Origin` for dynamic origin responses, and reject reflected arbitrary origins or invalid wildcard-plus-credentials configuration.
Reference: credentialed CORS policy.
```

Copy this shape when CORS turns browser credentials into cross-origin readable data.

```text
[medium] [go-security-review] internal/app/session.go:42
Issue: Axis: Browser Session Cookie Safety; the session cookie is set without `Secure`, `HttpOnly`, or `SameSite`.
Impact: The cookie is easier to steal through script exposure, send over non-TLS in misconfigured environments, or attach to cross-site requests.
Suggested fix: Set `Secure` in non-local environments, keep `HttpOnly`, choose the approved `SameSite` mode for the flow, and keep `Path` and `Domain` constrained.
Reference: session cookie contract.
```

Copy this shape when cookie attributes weaken the browser session boundary.

## Reject
```text
Issue: CORS is too open, so anyone can call the API.
```

Reject because CORS controls browser read access, not whether a non-browser client can call the server.

```text
Suggested fix: Add an authz check instead of CSRF.
```

Reject when the route already authorizes the victim user but lacks protection against cross-site browser submission.

## Agent Traps
- Do not let "authenticated route" hide CSRF risk on cookie-attached unsafe requests.
- Do not flag permissive CORS on a non-browser, non-credentialed, public resource without naming a sensitive browser impact.
- Do not describe literal `Access-Control-Allow-Origin: *` with credentials as browser-readable sensitive data unless the implementation actually echoes the attacker-controlled origin.
- Do not assume `SameSite=Lax` protects state-changing `GET`.
- Do not ignore missing `Vary: Origin` when responses differ by origin.
- Do not absorb chi fallback or OPTIONS routing mechanics; name the security impact and hand off router-specific proof when needed.

## Validation Shape
- Add handler tests for cross-site `Origin`, missing CSRF token, invalid token, valid token, and method safety on cookie-authenticated routes.
- Add CORS tests proving untrusted origins are rejected, trusted origins are allowed exactly, dynamic origin responses set `Vary: Origin`, and invalid wildcard-plus-credentials behavior cannot accidentally turn into origin reflection.
- Add cookie tests for `Secure`, `HttpOnly`, `SameSite`, `Path`, and `Domain` on session creation and renewal.

## Repo-Local Anchors
- `internal/infra/http/router.go` owns HTTP middleware and explicit CORS preflight behavior.
- `internal/infra/http/router_test.go` has CORS and security-header patterns that should stay fail-closed when browser behavior changes.
