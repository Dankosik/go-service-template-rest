# syntax=docker/dockerfile:1

FROM --platform=$BUILDPLATFORM golang:1.26.2-bookworm@sha256:4f4ab2c90005e7e63cb631f0b4427f05422f241622ee3ec4727cc5febbf83e34 AS build

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
