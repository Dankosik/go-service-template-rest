# Server Protocol

Use this reference when server composition can change which status, metadata,
or standard protocol behavior reaches the client.

## Decision

- Inventory unary and streaming chains from outermost interceptor through the
  generated handler. Give each layer one concern and require parity unless a
  protocol-specific exception is accepted.
- Classify error provenance before mapping: grpc-go/library, standard service,
  trusted service policy, application/domain handler, and unexpected failure.
  Preserve deliberate canonical protocol or policy statuses at their owner;
  sanitize handler and unexpected errors at their own boundaries.
- Keep authentication and authorization rejection outside a handler-only
  sanitizer, or give the policy boundary an explicit safe mapping. A rejected
  policy must prevent handler entry and retain `UNAUTHENTICATED` versus
  `PERMISSION_DENIED`.
- Register every service before serving. Apply health, reflection, and other
  standard services only under explicit auth, admission, logging, and
  disclosure decisions while preserving their standard status semantics.
- Derive method identity from registered descriptors or another bounded
  authority. Peer-controlled method paths, metadata, payloads, and raw error
  text are not trusted labels or caller detail.

## Review

Report one global sanitizer that converts trusted or standard statuses to
`INTERNAL`, unary/stream chain drift, policy slices that never reach bootstrap,
recovery that leaks panic detail, auth rejection followed by handler entry,
unknown-health behavior rewritten by application mapping, or services
registered after `Serve`.

## Proof

Use generated clients against the composed server. Exercise unary and streaming
policy rejection, raw handler status, mapped domain error, context error,
panic, admission rejection, valid/invalid metadata, and standard health
`Check`/`Watch` behavior. Assert final code and safe detail, response metadata,
handler-entry oracle, completion log status, and absence of secret canaries.

Vendor authority:
[Status codes](https://grpc.io/docs/guides/status-codes/),
[Authentication](https://grpc.io/docs/guides/auth/),
[Health checking](https://grpc.io/docs/guides/health-checking/), and
[grpc-go server options](https://pkg.go.dev/google.golang.org/grpc#ServerOption).
