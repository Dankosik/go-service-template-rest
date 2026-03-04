# 10 Context Goals And Non-Goals

## Context
`privacy-sanitization-service` is an internal sidecar before upstream LLM calls. Repository policy targets private/internal networking and deterministic sanitization behavior.

Current Railway deploy works, but deployment/runtime settings remain partially default and need production hardening.

## Goals
1. Define a stable production baseline for Railway service settings (networking, scale, build, deploy).
2. Reduce rollout and recovery risk (health checks, graceful replacement, restart behavior).
3. Avoid uncontrolled cost/performance drift by right-sizing replica limits and replica count policy.
4. Make deployment settings reproducible and reviewable in repo (`config-as-code`).
5. Keep internal-service security posture (private-by-default exposure).
6. Keep the target architecture intentionally simple (no unnecessary distributed complexity).

## Non-Goals
- Redesign business logic or API behavior of sanitization flow.
- Introduce multi-cloud orchestration.
- Change detector stack or domain policy (`redact`/`tokenize`/`block`) in this scope.
- Introduce advanced resilience mechanisms without clear incident or capacity evidence.

## Constraints
- Existing runtime currently exposes `GET /health/live` and `GET /health/ready` (target `GET /health` is future scope).
- Existing hardened Dockerfile is located at `build/docker/Dockerfile`.
- Deployment remains GitHub-driven with CI gating (`Wait for CI` already enabled).
- Service must preserve graceful shutdown semantics and not log sensitive raw content.
- Simplicity-first rule: prefer the minimum control set that materially improves reliability.
