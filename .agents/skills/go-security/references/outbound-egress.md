# Outbound Egress

## Load When

Load this when the service dials something new — a provider API, a webhook
delivery, or a destination influenced by a caller, a tenant, or config.

## Decide

- `internal/infra/httpclient` is the outbound transport. One client is built per
  fixed provider authority and pins it: scheme and host are enforced on every
  request, redirects are refused with `http.ErrUseLastResponse`, and response
  header and body sizes are capped, with an oversized body surfacing as
  `*ResponseTooLargeError`.
- Its address check runs in the dialer's `ControlContext`, after resolution.
  That placement is the guarantee — it is what survives DNS rebinding, and it is
  why `Proxy` is set to nil: a proxy resolves and dials on the client's behalf,
  past the gate.
- `TargetClass` is the decision. `ExternalHTTPS` permits only HTTPS resolving to
  a public address; `PrivateHTTP` permits plaintext only under the private DNS
  suffix the caller names, and there is deliberately no default suffix for it.
- `MaxConnsPerHost` is the bulkhead between one slow provider and everything
  else, not a tuning knob: unset, one dependency can absorb the whole in-flight
  allowance while requests that never touch it are shed.
- A per-request, caller-supplied destination is outside what this client
  provides, because pinning is what makes its guarantee meaningful. Arbitrary
  destinations are a new decision — a registered endpoint identifier the service
  resolves itself, or an accepted egress design carrying its own destination
  policy and proof — not a configuration value on the existing client.
- `internal/infra/grpcclient` is the same shape for native gRPC: a fixed target
  with finite per-call header and message bounds.

## Reject

- `http.Get`, `http.DefaultClient`, or a bare `&http.Client{Timeout: ...}` on a
  request path: it follows redirects, honors `HTTP_PROXY`, and carries no
  address gate and no response bound. A timeout limits blast radius; it does not
  authorize a destination.
- Validating a destination at registration and dialing it later: the name can
  resolve somewhere else by then, which is the reason the gate sits at dial.

## Prove

`httptest.Server` or a fake transport, asserting rejection at the boundary that
owns it: refused scheme, refused resolved address, refused redirect, and the
response-size limit. `internal/infra/httpclient/client_test.go` is where those
cases live. Where safe egress depends on network policy outside the diff, name
it as a residual risk rather than implying the application check covers it.
