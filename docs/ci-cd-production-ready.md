# CI/CD Production-Ready Setup

This checklist defines repository settings that must be enabled so workflow files are enforced as hard gates.

## Required Branch Protection (for `main`)

Enable branch protection for `main` with:
- Require a pull request before merging.
- Require status checks to pass before merging.
- Require branches to be up to date before merging.
- Include administrators.
- Do not allow force pushes.
- Do not allow deletions.

Automation shortcut for cloned repositories:

```bash
make gh-protect BRANCH=main
```

This command configures branch protection via GitHub API (requires `gh` auth and admin permissions).
Before running it, ensure `.github/CODEOWNERS` does not contain the template placeholder (`@your-org/your-team`).
Recommended bootstrap: run template initialization first. `make template-init` auto-infers CODEOWNER from `git remote origin` and replaces placeholder owners when possible.

```bash
make template-init
# optional explicit override:
CODEOWNER=@your-org/your-team make template-init
```

Required status checks from `.github/workflows/ci.yml`:
- `repo-integrity`
- `lint`
- `openapi-contract`
- `test`
- `test-race`
- `test-coverage`
- `test-integration`
- `migration-validate`
- `go-security`
- `secret-scan`
- `container-security`

Notes:
- `openapi-breaking` runs only for pull requests and should also be required if API compatibility is mandatory.
- `migration-validate` is conditional and returns a successful skip when no migration files changed.
- `repo-integrity` includes required guardrails check (`make guardrails-check`) and skills mirror check (`make skills-check`), and should not be downgraded to optional.
- `go-security` runs `govulncheck` plus `gosec -exclude-generated` to avoid false positives from codegen artifacts.
- `secret-scan` runs `gitleaks` against git history with redaction enabled.
- integration jobs run with `REQUIRE_DOCKER=1` to fail fast if Docker is unavailable in CI runners.

## Pull Request Policy

- Require at least one approved review.
- Dismiss stale approvals when new commits are pushed.
- Require conversation resolution before merge.
- Disable direct pushes to `main`.
- Require review from Code Owners.

## Repository Guardrails (must exist in default branch)

- `AGENTS.md`
- `README.md`
- `Makefile`
- `.editorconfig`
- `.gitattributes`
- `.golangci.yml`
- `.redocly.yaml`
- `.github/CODEOWNERS`
- `.github/dependabot.yml`
- `.github/pull_request_template.md`
- `.github/workflows/ci.yml`
- `.github/workflows/cd.yml`
- `.github/workflows/nightly.yml`
- `CONTRIBUTING.md`
- `SECURITY.md`
- `LICENSE`
- `env/.env.example`
- `env/docker-compose.yml`
- `build/docker/Dockerfile`
- `build/docker/tooling-images.Dockerfile`

## Release and Artifact Trust

`cd.yml` already performs:
- Trivy image scan (`CRITICAL,HIGH`) before image push.
- CycloneDX SBOM generation.
- Cosign keyless signing via OIDC.
- Build provenance attestation (`actions/attest-build-provenance`).

Repository settings to verify:
- GitHub Actions has permission to create attestations.
- Package permissions allow publishing to GHCR for this repository.
- OIDC token requests are not restricted by organization policy for this repository.

## Environment and Deployment Safety

If deployment steps are added later (for Kubernetes/Helm/Terraform/Cloud), enforce:
- GitHub Environment protection with required reviewers for production.
- Manual approval gate before production deployment.
- Separate non-production and production credentials.
- Rollback procedure documented and linked in release PR.

## Operational Cadence

- Keep `nightly.yml` enabled to detect flaky tests and delayed regressions.
- Treat nightly failures as release blockers.
- Review and rotate pinned tool versions (`golangci-lint`, `govulncheck`, `gosec`, `gitleaks`, `Trivy`, `cosign`) on a regular schedule.
