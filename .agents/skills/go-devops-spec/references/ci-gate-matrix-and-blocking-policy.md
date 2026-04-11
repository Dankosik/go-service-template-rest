# CI Gate Matrix And Blocking Policy

## When To Load
Load this when defining or reviewing CI tiers, required jobs, local/CI command parity, timeout/cancellation semantics, nightly gates, or release preflight gates.

## Local Source Of Truth
- `.github/workflows/ci.yml` defines merge-time checks: `repo-integrity`, `lint`, `openapi-contract`, `openapi-breaking` for pull requests, `test`, `test-race`, `test-coverage`, `test-integration`, `migration-validate`, `go-security`, `secret-scan`, and `container-security`.
- `.github/workflows/nightly.yml` defines slower reliability evidence: flake detection, fuzz smoke, race, integration, OpenAPI checks, vulnerability/static checks, build, and Trivy scan.
- `.github/workflows/cd.yml` defines publish preflight for tag releases and publish-after-successful-main-CI behavior.
- `Makefile` is the local command source: `make check`, `make check-full`, `make ci-local`, `make docker-ci`, `make migration-validate`, and `make docker-container-security`.

## Enforceable Policy Examples
- Required branch protection contexts must match stable CI job names exactly; if a CI job is renamed, update the required-status configuration and `scripts/ci/required-guardrails-check.sh` in the same change.
- Merge policy blocks on `repo-integrity`, `lint`, `openapi-contract`, `test`, `test-race`, `test-coverage`, `test-integration`, `migration-validate`, `go-security`, `secret-scan`, and `container-security`; `openapi-breaking` is PR-only and should be required when API compatibility policy says breaking checks are mandatory for the project.
- Fast local validation may use `make check`; full local parity must use `make check-full` or `make docker-ci` when Docker-backed integration, migration rehearsal, or image scanning evidence is required.
- Timed-out, cancelled, missing, or path-filter-skipped required checks are release-blocking until rerun or explicitly accepted through exception governance.
- Nightly failures are release blockers only when they affect a changed path, reveal an active regression, or the release policy explicitly promotes nightly reliability evidence into a hard gate.

## Non-Enforceable Anti-Patterns
- "Run the usual checks" without naming commands, CI jobs, and required artifacts.
- Replacing `make` targets with inline one-off commands in CI unless the target is updated or the drift is explicitly justified.
- Treating `workflow_dispatch` success as equivalent to PR or tag-triggered release evidence when the trigger lacks the same ref, permissions, or base comparison.
- Adding path filters to required workflows without accounting for GitHub's pending-check behavior when required checks are skipped.
- Making `nightly` advisory forever while still using it rhetorically as release confidence.

## Evidence Artifacts
- GitHub Actions run URL, workflow name, job name, commit SHA, and conclusion.
- Coverage artifact: `coverage.out` plus `.artifacts/test/junit.xml` and `.artifacts/test/test2json.json` when `test-coverage` runs.
- Trivy output for `container-security` or nightly/release image scan.
- Migration rehearsal log from `make migration-validate` or `make docker-migration-validate`.
- Release preflight log from `.github/workflows/cd.yml` before any tag publish.

## Hand-Off Boundary
Do not decide API compatibility, schema safety, or security acceptance in this reference. Record the delivery gate consequence and route the underlying API/schema/security decision to the owning spec lane.

## Exa Source Links
- GitHub Docs: [Workflow syntax for GitHub Actions](https://docs.github.com/actions/using-workflows/workflow-syntax-for-github-actions)
- GitHub Docs: [Control the concurrency of workflows and jobs](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency)
- GitHub Docs: [Events that trigger workflows](https://docs.github.com/en/actions/automating-your-workflow-with-github-actions/events-that-trigger-workflows)
- GitHub Docs: [About protected branches](https://docs.github.com/repositories/configuring-branches-and-merges-in-your-repository/managing-protected-branches/about-protected-branches)

