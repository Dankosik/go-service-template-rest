# SSRF, Outbound, And Redirect Safety Review

## Behavior Change Thesis
When loaded for symptom "caller-influenced data selects an outbound network target," this file makes the model review scheme, host, redirect, DNS, IP class, timeout, and response bounds instead of likely mistake saying only "validate URL" or "use a custom client."

## When To Load
Load this when changed Go code makes outbound HTTP or network calls based on user, tenant, partner, webhook, import, image, URL preview, redirect, callback, or config input. Also load it when redirect policy, proxy policy, resolver behavior, or custom `http.Client` behavior changes.

## Decision Rubric
- Name which destination becomes reachable that the attacker could not reach directly: metadata service, loopback, private network, admin port, partner API, or internal DNS name.
- Prefer pre-registered endpoint IDs, allowlisted origins, or service-owned target selection over arbitrary per-request URLs.
- Require explicit schemes, usually HTTPS, and reject unsupported schemes.
- Use a dedicated `http.Client` with `Timeout`, request context propagation, and `CheckRedirect`.
- Reject or revalidate every redirect target; initial host validation is not enough.
- When arbitrary external egress is allowed, require host resolution plus private, loopback, link-local, multicast, and metadata IP rejection close to the request.
- Bound response body size and validate content type before processing remote imports or previews.
- Escalate when safe SSRF policy depends on network egress controls or platform routing not visible in the diff.

## Imitate
```text
[high] [go-security-review] internal/app/avatar.go:66
Issue: Axis: SSRF And Outbound Safety; `FetchAvatar` accepts a full request URL and calls `http.Get` with the default redirect policy.
Impact: A tenant user can point the service at an internal admin endpoint or a trusted external URL that redirects internally.
Suggested fix: Require pre-registered HTTPS origins or an allowlisted host ID, use a custom client with timeout, disable or revalidate redirects, and reject private or metadata IP resolutions before the request.
Reference: outbound fetch boundary.
```

Copy this shape when both arbitrary target selection and redirect behavior matter.

```text
[medium] [go-security-review] internal/app/webhook.go:112
Issue: Axis: SSRF And Redirect Safety; the webhook validator checks the initial hostname but delivery follows redirects without rechecking the target.
Impact: A partner-controlled endpoint can pass registration and later redirect to a private address.
Suggested fix: Set `CheckRedirect` to reject redirects or re-run the same scheme, host, and IP policy on every redirect target.
Reference: webhook delivery target policy.
```

Copy this shape when registration-time validation does not protect delivery-time behavior.

## Reject
```text
Issue: Possible SSRF.
```

Reject because it does not say which network target becomes reachable or which client behavior allows it.

```text
Suggested fix: Parse the URL and make sure it has a host.
```

Reject because syntactic URL parsing does not constrain redirects, DNS, private IPs, metadata addresses, or timeouts.

## Agent Traps
- Do not treat allowlisting the initial hostname as sufficient when redirects are followed.
- Do not forget DNS rebinding or post-resolution IP class checks when arbitrary hostnames are allowed.
- Do not rely on application checks alone for high-risk egress; name network policy as a residual risk or escalation when needed.
- Do not turn every outbound call into SSRF; prove caller influence over target selection or redirect behavior.
- Do not return raw upstream errors or bodies to clients when they can expose internal topology.

## Validation Shape
- Use `httptest.Server` or a fake transport to test scheme, host, port, redirect, and private-IP rejection.
- Test registration and delivery separately for webhooks or callbacks.
- Test context cancellation and client timeout behavior.
- Test response size limits before parsing, storing, or returning remote content.

## Repo-Local Anchors
- The repo's HTTP server config treats timeouts and request limits as first-class; outbound clients should have equally explicit budgets when introduced.
- API contract changes involving webhook URL acceptance, callback registration, or error semantics should be handed off when local review cannot infer policy.
