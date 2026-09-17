# Claim To Proof Mapping

## Load When

Load when a positive claim names a gate, package/repository scope, readiness, or
current environment behavior and the evidence boundary is not obvious.

## Decide

Proof leaves are independent. [Validation Routing](../../../../docs/validation-routing.md)
owns command composition and execution timing; its selected result proves only
the claim and surface it actually exercised. Generated, real-store, race,
security, performance, CI, and release claims remain distinct.
A zero-exit selector that runs no named test is not evidence, and a skipped
Docker scenario does not establish an integration claim.

Cached unit results support unchanged code behavior, not a claim that Docker,
database, image, or other external state was exercised now. A focused package
or reproducer does not prove unrelated packages, lint, drift, regression, or
race behavior.
Apply the [Evidence Contract](../../../../docs/spec-first-workflow/shared/evidence-contract.md#execution-evidence)
when deciding whether a scoped result or aggregate receipt remains reusable.

## Prove

Return [Evidence Result
V1](../../../../docs/spec-first-workflow/interfaces/evidence-result-v1.md) with the
exact command, result, duration, and scope actually exercised.
Name the one remaining command or owner when evidence is narrower than the
claim.
