# API-Facing Security Semantics

## Behavior Change Thesis
When loaded for REST or OpenAPI-visible security behavior, this file makes the model choose contract-visible fail-closed status, CORS, idempotency, limit, and problem-response requirements instead of likely mistake: ad hoc status codes, `200` error bodies, permissive CORS, or retry ambiguity.

## When To Load
Load this when security requirements touch REST/OpenAPI behavior, status codes, CORS, method policy, request size or media limits, problem responses, rate-limit responses, retry/idempotency, management endpoints, or browser-facing headers.

## Decision Rubric
- Put client-visible security behavior in the API contract or task-local API design: security scheme, per-operation security, protected/public distinction, supported methods, media types, request limits, and problem responses.
- Use `401` for missing or invalid authentication and `403` for authenticated callers without permission, unless a resource-existence concealment policy explicitly chooses otherwise.
- Use `405` with `Allow` when method semantics require it; do not let disallowed methods fall through to another handler.
- Use `413` for request size limits, `415` for unsupported request media types, `406` only when response negotiation is intentionally constrained, `429` for rate or abuse limits, and `503` only when temporary unavailability is safe to disclose and retryable.
- For retry-unsafe create or mutate operations, define idempotency-key scope, caller/tenant binding, TTL, conflict response, replay behavior, and storage boundary.
- For browser-callable APIs, define CORS allowlist policy and security headers. If CORS is unsupported, reject preflight explicitly.

## Imitate
- "Protected operation without a token returns `401` and runs no handler side effect; valid token without object permission returns `403` before repository mutation." Copy the authn/authz split plus no-side-effect rule.
- "Credentialed browser clients may use only the named origin/method/header allowlist; unsupported preflight fails closed." Copy exact CORS allowlisting rather than a generic browser note.
- "Retry-unsafe creation requires an idempotency key scoped to caller and tenant; same key/same payload replays, same key/different payload conflicts." Copy the retry ambiguity removal.

## Reject
- "Return `200` with an error body." This hides security denial from clients, caches, metrics, and tests.
- "Use `404` for all authz failures." Concealment is a policy choice, not a blanket substitute for authorization semantics.
- "Allow `*` CORS for convenience." Wildcard origins are unsafe for credentialed or sensitive browser-callable APIs.
- "Clients can retry if they want." Non-idempotent retries need a contract, not optimism.

## Agent Traps
- Do not let this reference become full API modeling. Use it only for security-visible semantics; hand off resource modeling to API-contract work when needed.
- Do not duplicate identity rules here. Load the auth reference when the hard question is who may access what.
- Do not expose service names, stack traces, SQL errors, panic values, secrets, or internal hosts through problem details.

## Validation Shape
- Contract or handler tests cover in-scope `401`, `403`, `405`, `413`, `415`, `429`, and `503` behavior and assert no side effects on denials.
- Problem-response tests assert the repo-approved format and sanitized detail.
- CORS tests assert unsupported preflight fails closed and supported origins, methods, and headers are exactly allowlisted.
- Idempotency tests assert replay, conflict, caller/tenant scoping, and TTL behavior where retry-unsafe mutation exists.

## Repo-Local Anchors
- `api/openapi/service.yaml` is the REST contract source of truth and currently defines shared problem responses and a `bearerAuth` component.
- `internal/infra/http/router.go` explicitly handles `NotFound`, `MethodNotAllowed`, `OPTIONS`, CORS preflight rejection, and request body limits.
- `internal/infra/http/router_test.go` already includes fail-closed CORS preflight and body-limit assertions that can guide future tests.
