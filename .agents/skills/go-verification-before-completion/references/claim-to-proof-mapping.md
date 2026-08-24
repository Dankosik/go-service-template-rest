# Claim To Proof Mapping

## Load When

Load when a positive claim names a gate, package/repository scope, readiness, or
current environment behavior and the evidence boundary is not obvious.

## Decide

Leaf gates are independent:

| Gate | Proves | Does not prove |
| --- | --- | --- |
| focused `go test -vet=off ./pkg` | that package's tests | formatting, lint, other packages |
| `make unit-check PKG=./pkg FILES='...'` | package format, tests, and the small lint set | full policy, other packages, race |
| `make verify` | the applicable non-overlapping worktree surfaces printed by `make plan` | skipped or not-applicable surfaces, full-repository policy |
| `make check` | full format, `lint-all`, `test-all`, tidy, generated drift | race, Docker, security, template matrices |
| matching `*-check` | one generated-contract surface | compatibility unless its breaking check also ran |
| matching `test-integration-{db,messaging,process,race}` | that real integration surface | sibling integration surfaces or image lifecycle |
| `REQUIRE_DOCKER=1 ALLOW_HEAVY=1 make test-integration` | full non-race integration pack | race or production image lifecycle |
| `ALLOW_HEAVY=1 make migration-validate` | migration and exact-image runtime rehearsal | data recoverability or rollout safety |

`make build` is not implied by test targets. `make lint` and `make test` require
`PKG` and do not default to `./...`. Full policy is `lint-all`; whole-program
analysis is `lint-deep` and needs `ALLOW_HEAVY=1` or CI. Security surfaces
remain separate. Performance uses the benchmarking owner.

Go exits zero for `-run` matching no tests and packages with no tests. When one
named test carries the claim, verify a real run event as the OpenAPI contract
check does. Integration may skip without Docker unless `REQUIRE_DOCKER=1`; the
required integration command fails closed instead.

Cached unit results support unchanged code behavior, not a claim that Docker,
database, image, or other external state was exercised now. Use `-count=1` when
fresh environment interaction is the claim. A focused package or reproducer
does not prove unrelated packages, lint, drift, regression, or race behavior.
Reuse a receipt only when candidate, scope, command, and environment match.

## Prove

Return [Evidence Result
V1](../../../../docs/spec-first-workflow/interfaces/evidence-result-v1.md) with the
exact command, result, duration, and scope actually exercised.
Name the one remaining command or owner when evidence is narrower than the
claim.
