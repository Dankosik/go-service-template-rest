# Container Runtime Hardening

## When To Load
Load this when specifying Dockerfile expectations, image build evidence, runtime user/entrypoint policy, minimal images, `.dockerignore`, image scanning, Kubernetes `securityContext`, or runtime-hardening exceptions.

## Local Source Of Truth
- `build/docker/Dockerfile` uses a multi-stage Go build, pinned base images by digest, `CGO_ENABLED=0`, `-trimpath`, `-mod=readonly`, `-buildvcs=false`, stripped deterministic build flags, `timetzdata`, distroless static nonroot runtime, `USER nonroot:nonroot`, `STOPSIGNAL SIGTERM`, and exec-form `ENTRYPOINT`.
- `.dockerignore` excludes repository metadata, env files, docs, scripts, tests, coverage, temp files, dist, vendor, and Markdown from the image build context.
- `.github/workflows/ci.yml`, `.github/workflows/nightly.yml`, and `.github/workflows/cd.yml` build and scan container images with Trivy.
- `Makefile` exposes `docker-build`, `docker-run`, `docker-container-security`, and `docker-ci`.

## Enforceable Policy Examples
- Runtime images must not contain the Go toolchain, package manager, shell, test tree, or source tree unless a named debugging or compliance exception approves it.
- Dockerfile changes must preserve non-root execution, exec-form entrypoint, minimal runtime image, pinned base image digest, deterministic build flags, and explicit Go version control, or record a scoped exception.
- Secret material must not enter build args, image layers, logs, or committed Dockerfiles; when build-time secrets are unavoidable, require BuildKit secret mounts and proof that the secret is not present in the final image.
- Container release is blocked on HIGH/CRITICAL Trivy findings unless an exception names owner, CVE/finding, reason, expiry, and compensating control.
- Kubernetes runtime specs, when present, should require `runAsNonRoot`, `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, dropped capabilities, no privileged mode, and `RuntimeDefault` seccomp unless workload constraints justify an exception.

## Non-Enforceable Anti-Patterns
- "Use a secure image" without naming base-image, user, entrypoint, capabilities, filesystem, and scan gates.
- Floating base tags without digest or update policy.
- Debug shells in production images without a bounded exception.
- Passing credentials through `ARG` or `ENV` in Docker builds.
- Copying the entire repo into the runtime stage rather than copying only the compiled artifact and required runtime data.
- Declaring Kubernetes hardening while leaving the manifest fields unset and unenforced.

## Evidence Artifacts
- Docker build log that names `build/docker/Dockerfile` and target image reference.
- Container scan log from Trivy with exit-code policy.
- Dockerfile diff showing preserved multi-stage, non-root, pinned digest, and deterministic build flags.
- `.dockerignore` diff when build context changes.
- Kubernetes manifest, Helm output, or admission-policy evidence showing runtime securityContext enforcement when Kubernetes is the deployment target.

## Hand-Off Boundary
Do not decide service architecture, runtime dependency needs, or application-level trust-boundary behavior here. Delivery owns image/runtime enforcement; application architecture and security specs own why a runtime exception is necessary.

## Exa Source Links
- Docker Docs: [Building best practices](https://docs.docker.com/build/building/best-practices/)
- Docker Docs: [Build secrets](https://docs.docker.com/build/building/secrets/)
- Docker Docs: [Distroless images](https://docs.docker.com/dhi/core-concepts/distroless)
- Kubernetes Docs: [Configure a Security Context for a Pod or Container](https://kubernetes.io/docs/tasks/configure-pod-container/security-context/)
- Kubernetes Docs: [Pod Security Standards](https://kubernetes.io/docs/concepts/security/pod-security-standards)
- Kubernetes Docs: [Application Security Checklist](https://kubernetes.io/docs/concepts/security/application-security-checklist)

