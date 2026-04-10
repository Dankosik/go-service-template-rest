# Compatibility And Versioning

## When To Read This
Read this when a contract change affects existing clients, status codes, error shapes, pagination defaults, enum values, field nullability, URI versioning, deprecation, sunset, coexistence between old and new endpoints, or rollout across mixed versions. Keep the output at API evolution semantics; do not design deployment pipelines or database migrations here.

## Compact Principles
- Treat APIs as contracts across wire shape, client source expectations, and semantics. A syntactically additive change can still break clients.
- Classify each change as `additive`, `behavior-change`, or `breaking`, and call out the client assumption that makes it so.
- Preserve status-code, error-type, retry, idempotency, precondition, async, pagination, and consistency semantics inside a major version unless the compatibility plan explicitly accepts the risk.
- Major API version belongs in the URI prefix by this skill's default. Do not expose minor or patch versions in REST paths unless the API already has a different established policy.
- Distinguish the OpenAPI Specification version from the API product version. Updating `openapi: 3.1.1` is not the same as releasing `/v2`.
- Adding response fields is safer when clients are told to ignore unknown fields. It is still risky for strict response decoders and generated SDKs.
- Adding enum response values is a compatibility decision. Either document that clients must tolerate unknown values or avoid expanding a supposedly closed enum in place.
- Tightening validation, making optional fields required, changing default values, changing null-vs-omitted behavior, or changing timestamp precision can be breaking.
- Adding pagination to an unpaginated collection can be breaking because old clients may stop seeing full result sets.
- Deprecation is not shutdown. Use deprecation signaling and documentation for "do not use"; use sunset signaling when a resource or API is expected to become unavailable.
- When old and new endpoints coexist, define which is authoritative for validation, state transitions, idempotency key scope, ETag space, error mapping, and consistency.
- Compatibility plans should include client migration notes, coexistence duration, deprecation/sunset headers when relevant, and validation evidence.

## Good Examples
```http
HTTP/1.1 200 OK
Deprecation: @1775001600
Link: </docs/migrate-orders-v2>; rel="deprecation"; type="text/html"
Sunset: Tue, 30 Jun 2026 23:59:59 GMT
Link: </docs/orders-v1-sunset>; rel="sunset"; type="text/html"
```

Contract note: `Deprecation` communicates lifecycle discouragement; `Sunset` communicates expected unavailability. They can appear together only when both meanings are true.

```text
Change: add optional response field `risk_level` to GET /v1/orders/{id}`.
Compatibility class: additive with semantic risk.
Client impact: clients with strict JSON decoders may reject the field; clients treating unknown enum-like strings as fatal may need guidance.
Decision: add in v1 only if API policy already requires ignoring unknown response fields; otherwise stage behind a documented preview or v2.
```

## Bad Examples
```text
Change: GET /v1/orders now returns the first 50 orders by default and a next_cursor.
Compatibility class: additive because the response only added pagination fields.
```

Why it is bad: the behavior changed from complete result set to partial result set, so old clients can silently miss data.

```text
Change: 409 conflicts now return 422 validation_error.
Compatibility class: internal cleanup.
```

Why it is bad: status and problem type are client-visible control flow.

## Edge Cases That Often Fool Agents
- "Additive" is not automatically safe when clients use closed-world schemas, exhaustive enum switches, strict generated models, or response hashing.
- Changing a field's format while keeping it a string is still a breaking semantic change if clients parse it.
- Changing a default page size, sort order, retry window, or consistency guarantee can break reasonable clients even if no schema changed.
- Removing or renaming a field is usually equivalent to remove-and-add, not a harmless label cleanup.
- Making a nullable field non-null, or an omitted field always present with `null`, can break clients that distinguish presence.
- Sunset dates are hints, not guarantees. Do not imply a client can safely wait until the exact timestamp.
- Deprecation and Sunset scope matters: a header on one resource applies to that resource unless the API documents a broader scope.
- A replacement endpoint should not silently diverge on idempotency, ETags, or error mapping during coexistence unless the migration plan makes that visible.

## Source Links Gathered Through Exa
- Google AIP-180, Backwards compatibility: https://google.aip.dev/180
- Google AIP-185, API Versioning: https://google.aip.dev/185
- RFC 9745, Deprecation HTTP Response Header Field: https://www.rfc-editor.org/rfc/rfc9745
- RFC 8594, Sunset HTTP Header Field: https://www.rfc-editor.org/rfc/rfc8594.html
- OpenAPI Specification v3.1.1, OAS versioning and API description semantics: https://spec.openapis.org/oas/v3.1.1.html
- JSON Schema 2020-12 Core: https://json-schema.org/draft/2020-12/json-schema-core
