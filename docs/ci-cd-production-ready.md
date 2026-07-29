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
- GitHub Rulesets require the stable `ci-required` job from
  `.github/workflows/ci.yml` and the repository's code-scanning gate;
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

`.github/workflows/cd.yml` builds one image, validates migrations and runtime
behavior against that image, scans it, generates its SBOM, pushes an immutable
build tag to obtain the registry digest, signs and attests that digest, verifies
the evidence, and only then promotes `main`, version, or `latest` tags.

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
- `ci-required` is the stable merge-admission context. It fails unless every
  applicable local CI job succeeds. It accepts PR-only jobs as skipped outside
  pull requests and runtime-heavy jobs as skipped only after the fail-closed
  path classifier proves a Markdown-only instruction, `docs/`, or `specs/`
  change.
- Markdown-only routing also skips repo-integrity Go setup,
  module/format/generated checks, and Dependency Review. Those paths cannot
  alter module manifests; repository integrity, the secret job with its pinned
  Go-based scanner, and `ci-required` remain mandatory.
- Coverage owns ordinary suite execution; race and integration remain separate
  because they prove different risks. Mandatory lint owns `govet`, so repeated
  current-tree test commands disable their duplicate default vet pass.
- Delivery quality checks GitHub Actions semantics with actionlint,
  medium-or-higher severity and high confidence workflow-security findings with
  zizmor, tracked shell scripts with digest-pinned ShellCheck, and the
  production Dockerfile with BuildKit's native checks. The job is required for
  every non-docs-only change and enters through `make delivery-quality`.
- Race, coverage, and integration restore a runtime-only Go cache. Tool jobs
  include `tools/go.sum`; every lint-consuming CI, template-proof, and
  release-preflight job uses a SHA-pinned official installer for the exact
  golangci-lint version in that module and still enters through its Make or
  template-check owner.
- OpenAPI and SQLC sources own generated output and drift checks.
- The container job builds one production image; migration validation and
  Trivy reuse that exact tag. Migration validation owns reversible rehearsal plus
  production-image
  migration, startup, readiness, embedded-version, read-only runtime, and
  SIGTERM proof.
- Go vulnerability, source security, secret, dependency, and container scans
  remain separate because they inspect different authorities.
- `.github/workflows/cd.yml` owns release preflight and GHCR publication.
  Publication jobs are admitted only when the repository variable
  `ENABLE_GHCR_PUBLISH` is exactly `true`; derived repositories otherwise
  perform no registry login, push, signing, or attestation.
- Publish jobs install the pinned Cosign release directly, without a Go
  toolchain, and the second Trivy invocation reuses the binary installed by the
  vulnerability scan. SBOM generation, digest signing, attestations, and both
  verifier paths remain mandatory.
- `railway.toml` owns only generic, non-secret Railway source-build policy.

Timed-out, cancelled, missing, or failed required checks are not passing
evidence. Empty, mixed, unrecognized, manually dispatched, or unresolvable path
sets run the full matrix. Secret and repository-integrity proof remains active
for Markdown-only changes.

## GitHub merge and release governance

Add `ci-required` to a ruleset only after it has reported successfully in the
repository. Keep the existing required contexts during that first run, require
both old and new contexts for one pull request, then retire the individual
local CI contexts. Keep an independently managed code-scanning context such as
`Analyze (go)` required because a job in `ci.yml` cannot aggregate a different
workflow.

For `main`, require pull requests, strict up-to-date checks, resolved
conversations, and block deletion and force-push. A single-maintainer public
template may use zero required approvals because authors cannot approve their
own pull requests. Organization-owned production services should normally
require one approval and code-owner approval for critical paths.

Protect `v*` tags with a separate tag ruleset. Restrict creation, update, and
deletion to a named release actor; do not enable the creation restriction until
that actor and its emergency recovery path are proven.

Rulesets are external administrative state. The template documents the policy
but does not ship a credentialed script that rewrites repository settings.
Enable the repository setting that requires full commit SHAs for third-party
actions. It does not enforce reusable-workflow pins, so keep those full-SHA
references reviewable and Dependabot-managed.

## Optional organization profile

The public template remains complete and must not call an original
organization's workflows. An organization may later replace only two stable
policy boundaries with full-SHA-pinned reusable workflows:

- `go-policy.yml`: no inputs or secrets; dependency review plus
  `make go-security`; pull requests run `make secret-scan` with the trusted base
  SHA, while main and release run `make secret-scan-history`; read-only
  permissions and one final `required` job;
- `sign-attest-image.yml`: one validated
  `ghcr.io/<organization>/<service>@sha256:...` input; scan, CycloneDX SBOM,
  keyless signing, provenance and SBOM attestations, and verification for that
  digest.

Service tests, OpenAPI and migration policy, Docker context, image build and
tags, triggers, concurrency, environments, deployment, and rollback stay in
the service repository. `go-policy.yml` needs only `contents: read`.
`sign-attest-image.yml` and its caller grant only `contents: read`,
`packages: write`, `id-token: write`, and `attestations: write`; a called
workflow cannot elevate caller permissions. Do not use `secrets: inherit`.
Constrain OIDC trust to the caller organization or repository and the exact
central workflow path and SHA.

Publish reusable workflow releases with immutable component tags for operators,
but callers reference the corresponding full commit SHA and retain the tag in a
comment. The existing GitHub Actions Dependabot configuration can propose pin
updates. Test a new pin on public, private, fork, Dependabot, main-push, release,
and permission-denial fixtures before canary adoption.

Organization ruleset-required workflows are an optional later enforcement
layer, not a template prerequisite. Enable them only after plan and visibility
eligibility, check naming, repository creation, merge-group behavior, and
ruleset-pin rollback are proven in Evaluate mode.
Do not both call the policy workflow explicitly and require the same workflow
through a ruleset in one repository.

## Local and CI checks

- `make check` — ordinary edit loop.
- `make ci-local` — broad native CI aggregate.
- `make secret-scan` — current worktree plus base-to-HEAD commits;
  `make secret-scan-history` — full-history main/release proof.
- `make migration-validate` — disposable PostgreSQL, reversible migration
  rehearsal, production-image migrator, startup, readiness, and SIGTERM.
- `make check-full` — native, Docker-backed integration, runtime image, migration,
  and container security.
- `make pr-check BASE_REF=origin/main` — full proof plus template,
  downloaded-module, and base-relative OpenAPI compatibility.
- CI's parallel `template-minimal-feature` and `template-postgres-feature` jobs
  initialize temporary generated services independently. The PostgreSQL job
  generates its first POST contract and sqlc query, and proves valid,
  rejected, commit, and rollback paths against the pinned PostgreSQL image.
  The fixture is not shipped in the base runtime.
- Nightly does not rerun deterministic merge gates on the same commit. It owns
  repeated flake/fuzz execution, fresh container integration, benchmark
  lifecycle verification, current vulnerability analysis, and a fresh runtime
  image scan.

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
