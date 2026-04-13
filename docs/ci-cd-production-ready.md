# CI/CD Production-Ready Checklist

Use this checklist when adopting the template in a real service repository.

## One-Time Repository Setup

1. Run `make template-init` after clone.
2. Verify `go.mod` module path is no longer `github.com/example/go-service-template-rest`.
3. Replace CODEOWNERS placeholders with real owners.
4. Authenticate GitHub CLI (`gh auth login`) and run `make gh-protect BRANCH=main`.
5. Run `make gh-protect-check BRANCH=main` to audit required status contexts without mutating settings.

## Required Branch Protection Checks

- `repo-integrity`
- `lint`
- `openapi-contract`
- `openapi-breaking`
- `test`
- `test-race`
- `test-coverage`
- `test-integration`
- `migration-validate`
- `go-security`
- `secret-scan`
- `container-security`

These are configured by `scripts/dev/configure-branch-protection.sh`.

PR-only Dependency Review and nightly Trivy repository filesystem/config scanning are intentionally informational on first introduction. They are not required branch-protection contexts.

## Ongoing Gate Expectations

- Keep generated artifacts in sync (`openapi`, `sqlc`, `mockgen`, `stringer`).
- Keep agent mirrors and skill mirrors in sync (`make agents-check`, `make skills-check`).
- Keep docs-drift gate green when behavior/contract/CI-sensitive files change.
- Keep required security gates green (`govulncheck`, `gosec`, `gitleaks`, Trivy image scanning).
- Review informational Dependency Review and nightly Trivy repository filesystem/config findings before deciding whether a future task should make them blocking.
- Keep `check` and `check-full` usable for local developer feedback loops.
