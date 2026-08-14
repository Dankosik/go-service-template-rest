# PostgreSQL-Backed HTTP Idempotency Research

status: ready for Specification
Valid as of: 2026-08-11
Repository baseline: clean `main` at `40e6d212799ae8677b675339929c559246536181`
Boundary: Research only. This note does not select a final architecture, define Specification, create tasks or migrations, or authorize code.
Independent review: PASS on 2026-08-11 after focused repair of evidence locators and authority labels.

## Accepted outcome and stop condition

Research an optional template-init-selected capability, provisionally
`HTTP_IDEMPOTENCY=postgres`, through which an HTTP business operation can declare its
idempotency scope and use a ready PostgreSQL-backed mechanism without recreating
duplicate detection, concurrency control, replay, retention, cleanup, or recovery.

Stop when the repository baseline, external contract evidence, candidate families,
atomicity boundary, reusable responsibilities, endpoint-owned semantics, evidence
limits, and inputs needed by Specification are explicit enough that a fresh
Specification session does not have to repeat discovery. Do not cross the Research
macro-phase boundary.

## Research disposition

The capability is feasible, but the research does not justify selecting one complete
architecture.

The strongest established boundary is narrower than “HTTP middleware provides
idempotency”:

- A transport layer can parse and expose the key, name the generated operation, map
  outcomes to Problem responses, and replay a transport representation after the
  business result is known.
- It cannot atomically bind a handler-owned PostgreSQL mutation to an idempotency
  record after the handler returns. A process failure between those commits leaves a
  completed business effect with no durable replay evidence.
- Atomic duplicate suppression for business effects in the same PostgreSQL database
  is reusable when the idempotency persistence step joins the exact caller-owned
  `pgx.Tx` that owns the feature mutation. The current inbox and outbox already prove
  that ownership shape.
- Effects outside that transaction—outbound HTTP, broker publication without the
  outbox, files, or another datastore—remain endpoint-owned distributed recovery.

This eliminates middleware-only atomicity and the current status quo as sufficient
answers. It does not select between a one-transaction record, a separately visible
in-progress state machine with fencing/recovery, or a composition of HTTP and
transaction-bound parts. Specification must first fix the observable semantics.

There is also no finalized interoperable keyed-idempotency standard whose defaults
the template can inherit. The current IETF `Idempotency-Key` document is an expired
Internet-Draft, and deployed providers disagree on syntax, scope, conflict status,
concurrent behavior, replay, errors, and retention. Those are local contract decisions,
not implementation details.

Evidence language in this note is deliberate: **established** means supported by the
current repository or a primary/current external owner; **inference** means a stated
consequence of those facts; **conflict** means credible owners specify incompatible
behavior; **absence** means the bounded search found no current owner or implementation;
and **unknown** names endpoint or deployment evidence still required. A provider's
public contract proves behavior at its boundary, not its hidden implementation.

## Open-item map

| Item | Question | Method | Established result or remaining decision | Downstream owner |
| --- | --- | --- | --- | --- |
| R1 | What already exists in the repository? | Current-state baseline | OpenAPI, generated strict handlers, Problem rendering, request limits, bounded outbound retries, caller-owned PostgreSQL transactions, inbox/outbox precedents, migrations, telemetry, bootstrap supervision, and template-init profiles exist. No inbound HTTP idempotency exists. | Constrains Specification and later design. |
| R2 | Is there a standard contract to adopt? | Current external contract | No finalized keyed-idempotency RFC or common provider contract. The expired draft and OASIS committee specification are guidance only. | Specification must own wire behavior. |
| R3 | What is the durable identity? | Provider comparison plus input closure | Production practice consistently combines caller key with authenticated client/tenant and stable operation scope. Resource, API-version, environment, region, and expiry boundaries remain endpoint/deployment decisions. | Specification. |
| R4 | What does a fingerprint prove? | External contract and semantic counterexample | It detects reuse of one caller intent for different semantic input; it does not replace the caller key. No universal canonicalization exists. | Endpoint semantics in Specification. |
| R5 | What happens for duplicates and concurrency? | Cross-source conflict plus PostgreSQL semantics | Unique constraints arbitrate ownership but do not define HTTP wait, reject, replay, takeover, or timeout policy. Provider behavior diverges. | Specification, then data/reliability design. |
| R6 | Can lost response and ambiguous commit be recovered? | Repository failure path plus PostgreSQL semantics | Yes for same-transaction PostgreSQL effects when the scoped record/result commits with the feature mutation and retries query the authoritative writer. A commit error is not proof of rollback. | Specification plus later transaction design/proof. |
| R7 | What is replayed? | Provider contract comparison | No common model: providers freeze the first response, return current state, or resume/reexecute. Stored bytes, stable headers, failure classes, correlation data, and oversize behavior are open. | Endpoint semantics in Specification. |
| R8 | How long is evidence retained? | Conflict/freshness comparison | There is no universal TTL. Published windows range from one hour to 30 days or endpoint-specific periods. Cleanup, expiry behavior, privacy, and capacity remain local obligations. | Specification and later data/operations design. |
| R9 | Is there a mature Go dependency to adopt? | Solution discovery and source inspection | No mature exact-fit Go library was found that provides PostgreSQL durability, semantic fingerprint conflicts, bounded HTTP replay, retention, and completion inside the caller's `pgx.Tx`. Mature adjacent workflow/job systems solve different problems. | Candidate evidence for later design; refresh before dependency selection. |
| R10 | Can atomic business-effect idempotency be reusable without transport owning the transaction? | Sequencing falsification against current ownership | Yes, for effects in the caller's PostgreSQL transaction, through a transaction-bound primitive invoked by the operation/repository adapter. No, for middleware acting only before/after an independently committed handler. | Hard ownership constraint for Specification and design. |

