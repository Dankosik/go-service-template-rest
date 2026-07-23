# CI/CD Production-Ready Checklist

Use this checklist when creating a real service from the template. Repository
tooling supplies build and verification capabilities; it does not connect or
deploy to a specific infrastructure project.

## Initialize the derived repository

```bash
make template-init \
  MODULE=github.com/your-org/your-service \
  CODEOWNER=@your-org/backend
make template-init-check
```

Confirm that:

- the template module no longer appears in Go imports or `.golangci.yml`;
- `.github/CODEOWNERS` names real owners;
- secrets remain in the chosen platform or secret manager;
- GitHub Rulesets require the current blocking jobs from
  `.github/workflows/ci.yml`;
- critical workflow, Dockerfile, migration, and deployment changes require an
  appropriate owner review;
- creation, update, and deletion of release tags such as `v*` are protected.

The template intentionally does not carry an API client that rewrites repository
rules or deployment infrastructure.

## Choose one deployment evidence path

### Source build

The deployment platform checks out a selected commit and builds
`build/docker/Dockerfile`. For Railway, `railway.toml` supplies the generic
source-build profile and `RAILWAY_GIT_COMMIT_SHA` becomes the default version in
the application startup log.

Evidence for this path is the selected commit, platform build record, deployment
record, reported application version, readiness result, and operator-owned
post-deploy checks. A GHCR attestation does not certify this independent build.

### Published image

`.github/workflows/cd.yml` builds one image, scans it, generates its SBOM, pushes
an immutable build tag to obtain the registry digest, signs and attests that
digest, verifies the evidence, and only then promotes `main`, version, or
`latest` tags.

The workflow writes the immutable `image@sha256:...` reference to the job
summary. Deploy that digest rather than resolving a mutable tag later.

## Verify a published image

Set values from the release workflow output:

```bash
IMAGE=ghcr.io/your-org/your-service@sha256:...
REPOSITORY=your-org/your-service
WORKFLOW_IDENTITY=https://github.com/your-org/your-service/.github/workflows/cd.yml@refs/tags/v1.2.3
```

Verify the keyless image signature, build provenance, and digest-bound
CycloneDX SBOM:

```bash
cosign verify \
  --certificate-identity "${WORKFLOW_IDENTITY}" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  "${IMAGE}"

gh attestation verify "oci://${IMAGE}" --repo "${REPOSITORY}"
gh attestation verify "oci://${IMAGE}" \
  --repo "${REPOSITORY}" \
  --predicate-type https://cyclonedx.org/bom
```

Verification establishes artifact identity and build origin; it does not prove
that the artifact is vulnerability-free or that a deployment is healthy.

## Gate ownership

- `.github/workflows/ci.yml` owns pull-request and push validation.
- Dependency Review blocks newly introduced runtime vulnerabilities at high or
  critical severity.
- Coverage owns ordinary suite execution; race and integration remain separate
  because they prove different risks.
- OpenAPI and SQLC sources own generated output and drift checks.
- Migration validation owns reversible rehearsal plus production-image
  migration, startup, readiness, embedded-version, read-only runtime, and
  SIGTERM proof.
- Go vulnerability, source security, secret, dependency, and container scans
  remain separate because they inspect different authorities.
- `.github/workflows/cd.yml` owns release preflight and GHCR publication.
- `railway.toml` owns only generic, non-secret Railway source-build policy.

Timed-out, cancelled, missing, or failed required checks are not passing
evidence. An intentionally skipped migration job is acceptable only when its
path detector reports that no migration, runtime-image, startup, dependency, or
workflow owner changed.

## Local and CI checks

- `make check` — ordinary edit loop.
- `make ci-local` — broad native CI aggregate.
- `make migration-validate` — disposable PostgreSQL, reversible migration
  rehearsal, production-image migrator, startup, readiness, and SIGTERM.
- `make check-full` — native, Docker-backed integration, runtime image, migration,
  and container security.
- `make pr-check BASE_REF=origin/main` — full proof plus base-relative OpenAPI
  compatibility.

Use the smallest command that proves the current change while iterating. Before
claiming image, migration, or deployment readiness, use the Docker-backed gate
or equivalent CI evidence.

## Deployment setup owned by the adopter

For each environment, record and verify:

- the selected repository commit or immutable image digest;
- deployment service, region, domain, and dependency network path;
- secret and database-reference ownership;
- CPU, memory, replica, autoscaling, and cost limits;
- private or authenticated access to `/metrics`;
- continuous monitoring in addition to deployment healthchecks;
- rollout, rollback, retention, and incident authority;
- Railway `Wait for CI` when using GitHub-triggered source deployments.

Do not copy example capacity values into production without a workload,
measurement, and budget owned by that service.
