# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.25.7-bookworm@sha256:564e366a28ad1d70f460a2b97d1d299a562f08707eb0ecb24b659e5bd6c108e1 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build \
      -trimpath \
      -mod=readonly \
      -buildvcs=false \
      -ldflags="-s -w -buildid=" \
      -tags=timetzdata \
      -o /out/service ./cmd/service

FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1

WORKDIR /
COPY --from=build /out/service /service

EXPOSE 8080
STOPSIGNAL SIGTERM
USER nonroot:nonroot
ENTRYPOINT ["/service"]
