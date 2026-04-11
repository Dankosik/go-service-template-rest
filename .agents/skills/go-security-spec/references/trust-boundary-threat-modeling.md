# Trust Boundary And Threat Modeling Examples

## When To Load
Load this when designing security requirements before coding for a new or changed flow, when trust boundaries are implicit, when external/partner/internal/async/third-party boundaries are crossed, or when a threat-control matrix needs examples.

## Selected Controls
- Require a boundary map that names external clients, partner APIs, internal services, workers, queues, databases, caches, secret stores, observability sinks, and third-party APIs.
- Classify each input as untrusted unless the producing boundary and verification control are explicit.
- Require one attacker path per meaningful boundary. Use prompts such as spoofing, tampering, repudiation, information disclosure, denial of service, and elevation of privilege.
- Convert mitigated threats into requirements that name the enforcement point and proof obligation.
- Use repository anchors when available: OpenAPI contract in `api/openapi/service.yaml`, HTTP boundary controls in `internal/infra/http`, config secret policy in `docs/configuration-source-policy.md`, and disclosure process in `SECURITY.md`.

## Rejected Controls
- Reject generic statements such as "validate input" or "use auth" when they do not identify an attacker path, boundary, enforcement point, and proof.
- Reject trusting internal service traffic by default. Internal callers still need authentication, authorization, and tenant or data-scope checks where they can trigger side effects.
- Reject threat models that omit async workers, webhooks, outbound HTTP calls, telemetry exporters, caches, or generated API surfaces because they are "not user facing".
- Reject a single mitigation that only lives in client-side code, generated code comments, or documentation.

## Fail-Closed Examples
- If the flow cannot identify a caller, subject, tenant, and request boundary, the requirement status is blocked and the service must deny protected actions.
- If a third-party callback signature cannot be verified, discard the callback and record a security event without running side effects.
- If data classification is unknown for a field crossing a boundary, treat it as sensitive until a lower classification is explicitly approved.
- If a mitigation depends on a gateway or service mesh that is not present in the repo, record it as an external dependency and require service-local defense until the dependency is proven.

## Testable Requirements
- Given every affected endpoint or worker input, a test or review checklist names the boundary, data source, and expected verification control.
- Given a forged internal header such as `X-User-Id` or `X-Tenant-Id`, protected service logic rejects the request unless the header is bound to a verified upstream identity contract.
- Given an unauthenticated or unsigned async message, the worker refuses side effects and records a bounded security event.
- Given a degraded auth, policy, secret, cache, or third-party dependency, protected actions deny by default unless an approved safer degradation is documented.

## Repo-Local Anchors
- `api/openapi/service.yaml` currently declares `security: []` globally and a `bearerAuth` JWT scheme as a component, so security requirements must distinguish public baseline endpoints from future protected operations.
- `internal/infra/http/router.go` applies request framing, body limit, security headers, access logging, recovery, request correlation, and explicit CORS preflight rejection.
- `docs/configuration-source-policy.md` defines YAML as non-secret config and `APP__...` environment variables as the secret-value boundary.
- `SECURITY.md` defines private vulnerability reporting and disclosure expectations.

## Exa Source Links
- OWASP Threat Modeling Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Threat_Modeling_Cheat_Sheet.html
- OWASP ASVS V1 Architecture, Design and Threat Modeling: https://github.com/OWASP/ASVS/blob/master/4.0/en/0x10-V1-Architecture.md
- OWASP Microservices based Security Architecture Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Microservices_based_Security_Arch_Doc_Cheat_Sheet.html
- OWASP API Security Top 10 2023 overview: https://owasp.org/API-Security/editions/2023/en/0x11-t10/
