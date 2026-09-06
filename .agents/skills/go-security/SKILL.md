---
name: go-security
description: "Attacker paths. Use when identity, authorization, tenancy, tokens, secrets, injection, SSRF, abuse, or another trust boundary changes what an attacker can reach."
metadata:
  invocation: model
  kind: method
---

# Go Security

Security work walks **attacker paths** from trust boundary through principal,
authority, asset, enforcement point, attacker action, and observable denial.

`boundary -> principal -> enforcement -> action -> failure -> negative proof`

For a delegated Decision or Review, or when the active artifact requires its
result interface, load the
[shared specialist contract](../../contracts/specialist-contract.md).
From every caller-controlled entrypoint to an asset or observable denial, build
`AttackerPath{boundary, principal, asset, enforcement, action, failure,
denial_proof}` across route exposure, verified identity, objects reached,
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
envelope; no findings still requires the focused deny proof. Load [access
control](../../../docs/universal-disciplines/auth-access-control/SKILL.md) when
the credential, permission, or revocation mechanism itself is open.
