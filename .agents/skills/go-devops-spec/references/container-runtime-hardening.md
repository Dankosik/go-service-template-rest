# Container Runtime Hardening

## Behavior Change Thesis
When loaded for symptom "the spec touches Dockerfile, image contents, or runtime security posture," this file makes the model choose the repository's minimal, non-root, digest-pinned image baseline and scan gates instead of likely mistake "use a secure image" or adding generic Kubernetes hardening with no enforcement surface.

## When To Load
Load for Dockerfile expectations, image build evidence, runtime user/entrypoint policy, minimal images, `.dockerignore`, image scanning, Kubernetes `securityContext`, or runtime-hardening exceptions.

## Local Source Of Truth
- `build/docker/Dockerfile` uses a multi-stage Go build, pinned base images by digest, `CGO_ENABLED=0`, `-trimpath`, `-mod=readonly`, `-buildvcs=false`, stripped deterministic build flags, `timetzdata`, distroless static nonroot runtime, `USER nonroot:nonroot`, `STOPSIGNAL SIGTERM`, and exec-form `ENTRYPOINT`.
- `.dockerignore` excludes repository metadata, env files, docs, scripts, tests, coverage, temp files, dist, vendor, and Markdown from the image build context.
- `.github/workflows/ci.yml`, `.github/workflows/nightly.yml`, and `.github/workflows/cd.yml` build and scan images with Trivy.
- `Makefile` exposes `docker-build`, `docker-run`, `docker-container-security`, and `docker-ci`.

## Decision Rubric
- Runtime images must not contain the Go toolchain, package manager, shell, test tree, or source tree unless a named debugging/compliance exception approves it.
- Dockerfile changes must preserve non-root execution, exec-form entrypoint, minimal runtime image, pinned base image digest, deterministic build flags, and explicit Go version control, or record a scoped exception.
- Secret material must not enter build args, image layers, logs, or committed Dockerfiles; unavoidable build-time secrets require BuildKit secret mounts and proof that the secret is absent from the final image.
- Container release is blocked on HIGH/CRITICAL Trivy findings unless an exception names owner, finding/CVE, affected artifact, expiry, and compensating control.
- Kubernetes `securityContext` policy is relevant only when Kubernetes manifests are in scope; then require `runAsNonRoot`, `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, dropped capabilities, no privileged mode, and `RuntimeDefault` seccomp unless workload constraints justify an exception.

## Imitate
- "A Dockerfile change must preserve `USER nonroot:nonroot`, exec-form `ENTRYPOINT`, digest-pinned base images, and deterministic Go build flags; evidence is the Dockerfile diff plus `make docker-container-security`." Copy the baseline-preservation rule.
- "A new runtime dependency that needs a shell is an exception request, not a default image expansion." Copy the minimal-image posture.
- "Kubernetes hardening belongs in the spec only when a manifest/admission surface will enforce it." Copy the enforcement-surface test.

## Reject
- "Use a secure base image." This does not name digest pinning, runtime user, entrypoint, image contents, or scan gates.
- "Pass the private token through `ARG` during docker build." This risks layer/log leakage; require BuildKit secret mounts or redesign.
- "Copy the whole repo into the runtime image." This violates the compiled-artifact-only runtime stage.

## Agent Traps
- Do not confuse build-stage tools with allowed runtime contents.
- Do not require Kubernetes fields for a non-Kubernetes deployment unless a platform-specific admission/control surface is planned.
- Do not accept a scan exception without owner, finding, expiry, and rescan condition.

## Validation Shape
Use Docker build logs naming `build/docker/Dockerfile`, Trivy scan output with exit-code policy, Dockerfile diff preserving baseline controls, `.dockerignore` diff when build context changes, and Kubernetes manifest/Helm/admission evidence only when Kubernetes is the deployment target.

## Hand-Off Boundary
Do not decide service architecture, runtime dependency needs, or application-level trust-boundary behavior here. Delivery owns image/runtime enforcement; application architecture and security specs own why a runtime exception is necessary.
