# syntax=docker/dockerfile:1

# Tooling image catalog consumed by scripts/dev/docker-tooling.sh.
# Keep these references pinned by digest; Dependabot updates this file.
FROM golang:1.25.7-bookworm@sha256:564e366a28ad1d70f460a2b97d1d299a562f08707eb0ecb24b659e5bd6c108e1 AS go_toolchain
FROM node:20.19.0-bookworm@sha256:a5fb035ac1dff34a4ecaea85f90f7321185695d3fd22c12ba12f4535a4647cc5 AS node_toolchain
FROM golangci/golangci-lint:v2.10.1@sha256:ea84d14c2fef724411be7dc45e09e6ef721d748315252b02df19a7e3113ee763 AS golangci_lint_tool
FROM postgres:17@sha256:2cd82735a36356842d5eb1ef80db3ae8f1154172f0f653db48fde079b2a0b7f7 AS postgres_tool
FROM migrate/migrate:v4.19.0@sha256:d5c978181e3bfa55cc50e3bd8d7da3d87418a87693453250a8804b81ee6494db AS migrate_tool
FROM aquasec/trivy:0.65.0@sha256:a22415a38938a56c379387a8163fcb0ce38b10ace73e593475d3658d578b2436 AS trivy_tool
