# Client Channel

Use this reference when client construction or channel policy can change
identity, reachability, replay, or resource ownership.

## Decision

- Preserve the caller's resolver target and require explicit transport
  credentials. Plaintext is an explicit accepted boundary using
  `insecure.NewCredentials`, not an implicit fallback.
- Treat `grpc.NewClient` construction as lazy channel configuration, not
  connectivity or readiness proof. Separate any required startup connectivity
  gate from the constructor contract.
- Give each target one deliberate long-lived `ClientConn` owner, share it among
  generated clients, and close it with the owning dependency lifecycle.
- Keep deadlines at the operation or stream contract. Select `WaitForReady`
  only when queueing until readiness is required and bounded by the caller's
  deadline.
- Treat retry as method policy. grpc-go transparent retry and resolver-provided
  service config may still exist without a template default; configured retry
  requires replay safety, attempts/backoff inside one budget, and attempt
  observability.
- Bind TLS roots and expected server identity to the target. Preserve hostname
  verification through every wrapper and deployment route.

## Review

Report per-call connections, nil or implicit credentials, constructor success
claimed as reachability, stripped resolver schemes, hidden `Close`, a global
retry/`WaitForReady` default without per-method policy, deadlines that cannot
cover all attempts, or TLS tests that prove only certificate parsing.

## Proof

Prove lazy construction with an unreachable target, exact target preservation,
shared connection ownership, explicit plaintext rejection/acceptance, trusted
CA success, hostname mismatch failure, cancellation/deadline propagation, and
live send/receive/metadata limits. Add retry or `WaitForReady` scenarios only
when they are part of the accepted client contract.

Vendor authority:
[grpc-go client anti-patterns](https://github.com/grpc/grpc-go/blob/master/Documentation/anti-patterns.md),
[Authentication](https://grpc.io/docs/guides/auth/),
[Retry](https://grpc.io/docs/guides/retry/), and
[Wait-for-Ready](https://grpc.io/docs/guides/wait-for-ready/).
