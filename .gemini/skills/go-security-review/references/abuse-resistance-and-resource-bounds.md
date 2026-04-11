# Abuse Resistance And Resource Bounds Review

## Behavior Change Thesis
When loaded for symptom "a caller can scale work, cost, or retries," this file makes the model name the exhausted resource and choose pre-work bounds, throttles, budgets, or backpressure instead of likely mistake vague DoS or rate-limit advice.

## When To Load
Load this when changed Go code touches request size, header size, multipart parsing, pagination, search filters, batching, retries, outbound calls, goroutine fan-out, queueing, expensive transforms, file processing, password reset, OTP, webhooks, third-party paid APIs, or rate-limit semantics.

If password reset or OTP token generation, hashing, expiry, or replay is primary, load `token-and-credential-flow-review.md`. Use this file when the account-recovery risk is repeatability, enumeration cost, provider spend, or missing throttling.

## Decision Rubric
- Name the resource: CPU, memory, network, storage, goroutines, file descriptors, DB connections, queue slots, provider calls, or money.
- Require limits before parsing or expensive work for bodies, headers, multipart files, arrays, filters, page sizes, batches, recursion, and archives.
- Clamp client-provided limits to server-owned maxima and reject impossible values.
- Tie expensive work to caller context and explicit deadlines; avoid zero-timeout outbound clients on request paths.
- Require retry budgets, backoff, idempotency, and stop conditions before repeated side effects.
- Add per-subject and per-client throttles before reset, OTP, webhook, export, and paid-provider calls.
- Use bounded concurrency, bounded queues, and backpressure for fan-out.
- Keep fail-closed behavior on security dependencies; degraded mode must not bypass authorization or tenant isolation.

## Imitate
```text
[high] [go-security-review] internal/app/search.go:91
Issue: Axis: Abuse Resistance And Resource Bounds; `limit` is parsed from the query and passed directly into `ListRecent` without a maximum.
Impact: An authenticated caller can request very large result sets and pin database and response memory.
Suggested fix: Clamp to the API maximum at the HTTP boundary and add a regression test for `limit=max+1`.
Reference: pagination limit contract.
```

Copy this shape when the caller controls result size.

```text
[high] [go-security-review] internal/app/reset.go:44
Issue: Axis: Abuse Resistance And Resource Bounds; every password-reset request sends an SMS before any per-account or per-IP throttle.
Impact: An unauthenticated caller can drive third-party SMS cost and lockout noise.
Suggested fix: Add a fail-closed throttle before the provider call and return a generic response that does not disclose account existence.
Reference: account recovery abuse boundary.
```

Copy this shape when the risk is repeatable provider cost, not token cryptography.

```text
[medium] [go-security-review] internal/app/import.go:133
Issue: Axis: Abuse Resistance And Resource Bounds; the importer starts one goroutine per submitted item with no batch cap or worker limit.
Impact: A tenant can submit a large array and exhaust goroutines or downstream connections.
Suggested fix: Enforce a maximum item count and process through a bounded worker pool tied to the request context.
Reference: import fan-out bound.
```

Copy this shape when concurrency is attacker-scaled.

## Reject
```text
Issue: This could DoS the service.
```

Reject because it does not identify the exhausted resource or the control that fits the local path.

```text
Suggested fix: Add rate limiting.
```

Reject when the safer local correction is a size cap, batch cap, timeout, worker bound, or provider-call throttle before work begins.

## Agent Traps
- Do not confuse a default page size with a maximum page size.
- Do not apply limits after `io.ReadAll`, multipart parsing, archive extraction, or remote content download.
- Do not suggest global rate limiting when the abuse path needs per-subject, per-tenant, or per-provider throttling.
- Do not let fail-open fallback bypass auth, tenant isolation, or anti-abuse controls.
- Do not take ownership of detailed worker-pool correctness; hand off lifecycle and race concerns to `go-concurrency-review`.

## Validation Shape
- Add tests for oversized body, oversized multipart file, max+1 page size, max+1 batch size, invalid negative limits, and filter complexity limits when touched.
- Add fake provider tests that prove throttling occurs before third-party calls.
- Add context-cancellation tests for outbound and expensive processing paths.
- Add race or leak-oriented tests when concurrency bounds change.
- Run targeted package tests plus `make test-race` when goroutine or queue behavior changes.

## Repo-Local Anchors
- `internal/infra/http/server.go` exposes `ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, and `MaxHeaderBytes`.
- `internal/infra/http/middleware.go` rejects conflicting request framing and wraps bodies with `http.MaxBytesReader`.
- `internal/config/validate.go` validates timeout and pool-size ranges for HTTP and backing stores.
- `Makefile` includes `make test-race`, `make go-security`, and `make ci-local` for broader validation.
