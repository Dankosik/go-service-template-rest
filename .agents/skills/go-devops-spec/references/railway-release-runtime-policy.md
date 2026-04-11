# Railway Release Runtime Policy

## Behavior Change Thesis
When loaded for symptom "the delivery spec needs platform rollout or runtime policy for this repo's deployment target," this file makes the model choose repo-reviewable Railway health, overlap, draining, restart, and capacity evidence instead of likely mistake generic Kubernetes rollout advice or "monitor after deploy" with no platform-enforced gate.

## When To Load
Load for Railway deployment policy, release promotion/rollback criteria, healthcheck wiring, overlap/draining windows, restart policy, production replica/capacity baseline, or platform drift in `railway.toml`.

## Local Source Of Truth
- `railway.toml` is the non-secret deployment-policy source of truth and says policy changes must pass PR review plus `make guardrails-check`.
- `railway.toml` pins `builder = "DOCKERFILE"` and `dockerfilePath = "build/docker/Dockerfile"`.
- `railway.toml` sets `/health/ready`, `healthcheckTimeout = 180`, `restartPolicyType = "ON_FAILURE"`, `restartPolicyMaxRetries = 5`, `overlapSeconds = 45`, and `drainingSeconds = 30`.
- `railway.toml` records baseline assertions: production replicas `>=2` and per-replica baseline `2 vCPU / 2 GiB`.
- `scripts/ci/required-guardrails-check.sh` fails if these Railway policy assertions drift.

## Decision Rubric
- Treat `railway.toml` as the reviewable policy surface; secrets and environment-specific secret values stay in Railway variables, not repository files.
- Deployment specs must tie promotion/rollback to `/health/ready`, overlap/draining windows, restart policy, and objective post-deploy signals.
- If release risk depends on capacity, name the replica and per-replica baselines and the evidence that the target environment satisfies them.
- If the spec proposes a different builder, Dockerfile path, healthcheck, overlap, draining, or restart policy, require a guardrail update or an explicit exception.
- Do not import Kubernetes concepts unless the deployment target actually changes or a Kubernetes manifest/admission surface is part of the plan.

## Imitate
- "Railway release promotion requires build through `build/docker/Dockerfile`, `/health/ready` passing within 180 seconds, 45-second overlap, 30-second draining, and `ON_FAILURE` restart policy with max retries 5." Copy the repo-specific platform knob naming.
- "Production readiness evidence includes target service variables present in Railway, replica baseline `>=2`, and per-replica `2 vCPU / 2 GiB` capacity evidence; secrets are verified in Railway, not written to repo." Copy the secret boundary and capacity proof.
- "A healthcheck path change must update `railway.toml`, the guardrails checker, and the service route/spec that owns readiness semantics." Copy the cross-surface drift rule.

## Reject
- "Use a Kubernetes rolling update with readiness probes." This is wrong for Railway unless the deployment platform changes.
- "Deploy and monitor logs manually." This lacks healthcheck, overlap/draining, restart, and rollback gates.
- "Store production API keys in `railway.toml` for reproducibility." This violates the non-secret policy surface.

## Agent Traps
- Do not treat `railway.toml` as a secret/config dumping ground; it is non-secret deployment policy.
- Do not let a Dockerfile hardening decision bypass Railway's pinned Dockerfile path.
- Do not invent deployment-platform capabilities; if Railway behavior is uncertain and it materially affects release safety, record a verification blocker rather than filling in generic cloud behavior.

## Validation Shape
Use `make guardrails-check`, the `railway.toml` diff, Railway deployment logs/status, `/health/ready` evidence during the promotion window, restart/rollback event evidence, and environment evidence for replica/capacity baselines and required variables.

## Hand-Off Boundary
Do not define application readiness semantics, capacity model, or secret values here. Delivery owns the platform evidence and drift controls; application, reliability, and security specs own those underlying decisions.
