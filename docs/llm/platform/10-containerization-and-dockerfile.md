# Containerization and Dockerfile instructions for LLMs

## Load policy
- Load: Optional.
- Use when:
  - Creating or changing Dockerfiles for Go services.
  - Reviewing image hardening, runtime user model, startup behavior, and shutdown semantics.
  - Choosing static vs dynamic linking strategy (`CGO_ENABLED=0/1`) and runtime base image.
  - Defining reproducible container build defaults, image pinning, and release artifact rules.
  - Auditing container anti-patterns and merge/release gates.
- Do not load when: Task is internal code refactoring with no build/runtime container impact.

## Purpose
- This document defines repository defaults for production containerization of Go services.
- Goal: deterministic builds, minimal attack surface, predictable runtime behavior, and reviewable exceptions.
- Defaults are mandatory unless an ADR explicitly approves an exception.

## Baseline assumptions
- Primary artifact is a Go service binary built in CI and shipped as OCI image.
- Runtime target is Kubernetes first, Docker-compatible second.
- Default service port is `8080` (non-privileged port for non-root runtime).
- Default runtime profile is distroless + non-root + exec-form startup.
- Debugging in production must not rely on shell tools inside runtime image.

## Required inputs before generating Dockerfile
Resolve these first. If unknown, apply defaults and state assumptions.

- Does the service require cgo or native shared libraries?
- Does the service need custom CA trust chain or only public CAs?
- Does runtime logic require timezone database (`time.LoadLocation`)?
- Does the service need writable filesystem paths beyond `/tmp`?
- Is multi-arch image output required (`linux/amd64` + `linux/arm64`)?
- Must release build be fully reproducible by digest and attestation policy?

## Mandatory Dockerfile defaults

### Default profile (MUST)
- Dockerfile MUST use multi-stage build: separate `build` and `runtime` stages.
- Runtime stage MUST contain only the executable and required runtime files.
- Runtime base MUST be minimal and non-root by default: `gcr.io/distroless/static-debian12:nonroot`.
- Service startup MUST use exec-form `ENTRYPOINT` (JSON array), not shell form.
- Runtime user MUST be non-root (via `:nonroot` tag or explicit `USER`).
- Dockerfile MUST avoid package manager installs in runtime stage.
- Dockerfile MUST not embed secrets in `ARG`, `ENV`, copied files, or build layers.

### Canonical default Dockerfile (static/pure Go)
```dockerfile
# syntax=docker/dockerfile:1

ARG GO_VERSION=1.24.0
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-bookworm AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build \
      -trimpath \
      -mod=readonly \
      -buildvcs=false \
      -ldflags="-s -w -buildid=" \
      -tags=timetzdata \
      -o /out/service ./cmd/service

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=build /out/service /service
EXPOSE 8080
ENTRYPOINT ["/service"]
```

### Required companion defaults
- `.dockerignore` is mandatory and MUST exclude at least `.git/`, local build artifacts, `.env*`, test/output folders, and other non-runtime files.
- Base images SHOULD be pinned by digest in release pipeline for reproducibility.
- Base image digest update cadence MUST be explicit (automated PR or scheduled maintenance).

## Static vs dynamic linking decision rules
Apply in order.

1. Default to static linking (`CGO_ENABLED=0`) for Go services.
2. If cgo is required, switch to dynamic profile explicitly and document why.
3. If dynamic profile is used, runtime base MUST include required libc/ssl dependencies.
4. If cgo is enabled, runtime image default changes to `gcr.io/distroless/base-debian12:nonroot`.
5. If required shared libs are unknown, fail generation and request explicit dependency list.

### Dynamic (cgo) profile requirements
- Build stage MUST set `CGO_ENABLED=1`.
- Runtime MUST not use distroless `static` when libc dependencies are needed.
- Review MUST include evidence that runtime libs resolve (for example, `ldd` check during CI).

## CA certificates and timezone defaults

### CA certificates
- Runtime MUST have CA trust store for outbound TLS.
- Distroless `static` and `base` are acceptable defaults because they include CA certificates.
- If using `scratch`, Dockerfile MUST explicitly copy CA bundle from builder stage.
- `InsecureSkipVerify` as workaround for missing CA is forbidden.

### Timezone data
- Default build SHOULD include `-tags timetzdata` for portable tzdb fallback.
- If service uses local timezone logic, runtime MUST provide tzdata or embedded tzdb.
- If timezone is irrelevant and UTC-only behavior is guaranteed, this may be explicitly documented as exception.

