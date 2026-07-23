# Contributing

Keep changes deterministic, reviewable, and owned by the narrowest package or
repository surface that can prove them.

## Start a derived service

Run the template initializer once, before the first service change:

```bash
make template-init \
  MODULE=github.com/your-org/your-service \
  CODEOWNER=@your-org/backend
```

The command rewrites the Go module and module-qualified lint rules, updates
CODEOWNERS, preserves an existing `.env`, creates it from `.env.example` only
when absent, and rejects invalid input before mutation. Verify that contract
with `make template-init-check`.

## Validate a change

Use the smallest command that proves the claim:

```bash
make check          # format, lint, unit tests
make ci-local       # broad host-toolchain CI aggregate
make check-full     # ci-local plus Docker-backed integration and image gates
make pr-check BASE_REF=origin/main
```

Focused commands remain available:

```bash
make test-race
make test-integration
make test-report
make openapi-check
make sqlc-check
make migration-validate
make go-security
make secret-scan
```

`make check-full` and the Docker-backed focused commands require a reachable
Docker daemon. Do not describe a host-only result as full container or
migration evidence.

For performance work, follow
[Benchmarking](docs/benchmarking.md) and run only the benchmark level that
matches the claim. When benchmark infrastructure changes, run
`make benchmark-infra-check`.

## Pull requests and repository policy

- Keep PR scope focused and reversible.
- Include exact validation evidence and any unverified remainder.
- Update docs with behavior, contract, CI, or operational changes.
- Configure required reviews and status checks with GitHub Rulesets or
  organization policy. The repository does not mutate its own GitHub settings.
- Treat `.github/workflows/ci.yml` as the source of truth for current check
  names instead of copying a permanent list into scripts or docs.

For Railway policy changes:

- keep `railway.toml` non-secret and PR-reviewed;
- keep the canonical build path in `build/docker/Dockerfile`;
- validate the affected migration, runtime-image, and deployment surfaces;
- reopen the relevant design before changing rollout timing, retry, replica, or
  capacity policy.

## Code and generated sources

- Format Go through `make fmt`; verify with `make fmt-check`.
- Prefer explicit Go and existing repository seams over new framework layers.
- Use the placement guide in
  [Project Structure & Module Organization](docs/project-structure-and-module-organization.md#3-deterministic-placement-algorithm).
- Keep business logic in `internal/<feature>`, HTTP mapping in
  `internal/infra/http`, persistence adapters in `internal/infra/postgres`,
  configuration in `internal/config`, and composition in
  `cmd/service/internal/bootstrap`.
- Regenerate OpenAPI and SQLC output through their owning targets. Never edit
  generated output by hand.

## Security and ownership

Do not open public issues for undisclosed vulnerabilities; follow
`SECURITY.md`. Before enabling required code-owner reviews in a derived
repository, confirm `.github/CODEOWNERS` names real users or teams with access.
