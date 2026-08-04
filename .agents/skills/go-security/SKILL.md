---
name: go-security
description: "Security: Use for trust boundaries, identity, authorization, tenant isolation, tokens, secrets, injection, SSRF, or review. Own policy; Skip non-security API, data, reliability, or placement."
---

# Go Security

Security work walks **attacker paths**: every reachable route from an attacker's position through identity, authority, asset, enforcement point, and attacker action to an observable failure — and silence is never evidence that a path is closed.

`trust boundary -> principal -> enforcement point -> attacker action -> observable failure -> negative proof`

The system fails closed: missing identity, ambiguous tenant, or absent policy denies rather than guesses. A control proves nothing until a negative test exercises its deny path, because allow-path tests pass equally well against a bypassed control — so every control this work accepts or changes carries one.

Load the [shared specialist contract](../specialist-contract.md). Reconstruct every reachable attacker path across the affected surfaces: the declared exposure of each route, the verified principal, the objects reached with it, outbound destinations, secrets, and the bounds on caller-driven work.

Decide against what this service already enforces rather than a generic threat list. `internal/infra/oidcjwt` verifies bearer tokens, `api/openapi/service.yaml` requires one by default, `internal/infra/httpclient` pins outbound destinations, and `internal/config` separates secret from non-secret configuration. The [reference selector](references/index.md) covers only pressures where those controls or a current standard override the obvious answer; load one reference by default and another only for an independent pressure.

## Choose The Branch

- **Decision** — select when security policy is absent or changing. Complete when every reachable path reaches a shared Decision disposition with fail-closed behavior and focused negative proof.
- **Review** — select when changed code must conform to accepted security policy. Follow every reachable path into the shared finding envelope, naming any boundary or proof blocker; no finding requires focused negative proof. Missing policy returns to the named Security Decision owner.

Hand non-security contract semantics to `go-api-contract`, data authority to `go-data-architecture`, and placement to `go-implementation-ownership`. Load [`auth-access-control`](../../../docs/universal-disciplines/auth-access-control/SKILL.md) when the decision turns on credential lifecycle, session or token mechanism, permission-model shape, or how fast a revocation must take effect: it forces the mechanism to be derived from the required revocation window and permissions to be enumerable data, instead of role checks behind a long-lived token whose claims are treated as current.
