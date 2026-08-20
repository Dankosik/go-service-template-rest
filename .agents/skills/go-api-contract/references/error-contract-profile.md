# Error Contract Profile

## Load When

Load this when an operation gains, moves, or removes an error response, or when
a `Problem` body needs a field it does not already have.

## Decide

- `internal/problem` is the single source of `Code`, `Title`, `TypeURI`, and
  `Status`. Answer a new failure with an existing `problem.Code`; add a catalog
  entry only when no published status fits.
- `code` is the stable machine contract. `type` is an RFC section URI shared by
  every service built from this template, so it identifies the HTTP semantic,
  not this API's problem class. `title` and `detail` are for humans, and `detail`
  must be the service's own wording — never a wrapped `err.Error()`.
- The published `Problem` members are `code`, `type`, `title`, `status`,
  `detail`, `instance`, `request_id`. Contracts declare `additionalProperties:
  false`, so a field-level error list, a `current_state` hint, or a retry hint is
  a schema change with a compatibility class, not an extension you add in the
  handler.
- A feature package classifies its own errors through `problem.Mapper` and
  `problem.Classify`; depguard forbids it from importing the transport adapter,
  so the mapper is the seam, not a switch inside each operation.
- Declare every status a client can receive, not only the ones the handler
  writes. The middleware chain in `internal/infra/http` answers before a request
  reaches an operation — body limit, shedding, timeout, panic, and rate limit
  once a `RateLimiter` is wired — so those statuses are client-visible whether or
  not the operation lists them. `403`, `409`, and `422` are the catalog entries
  no runtime path produces.
- Every operation carries `x-security-decision` with `exposure` of `public`,
  `protected`, or `blocked` plus a rationale. A `protected` operation must also
  declare `401` and `403` `application/problem+json` responses.
- Pair a retryable status with a `Retry-After` header the contract marks
  required, as the reference contract does on `429` and `503`.
  `problem.Mapped.RetryAfter` carries the value; without a hint a client library
  either gives up or retries immediately.

## Reject

- A hand-built problem body in a handler: it bypasses `problem.ForCode`, so
  `status`, `title`, and `type` can disagree with the catalog every other
  operation answers from — the exact drift that once advertised a `409` slug
  conflict as a server fault.
- Concealing a resource with `404` on one operation and `403` on another: split
  answers let a caller distinguish the two by probing.

## Prove

`make openapi-check` runs `TestOpenAPIRuntimeContract*`.
`...OperationsDeclareSecurityDecisions` iterates the spec automatically;
`...ResponsesMatchSpec` does not — its case table is hand-maintained, so a new
operation ships with no response-contract coverage and nothing fails to say so.
Add a case per reachable status for every operation you add.
