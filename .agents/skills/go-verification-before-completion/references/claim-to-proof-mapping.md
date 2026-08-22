# Claim To Proof Mapping

## Load When

Load when a positive claim names a gate, package/repository scope, readiness, or
current environment behavior and the evidence boundary is not obvious.

## Decide

Leaf gates are independent:

| Gate | Proves | Does not prove |
| --- | --- | --- |
| `make fmt-check` | Go formatting | behavior or analysis |
| `make lint` | mandatory static analysis | tests, deep lint, generated drift, Docker |
| `make test` | ordinary Go tests | race, integration, images, external state |
| matching `*-check` | one generated-contract surface | compatibility unless its breaking check also ran |
| `REQUIRE_DOCKER=1 make test-integration` | real integration paths in scope | production image lifecycle |
| `make migration-validate` | migration and exact-image runtime rehearsal | data recoverability or rollout safety |

`make build` is not implied by test targets. `make lint` omits `lint-deep`.
Security surfaces remain separate. Performance uses the benchmarking owner.

Go exits zero for `-run` matching no tests and packages with no tests. When one
named test carries the claim, verify a real run event as the OpenAPI contract
check does. Integration may skip without Docker unless `REQUIRE_DOCKER=1`; the
required integration command fails closed instead.

Cached unit results support unchanged code behavior, not a claim that Docker,
database, image, or other external state was exercised now. Use `-count=1` when
fresh environment interaction is the claim. A focused package or reproducer
does not prove unrelated packages, lint, drift, regression, or race behavior.
Another checkout/session receipt applies only after matching candidate identity
and unchanged preconditions.

## Prove

Return [Evidence Result
V1](../../../../docs/spec-first-workflow/interfaces/evidence-result-v1.md) with the
exact command, exit/result, cached/fresh state, and scope actually exercised.
Name the one remaining command or owner when evidence is narrower than the
claim.
