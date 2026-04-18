Phase: research
Phase status: completed
Question:
- What Railway mechanism should own production migrations so deploys are automatic but not tied to every app replica startup?
- What part of GitHub-triggered Railway deploy behavior is code-configurable versus operator-configured in the Railway UI?

Sources:
- local repo: `railway.toml`, `docs/railway-deployment-profile.md`, `build/docker/Dockerfile`, `Makefile`, `.github/workflows/ci.yml`
- external: official Railway docs fetched through `exa`

Fan-in:
- Railway officially supports `preDeployCommand` between build and promotion for migrations.
- Config-as-code can set `deploy.preDeployCommand` in `railway.toml`.
- GitHub-connected Railway services auto-deploy on pushes to the configured branch, and `Wait for CI` is a per-service operator toggle, not a repository file setting.
- For Dockerfile/image deployments, start-like command overrides run in exec form; direct binaries are safer than shell-dependent migration scripts for this distroless image.

Completion marker: enough platform evidence exists to approve a one-migrator pre-deploy design without reopening research.
Stop rule: do not write implementation code in this phase file.
Next action: record the release-safe decision and its non-goals in `spec.md`.
