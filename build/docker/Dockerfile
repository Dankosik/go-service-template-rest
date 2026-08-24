# syntax=docker/dockerfile:1

ARG SOURCE_DATE_EPOCH=0

FROM --platform=$BUILDPLATFORM golang:1.27.0-bookworm@sha256:484ef6066fa69acb059fdfeda7ba2b8f7391f2ef6abc6f9b8411e669ebd56466 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

ARG APP_VERSION=dev
ARG VCS_REF=unknown
ARG RAILWAY_GIT_COMMIT_SHA
ARG SOURCE_DATE_EPOCH
ARG PGO_PROFILE=off
ARG PGO_MANIFEST
ARG PGO_ACCEPTED_BUILD_FINGERPRINT
ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0
RUN --mount=type=bind,target=. \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    if [ "${PGO_PROFILE}" = "auto" ] || [ -z "${PGO_PROFILE}" ]; then \
      echo "PGO_PROFILE must be off or name an explicit representative CPU profile" >&2; \
      exit 2; \
    fi && \
    if [ "${PGO_PROFILE}" != "off" ]; then \
      test -n "${PGO_MANIFEST}" && \
      test -n "${PGO_ACCEPTED_BUILD_FINGERPRINT}" && \
      PGO_TARGET_GOOS="${TARGETOS:-linux}" PGO_TARGET_GOARCH="${TARGETARCH:-amd64}" \
        bash ./scripts/dev/benchmark.sh verify-pgo \
          "${PGO_PROFILE}" "${PGO_MANIFEST}" "${PGO_ACCEPTED_BUILD_FINGERPRINT}"; \
    fi && \
    module_path="$(go list -m)" && \
    build_packages="./cmd/service" && \
# profile:messaging-nats-jetstream:start
    if [ -d ./cmd/worker ]; then build_packages="${build_packages} ./cmd/worker"; fi && \
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
    if [ -d ./cmd/outbox-relay ]; then build_packages="${build_packages} ./cmd/outbox-relay"; fi && \
# profile:outbox-postgres:end
# profile:jobs-postgres:start
    if [ -d ./cmd/jobs-worker ]; then build_packages="${build_packages} ./cmd/jobs-worker"; fi && \
# profile:jobs-postgres:end
    if [ -d ./cmd/migrate ]; then build_packages="${build_packages} ./cmd/migrate"; fi && \
    resolved_version="${APP_VERSION}" && \
    if [ "${resolved_version}" = "dev" ] && [ -n "${RAILWAY_GIT_COMMIT_SHA}" ]; then \
      resolved_version="sha-$(printf '%.12s' "${RAILWAY_GIT_COMMIT_SHA}")"; \
    fi && \
    resolved_commit="${RAILWAY_GIT_COMMIT_SHA:-${VCS_REF}}" && \
    ldflags="-s -w -buildid= -X ${module_path}/internal/config.buildVersion=${resolved_version} -X ${module_path}/internal/config.buildCommit=${resolved_commit}" && \
    GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
      -pgo="${PGO_PROFILE}" \
      -trimpath \
      -mod=readonly \
      -buildvcs=false \
      -ldflags="${ldflags}" \
      -o /out/ ${build_packages}
# profile:database-postgres:start
RUN --mount=type=bind,target=. \
    mkdir -p /out/migrations && \
    if [ -d /src/migrations ]; then cp -R /src/migrations/. /out/migrations/; fi
# profile:database-postgres:end
RUN find /out -exec touch -d "@${SOURCE_DATE_EPOCH}" {} +

FROM gcr.io/distroless/static-debian12:nonroot@sha256:1b7b9f0f0e0a1d2155f531db587cc48ec26aaf97ab64364225f5bf18a054e66a

ARG APP_VERSION=dev
ARG VCS_REF=unknown
ARG RAILWAY_GIT_COMMIT_SHA
ARG SOURCE_URL=""
LABEL org.opencontainers.image.version="${RAILWAY_GIT_COMMIT_SHA:-$APP_VERSION}" \
      org.opencontainers.image.revision="${RAILWAY_GIT_COMMIT_SHA:-$VCS_REF}" \
      org.opencontainers.image.source="${SOURCE_URL}"

WORKDIR /
COPY --from=build /out/service /service
# profile:messaging-nats-jetstream:start
COPY --from=build /out/worker /worker
# profile:messaging-nats-jetstream:end
# profile:outbox-postgres:start
COPY --from=build /out/outbox-relay /outbox-relay
# profile:outbox-postgres:end
# profile:jobs-postgres:start
COPY --from=build /out/jobs-worker /jobs-worker
# profile:jobs-postgres:end
# profile:database-postgres:start
COPY --from=build /out/migrate /migrate
COPY --from=build /out/migrations /migrations
# profile:database-postgres:end

EXPOSE 8080
STOPSIGNAL SIGTERM
USER nonroot:nonroot
ENTRYPOINT ["/service"]
