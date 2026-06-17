# syntax=docker/dockerfile:1

# Tooling image catalog consumed by scripts/dev/docker-tooling.sh.
# Keep these references pinned by digest; Dependabot updates this file.
FROM golang:1.26.4-bookworm@sha256:5d2b868674b57c9e48cdd39e891acce4196b6926ca6d11e9c270a8f85106203d AS go_toolchain
FROM node:20.20.0-bookworm@sha256:65b74d0fb42134c49530a8c34e9f3e4a2fb8e1f99ac4a0eb4e6f314b426183a2 AS node_toolchain
FROM postgres:17@sha256:2cd82735a36356842d5eb1ef80db3ae8f1154172f0f653db48fde079b2a0b7f7 AS postgres_tool
FROM migrate/migrate:v4.19.1@sha256:cc4ad8e19d66791e3689405d9a028ce6e9614f32032db14acda1469f7201d6e4 AS migrate_tool
FROM aquasec/trivy:0.69.2@sha256:3d1f862cb6c4fe13c1506f96f816096030d8d5ccdb2380a3069f7bf07daa86aa AS trivy_tool