## Current repository baseline

### HTTP contract and generated handler seam

- `api/openapi/service.yaml:1` is the canonical OpenAPI 3.0.3 authority; generated
  bindings are derived. Endpoint opt-in, the key parameter, and concrete response
  alternatives therefore have an operation-level contract owner.
- `internal/infra/http/handlers.go:19-25` exposes `Handlers.API` as the service
  composition seam.
- `internal/openapi/openapi.gen.go:420-480` defines a `StrictMiddlewareFunc` that
  receives the typed request/response and `operationID`. The generated wrapper
  serializes the response after the strict handler returns.
- `internal/infra/http/router.go:35-60` constructs the strict server but currently
  passes no strict middleware. `docs/repo-architecture.md:275-283` says shared HTTP
  policy changes the hardened chain, while service-only policy should wrap the API
  handler rather than creating an unowned middleware registry.
- Consequence: generated operation metadata can carry HTTP policy, but it is not a
  transaction owner and cannot infer tenant or semantic fingerprint fields.

### Problems, limits, timeouts, and retries

- `internal/infra/http/problem.go:105-129` renders the repository's generated
  RFC 9457-style Problem shape and supplies the current request ID.
- `internal/problem/problem.go:24-61` has generic 409, 422, 429, 503, and 504
  statuses, but there are no idempotency-specific stable Problem codes. Same-key
  mismatch, in-progress, store unavailable, and ambiguous outcome still need distinct
  contract treatment.
- `internal/infra/http/domain_errors.go:14-86` is the existing domain/failure-to-
  Problem mapping seam, including `Retry-After` for mapped retryable failures.
- `internal/infra/http/harden.go:44-55` orders correlation, tracing, security,
  access logging, body limit, timeout, in-flight admission, rate limiting, recovery,
  and the API handler.
- `internal/config/http_config.go:55-98` defaults to a 16 KiB header limit, 1 MiB
  request body limit, 8 second request timeout, 10 second write timeout, 256 in-flight
  requests, and 4096 connections. The request-body ceiling is not a stored-result
  ceiling; there is no bounded server response-capture contract today.
- `internal/infra/http/middleware_timeout.go:10-55` cancels the request context and
  can emit a fallback 504 only if the handler has not committed a response. It cannot
  forcibly stop work that ignores cancellation.
- `internal/infra/http/middleware_inflight.go:19-69` rejects rather than queues when
  the global in-flight bound is full, using 503 and `Retry-After: 1`.
- `internal/infra/http/middleware_access_log.go:13-74` records bounded route, status,
  duration, and Problem code; it does not log arbitrary headers or bodies. Raw keys,
  fingerprints, tenants, and replay bodies do not belong in routine logs or labels.
- `internal/infra/httpclient/retry.go:21-247` provides bounded outbound retries with
  jitter and remaining-budget checks. Unsafe methods require a nonblank
  `Idempotency-Key` and rewindable body; only 429/502/503/504 and transport failures
  qualify. This is evidence that callers already treat the header as a repeat-safety
  assertion, not evidence of an inbound guarantee.

### PostgreSQL transaction ownership and recovery precedents

- `internal/infra/postgres/transaction.go:13-108` owns begin/rollback/commit around a
  caller callback. It exposes the exact `pgx.Tx`, classifies 40001/deadlock as
  retryable, returns `ErrCommitUnknown` when commit may have landed, and intentionally
  does not retry because the database wrapper cannot know whether all caller effects
  are safe to repeat.
- `internal/config/postgres_config.go:43-152` defaults to 25 connections, 1 second
  acquire timeout, and 8 second statement and idle-in-transaction limits. A duplicate
  that waits on a unique conflict can consume a request slot and database connection
  until nearly the request deadline; leaving that behavior implicit would not be a
  bounded concurrency contract.
- `internal/infra/postgresinbox/inbox.go:1-53` requires a caller-owned `pgx.Tx` and
  uses a composite primary-key claim with `INSERT ... ON CONFLICT DO NOTHING`.
  It neither starts nor commits the transaction. This is direct evidence that a
  reusable persistence primitive can join a feature mutation without exposing PGX to
  the feature API.
- `docs/postgres-idempotent-inbox.md` limits the guarantee to the claim and effects in
  the same PostgreSQL transaction. It deliberately has no TTL, cleanup, result,
  fingerprint, state machine, telemetry, ordering, or external-effect guarantee.
  `specs/inbox-idempotent-consumption/spec.md:46` explicitly excluded generic HTTP
  idempotency because the actor, key, and lifetime differ.
- `internal/infra/postgresoutbox/store_append.go:13-38` likewise appends through the
  caller's transaction. `internal/infra/postgresoutbox/store_receipt.go:15-94` keeps
  immutable versioned fingerprint evidence and reconciles unknown commits on the
  writer primary; absence on a replica never authorizes a repeat. This is useful
  precedent, not a reusable HTTP result store.
- `examples/reference-service/internal/article/article.go:62-90` and the PostgreSQL
  unit-of-work integration test show that a feature may own an `Atomically` port while
  the concrete adapter binds it to `Pool.InTx`. There is no repository-wide generic
  unit-of-work abstraction to extend.

