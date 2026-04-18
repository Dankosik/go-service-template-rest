# Railway Pre-Deploy And GitHub Autodeploy Research

Question:
- How should this template apply database migrations automatically on Railway without making every service start compete to migrate?
- What GitHub-triggered Railway behavior can the template encode directly?

Repository facts:
- `railway.toml` already owns Railway deployment policy and is enforced by `scripts/ci/required-guardrails-check.sh`.
- The canonical runtime image is `build/docker/Dockerfile`, which currently ships only `/service` in a distroless final stage.
- `Makefile` / CI validate migrations with `go tool migrate` but do not run them as part of deployment.
- `.github/workflows/ci.yml` already runs on `push`, which is a prerequisite for Railway `Wait for CI`.

External findings from official Railway docs fetched through `exa`:
- Railway `preDeployCommand` runs after build and before promotion. It has access to environment variables and the private network, and deployment does not proceed if the command fails.
- `preDeployCommand` is configurable in `railway.toml` / config-as-code under `[deploy]`.
- Config defined in `railway.toml` overrides dashboard build/deploy settings for that deployment only.
- GitHub-linked Railway services auto-deploy when commits land on the configured branch.
- `Wait for CI` is an operator toggle in Railway service settings; when enabled, Railway waits for push-triggered GitHub workflows to finish and skips deploys on CI failure.

Implications:
- The right release-safe owner for automatic migrations in this template is a single pre-deploy migrator, not app startup and not build-time DDL.
- The template must ship migration execution dependencies inside the image because Railway pre-deploy runs against the built image.
- The current distroless image should use a direct migration binary instead of shell-wrapped scripts.
- The template can encode the migrator command in `railway.toml`, but it can only document, not force, the GitHub repo link and `Wait for CI` toggle.

Sources:
- Railway docs: `https://docs.railway.com/deployments/pre-deploy-command`
- Railway docs: `https://docs.railway.com/reference/config-as-code`
- Railway docs: `https://docs.railway.com/deployments/github-autodeploys`
