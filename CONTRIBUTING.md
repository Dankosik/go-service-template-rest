# Contributing

This repository is a Go REST service template optimized for beginners and AI-assisted workflows. Keep changes deterministic, reviewable, and production-oriented.

## Quick Start for Contributors

1. Bootstrap local tooling and environment:

```bash
make setup
```

2. Initialize module path once after clone:

```bash
make init-module MODULE=github.com/your-org/your-service
```

3. Run baseline checks before opening a PR:

```bash
make fmt-check
make lint
make test
make openapi-check
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

- Go formatting is mandatory (`gofmt` via `make fmt`/`make fmt-check`).
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

After cloning this template into a new repository, update CODEOWNERS to your actual team handles.
