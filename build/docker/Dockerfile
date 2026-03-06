# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26.1-bookworm@sha256:c7a82e9e2df2fea5d8cb62a16aa6f796d2b2ed81ccad4ddd2bc9f0d22936c3f2 AS build

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

FROM gcr.io/distroless/static-debian12:nonroot@sha256:a9329520abc449e3b14d5bc3a6ffae065bdde0f02667fa10880c49b35c109fd1

WORKDIR /
COPY --from=build /out/service /service

EXPOSE 8080
STOPSIGNAL SIGTERM
USER nonroot:nonroot
ENTRYPOINT ["/service"]
