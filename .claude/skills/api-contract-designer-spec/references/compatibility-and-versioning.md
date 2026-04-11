# Compatibility And Versioning

## When To Load
Load this when a contract change affects existing clients, status codes, error shapes, pagination defaults, enum values, field nullability, URI versioning, deprecation, sunset, or coexistence between old and new endpoints.

## Decision Rubric
- Classify each change as `additive`, `behavior-change`, or `breaking`, and name the client assumption that makes it so.
- Treat status codes, problem types, retry/idempotency/precondition behavior, async behavior, pagination, sorting, and consistency as compatibility surface, not cleanup.
- Keep the major API version in the URI prefix by this skill's default. Do not put minor or patch versions in REST paths unless the API already does.
- Distinguish OpenAPI document version from API product version. `openapi: 3.1.1` is not `/v2`.
- Adding response fields is safer only when clients are expected to ignore unknown fields. Strict decoders and generated SDKs can still break.
- Adding enum values is a compatibility decision. Document unknown-value tolerance or avoid expanding a closed enum in place.
- Tightening validation, making optional fields required, changing defaults, changing null-vs-omitted behavior, or changing timestamp precision can be breaking.
- Adding pagination to an unpaginated collection can be breaking because old clients may silently miss data.
- Deprecation means "do not use"; Sunset means "may become unavailable." Use both only when both meanings are true.
- When old and new endpoints coexist, define which is authoritative for validation, state transitions, idempotency key scope, `ETag` space, error mapping, and consistency.

## Imitate
```http
HTTP/1.1 200 OK
Deprecation: @1775001600
Link: </docs/migrate-orders-v2>; rel="deprecation"; type="text/html"
Sunset: Tue, 30 Jun 2026 23:59:59 GMT
Link: </docs/orders-v1-sunset>; rel="sunset"; type="text/html"
```

Good: lifecycle discouragement and expected unavailability are separate signals with separate documentation links.

```text
Change: add optional response field `risk_level` to `GET /v1/orders/{id}`.
Compatibility class: additive with semantic risk.
Client impact: strict JSON decoders may reject the field; exhaustive enum-like handling may need guidance.
Decision: add in v1 only if the API policy already requires ignoring unknown response fields; otherwise stage behind a preview or v2.
```

Good: "additive" is not treated as automatically safe.

## Reject
```text
Change: GET /v1/orders now returns the first 50 orders by default and a next_cursor.
Compatibility class: additive because the response only added pagination fields.
```

Bad: behavior changed from complete result set to partial result set.

```text
Change: 409 conflicts now return 422 validation_error.
Compatibility class: internal cleanup.
```

Bad: status and problem type are client-visible control flow.

## Agent Traps
- Changing a field format while keeping it a string is still breaking if clients parse it.
- Changing default page size, sort order, retry window, idempotency TTL, or freshness guarantee can break clients without a schema diff.
- Removing or renaming a field is usually remove-and-add, not harmless cleanup.
- Making a nullable field non-null, or always returning an omitted field as `null`, can break presence-sensitive clients.
- Sunset dates are hints, not guarantees. Do not imply clients can safely wait until the exact timestamp.
- A replacement endpoint should not silently diverge on idempotency, `ETag`, or error mapping during coexistence.
