# CI/CD Production-Ready Checklist

Use this checklist when adopting the template in a real service repository.

## One-time repository setup

1. Initialize the derived repository:

   ```bash
   make template-init \
     MODULE=github.com/your-org/your-service \
     CODEOWNER=@your-org/backend
   make template-init-check
   ```

2. Confirm the template module no longer appears in Go imports or
   `.golangci.yml`, and confirm `.github/CODEOWNERS` names real owners.
3. Configure GitHub Rulesets or organization policy to require pull requests,
   code-owner review where appropriate, and the blocking checks currently
   defined by `.github/workflows/ci.yml`.
4. Configure repository secrets and external deployment integration. Keep
   secrets out of tracked files.
5. If Railway deploys from GitHub, connect the intended repository and branch
   and enable Railway `Wait for CI` when promotion must wait for CI.

The repository intentionally does not carry a branch-protection API client.
Exact check names change with the workflow and should not be duplicated in a
local policy script.

## Gate ownership

- `.github/workflows/ci.yml` owns pull-request and push validation.
- The coverage job owns the ordinary test-suite execution; race and integration
  remain separate because they prove different failure modes.
- The lint job owns configured analyzer and vet-class evidence.
- OpenAPI and SQLC sources own generated output; drift checks compare it.
- Migration validation owns reversible schema rehearsal and the runtime
  `/migrate` entrypoint.
- The runtime Dockerfile owns image composition; container security scans that
  same image.
- Go vulnerability, source security, secret, dependency, and container scans
  remain separate because they inspect different authorities.
- `.github/workflows/cd.yml` owns release validation and image publication.
- `railway.toml` owns non-secret Railway deployment policy.

Dependency Review and any explicitly informational scheduled scans are
advisory until repository policy deliberately promotes them. Do not silently
describe them as blocking.

## Ongoing checks

- Use `make check` for the ordinary edit loop.
- Use `make ci-local` for the broad host-toolchain aggregate.
- Use `make check-full` before a claim that includes integration, migration,
  runtime image, or container security.
- Use `make pr-check BASE_REF=origin/main` when the claim includes base-relative
  OpenAPI compatibility.
- Review GitHub Rulesets after CI job renames or removals so obsolete contexts
  do not block merges and new blocking contexts are admitted intentionally.
- Preserve exact proof and record any unavailable external or Docker-backed
  evidence in the PR or release packet.
