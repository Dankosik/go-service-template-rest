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
when absent, and rejects invalid input before mutation. For explicitly requested
initializer-contract verification, use `ALLOW_HEAVY=1 make template-init-check`;
that matrix is not a routine service-development gate.

## Validate a change

Ordinary local Go development finishes after the agreed behavior is implemented,
a matching build and relevant unit tests pass, known in-scope defects are fixed,
and any applicable final review is resolved. For the main service and the
ordinary root-module suite:

```bash
make build
ALLOW_FULL=1 make test-all
```

[Go Validation](docs/validation/go.md) owns affected-package scope and additional
retained executable builds. `test-all` here is ordinary testing without race or
integration tags, not `make check`. Instruction-only changes use static review
and the matching [structural check](docs/validation/instructions.md), not Go tests.

`make prove PKG=./internal/<package> FILES='internal/<package>/*.go'` remains an
optional standalone diagnostic. `make verify` is an explicitly selected expanded
route; it may require Docker, integration, migration, or image work. `make plan`
only explains that route. Neither is a mandatory local completion step.

`ALLOW_FULL=1 make check` remains an explicitly required full-repository gate.
Do not also run its `fmt-check`, `lint-all`, or `test-all` leaves. `make test` and
`make lint` require `PKG` and do not default to `./...`.

[AGENTS.md](AGENTS.md#validation-budget) owns the ordinary local stop rule.
Load the [Evidence Contract](docs/spec-first-workflow/shared/evidence-contract.md)
for exceptions, reuse, additional required proof, or unavailable infrastructure.
Do not create runners or environments for extra confidence. Missing optional
Docker/provider observations are material gaps to disclose, not blockers to
repair. Existing CI/release gates and explicit task requirements remain intact;
a known real defect still requires correction.

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

When local secret verification is required, use `make secret-scan`. The existing
pull-request scan remains intact. Main and release use `make secret-scan-history`;
do not replace that historical gate with a faster scan or broader baseline.

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
- retain applicable CI/release gates and explicitly requested migration,
  runtime-image, and deployment verification;
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
