# Outbound Egress

## Load When

Load this when the service dials something new — a provider API, a webhook
delivery, or a destination influenced by a caller, a tenant, or config.

## Decide

- `internal/infra/httpclient` builds one client per fixed provider authority.
  Scheme and host are enforced on every request, redirects are refused with
  `http.ErrUseLastResponse`, and caller-supplied correlation headers are
  stripped. Construction requires provider-wide header, decoded-body, and
  request-concurrency ceilings; a smaller non-streaming operation may also own
  its deadline and decoded-body limit.
- Its address check runs in the dialer's `ControlContext`, after resolution.
  That placement is the guarantee — it is what survives DNS rebinding, and it is
  why `Proxy` is set to nil: a proxy resolves and dials on the client's behalf,
  past the gate.
- `NewExternalHTTPS` is the default. `NewPrivateHTTPS` is only for an existing
  deployment-owned private route and requires that platform's private DNS
  suffix; there is deliberately no default suffix and no plaintext variant.
- Provider adapters or official SDKs choose the limits and retry eligibility.
  The shared client enforces the accepted non-streaming ceilings but does not
  invent provider values, parse responses, or retry.
- A per-request, caller-supplied destination is outside what this client
  provides, because pinning is what makes its guarantee meaningful. Arbitrary
  destinations are a new decision — a registered endpoint identifier the service
  resolves itself, or an accepted egress design carrying its own destination
  policy and proof — not a configuration value on the existing client.
- `internal/infra/grpcclient` is the same shape for native gRPC: a fixed target
  with finite per-call header and message bounds.

## Reject

- `http.Get`, `http.DefaultClient`, or a bare `&http.Client{Timeout: ...}` on a
  fixed-target request path: it follows redirects, honors `HTTP_PROXY`, and
  carries no address gate. A timeout limits blast radius; it does not authorize
  a destination.
- Validating a destination at registration and dialing it later: the name can
  resolve somewhere else by then, which is the reason the gate sits at dial.

## Prove

Use a focused client test that asserts refused scheme, authority, resolved
address, redirect, and stale correlation headers. Provider response limits are
proved beside their parser or SDK. Where safe egress depends on network policy outside the diff, name
it as a residual risk rather than implying the application check covers it.
