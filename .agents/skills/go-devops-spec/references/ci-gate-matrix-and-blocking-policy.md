# CI Gate Matrix And Blocking Policy

## Behavior Change Thesis
When loaded for symptom "the delivery spec needs CI tiers, required jobs, or release evidence," this file makes the model choose repository-owned gate names, fail-closed status semantics, and local/CI parity instead of likely mistake "run the usual checks" or treating advisory/nightly/manual evidence as equivalent to merge evidence.

## When To Load
Load for CI gate matrices, required status checks, local/CI command parity, timeout/cancellation semantics, nightly evidence, or release preflight gates.

## Local Source Of Truth
- `.github/workflows/ci.yml` defines merge-time jobs: `repo-integrity`, `lint`, `openapi-contract`, PR-only `openapi-breaking`, `test`, `test-race`, `test-coverage`, `test-integration`, `migration-validate`, `go-security`, `secret-scan`, and `container-security`.
- `.github/workflows/nightly.yml` defines slower reliability evidence: flake detection, fuzz smoke, race, integration, OpenAPI checks, vulnerability/static checks, build, and Trivy scan.
- `.github/workflows/cd.yml` defines publish preflight for tag releases and publish-after-successful-main-CI behavior.
- `Makefile` is the local command source for parity claims: `make check`, `make check-full`, `make ci-local`, `make docker-ci`, `make migration-validate`, and `make docker-container-security`.

## Decision Rubric
- Name the exact gate tier and job/target that enforces it; avoid prose-only gate names.
- Required branch-protection contexts must match stable CI job names; when a job is renamed, update branch protection configuration and `scripts/ci/required-guardrails-check.sh` in the same change.
- Treat timed-out, cancelled, missing, or path-filter-skipped required checks as blocking delivery evidence until rerun or covered by exception governance.
- Use `make check` for fast local confidence; require `make check-full` or `make docker-ci` when Docker-backed integration, migration rehearsal, or image scanning evidence matters.
- Treat nightly failures as release blockers only when they affect a changed path, reveal an active regression, or the release policy explicitly promotes nightly reliability evidence into a hard gate.

## Imitate
- "For a PR touching normal service code, merge is blocked by `repo-integrity`, `lint`, `openapi-contract`, `test`, `test-race`, `test-coverage`, `test-integration`, `migration-validate`, `go-security`, `secret-scan`, and `container-security`; evidence is the GitHub Actions job conclusion on the merge SHA." Copy the exact-context habit.
- "`openapi-breaking` is PR-only and required only if API compatibility policy makes breaking-change detection a hard gate; delivery records the check consequence, not the API policy." Copy the ownership boundary.
- "Release preflight for tags must use `.github/workflows/cd.yml` evidence and cannot be replaced by a successful `workflow_dispatch` run unless the spec proves same ref, permissions, and comparison base." Copy the trigger-sensitive evidence rule.

## Reject
- "Run the normal CI suite before merge." This hides which job is blocking and cannot be mapped to branch protection.
- "Nightly passed last week, so release can proceed despite red PR CI." This confuses advisory reliability history with merge-time evidence.
- "Add path filters to required workflows to save time." This can leave required checks pending or missing unless the spec handles skipped-check semantics.

## Agent Traps
- Do not invent a CI tier if no repository command or workflow can produce its artifact.
- Do not replace Makefile targets with inline CI commands unless the spec explicitly accepts local/CI drift.
- Do not treat `workflow_dispatch` success as equivalent to PR, push, or tag-triggered evidence without checking trigger, ref, permissions, and base comparison.

## Validation Shape
Use GitHub Actions run URL, workflow name, job name, commit SHA, conclusion, coverage artifacts (`coverage.out`, `.artifacts/test/junit.xml`, `.artifacts/test/test2json.json`), Trivy output, migration rehearsal logs, and release preflight logs.

## Hand-Off Boundary
Do not decide API compatibility, schema safety, or security acceptance here. Record the delivery gate consequence and route the underlying API, data, or security decision to the owning spec lane.
