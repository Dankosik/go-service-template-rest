# Abuse Resistance And Resource Bounds Review

## When To Load
Load this when changed Go code touches request size, header size, multipart parsing, pagination, search filters, batching, retries, outbound calls, goroutine fan-out, queueing, expensive transforms, file processing, password reset, OTP, webhooks, third-party paid APIs, or rate-limit semantics.

## Attacker Preconditions
- The attacker can repeat a request, increase a size/count/depth parameter, trigger fan-out, submit large payloads, cause retries, or invoke a paid or expensive dependency.
- The code path consumes CPU, memory, network bandwidth, storage, goroutines, file descriptors, database connections, queue slots, or third-party billable operations.
- Existing limits are missing, too high, bypassable through batching, or applied after expensive work has begun.

## Review Signals
- `io.ReadAll`, multipart parsing, JSON decoding, archive extraction, or template rendering happens before size limits.
- Pagination defaults are present but maximum page size is missing.
- Filters, batch arrays, recursion depth, or GraphQL-like operations are unbounded.
- Retries lack a budget, backoff, idempotency, or stop condition.
- Outbound calls lack context propagation or client timeout.
- Password reset, OTP, email/SMS, webhook, or export endpoints lack per-subject and per-client throttles.
- Concurrency or queue limits are removed from expensive paths.
- Fail-open fallback weakens auth, tenant isolation, or anti-abuse controls when a dependency fails.

## Bad Finding Examples
- "Add rate limiting."
- "This could DoS the service."
- "Bound this."

These are too vague unless they name the exhausted resource and a correction that fits the local path.

## Good Finding Examples
- "[high] [go-security-review] internal/app/search.go:91 Axis: Abuse Resistance And Resource Bounds; `limit` is parsed from the query and passed directly into `ListRecent` without a maximum. An authenticated caller can request very large result sets and pin database and response memory. Clamp to the API maximum at the HTTP boundary and add a regression test for `limit=max+1`."
- "[high] [go-security-review] internal/app/reset.go:44 Axis: Abuse Resistance And Resource Bounds; every password-reset request sends an SMS before any per-account or per-IP throttle. An unauthenticated caller can drive third-party SMS cost and lockout noise. Add a fail-closed throttle before the provider call and return a generic response that does not disclose account existence."
- "[medium] [go-security-review] internal/app/import.go:133 Axis: Abuse Resistance And Resource Bounds; the importer starts one goroutine per submitted item with no batch cap or worker limit. A tenant can submit a large array and exhaust goroutines or downstream connections. Enforce a maximum item count and process through a bounded worker pool tied to the request context."

## Smallest Safe Correction
- Enforce max request body, header, file, array, page-size, batch, and recursion limits before parsing or expensive work.
- Clamp client-provided limits to server-owned maxima and reject impossible values.
- Tie work to `context.Context` deadlines and use explicit `http.Client` or database timeouts.
- Add per-subject and per-client throttles around reset, OTP, webhook, export, and third-party paid operations.
- Use bounded concurrency, bounded queues, and backpressure for fan-out.
- Keep fail-closed behavior on security dependencies; degraded mode must not bypass authorization or tenant isolation.

## Validation Ideas
- Add tests for oversize body, oversize multipart file, max+1 page size, max+1 batch size, and invalid negative limits.
- Add fake provider tests that prove throttling occurs before third-party calls.
- Add context-cancellation tests for outbound and expensive processing paths.
- Add race or leak-oriented tests when concurrency bounds change.
- Run targeted package tests plus `make test-race` when goroutine or queue behavior changes.

## Repo-Local Anchors
- `internal/infra/http/server.go` exposes `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, and `MaxHeaderBytes`.
- `internal/infra/http/middleware.go` rejects conflicting request framing and wraps bodies with `http.MaxBytesReader`.
- `internal/config/validate.go` validates timeout and pool-size ranges for HTTP and backing stores.
- `Makefile` includes `make test-race`, `make go-security`, and `make ci-local` for broader validation.

## Exa Source Links
- OWASP API4:2023 Unrestricted Resource Consumption: https://owasp.org/API-Security/editions/2023/en/0xa4-unrestricted-resource-consumption/
- OWASP Input Validation Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html
- OWASP File Upload Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html
- Go `net/http` package docs: https://pkg.go.dev/net/http
