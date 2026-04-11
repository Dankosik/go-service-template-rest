# Contributing

This repository is a Go REST service template optimized for beginners and AI-assisted workflows. Keep changes deterministic, reviewable, and production-oriented.

## Quick Start for Contributors

1. Bootstrap local environment:

```bash
make bootstrap
```

2. Run quick checks before opening a PR:

```bash
make check
```

3. Run full CI-like checks when needed:

```bash
make check-full
```

4. If this repository was cloned as a new service, run template rewiring before the first PR (module path/CODEOWNERS/skills mirrors):

```bash
make template-init
# optional explicit modes:
make template-init-native
make template-init-docker
# manual fallback:
make init-module MODULE=github.com/your-org/your-service CODEOWNER=@your-org/your-team
```

5. Apply required branch protection/status checks (repo admin):

```bash
make gh-protect BRANCH=main
```

If this fails due placeholder CODEOWNERS, run `make template-init` with explicit `CODEOWNER=@your-org/your-team`.

If your change is concurrency- or integration-sensitive, also run:

```bash
make test-race
make test-integration
```

## Railway Deployment Policy Changes

- Treat `railway.toml` as the deployment policy source of truth.
- Keep secrets out of `railway.toml`; use Railway variables/secrets.
- Any deployment policy change must be PR-reviewed and traceable.
- Run before PR:

```bash
make guardrails-check
```

- If you need to change locked rollout/capacity policy values (`180s`, `45s`, `30s`, `5 retries`, replica/capacity baseline), reopen spec first.

## Pull Request Rules

- Use the PR template and fill all required sections.
- Keep PR scope focused and reversible.
- Include concrete test evidence in PR description.
- Update docs in the same PR when behavior, contract, CI policy, or operations change.

## Commit and Code Style Expectations

- Go formatting is mandatory (`goimports` via `make fmt`/`make fmt-check`).
- Prefer explicit, readable Go code over framework-heavy abstractions.
- Before adding production feature code, use the placement guide in [Project Structure & Module Organization](docs/project-structure-and-module-organization.md#4-where-to-put-new-code). The local boundary bullets below are only a summary, not the full placement rule set.
- Keep module boundaries intact:
  - business logic and consumer-owned ports: `internal/app/<feature>`
  - HTTP mapping and generated-route policy: `internal/infra/http`
  - data adapters and SQLC mapping: `internal/infra/postgres`
  - config snapshot policy: `internal/config`
  - broad integration scenarios: `test/`
- Do not merge generated drift:
  - run `make openapi-generate`
  - verify with `make openapi-drift-check`
  - when interface seams change, run `make mocks-generate`
  - verify with `make mocks-drift-check`
  - when internal integer enums change, run `make stringer-generate`
  - verify with `make stringer-drift-check`
  - when SQL queries or migrations change, run `make sqlc-generate`
  - verify with `make sqlc-check`

## Security and Disclosure

- Do not open public issues for undisclosed vulnerabilities.
- Follow the process in `SECURITY.md`.

## Ownership

Critical paths are owner-protected via `.github/CODEOWNERS`.

After cloning this template into a new repository, verify CODEOWNERS points to real owners before enabling required code owner reviews.
