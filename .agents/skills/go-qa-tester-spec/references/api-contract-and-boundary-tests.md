# API Contract And Boundary Tests

## Behavior Change Thesis
When loaded for symptom "the change is REST/OpenAPI or client-boundary visible", this file makes the model choose boundary-observable contract proof instead of likely mistake "prove handlers with internal unit tests or invent missing HTTP semantics."

## When To Load
Load this when planned behavior affects OpenAPI, generated bindings, HTTP method/status semantics, validation, limits, idempotency keys, auth/tenant/object boundaries, problem details, async acceptance, or runtime route/handler conformance.

## Decision Rubric
- Use `api/openapi/service.yaml`, generated API docs, `internal/api/README.md`, and runtime route tests as local contract sources before naming external standards.
- Contract proof must name status, headers, body schema/problem details, request decoding, and generated/runtime artifact expectations when those are affected.
- Boundary proof must vary the caller-controlled dimension: credential, tenant, object ID, idempotency key, cursor/filter, request size, unknown field, or async operation ID.
- Idempotent write proof should include first request, same key/same payload replay, same key/different payload conflict, and concurrent same-key attempts when concurrency is in scope.
- Async `202 Accepted` proof should include accepted response, operation identity, invalid-before-acceptance rejection, polling/terminal states if approved, and failure exposure if the contract owns it.
- OpenAPI drift proof should use repository targets, not ad hoc command substitutes.
- Missing method/status/error/idempotency/concealment policy is an API-spec blocker, not a QA decision.

## Imitate
| Contract Surface | Required Rows | Selected Proof | Observable To Copy |
| --- | --- | --- | --- |
| Request validation | valid body; missing field; invalid type; unknown field if strict; oversized body if limit changed | Contract | status, problem payload shape, field path if specified, no partial side effect |
| Auth and ownership | no credential; expired/invalid credential; wrong tenant/object owner; authorized actor | Contract or integration | 401/403/concealment status per approved policy, no leaked data, no write |
| Idempotent write | first request; same key/same payload; same key/different payload; concurrent same key | Contract plus integration if durable | stable response or operation, conflict where specified, exactly one durable side effect |
| OpenAPI drift | generated bindings compile; runtime route contract check; lint/validate | Repository contract commands | clean generated diff, `internal/api` tests pass, runtime contract check passes |

## Reject
- "Unit test the handler" as the only API contract proof when middleware, decoding, generated bindings, headers, or OpenAPI drift can change client-visible behavior.
- "Check 400 on bad input" without problem body, field path, strictness, size, and no-side-effect expectations when those are in scope.
- "Test authorization happy path" without wrong tenant/object or missing/invalid credential when the boundary changed.
- "OpenAPI looks updated" without repository drift/runtime/lint/validate commands.

## Agent Traps
- Do not import generic REST status choices from external examples. Use approved API decisions.
- Do not route storage-backed idempotency only to API contract tests; durable duplicate suppression may also need integration proof.
- Do not confuse generated-code compile success with runtime route conformance.
- Do not let security boundary proof leak into a broad security checklist. Keep it to caller-visible boundary behavior unless a separate security specialist owns more.

## Validation Shape
API strategy is ready when each boundary row names the public observable and the repository-supported contract or runtime check that would catch drift.
