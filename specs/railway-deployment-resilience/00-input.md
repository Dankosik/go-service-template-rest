# 00 Input

## User Request
- Service is already auto-deployed to Railway from GitHub `main` after CI.
- Need production-readiness hardening for Railway settings without changing product behavior.
- Focus areas: `Networking`, `Scale`, `Build`, `Deploy`, retries/recovery behavior, replica strategy, and `Config-as-code`.
- Current state from screenshots (2026-03-04):
  - `Wait for CI`: enabled.
  - Scale: `1` replica in `EU West (Amsterdam)`.
  - Replica limits: `CPU 32 vCPU`, `Memory 32 GB` (max plan limits).
  - Builder: `Railpack` (`go@1.26.0`), no custom build/start/pre-deploy command.
  - `Teardown`: disabled.
  - `Healthcheck Path`: not configured.
  - `Serverless`: disabled.
  - `Restart Policy`: `On Failure`, retries `10`.
  - `Config-as-code`: not configured.

## Scope
- Railway deployment policy and runtime settings hardening for production.
- Operational safety defaults for rollout/restart/health checks.
- Reproducibility of deploy config in repository.

## Explicit Decision
- Keep deployment model as Railway GitHub-based auto-deploy; do not redesign platform.