## Reproducible build defaults
- Build MUST use `-trimpath` and `-mod=readonly`.
- Build SHOULD set `-buildvcs=false` by default in Docker context to avoid hidden `.git` dependency.
- Build SHOULD use stable ldflags for reproducibility (`-buildid=` and optionally `-s -w`).
- Go toolchain version MUST be explicit (`ARG GO_VERSION=x.y.z`), not floating major-only tag.
- Runtime base SHOULD be pinned by digest for release.
- CI SHOULD publish SBOM and provenance attestations for release images.
- BuildKit provenance/SBOM emission SHOULD be enabled for release workflows.

## Startup command and signal handling baseline

### Command shape defaults
- Main service process MUST run as PID 1 directly (`ENTRYPOINT ["/service"]`).
- Shell-form startup (`ENTRYPOINT /service`, `CMD service`) is forbidden by default.
- Wrapper scripts are exception-only and MUST `exec` the final process.

### Signal and termination defaults
- Service code MUST handle SIGTERM for graceful shutdown (for example via `signal.NotifyContext`).
- Container stop flow MUST assume orchestrator sends `SIGTERM` then `SIGKILL` after grace timeout.
- Dockerfile MAY set `STOPSIGNAL SIGTERM` explicitly when runtime requires clarity.

## Image size vs operability trade-offs

| Runtime base | Default usage | Pros | Risks / constraints |
|---|---|---|---|
| Distroless `static` | Default for pure Go | Small, minimal attack surface, includes CA/tzdata, non-root tag | No shell/package manager in runtime |
| Distroless `base` | Default for cgo workloads | Includes glibc/libssl, still minimal and non-root | Larger than `static`, still no shell |
| `scratch` | Opt-in only | Minimal size | Must manage CA/tzdata/files manually; easy to break TLS/timezone behavior |
| Full/minimal distro (Debian/Alpine) | Exception-only | Easier interactive debugging | Larger surface area, extra CVE noise, stricter hardening required |

Decision policy:
- Choose smallest runtime that still satisfies real runtime dependencies.
- Do not switch to heavier base image only for convenience debugging.
- If heavier base is used, exception note MUST include security and operability rationale.

## Runtime hardening baseline

### Container-level defaults
- Run as non-root user.
- Read-only root filesystem by default.
- Writable paths restricted to explicit mounts and `/tmp`.
- No privilege escalation.
- Drop all Linux capabilities by default.
- Add back only `NET_BIND_SERVICE` when binding privileged ports is unavoidable.
- Avoid privileged mode, hostPID, hostNetwork, and hostPath unless explicitly approved.

### Kubernetes baseline (target: Restricted)
- `securityContext.allowPrivilegeEscalation: false`
- `securityContext.runAsNonRoot: true`
- `securityContext.seccompProfile.type: RuntimeDefault`
- `securityContext.capabilities.drop: ["ALL"]`
- `readOnlyRootFilesystem: true` (when service behavior permits)

## Anti-patterns (review blockers)
- Single-stage Dockerfile that leaves compiler/toolchain/cache/source in runtime image.
- Runtime container running as root or requiring `runAsUser: 0`.
- Shell-heavy startup chain (`sh -c`, `bash -c`) for main process.
- Missing CA trust store in image while service performs outbound TLS.
- Ignoring timezone requirements (`time.LoadLocation`) without tzdata or embedded tzdb.
- Installing shell/package manager into production distroless runtime image.
- Copying `.git`, secrets, or local env files into image.
- Using mutable `latest` tags for release artifacts without digest policy.
- Disabling TLS verification to compensate for broken image trust store.
- Patching packages inside running containers instead of rebuilding/redeploying image.

## Review criteria (merge gate)
- Dockerfile is multi-stage and runtime stage contains only required runtime artifacts.
- Runtime base is minimal and matches linking mode (`static` vs `base` for cgo).
- Runtime user is non-root and startup is exec-form entrypoint.
- CA cert and timezone needs are explicitly satisfied.
- Build flags include reproducibility defaults (`-trimpath`, `-mod=readonly`, version pinning).
- `.dockerignore` exists and blocks secret/context leakage.
- Runtime hardening baseline is enforced in deployment manifests.
- No shell-based PID1 startup, no root runtime, no embedded secrets.
- Exception paths (scratch/full distro/cgo deviations) are documented with rationale and risk.
- Release pipeline emits and stores required image security metadata (SBOM/provenance/signature) according to repo policy.

