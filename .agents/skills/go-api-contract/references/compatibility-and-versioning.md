# Compatibility And Versioning

## Load When

Load this when a change touches an already-published operation: a status, error
`code`, enum value, default, nullability, pagination behavior, consistency
guarantee, or a deprecation and removal plan.

## Decide

- Classify each change as `additive`, `behavior-change`, or `breaking`, and name
  the client assumption that makes it so. "Additive" is a claim about client
  tolerance, not about the diff: contracts here declare `additionalProperties:
  false`, and generated SDKs decode strictly.
- Treat as `breaking` regardless of what the differ reports: introducing
  pagination on a collection that previously returned everything; changing a
  default page size, sort order, or retry window; re-mapping a status or an error
  `code`; narrowing a consistency or freshness guarantee; changing timestamp
  precision or null-vs-omitted behavior.
- `response-non-success-status-removed` is `info` in the oasdiff checkset;
  only `response-success-status-removed` is `error`. Removing or re-mapping an
  error response is a hand-classified change with no automated guard.
- The OpenAPI document version is not the API version. Contracts here are
  `openapi: 3.0.3`, so optionality is `nullable: true`; rewriting it as 3.1's
  `type: [string, "null"]` is a document-format migration affecting generation,
  not a product version change.
- `Deprecation` (RFC 9745) is a Structured Field Date — `Deprecation:
  @1688169599`. `Sunset` (RFC 8594) is an HTTP-date — `Sunset: Tue, 30 Jun 2026
  23:59:59 GMT`. They are not interchangeable syntaxes, and RFC 9745 requires the
  Sunset instant to be no earlier than the Deprecation instant.
- While an old and a new surface coexist, name which one is authoritative for
  validation, state transitions, idempotency key scope, and error mapping, and
  record the removal criteria with the change that introduced the coexistence.

## Reject

- Listing a check id in `api/openapi/breaking-changes-approvals.txt` to make the
  gate green: the file silences that check for the whole diff, so it must carry
  the accepted breaking change and its migration, not the desire to merge.
- Treating format, lint, or unit tests as compatibility proof: none runs the
  breaking check. Pull-request CI supplies the exact base spec with `git show`.

## Prove

`make openapi-breaking BASE_OPENAPI=<base spec>` directly, or the pull-request
CI breaking step. A pass proves no schema-detectable
regression against that base. For every change in the hand-classified list
above, the proof is a stated client assumption and a migration or version
decision, because the gate cannot see it.
