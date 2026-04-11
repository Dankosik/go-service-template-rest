# SSRF, Outbound, And Redirect Safety Review

## When To Load
Load this when changed Go code makes outbound HTTP or network calls based on user, tenant, partner, webhook, import, image, URL preview, redirect, callback, or config input. Also load it when redirect policy or custom `http.Client` behavior changes.

## Attacker Preconditions
- The attacker can influence a URL, scheme, host, port, path, webhook target, proxy setting, redirect location, DNS name, or upstream service selection.
- The service can reach internal networks, metadata services, private admin ports, or privileged partner APIs that the attacker cannot reach directly.
- The outbound response, timing, status, side effect, or cost can be observed or abused.

## Review Signals
- `http.Get`, `http.Post`, `http.DefaultClient`, or a zero-timeout client is used on security-sensitive outbound paths.
- User-provided full URLs are accepted when only a host, domain ID, or pre-registered endpoint is needed.
- Scheme, host, port, redirect chain, DNS resolution, or post-resolution IP class is not constrained.
- Redirects are followed by default after initial validation.
- Private, loopback, link-local, multicast, or cloud metadata ranges are not blocked when arbitrary external destinations are allowed.
- Response body size and content type are unchecked for remote imports or previews.
- Outbound errors leak internal topology or raw upstream responses to clients.

## Bad Finding Examples
- "Possible SSRF."
- "Validate URL."
- "Use a custom HTTP client."

These do not explain which destination becomes reachable or which client behavior makes the bypass possible.

## Good Finding Examples
- "[high] [go-security-review] internal/app/avatar.go:66 Axis: SSRF And Outbound Safety; `FetchAvatar` accepts a full request URL and calls `http.Get` with the default redirect policy. A tenant user can point the service at an internal admin endpoint or a trusted external URL that redirects internally. Require pre-registered HTTPS origins or an allowlisted host ID, use a custom client with timeout, disable or revalidate redirects, and reject private or metadata IP resolutions before the request."
- "[medium] [go-security-review] internal/app/webhook.go:112 Axis: SSRF And Redirect Safety; the webhook validator checks the initial hostname but delivery follows redirects without rechecking the target. A partner-controlled endpoint can pass registration and later redirect to a private address. Set `CheckRedirect` to reject redirects or re-run the same scheme/host/IP policy on every redirect target."

## Smallest Safe Correction
- Prefer pre-registered webhook endpoint IDs or allowlisted origin names over arbitrary per-request URLs.
- Require explicit schemes, usually HTTPS, and reject unsupported schemes.
- Use a dedicated `http.Client` with `Timeout`, request context propagation, and `CheckRedirect`.
- Revalidate or reject redirects. Treat every redirect target as a new outbound decision.
- Resolve hostnames and reject private, loopback, link-local, multicast, and cloud metadata IPs when arbitrary external egress is allowed. If this policy is hard to implement locally, escalate to security design.
- Bound response size with a limiting reader and validate content type before processing.
- Keep network egress controls in mind; application checks do not replace network policy for high-risk SSRF surfaces.

## Validation Ideas
- Unit-test scheme, host, port, redirect, and private-IP rejection using a local `httptest.Server`.
- Test that context cancellation and client timeout are honored.
- Test that response size limits stop processing before parsing or storing remote content.
- Add regression tests for webhook registration and delivery separately, because time-of-registration checks are not enough.

## Repo-Local Anchors
- The repo's HTTP server code already treats timeouts and request limits as first-class config; outbound clients should have equally explicit budgets when introduced.
- API contract changes involving webhook URL acceptance, callback registration, or error semantics should be handed off to API/security spec work when local review cannot infer policy.

## Exa Source Links
- OWASP SSRF Prevention Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html
- OWASP API7:2023 Server Side Request Forgery: https://owasp.org/API-Security/editions/2023/en/0xa7-server-side-request-forgery/
- Go `net/http` package docs: https://pkg.go.dev/net/http
