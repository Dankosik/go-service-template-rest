# Source Of Truth Ownership

- `env/migrations/`: canonical schema change source
- `internal/infra/postgres/`: DSN normalization rules and migration execution mechanics
- `cmd/migrate/`: orchestration entrypoint that turns template config into one migration run
- `railway.toml`: deployment policy hook that decides when Railway invokes the migrator
- `docs/railway-deployment-profile.md`: operator-facing explanation of the Railway deployment baseline and prerequisites

# Explicit Non-Owners

- `cmd/service/` is not the owner of production migrations
- `.github/workflows/cd.yml` is not the deploy-time schema migrator for Railway repo-connected services
- Railway dashboard GitHub connection state is not repo-owned; the template only documents it

# Dependency Direction

- `cmd/migrate/` may depend on `internal/config` and `internal/infra/postgres`
- `internal/infra/postgres` must not depend on Railway-specific packages or dashboard state
- docs and guardrails may reference `railway.toml`, but deployment semantics still execute through Railway itself
