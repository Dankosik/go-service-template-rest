# CI/CD Production-Ready Checklist

Use this checklist when adopting the template in a real service repository.

## One-Time Repository Setup

1. Run `make template-init` after clone.
2. Verify `go.mod` module path is no longer `github.com/example/go-service-template-rest`.
3. Replace CODEOWNERS placeholders with real owners.
4. Authenticate GitHub CLI (`gh auth login`) and run `make gh-protect BRANCH=main`.

## Required Branch Protection Checks

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

These are configured by `scripts/dev/configure-branch-protection.sh`.

## Ongoing Gate Expectations

- Keep generated artifacts in sync (`openapi`, `sqlc`, `mockgen`, `stringer`).
- Keep agent mirrors and skill mirrors in sync (`make agents-check`, `make skills-check`).
- Keep docs-drift gate green when behavior/contract/CI-sensitive files change.
- Keep security gates green (`govulncheck`, `gosec`, `gitleaks`, Trivy).
- Keep `check` and `check-full` usable for local developer feedback loops.
