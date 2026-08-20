# Claim To Proof Mapping

## Load When

Load when a positive claim names a gate, package/repository scope, readiness, or
current environment behavior and the evidence boundary is not obvious.

## Decide

The aggregates are not nested supersets:

| Gate | Proves | Does not prove |
| --- | --- | --- |
| `make check` | structure, format, lint, unit tests | build, deep lint, generated drift, security, Docker |
| `make ci-local` | host CI including deep lint, race, coverage, generated checks, Go security, secrets | integration, migrations, images, containers |
| `make check-full` | delivery, host CI, required Docker integration, image, migration, container security | base-relative template/module/API/proto breaking checks |
| `make pr-check BASE_REF=...` | base-relative PR surface | production/runtime evidence |

`make build` is not implied by test targets. `make lint` omits `lint-deep`.
Security surfaces remain separate. Performance uses the benchmarking owner.

Go exits zero for `-run` matching no tests and packages with no tests. When one
named test carries the claim, verify a real run event as the OpenAPI contract
check does. Integration may skip without Docker unless `REQUIRE_DOCKER=1`; the
required aggregate fails closed instead.

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
