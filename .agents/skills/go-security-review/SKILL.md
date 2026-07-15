---
name: go-security-review
description: "Use when changed Go crosses a trust boundary or affects authn, authz, tenant or object isolation, browser sessions, CORS/CSRF, tokens, credentials, injection, SSRF, paths, secrets, or abuse controls; Own security-policy enforcement defects; Skip when policy is unset or the primary issue is business meaning, chi wiring, or observability privacy."
---

# Go Security Review

Load the [shared specialist contract](../specialist-contract.md) for selection, evidence, return, and handoff. Review trust boundaries, authn/authz/isolation, session/token/CSRF/CORS, secrets, abuse, and auditability. Load [the reference selector](references/index.md) only when its pressure changes the result, and another reference only for an independent pressure. Escalate changed trust or access policy to `go-security-spec`. Return the owned decision or evidence-backed finding, forced consequence, and focused proof; stop rather than inventing another owner’s policy.