### Migrations, bootstrap, telemetry, and template lifecycle

- Canonical migrations live under `migrations/`; SQLC input and generated ownership
  remain under `internal/infra/postgres`. `internal/infra/postgresmigrate/migrate.go`
  exposes forward migration through the separate `cmd/migrate` path. The service does
  not apply schema at startup.
- `cmd/service/internal/bootstrap/startup_dependencies.go:147-179` creates one pool,
  registers readiness, and closes the pool under a bounded lifecycle. Pool saturation
  maps to a retryable 503 with a one-second `Retry-After`.
- `internal/background/background.go:1-220` is a lifecycle supervisor, not a job
  framework. It owns cancellation, panic containment, readiness-visible failure, and
  bounded join; a cleanup task would still need its own schedule, cross-replica
  coordination, retry, batching, and readiness policy.
- PostgreSQL tracing omits SQL text and connection details, and infra packages use
  closed low-cardinality vocabularies. Idempotency outcomes can follow that pattern;
  caller keys, tenant values, fingerprints, and stored results cannot be labels.
- `scripts/init-module.sh` currently owns `DATABASE`, `OUTBOX`, `INBOX`, and other
  profile dependencies, records choices in `template.lock`, removes inactive pack
  surfaces, and regenerates the shared SQLC package after profile resolution.
  `HTTP_IDEMPOTENCY` does not exist. A later PostgreSQL pack would have to depend on
  `DATABASE=postgres` and join the existing off/on/repeatability matrix; this research
  creates none of that plumbing.

## Current standards and production practice

### Standards status

