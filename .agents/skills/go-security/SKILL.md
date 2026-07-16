---
name: go-security
description: "Security: Use when trust boundaries, identity, authorization, tenant isolation, browser/token controls, secrets, abuse, injection, or SSRF needs a decision, or when changed Go crosses those boundaries. Own security policy and conformance review; Skip when the primary concern is non-security API, data, reliability, or implementation placement."
---

# Go Security

Load the [shared specialist contract](../specialist-contract.md). Keep identities, access rules, enforcement points, tenant/object isolation, input/output controls, secret handling, abuse bounds, and negative proof coherent.

## Choose The Branch

- **Decision** — select when security policy is absent or changing. Load the [decision selector](references/decision/index.md) for one result-changing threat. Complete when trust boundaries, rules, enforcement, failure semantics, proof, and blockers are explicit.
- **Review** — select when changed code must conform to accepted security policy. Load the [review selector](references/review/index.md) for the concrete attacker path. Complete when every affected trust boundary is dispositioned as a finding or no finding with the smallest fail-closed correction and focused proof; missing policy stays in the decision branch.

Hand non-security contract semantics to `go-api-contract`, data authority to `go-data-architecture`, and placement to `go-implementation-ownership`.
