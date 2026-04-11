# Codegen Contract And Docs Drift

## When To Load
Load this when a delivery spec must define generated-code drift, OpenAPI/codegen verification, sqlc/stringer/mock drift, docs drift, local/CI parity for generated artifacts, or compatibility-check blocking behavior.

## Local Source Of Truth
- `Makefile` exposes `openapi-generate`, `openapi-drift-check`, `openapi-runtime-contract-check`, `openapi-lint`, `openapi-validate`, `openapi-breaking`, `openapi-check`, `sqlc-check`, `mocks-drift-check`, `stringer-drift-check`, and `docs-drift-check`.
- `.github/workflows/ci.yml` runs OpenAPI generation/drift checks, runtime contract checks, schema validation, linting, PR breaking-change checks, SQLC checks through `repo-integrity`, and docs drift.
- `scripts/ci/docs-drift-check.sh` maps behavior, contract, CI, Docker, migration, Makefile, and script changes to documentation-update obligations.
- `scripts/dev/docker-tooling.sh` mirrors drift checks for zero-setup Docker execution.

## Enforceable Policy Examples
- Generated artifacts must be regenerated from source and committed in the same change; CI fails on tracked or untracked drift after generation.
- OpenAPI changes must pass generation, drift detection, compile checks, runtime contract checks, validation, linting, and PR breaking checks when a base OpenAPI spec exists.
- SQL query files and generated SQLC files must have matching stems; stale generated SQLC files are a blocking drift defect.
- Behavior, contract, CI-sensitive, Docker, Makefile, migration, or script changes require a docs update unless the drift checker explicitly excludes the file class.
- A spec may say `openapi-breaking` is conditional only if it names the base-ref availability rule and fallback behavior when the base spec is missing.

## Non-Enforceable Anti-Patterns
- "Regenerate if needed" without naming the generator and drift-check target.
- Accepting generated files that compile but differ from the canonical generator output.
- Treating docs drift as a reviewer memory task instead of a script-enforced check.
- Letting CI use `git diff` against a shallow or unavailable base ref without defining skip or fail behavior.
- Using API-breaking results to decide API policy inside this delivery skill; delivery only records whether the check blocks.

## Evidence Artifacts
- CI logs from `openapi-contract`, `openapi-breaking`, and `repo-integrity`.
- Local command output from `make openapi-check`, `make sqlc-check`, `make mocks-drift-check`, `make stringer-drift-check`, or `make docs-drift-check BASE_REF=... HEAD_REF=...`.
- Clean `git diff --exit-code` on generated output paths after generators run.
- Docs change path listed beside the behavior or contract change that triggered the docs drift policy.

## Hand-Off Boundary
Do not define API semantics, request/response compatibility rules, SQL query ownership, or schema design here. Record the delivery check and hand those decisions to API, data, or application architecture specs.

## Exa Source Links
- GitHub Docs: [Workflow syntax for GitHub Actions](https://docs.github.com/actions/using-workflows/workflow-syntax-for-github-actions)
- GitHub Docs: [Events that trigger workflows](https://docs.github.com/en/actions/automating-your-workflow-with-github-actions/events-that-trigger-workflows)
- GitHub Docs: [About protected branches](https://docs.github.com/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)

