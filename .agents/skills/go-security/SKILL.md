---
name: go-security
description: "Security: Use for identity, authorization, tenancy, tokens, secrets, injection, SSRF, or trust boundaries. Own policy; Skip placement."
metadata:
  invocation: model
  kind: method
---

# Go Security

Security work walks **attacker paths** from trust boundary through principal,
authority, asset, enforcement point, attacker action, and observable denial.

`boundary -> principal -> enforcement -> action -> failure -> negative proof`

Load the [shared specialist contract](../../contracts/specialist-contract.md). Reconstruct
every reachable path across route exposure, verified identity, objects reached,
outbound destinations, secrets, and caller-controlled work. Missing identity,
ambiguous tenant, or absent policy denies. Every accepted control needs focused
negative proof because an allow test also passes against a bypass.

Decide against existing owners: `internal/infra/oidcjwt` verifies tokens,
`api/openapi/service.yaml` declares default auth, `internal/infra/httpclient`
pins destinations, and `internal/config` separates secret inputs. Load the
[reference selector](references/index.md) for identity, exposure, interpreter
input, outbound destination, work amplification, or a secret sink.

For a **Decision**, disposition every reachable path with fail-closed behavior
and negative proof. For **Review**, follow each path into the shared finding
envelope; no findings still requires the focused deny proof.

Hand contract semantics to `go-api-contract`, data authority to
`go-data-architecture`, placement to `go-implementation-ownership`, and
credential, permission, or revocation mechanism to [access control](../../../docs/universal-disciplines/auth-access-control/SKILL.md).
