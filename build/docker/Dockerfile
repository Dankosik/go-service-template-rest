# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26.5-bookworm@sha256:1ecb7edf62a0408027bd5729dfd6b1b8766e578e8df93995b225dfd0944eb651 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG APP_VERSION=dev
ARG VCS_REF=unknown
ARG RAILWAY_GIT_COMMIT_SHA
ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0
RUN module_path="$(go list -m)" && \
    resolved_version="${APP_VERSION}" && \
    if [ "${resolved_version}" = "dev" ] && [ -n "${RAILWAY_GIT_COMMIT_SHA}" ]; then \
      resolved_version="sha-$(printf '%.12s' "${RAILWAY_GIT_COMMIT_SHA}")"; \
    fi && \
    ldflags="-s -w -buildid= -X ${module_path}/internal/config.buildVersion=${resolved_version}" && \
    GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
      -trimpath \
      -mod=readonly \
      -buildvcs=false \
      -ldflags="${ldflags}" \
      -tags=timetzdata \
      -o /out/service ./cmd/service
# profile:database-postgres:start
RUN module_path="$(go list -m)" && \
    resolved_version="${APP_VERSION}" && \
    if [ "${resolved_version}" = "dev" ] && [ -n "${RAILWAY_GIT_COMMIT_SHA}" ]; then \
      resolved_version="sha-$(printf '%.12s' "${RAILWAY_GIT_COMMIT_SHA}")"; \
    fi && \
    ldflags="-s -w -buildid= -X ${module_path}/internal/config.buildVersion=${resolved_version}" && \
    GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build \
      -trimpath \
      -mod=readonly \
      -buildvcs=false \
      -ldflags="${ldflags}" \
      -tags=timetzdata \
      -o /out/migrate ./cmd/migrate
RUN mkdir -p /out/migrations && \
    if [ -d /src/migrations ]; then cp -R /src/migrations/. /out/migrations/; fi
# profile:database-postgres:end

FROM gcr.io/distroless/static-debian12:nonroot@sha256:f5b485ea962d9bd1186b2f6b3a061191539b905b82ec395de78cbfae51f20e35

ARG APP_VERSION=dev
ARG VCS_REF=unknown
ARG SOURCE_URL=""
LABEL org.opencontainers.image.version="${APP_VERSION}" \
      org.opencontainers.image.revision="${VCS_REF}" \
      org.opencontainers.image.source="${SOURCE_URL}"

WORKDIR /
COPY --from=build /out/service /service
# profile:database-postgres:start
COPY --from=build /out/migrate /migrate
COPY --from=build /out/migrations /migrations
# profile:database-postgres:end

EXPOSE 8080
STOPSIGNAL SIGTERM
USER nonroot:nonroot
ENTRYPOINT ["/service"]
