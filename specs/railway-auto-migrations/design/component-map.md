# Affected Surfaces

- `cmd/migrate/`: new dedicated migration entrypoint for Railway pre-deploy and local replay of the same path
- `internal/infra/postgres/`: migration execution helper and shared DSN handling
- `build/docker/Dockerfile`: ship `/migrate` and `env/migrations/` in the final image
- `railway.toml`: declare the pre-deploy migration command
- `scripts/ci/required-guardrails-check.sh`: lock the new Railway migration policy into repo guardrails
- `docs/railway-deployment-profile.md`: document the new Railway deployment baseline

# Stable Surfaces

- `cmd/service/` runtime startup flow stays migration-free
- `env/migrations/` remains the schema source of truth
- `.github/workflows/ci.yml` keeps owning migration rehearsal in CI
- HTTP/readiness behavior and service bootstrap semantics remain unchanged

# Responsibility Changes

- production migration ownership moves from manual operator action to a single Railway pre-deploy command
- the runtime image now owns the dependencies needed to execute migrations, not just the service binary
