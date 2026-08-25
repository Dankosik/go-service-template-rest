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
with `ALLOW_HEAVY=1 make template-init-check`.

## Validate a change

Use the smallest command that proves the claim:

```bash
make prove PKG=./internal/<package> FILES='internal/<package>/*.go'
make plan
make verify
```

`ALLOW_FULL=1 make check` is the one full-repository owner. Do not also run
`fmt-check`, `lint-all`, or `test-all` beside it. `make test` and `make lint`
require `PKG` and do not default to `./...`.

Heavy commands remain available with an explicit grant:

```bash
ALLOW_HEAVY=1 make test-race
ALLOW_HEAVY=1 make test-integration
make mod-check
make openapi-check
make sqlc-check
ALLOW_HEAVY=1 make migration-validate
ALLOW_HEAVY=1 make govulncheck
ALLOW_HEAVY=1 make gosec
make secret-scan
ALLOW_HEAVY=1 make secret-scan-history
```

Docker-backed focused commands require a reachable Docker daemon. Do not
describe a host-only result as container or migration evidence.

`make root-mod-check` owns the service module. `make tools-mod-check` owns the
tool module. `make mod-check` runs both and remains the explicit combined gate.

Use `make secret-scan` for local and pull-request changes. Main and release use
`make secret-scan-history`; do not replace the historical gate with
a faster scan or a broader baseline.

For performance work, follow
[Benchmarking](docs/benchmarking.md) and run only the benchmark level that
matches the claim.

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

- Format Go through `make fmt`; verify with `make fmt-check` or `make check`.
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
