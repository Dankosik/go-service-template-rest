# API Contract And Boundary Tests

## When To Load
Load this when a planned change affects REST/OpenAPI contract behavior, generated API bindings, HTTP status semantics, validation, limits, idempotency, auth/tenant boundaries, async acceptance, problem details, or runtime route/handler conformance.

## Source Grounding
- Treat `api/openapi/service.yaml` and generated API docs as local contract sources when the change is API-visible.
- Use `internal/api/README.md`, `docs/build-test-and-development-commands.md`, `Makefile`, and CI workflows for repository commands and contract checks.
- Use OpenAPI and OWASP sources to calibrate boundary proof, not to invent product-specific status or error policy.

## Selected/Rejected Level Examples
| API obligation | Selected level | Rejected level | Why |
| --- | --- | --- | --- |
| OpenAPI shape, generated binding drift, or runtime contract route match | Contract | E2E-only | Contract checks catch spec/code/runtime drift directly and are cheaper than full runtime smoke. |
| Handler validation result visible to clients | Contract or handler-boundary integration | Pure use-case unit | The proof must include decoding, status, response body, and relevant headers. |
| Object ownership or tenant mismatch | Contract or integration with two actors/scopes | Authorized happy-path contract only | The security obligation is the denied cross-scope request. |
| Idempotency key behavior on public write endpoint | Contract plus integration when durable storage owns dedup | Unit-only idempotency helper | Public semantics and durable duplicate suppression must both be proven when they are part of the contract. |
| Async `202 Accepted` operation resource behavior | Contract plus process/integration proof | Immediate-success unit | The proof must cover accepted response, operation identity, polling/terminal states if specified, and failure exposure. |
| Parser or decoder robustness for request input | Contract rows, plus fuzz if parser has cheap deterministic invariant | Manual malformed examples only | Contract rows prove public response; fuzz broadens parser input space when the parser itself is risky. |

## Scenario Matrix Examples
| Contract surface | Required rows | Selected proof | Pass/fail observable |
| --- | --- | --- | --- |
| Request validation | valid body, missing field, invalid type, unknown field if strict, oversized body if limits changed | Contract | status, problem/error payload shape, field path if specified, no partial side effect |
| Auth and ownership | no credential, expired/invalid credential, wrong tenant/object owner, authorized actor | Contract or integration | 401/403/concealment status per approved policy, no leaked resource data, no write |
| Idempotent write | first request, replay same key/same payload, same key/different payload, concurrent same key | Contract plus integration if persistent | stable response/operation, conflict where specified, exactly one durable side effect |
| Pagination/filtering | empty result, boundary page size, invalid cursor/filter, deterministic order | Contract plus data integration if DB ordering matters | response schema, next cursor/link, stable ordering, validation error |
| Async operation | accepted, invalid request rejected before acceptance, retryable worker failure, terminal failure, poll unknown ID | Contract plus process proof | `202`/operation resource where specified, terminal state, error shape, retention behavior |
| OpenAPI drift | generated bindings compile, runtime route contract check, lint/validate | Repository contract commands | clean generated diff, `internal/api` tests pass, runtime contract test pass |

## Pass/Fail Observables
- Contract rows name status, headers, body schema, and generated/runtime artifact expectations when relevant.
- Boundary tests include negative and misuse paths when caller identity, tenant, object ID, size, or idempotency key is caller-controlled.
- API strategy does not decide missing product semantics; unresolved method/status/error/idempotency policy is an upstream API spec blocker.
- OpenAPI drift and generated artifact checks use repository targets rather than ad hoc command substitutes.
- Security-sensitive contract proof fails closed and checks absence of leaked sensitive or cross-tenant data.

## Exa Source Links
- [OpenAPI Specification v3.0.4](https://spec.openapis.org/oas/v3.0.4.html)
- [OWASP WSTG API Broken Object Level Authorization](https://owasp.org/www-project-web-security-testing-guide/latest/4-Web_Application_Security_Testing/12-API_Testing/02-API_Broken_Object_Level_Authorization)
- [testing package](https://pkg.go.dev/testing)
- [Go Fuzzing](https://go.dev/doc/fuzz/)
- [Go security best practices](https://go.dev/doc/security/best-practices)

