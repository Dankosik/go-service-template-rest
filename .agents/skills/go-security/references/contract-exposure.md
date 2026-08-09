# Contract Exposure

## When To Load

Load this when an operation is added, its `security:` changes, or a surface
becomes reachable by a caller class it was not serving before.

## Behavior Change Thesis

Without this file, a new operation is treated as public until someone asks for
authentication, and `security: []` reads as "no security decision yet". This
contract inverts that default: `api/openapi/service.yaml` declares
`security: [{bearerAuth: []}]` at the top level, so every operation is protected
unless it opts out — and `security: []` *is* the opt-out. An unreviewed one
ships an unauthenticated endpoint that the diff shows as two characters.

## Decision Rubric

- The global requirement is the default. The decision to record for
  `security: []` is why this operation is reachable without a credential, not
  whether authentication was implemented yet.
- Every operation carries `x-security-decision` with an `exposure` of `public`,
  `protected`, or `blocked` plus a rationale.
  `TestOpenAPIRuntimeContractOperationsDeclareSecurityDecisions` iterates the
  spec and fails the build when it is missing, so the annotation is the durable
  record; a rationale that lives only in a task artifact is not.
- The two shipped opt-outs are `/health/live` and `/health/ready`, justified as
  platform probes exposing only generic state. An opt-out returning anything
  derived from stored data is a different class and carries its own decision.
- `go-api-contract` owns what a protected operation must then declare — its
  `401` and `403` responses and the `internal/problem` codes they answer from.
  This file owns whether the operation is protected at all.
- The authentication surface is a removable template profile. The
  `# profile:authn-oidc-jwt:start` and `:end` markers are stripped by
  `scripts/init-module.sh` and gated by `scripts/ci/template-init-check.sh`, so
  an edit inside them has to leave both the kept and the stripped tree valid.
- Browser reachability is a decision, not a default. This service answers JSON
  behind an edge: `SecurityHeaders` sets `nosniff` and nothing else, and the
  router answers a CORS preflight with `cors preflight is not enabled` rather
  than a permissive header. Admitting browser callers means deciding CORS, and
  deciding CSRF as well once credentials ride on cookies.
- For a cookie-authenticated route, `net/http.CrossOriginProtection` (Go 1.25+)
  is the standard-library control, and what needs review is what it allows by
  design: safe methods always, and any request carrying neither
  `Sec-Fetch-Site` nor `Origin`.

## Reject

- Reporting a literal `Access-Control-Allow-Origin: *` alongside credentials as
  cross-origin data theft: browsers refuse that combination, so nothing is read.
  The defect worth a finding is a handler that reflects the caller's `Origin`,
  which works.
- Treating a path prefix as the boundary: `/admin` and `/internal` are strings.
  Reachability is what the contract declares and the router serves.

## Validation Shape

`make openapi-check` runs the contract tests, including the security-decision
sweep above. The runtime side is `authn_router_test.go`
(`TestHTTPAuthnBoundary`) and `request_errors_test.go`, which already assert an
unwired resolver answers 401 and that rejection details stay sanitized — a new
exposure decision extends those rather than proving reachability by hand.
