# Trust Boundary And Threat Modeling

## Behavior Change Thesis
When loaded for a flow with implicit trust boundaries, this file makes the model choose named boundary, attacker-path, enforcement, and proof requirements instead of likely mistake: generic "use auth" or "validate input" controls detached from where trust is crossed.

## When To Load
Load this when a new or changed flow crosses external, partner, internal-service, worker, queue, database, cache, secret-store, telemetry, or third-party API boundaries and the security owner or attacker path is still fuzzy.

## Decision Rubric
- Name every boundary crossed by the flow, including generated API surfaces, async workers, outbound calls, caches, telemetry exporters, and health or management endpoints.
- Treat every input as untrusted until the producing boundary and verification control are explicit.
- For each meaningful boundary, write one attacker path and one chosen mitigation with the enforcement point.
- Split "authentication", "authorization", "tenant isolation", "input integrity", "secret handling", and "abuse resistance" when they require different owners or proof.
- If a mitigation depends on a gateway, service mesh, identity provider, queue policy, or secret manager not visible in the repo, record it as an external dependency and require service-local defense or a blocker.

## Imitate
- "For the webhook callback boundary, require provider signature verification over the canonical payload before any side effect; unverifiable callbacks are discarded with a bounded security event." Copy the boundary, verification point, denial behavior, and side-effect order.
- "For internal HTTP calls carrying `X-User-Id` or `X-Tenant-Id`, require those headers to be bound to an authenticated upstream contract; otherwise the service derives identity from its own verified auth context." Copy the refusal to treat internal headers as magically trusted.
- "For fields leaving the service through telemetry, unknown data classification is treated as sensitive until approved otherwise." Copy the data-classification fail-closed default at a non-API boundary.

## Reject
- "Internal traffic is trusted because it is inside the VPC." This skips producer identity, authorization, and tenant or data-scope checks.
- "OpenAPI security will cover this later." This points at a possible artifact, not an enforcement owner or proof.
- "Validate input." This is not a threat response until it names the boundary, invalid cases, enforcement point, and negative proof.

## Agent Traps
- Do not omit async workers, outbound HTTP, telemetry, caches, or generated handlers because they are not user-facing.
- Do not let a single broad threat model replace concrete requirements. Requirements must still name who enforces the control and what denial looks like.
- Do not rely on client-side code, generated comments, docs, or dashboard-only monitoring as the sole mitigation.

## Validation Shape
- Boundary matrix: affected input or output surface -> producer or consumer boundary -> verification control -> owner -> negative proof.
- Fail-closed checks for unauthenticated requests, unsigned messages, unverified callbacks, untrusted headers, unknown data classification, and degraded auth, policy, secret, cache, or third-party dependencies.

## Repo-Local Anchors
- `api/openapi/service.yaml` currently declares `security: []` globally and a `bearerAuth` JWT scheme as a component, so security requirements must distinguish public baseline endpoints from future protected operations.
- `internal/infra/http/router.go` applies request framing, body limit, security headers, access logging, recovery, request correlation, and explicit CORS preflight rejection.
- `docs/configuration-source-policy.md` defines YAML as non-secret config and `APP__...` environment variables as the secret-value boundary.
- `SECURITY.md` defines private vulnerability reporting and disclosure expectations.
