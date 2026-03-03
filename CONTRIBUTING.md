# Contributing

This repository is a Go REST service template optimized for beginners and AI-assisted workflows. Keep changes deterministic, reviewable, and production-oriented.

## Quick Start for Contributors

1. Bootstrap local tooling and environment:

```bash
make setup
make setup-strict
# without make:
bash ./scripts/dev/setup.sh
bash ./scripts/dev/setup.sh --strict
# or explicit modes:
make setup-native
make setup-native-strict
make setup-docker
```

Setup auto-infers `CODEOWNER` from git `origin` when CODEOWNERS still has the template placeholder.
If you want explicit team ownership, set CODEOWNER during setup:

```bash
CODEOWNER=@your-org/your-team make setup
# without make:
CODEOWNER=@your-org/your-team bash ./scripts/dev/setup.sh
```

2. In most cloned repositories this step is not needed (setup handles module initialization).  
If setup reports skipped module initialization, run manual fallback once:

```bash
make init-module CODEOWNER=@your-org/your-team
# or explicit module:
make init-module MODULE=github.com/your-org/your-service CODEOWNER=@your-org/your-team
# zero-setup alternatives:
make docker-init-module MODULE=github.com/your-org/your-service CODEOWNER=@your-org/your-team
bash ./scripts/dev/docker-tooling.sh init-module github.com/your-org/your-service
```

3. Apply required branch protection/status checks (repo admin):

```bash
make gh-protect BRANCH=main
```

If this fails due placeholder CODEOWNERS, rerun setup with explicit `CODEOWNER=@your-org/your-team`.

4. Run baseline checks before opening a PR:

```bash
make ci-local
# zero-setup equivalent:
make docker-ci
```

If your change is concurrency- or integration-sensitive, also run:

```bash
make test-race
make test-integration
```

## Pull Request Rules

- Use the PR template and fill all required sections.
- Keep PR scope focused and reversible.
- Include concrete test evidence in PR description.
- Update docs in the same PR when behavior, contract, CI policy, or operations change.

## Commit and Code Style Expectations

- Go formatting is mandatory (`goimports` via `make fmt`/`make fmt-check`).
- Prefer explicit, readable Go code over framework-heavy abstractions.
- Keep module boundaries intact:
  - business logic: `internal/app`
  - transport/runtime wiring: `internal/infra/http`
- Do not merge generated drift:
  - run `make openapi-generate`
  - verify with `make openapi-drift-check`

## Security and Disclosure

- Do not open public issues for undisclosed vulnerabilities.
- Follow the process in `SECURITY.md`.

## Ownership

Critical paths are owner-protected via `.github/CODEOWNERS`.

After cloning this template into a new repository, verify CODEOWNERS points to real owners before enabling required code owner reviews.