- [RFC 9110](https://www.rfc-editor.org/rfc/rfc9110.html) defines method
  idempotency by intended effect. It does not define a keyed protocol for POST/PATCH,
  and a client or proxy cannot infer that a non-idempotent request is safe to retry.
- The [IETF `Idempotency-Key` draft revision 07](https://datatracker.ietf.org/doc/draft-ietf-httpapi-idempotency-key-header/07/)
  is dated 2025-10-15, expired 2026-04-18, and archived. It is not an RFC, and the
  field is absent from the [IANA HTTP Field Name Registry](https://www.iana.org/assignments/http-fields/http-fields.xhtml)
  as checked on 2026-08-11.
- The draft models the value as an [RFC 8941 Structured Field string](https://www.rfc-editor.org/rfc/rfc8941.html),
  which is quoted on the wire; common providers document unquoted opaque values. A
  strict draft parser would therefore be a compatibility choice, not a neutral
  standards choice.
- [OASIS Repeatable Requests 1.0](https://docs.oasis-open.org/odata/repeatable-requests/v1.0/repeatable-requests-v1.0.html)
  is a 2020 Committee Specification, not an OASIS Standard. Its multi-header protocol
  and client timestamp differ materially from the deployed single-key convention.
- [OpenAPI 3.0.3](https://spec.openapis.org/oas/v3.0.3) can declare a header parameter
  and operation responses. Vendor extensions can record metadata, but the generator
  is not required to enforce custom extensions. Declaration and runtime enforcement
  need separate proof.
- [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457.html) supplies the base Problem
  response semantics: `type` is the primary machine identifier; `detail` is not a
  parse target; and the wire status and Problem status must agree. The repository's
  stable `code` is a local extension owned by `internal/problem/problem.go:1-22`, not
  an RFC field. Both extension values and details must avoid prior-request disclosure.

### Keys, authenticated scope, and fingerprints

Established across the expired draft and production contracts:

- The caller generates a high-entropy key for one intended operation and reuses it
  only for retries of that intent. Keys should not contain personal or business data.
- Lookup scope includes the authenticated client/tenant and a stable business
  operation identity, not the raw key alone. [Stripe v2](https://docs.stripe.com/api-v2-overview)
  scopes by API, account, and sandbox; [Adyen](https://docs.adyen.com/development-resources/api-idempotency)
  by company account; [PayPal](https://developer.paypal.com/api/rest/reference/idempotency/)
  by API call type; and the [AWS Builders' Library](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/)
  describes customer identity plus client request ID.
- Authentication and current authorization must still precede replay. A key must not
  become a bearer capability for another tenant's retained response.
- A fingerprint detects same-key reuse for materially different input. It does not
  infer intent and cannot replace the caller key: two byte-identical requests can be
  two intentionally distinct operations.

No universal fingerprint exists. Hashing raw JSON makes insignificant whitespace or
field order conflict; hashing too few decoded fields can conflate different path,
query, headers, defaults, tenant, or API-version semantics. The endpoint contract must
identify semantic inputs and a canonicalization version. Only a digest need normally
be retained; retaining the raw request increases privacy and capacity cost.

Published key caps also conflict: [Stripe v1](https://docs.stripe.com/api/idempotent_requests)
permits 255 characters, [Adyen](https://docs.adyen.com/development-resources/api-idempotency)
64, and [PayPal](https://developer.paypal.com/api/rest/reference/idempotency/) uses
38 bytes for its proprietary field. UUIDv4 provides 122 random bits under
[RFC 9562](https://www.rfc-editor.org/rfc/rfc9562.html) and fits those examples, but
neither its syntax nor one provider's cap is yet a repository requirement.

### Mismatch and concurrent in-progress requests

The [expired draft](https://datatracker.ietf.org/doc/html/draft-ietf-httpapi-idempotency-key-header)
proposes 422 for the same key with a different fingerprint and 409 for a concurrent
in-progress request. Production contracts disagree: [OASIS](https://docs.oasis-open.org/odata/repeatable-requests/v1.0/repeatable-requests-v1.0.html)
uses 400 for mismatch; providers return provider-specific idempotency errors;
[Adyen](https://docs.adyen.com/development-resources/api-idempotency) may use 409 or
422 with a retry signal; and [PayPal](https://developer.paypal.com/api/rest/reference/idempotency/)
may fail the simultaneous second request.

The stable cross-source requirement is semantic, not the status number:

- mismatch is deterministic and non-retryable for that key and must disclose none of
  the previous tenant's data;
- in-progress behavior is explicitly bounded and tells the client whether and when a
  retry can help;
- the first owner's rollback or death cannot leave two live owners or a permanent
  unclassified limbo.

[PostgreSQL unique-index checking](https://www.postgresql.org/docs/18/index-unique-checks.html)
and [`INSERT ... ON CONFLICT`](https://www.postgresql.org/docs/18/sql-insert.html)
can atomically arbitrate a composite identity. A conflicting insert can wait for the
other transaction to commit or abort. That native behavior is useful coordination,
but it does not choose HTTP latency, retry advice, stale-owner takeover, or a lease
fence. An explicit committed `in_progress` row makes state visible before the business
transaction, but also creates abandoned-state and recovery obligations that the
single-transaction claim does not have.

### Lost responses, commit ambiguity, and bounded retries

A client can lose the response after the server commits. A connection can also fail
during `COMMIT`, leaving the application unable to decide from the call result whether
the transaction landed. `ErrCommitUnknown` already preserves this distinction.

When the scoped idempotency evidence/result and local business writes commit in the
same PostgreSQL transaction, an authoritative writer lookup on retry can establish one
of two durable states: both committed, or neither committed. A missing row on an
asynchronous replica is not authority to execute again. The 2025 AWS analysis of
[transaction visibility in PostgreSQL clusters with read replicas](https://aws.amazon.com/blogs/database/understanding-transaction-visibility-in-postgresql-clusters-with-read-replicas/)
shows why replica visibility cannot be assumed to follow the client's primary commit.

The guarantee also depends on the target durability policy. PostgreSQL documents that
[`synchronous_commit=off`](https://www.postgresql.org/docs/18/wal-async-commit.html)
can acknowledge work later lost after a crash. Target topology, commit policy,
failover, multi-region writes, and read routing are deployment inputs that must reopen
the proof if they differ from a single authoritative synchronous writer.

[PostgreSQL serialization-failure guidance](https://www.postgresql.org/docs/18/mvcc-serialization-failure-handling.html)
requires retrying the whole transaction after 40001 and often 40P01. The repository
correctly does not hide that retry in `Pool.InTx`; any later bounded retry must repeat
the complete operation under the same stable idempotency identity and only when all
effects are covered.

### Response replay and failure recovery

There is no common result model:

- [Stripe v1](https://docs.stripe.com/api/idempotent_requests) freezes the first status
  and body after execution begins, including 500 responses; pre-execution validation
  and conflicting concurrent requests are not stored. Keys may be pruned after at
  least 24 hours.
- [Stripe v2](https://docs.stripe.com/api-v2-overview) uses a 30-day window and may
  return updated state or continue/reexecute failed work instead of freezing every
  first result.
- [PayPal](https://developer.paypal.com/api/rest/reference/idempotency/) returns the
  latest status and uses endpoint-specific support/windows.
- [Shopify's payment API account](https://shopify.engineering/building-resilient-graphql-apis-using-idempotency/)
  models idempotency as a first-class operation input, locks concurrent requests, and
  records recovery points around local and remote steps.
- [AWS's production guidance](https://aws.amazon.com/builders-library/making-retries-safe-with-idempotent-APIs/)
  favors an explicit caller request token and semantically equivalent responses; it
  emphasizes late requests and same-token/different-intent validation.

Specification must therefore say whether replay means frozen bytes, a semantically
equivalent typed result, current resource state, or resumable execution. It must also
name replayable status classes and an allowlist of stable headers. Hop-by-hop headers,
authentication material, cookies, `Date`, trace correlation, and the prior request ID
must not be retained/replayed by default. A cached 500 does not prove that no local or
remote effect occurred.

The repository has no current stored-result bound. Specification needs a maximum for
retained status/body/header bytes and deterministic behavior when a result exceeds it.
Executing an advertised-idempotent operation and then silently failing to preserve
enough result to resolve a lost response weakens the contract.

### Retention, cleanup, privacy, and capacity

Published retention is product-specific:
[AWS Powertools](https://docs.aws.amazon.com/powertools/typescript/latest/features/idempotency/)
defaults to one hour, [Stripe v1](https://docs.stripe.com/api/idempotent_requests) at
least 24 hours, [Adyen](https://docs.adyen.com/development-resources/api-idempotency)
at least seven days, [Stripe v2](https://docs.stripe.com/api-v2-overview) 30 days, and
[PayPal](https://developer.paypal.com/api/rest/requests/) uses endpoint-specific
periods that can reach 45 days. These are comparison points, not a template default.

Expiry behavior is equally important. Stripe v1 may treat a pruned key as new work;
OASIS can reject a late retry using the client's first-sent timestamp. Deleting all
evidence and silently treating an old retry as new can duplicate a previously committed
effect. A tombstone or stale-request rejection preserves more history but adds contract,
trust, and storage cost.

[PostgreSQL routine-vacuuming guidance](https://www.postgresql.org/docs/18/routine-vacuuming.html)
establishes that deleting expired rows creates dead tuples and I/O rather than
immediately returning space. A retention number without an indexed expiry path,
bounded batch cleanup, autovacuum/capacity evidence, and stuck-state treatment is not
an operational policy. The existing background supervisor is a possible lifecycle
host, not proof of one safe cleanup topology across replicas.

Privacy constraints are direct:

- keys must not carry email, account, or other personal/business data;
- lookup and replay must remain scoped and reauthorized;
- request bodies should not be retained when a versioned digest suffices;
- response storage must be minimized, bounded, expired, and covered by erasure and
  incident-access policy;
- digests can remain personal data when linkable to a person or tenant;
- raw keys, fingerprints, tenants, bodies, and replay bytes must not enter logs,
  traces, metric labels, or unbounded exemplars.

These constraints agree with provider warnings and [GDPR Article 5](https://eur-lex.europa.eu/legal-content/EN/TXT/?uri=CELEX%3A32016R0679)
data-minimization and storage-limitation duties. No authoritative source supplies a
universal safe result-size ceiling.

### Engineering articles, incidents, and talks

- Stripe's 2017 engineering article, [Designing robust and predictable APIs with
  idempotency](https://stripe.com/blog/idempotency), remains useful production
  rationale for ambiguous network outcomes and bounded exponential backoff with
  jitter. It is not a current normative contract.
- Shopify's 2019 engineering account above is decision-changing because it exposes
  recovery-point and external-provider constraints that simple middleware examples
  omit.
- The 2025 AWS PostgreSQL replica-visibility article above is recent operational
  evidence that writer/replica routing belongs in the correctness boundary.
- A 2025/2026 practitioner talk, [Avoiding deja vu: building resilient APIs with
  idempotency](https://www.conroyp.com/speaking/avoiding-deja-vu-building-resilient-apis-with-idempotency),
  corroborates the need to separate client contract, storage state, replay, and
  cleanup, but its speaker page is not used as authority for any invariant.
- A July 2026 InfoWorld account, [Why an idempotency key isn't an idempotency
  guarantee](https://www.infoworld.com/article/4191741/why-an-idempotency-key-isnt-an-idempotency-guarantee.html),
  describes duplicate charges when the local key boundary did not include the remote
  payment effect. It is an attributable operational anecdote, not a public postmortem
  with inspectable incident evidence.

No first-party, technically detailed public incident report was located that could
select a universal PostgreSQL state machine. That absence is recorded rather than
replaced by unattributed incident claims or vendor-blog statistics.

## Atomicity feasibility test

### Test A: outer transport middleware records after handler success

```text
middleware reserves or looks up key
  -> handler begins and commits feature transaction
  -> process or connection fails before middleware stores the result
  -> retry finds no completed replay record and may execute the effect again
```

Disposition: fails atomic business-effect idempotency. Capturing `ResponseWriter`
bytes or using generated strict middleware does not close the gap. Beginning the
feature transaction in that middleware would move business transaction ownership into
transport and break the current repository boundary.

### Test B: operation-owned transaction includes reusable idempotency persistence

```text
generated handler validates transport contract and calls operation
  -> concrete operation/repository adapter owns Pool.InTx
  -> transaction-bound primitive arbitrates scoped key + fingerprint
  -> feature writes and durable result/evidence use the same pgx.Tx
  -> one commit makes both visible, or rollback makes neither visible
```

Disposition: locally feasible for same-PostgreSQL effects. `postgresinbox.Claim`,
`postgresoutbox.Store.Append`, `Pool.InTx`, and the reference unit-of-work adapter prove
the necessary ownership pattern. They do not prove the new row/state/result contract,
endpoint semantics, cleanup, or target-topology behavior.

### Test C: separately committed visible in-progress state

```text
reserve/lease key in transaction 1
  -> execute feature mutation and completion in transaction 2
  -> crash, timeout, or lease expiry between steps
  -> recovery must decide whether to resume, take over, fence, or reject
```

Disposition: feasible as a different state-machine family, not selected. It can
support fail-fast concurrent responses and long/remote workflows, but it introduces
abandoned reservations, lease/fencing, two-transaction recovery, and possible overlap.
Those costs are unjustified unless the accepted endpoint semantics require visibility
before the business transaction or recovery across external steps.

### Feasibility conclusion

Atomic business-effect idempotency can be reusable without transport owning the
feature transaction, but only through a caller-owned transaction seam and only for
effects covered by that transaction. Transport-only reuse can standardize the HTTP
envelope, not the atomic guarantee.

## Candidate map

| Candidate family | What it can own | Fit to intended outcome | Counter-evidence / decision-flip condition | Disposition |
| --- | --- | --- | --- | --- |
| Generic `net/http` middleware | Header parsing, coarse validation, response capture, lookup/replay around a handler | Minimal endpoint code for non-atomic cached results | Cannot atomically cover an independently committed handler; lacks endpoint tenant/fingerprint semantics; body capture needs a bound. Reconsider only when operations have no side effects or use an externally atomic owner. | Eliminated as the complete atomic mechanism; remains a possible complement. |
| Generated strict-handler wrapper | Typed request/response, `operationID`, pre/post handler policy | Better semantic access than outer middleware | Currently unwired; still cannot own the feature transaction without violating ownership. Vendor metadata enforcement needs generated proof. | Possible HTTP-contract complement, not a complete atomic owner. |
| Caller-owned transaction primitive | Scoped claim/fingerprint/result through caller's `pgx.Tx` | Directly matches same-database atomicity and current inbox/outbox patterns | Cannot cover external effects; endpoint must supply semantic scope/result. Reconsider if target effects do not share PostgreSQL or feature adapters cannot join one transaction. | Viable mechanism family; not selected. |
| PostgreSQL unique constraint with one-transaction record | Native key arbitration; duplicate waits until winner commits/rolls back | Smallest native atomic path; no abandoned committed in-progress row | Wait can consume request/connection budget; no immediate in-progress response; retention/result semantics still needed. | Viable state family; not selected. |
| PostgreSQL committed state machine/lease | Visible in-progress/completed/expired states, takeover and recovery | Supports fail-fast conflict and multi-step recovery | Adds lease expiry, fencing, stuck-state cleanup, overlap risk, and a commit gap unless completion joins the feature transaction. | Viable only if Specification requires those semantics. |
| Existing `postgresinbox` | Permanent exact-transaction duplicate claim | Proves ownership and DB arbitration | Wrong actor/key/lifetime; no fingerprint/result/replay/TTL/cleanup/telemetry. | Reuse pattern, not implementation unchanged. |
| Existing outbox receipt | Versioned fingerprint and writer-primary commit reconciliation | Proves stable identity and ambiguous-commit handling | No HTTP result/replay; immutable operational retention has a different lifecycle. | Reuse pattern, not implementation unchanged. |
| Mature workflow/job systems | Durable steps, retries, recovery, PostgreSQL transaction hooks | Useful when the operation is already an async workflow/job | Introduce a workflow runtime/queue and do not define this HTTP fingerprint/Problem/replay contract. | Substitute for a different endpoint model, not default pack machinery. |
| Exact-fit young Go libraries | Middleware/store/state examples, sometimes transaction completion hooks | Useful implementation-source evidence | Exact candidates were recent, pre-v1 or sparsely adopted, and did not establish the full contract/proof surface. | Reference only; refresh before design. |
| Status quo | Endpoint writes bespoke checks or relies on domain uniqueness | Zero new template machinery | Does not provide declared scope, semantic conflicts, lost-response replay, retention, or common recovery; duplicates policy across endpoints. | Does not meet intended outcome. |

### Go implementation-source sweep

The 2026-08-11 survey found no mature exact-fit dependency that should replace
repository/native PostgreSQL mechanisms by default:

- [`faustbrian/go-idempotency`](https://github.com/faustbrian/go-idempotency) is the
  closest API-shape reference because its store has transaction-aware completion and
  its HTTP wrapper is not sufficient for atomic business effects. At the cutoff it was
  created on 2026-07-15, untagged/pre-v1, and had no visible independent adoption.
  It also brings policy and dependencies beyond the repository's current stack. It is
  implementation-source evidence, not maturity evidence.
- [`eben-vranken/idempo`](https://github.com/eben-vranken/idempo) has a v1 tag,
  PostgreSQL storage, fingerprints, fencing, and HTTP replay, but was created on
  2026-05-28 and had limited adoption. Its middleware persists after the handler,
  owns a separate pool/runtime migration path, and documents uncached outcomes when
  post-handler persistence fails. It therefore leaves the atomic lost-response gap.
- [`riverqueue/river`](https://github.com/riverqueue/river) is an established
  PostgreSQL job system with transaction-aware insertion and unique-job facilities.
  It changes the operation to asynchronous job execution and does not supply HTTP
  semantic fingerprint conflicts or response replay.
- [DBOS Go](https://github.com/dbos-inc/dbos-transact-golang) provides PostgreSQL-
  durable workflow/checkpoint semantics and transaction integration. It is a much
  larger runtime and does not inherit this repository's HTTP key/scope/Problem/replay
  contract.
- A long tail of HTTP idempotency modules (`velmie/idempo`, `fco-gt/gopotency`, and
  similar packages) was recent or pre-v1 and primarily middleware/store oriented.
  The previously published `triaon/go-idempo` source was unavailable at the cutoff.
  None established the exact caller-owned transaction, target durability,
  maintenance, and behavioral contract required here.

The decision can flip if an exact candidate reaches a stable release, active
maintenance/adoption, supports completion through the caller's pgx transaction,
exposes bounded storage and cleanup, and allows repository-native Problem, telemetry,
bootstrap, and migration ownership without parallel policy.

## Reusable mechanism versus endpoint semantics

| Reusable pack responsibility candidate | Endpoint/application responsibility that cannot be inferred |
| --- | --- |
| Validate and carry a bounded opaque key according to one published grammar | Whether the key is required, optional, or forbidden for this operation |
| Persist a composite scope, fingerprint version/digest, state/evidence, expiry, and bounded result representation | Authenticated tenant/principal authority; stable business operation/resource scope; region/environment semantics |
| Arbitrate one owner with PostgreSQL constraints and expose classified outcomes | Whether concurrent duplicates wait, fail fast, resume, or return current state |
| Join persistence to a caller-owned `pgx.Tx` without starting or committing it | Which feature writes and outbox appends constitute the atomic business effect |
| Resolve lookup through the authoritative writer and preserve unknown commit outcomes | Whether and when the complete operation is safe to retry after 40001/40P01 or commit ambiguity |
| Store/replay only a bounded status/media type/body/stable-header representation | Which success/failure results are terminal, replayable, current, or resumable; semantic versus byte identity |
| Enforce retention metadata and provide bounded cleanup/telemetry hooks | Supported client retry horizon, post-expiry behavior, legal/erasure requirements, deployment cleanup ownership |
| Map internal outcome classes into the repository failure/Problem seam | Stable per-operation Problem types/statuses, `Retry-After`, and client correction text |
| Expose closed low-cardinality outcome/latency/storage signals with redaction | Operation-specific SLOs, alert thresholds, and business reconciliation signals |
| Join template-init dependency validation, lockfile, migrations, SQLC, bootstrap, and off/on proof | Target endpoint adoption, topology, deployment migration order, and any external provider idempotency/reconciliation |

The separation is a research result, not a package/file design. In particular, a
generic mechanism cannot derive tenant scope, decide which request fields express
intent, or decide whether a cached 500 is safe to replay.

## Evidence Specification must require

Specification should not select implementation files or SQL. It must make the
following observable behavior falsifiable and carry the listed evidence obligations:

1. Endpoint opt-in and exact OpenAPI contract: header name, grammar, maximum bytes,
   requiredness, examples, response headers, and missing/malformed Problems.
2. Authenticated isolation across tenant/principal, stable operation/resource scope,
   API version, environment, and any region boundary; a replay must reauthorize the
   current caller.
3. Semantic fingerprint inputs and canonicalization version, including path/query,
   decoded body/defaults, selected headers, and exclusions. Prove harmless JSON
   representation changes and material semantic changes classify correctly.
4. Same-key/same-intent replay and same-key/different-intent conflict without leaking
   prior request data.
5. Concurrent first requests: winner, loser wait/reject behavior, budget exhaustion,
   rollback handoff, process death, retry signal, and absence of two live owners.
6. Failure points before claim, during feature work, after feature work before commit,
   connection loss during commit, after commit before serialization, response loss,
   and service restart.
7. Whole-transaction handling of 40001/40P01 under a fixed stable identity, including
   the maximum attempt/time budget and the effects that make retry unsafe.
8. Authoritative primary/failover/read-routing behavior and durability assumptions;
   replicas must never authorize a new execution from stale absence.
9. Exact result model: frozen, semantic/current, or resumable; replayable statuses;
   media types; stable-header allowlist; fresh versus retained request ID/date/trace;
   and behavior for non-JSON or streaming operations.
10. Maximum retained body/header/result bytes and deterministic oversize behavior
    before the contract can be advertised. Prove storage amplification is bounded.
11. Retention start/end, late retries at and after expiry, clock authority, tombstone
    or treat-as-new behavior, cleanup batching/cadence, expiry index, autovacuum and
    capacity under representative churn, and stuck-state recovery if states exist.
12. Privacy classification, access control, erasure, backup retention, and telemetry
    redaction proving keys, tenants, fingerprints, request/response data, and secrets
    do not escape through logs, metrics, traces, or Problems.
13. Local-transaction guarantee boundary plus endpoint obligations for outbox,
    downstream idempotency, reconciliation, or compensation around every external
    effect.
14. Problem catalog and retry contract for missing/malformed key, mismatch,
    in-progress, storage unavailable/saturated, timeout, oversize result, and unknown
    commit outcome.
15. Template-init proof for disabled/enabled/invalid combinations, lockfile
    repeatability, retained/removed migrations and packages, one SQLC regeneration,
    compile of the generated handler seam, and separate migration-before-service
    deployment.
16. Closed telemetry vocabulary for first execution, replay, mismatch, in-progress,
    rollback/retry, unknown commit, cleanup, storage capacity, and latency; no raw key
    or tenant cardinality. Define which failures affect readiness and which are
    request-scoped degradation.

## Quantities and provenance

| Quantity | Current evidence | Disposition |
| --- | --- | --- |
| Request body / headers | Repository defaults: 1 MiB / 16 KiB | Input admission only; not a stored-result default. |
| HTTP request / write timeout | Repository defaults: 8s / 10s | Current concurrency budget pressure; not a chosen idempotency wait budget. |
| HTTP in-flight / PostgreSQL pool | Repository defaults: 256 / 25 | Duplicate waits can amplify contention; later design needs measurement. |
| PostgreSQL acquire / statement timeout | Repository defaults: 1s / 8s | Current failure envelope; no new timeout selected. |
| Key length | Provider examples: [PayPal 38 bytes](https://developer.paypal.com/api/rest/reference/idempotency/), [Adyen 64 chars](https://docs.adyen.com/development-resources/api-idempotency), [Stripe 255 chars](https://docs.stripe.com/api/idempotent_requests) | No universal value; Specification must choose exact grammar and bytes. |
| Retention | Published examples: [AWS Powertools 1h](https://docs.aws.amazon.com/powertools/typescript/latest/features/idempotency/), [Stripe v1 >=24h](https://docs.stripe.com/api/idempotent_requests), [Adyen >=7d](https://docs.adyen.com/development-resources/api-idempotency), [Stripe v2 30d](https://docs.stripe.com/api-v2-overview), [PayPal endpoint-specific up to 45d](https://developer.paypal.com/api/rest/requests/) | No universal value; derive from accepted retry/reconciliation horizon and privacy/capacity evidence. |
| Stored result | No repository or authoritative universal ceiling | Specification must choose and prove a ceiling; Research does not invent one. |
| Cleanup batch/cadence | No accepted workload or capacity target | Defer until retention and representative volume are known; benchmark rather than guess. |

## Evidence limits, conflicts, and refresh triggers

- Provider documentation proves public behavior, not internal table schemas, lock
  algorithms, response ceilings, or deployment topology.
- The IETF and OASIS documents are guidance, not finalized interoperable authority.
- No public incident with first-party evidence was found that selects one PostgreSQL
  model; operational anecdotes are labeled accordingly.
- No target business endpoint, tenant model, response distribution, retry horizon,
  data classification, regional topology, failover contract, or volume forecast was
  supplied. These are downstream inputs, not reasons to invent defaults in Research.
- The Go dependency survey is time-sensitive and package metadata is not proof of
  production correctness.

Refresh before Specification freezes the contract if the IETF draft advances or the
field enters IANA, and before Technical Design selects dependencies. Reopen the
correctness model when a concrete endpoint has external effects, streaming/large
results, long-running work, different tenant/resource semantics, multi-region writers,
replica reads, asynchronous commit, failover guarantees, or a retry window outside the
comparison range.

## Source ledger

### Repository authority

- `api/openapi/service.yaml`; `internal/openapi/openapi.gen.go`
- `internal/infra/http/{handlers,router,harden,problem,domain_errors,middleware_guards,middleware_timeout,middleware_inflight,middleware_access_log}.go`
- `internal/config/{http_config,postgres_config}.go`
- `internal/infra/httpclient/{client,retry}.go`
- `internal/infra/postgres/{postgres,transaction}.go`
- `internal/infra/postgresinbox/inbox.go`; `docs/postgres-idempotent-inbox.md`
- `internal/infra/postgresoutbox/{store_append,store_receipt}.go`
- `migrations/`; `internal/infra/postgresmigrate/migrate.go`
- `cmd/service/internal/bootstrap/`; `internal/background/background.go`
- `scripts/init-module.sh`; `scripts/ci/template-init-check.sh`; `template.lock`
- `docs/repo-architecture.md`; `docs/project-structure-and-module-organization.md`

### Current authoritative and production sources

- RFC 9110, RFC 8941, RFC 9457, RFC 9562; OpenAPI 3.0.3
- IETF `Idempotency-Key` draft revision 07 and IANA HTTP Field Name Registry
- OASIS Repeatable Requests 1.0 Committee Specification
- PostgreSQL 18 `INSERT`, unique-check, transaction retry, WAL durability, and vacuum
  documentation
- Stripe v1/v2, Adyen, PayPal, Shopify, and AWS published contracts/guidance
- GDPR Article 5 for personal-data minimization and storage limitation

### Candidate and operational sources

- `faustbrian/go-idempotency`, `eben-vranken/idempo`, River, and DBOS Go
  source/documentation; smaller candidates are bounded negative evidence only
- Stripe and Shopify engineering articles; AWS 2025 PostgreSQL replica analysis
- Conroy 2025/2026 talk page and InfoWorld July 2026 anecdote, used only as labeled
  corroboration

## Research stop rationale

All accepted questions have one of four dispositions: an established repository or
external constraint, a viable/eliminated candidate family, an explicit conflict that
Specification must resolve, or a named deployment/endpoint input with a refresh
condition. Further broad search is unlikely to change the macro-phase handoff. The
next action is Specification, not more Research or implementation.

## Standalone prompt for Specification

```text
Work in /Users/daniil/Projects/Opensource/go-service-template-rest using the structured
spec-first workflow. Enter the Specification macro phase for the optional
PostgreSQL-backed HTTP idempotency capability, provisionally selected at template init
as HTTP_IDEMPOTENCY=postgres.

Read AGENTS.md, docs/spec-first-workflow.md, the current Specification phase owner, and
specs/http-idempotency-postgres/research/synthesis.md. Treat that research as evidence,
not as a selected architecture. Do not repeat broad Research unless a named refresh
trigger has fired.

Produce the smallest falsifiable behavioral contract that lets an OpenAPI business
operation opt in, declare its authenticated tenant/operation/resource scope and
semantic request fingerprint, and execute through a reusable PostgreSQL-backed
mechanism without transport owning the feature transaction. Preserve the established
ceiling: atomicity is reusable only for the feature mutation, idempotency evidence or
bounded result, and any outbox append committed in the exact caller-owned PostgreSQL
transaction; external effects require endpoint-owned downstream idempotency, outbox,
reconciliation, or compensation.

Resolve the client-visible contract for the header grammar and bound; requiredness;
scope and reauthorization; fingerprint version/canonicalization; same-key mismatch;
concurrent in-progress behavior and retry advice; timeout, rollback, serialization or
deadlock retry, lost response, and ambiguous commit; authoritative writer/failover
reads; frozen versus semantic/current/resumable results; replayable statuses and
stable headers; fresh correlation data; stored-result ceiling and oversize behavior;
retention and late retry after expiry; cleanup and stuck-state semantics; privacy and
telemetry; Problem codes/statuses; migration-before-service deployment; bootstrap
degradation/readiness; and template-init disabled/enabled/invalid behavior.

Separate reusable capability behavior from endpoint-owned semantics. State every
bounded quantity with provenance or leave it as a named external input; do not copy a
provider TTL, key limit, or state machine as a default without accepted evidence.
Include acceptance scenarios for concurrency, rollback, process death, commit
ambiguity, response loss, expiry, cleanup, capacity/oversize, replica staleness,
external effects, redaction, and generated-profile composition.

Stop at the Specification macro-phase boundary. Do not write Technical Design, Test
Design, tasks, migrations, or code. Finish with the standalone prompt for the next
workflow phase selected by the router.
```
