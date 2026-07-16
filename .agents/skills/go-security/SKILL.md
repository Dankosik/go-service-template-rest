---
name: go-security
description: "Security: Use when trust boundaries, identity, authorization, tenant isolation, browser/token controls, secrets, abuse, injection, or SSRF needs a decision, or when changed Go crosses those boundaries. Own security policy and conformance review; Skip when the primary concern is non-security API, data, reliability, or implementation placement."
---

# Go Security

Load the [shared specialist contract](../specialist-contract.md). Reconstruct every reachable attacker path from affected trust-boundary surfaces in routes, handlers, identities, objects, stores, outbound calls, secrets, and abuse controls. Follow each path through identity, authority, asset, enforcement point, attacker action, and observable failure; silence is never evidence that a path is closed.

## Choose The Branch

- **Decision** — select when security policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing threat. Complete when shared Decision dispositions cover every reachable path with fail-closed behavior and focused negative proof.
- **Review** — select when changed code must conform to accepted security policy. Load the [review selector](references/review/index.md) for the concrete attacker path. Follow every reachable path into the shared finding envelope, naming any outside boundary or proof blocker; no finding requires focused negative proof. Missing policy ends this run with a named Security Decision handoff; conformance Review begins separately after acceptance.

Hand non-security contract semantics to `go-api-contract`, data authority to `go-data-architecture`, and placement to `go-implementation-ownership`.
