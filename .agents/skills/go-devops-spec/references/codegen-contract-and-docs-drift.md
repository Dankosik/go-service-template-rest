# Codegen Contract And Docs Drift

## Behavior Change Thesis
When loaded for symptom "generated artifacts, contracts, or docs may drift," this file makes the model choose generator-backed drift gates and docs-trigger policy instead of likely mistake "regenerate if needed" or relying on reviewer memory.

## When To Load
Load for OpenAPI/codegen verification, sqlc/stringer/mock drift, docs drift, local/CI parity for generated artifacts, or compatibility-check blocking behavior.

## Local Source Of Truth
- `Makefile` exposes `openapi-generate`, `openapi-drift-check`, `openapi-runtime-contract-check`, `openapi-lint`, `openapi-validate`, `openapi-breaking`, `openapi-check`, `sqlc-check`, `mocks-drift-check`, `stringer-drift-check`, and `docs-drift-check`.
- `.github/workflows/ci.yml` runs OpenAPI generation/drift checks, runtime contract checks, schema validation, linting, PR breaking-change checks, SQLC checks through `repo-integrity`, and docs drift.
- `scripts/ci/docs-drift-check.sh` maps behavior, contract, CI, Docker, migration, Makefile, and script changes to documentation-update obligations.
- `scripts/dev/docker-tooling.sh` mirrors drift checks for zero-setup Docker execution.

## Decision Rubric
- Generated artifacts must be regenerated from canonical source and committed in the same change; tracked or untracked drift after generation is blocking.
- OpenAPI changes must pass generation, drift detection, compile checks, runtime contract checks, validation, linting, and PR breaking checks when a base OpenAPI spec exists.
- SQL query files and generated SQLC files must have matching stems; stale generated SQLC files are blocking drift.
- Behavior, contract, CI-sensitive, Docker, Makefile, migration, or script changes require docs updates unless the drift checker explicitly excludes the file class.
- `openapi-breaking` may be conditional only when the spec names base-ref availability and fallback behavior when the base spec is missing.

## Imitate
- "A spec touching `api/openapi/service.yaml` must require `make openapi-check` plus PR `openapi-breaking` when a base spec exists; delivery records whether the check blocks, while API policy owns whether the change is allowed." Copy the split between delivery and API authority.
- "A Makefile or workflow change that alters behavior must either update docs or explain why `scripts/ci/docs-drift-check.sh` excludes that file class." Copy the script-backed docs policy.
- "Generated output is acceptable only after running the generator and observing a clean diff on generated paths." Copy the canonical-output proof.

## Reject
- "Regenerate generated files if they look stale." This lacks a deterministic generator and drift check.
- "Docs drift is a reviewer checklist item." The repo has a script gate; do not downgrade it to memory.
- "The API change compiles, so generated code is fine." Compile success does not prove generator output matches canonical source.

## Agent Traps
- Do not use `openapi-breaking` to decide API compatibility policy inside this delivery skill.
- Do not ignore shallow or missing base-ref behavior; skip/fail semantics must be stated when comparisons need a base.
- Do not invent drift targets absent from `Makefile` unless the spec also creates the repository enforcement surface.

## Validation Shape
Use CI logs from `openapi-contract`, `openapi-breaking`, and `repo-integrity`; local output from `make openapi-check`, `make sqlc-check`, `make mocks-drift-check`, `make stringer-drift-check`, or `make docs-drift-check BASE_REF=... HEAD_REF=...`; and clean generated-output diffs.

## Hand-Off Boundary
Do not define API semantics, request/response compatibility rules, SQL query ownership, or schema design here. Record the delivery check and hand those decisions to API, data, or architecture specs.
