# A retried PostgreSQL-backed HTTP mutation resolves once within its declared window

status: ready
Problem: A service can accept the same authenticated business mutation more than
once when a client loses the response, times out, retries concurrently, or cannot
tell whether PostgreSQL committed. The template has the transaction, OpenAPI,
Problem, migration, readiness, and profile seams needed to solve that once, but no
inbound HTTP idempotency contract. Without one, each endpoint invents incompatible
key scope, replay, expiry, and recovery behavior, and transport cannot atomically
cover a transaction it does not own.

## Scope and non-goals

In scope is an optional `HTTP_IDEMPOTENCY=postgres` template capability. A
non-streaming OpenAPI business operation may opt in only by declaring its current
authenticated authority, stable operation/resource scope, semantic fingerprint,
replayable terminal result, and all endpoint-owned bounds listed below. The reusable
capability validates and classifies the request, arbitrates one execution, joins the
idempotency evidence and bounded result to the caller-owned PostgreSQL transaction,
resolves retries from the authoritative writer, and exposes closed Problems and
operator evidence.

The capability is optional at template initialization but not optional on an opted
operation: that operation requires a valid key on every request. Health endpoints
and every operation that does not opt in retain their current behavior; merely
sending `Idempotency-Key` to such an operation creates no guarantee.

Non-goals:

- Transport does not begin, commit, retry, or otherwise own the feature
  transaction. It may validate the header, carry generated operation identity, map
  classified outcomes, and render a retained semantic result.
- The reusable atomic guarantee does not cover outbound HTTP, direct broker
  publication, files, another datastore, or any effect outside the exact PostgreSQL
  transaction. Such an endpoint must own downstream idempotency, a transactional
  outbox, reconciliation, or compensation before it may advertise an end-to-end
  retry guarantee.
- `202 Accepted`, resumable workflows, streaming responses, and results that cannot
  be represented within a declared finite ceiling are not supported by this base
  capability. A concrete endpoint with one reopens Specification and distributed
  flow design.
- The stored result is not current resource state and is not a cache. The feature's
  ordinary reads remain authoritative for current state.
- This specification selects observable behavior, not SQL, a state-machine
  representation, package placement, cleanup topology, or a dependency.

## Mandatory declarations and quantities

An operation cannot opt in, and a deployment cannot claim the capability ready,
until the following applicable inputs have named values. The template supplies no
numeric default for them.

| Input | Owner and provenance | Required contract |
| --- | --- | --- |
| `K_key_bytes` | API owner, from supported clients and the deployed aggregate header budget | Positive maximum bytes for this operation's ASCII key. It must fit below the server's configured total-header limit with the operation's other headers. Provider limits are comparison evidence only. |
| `B_result_bytes` | Endpoint and capacity owners, from the maximum encoded replay result and representative storage volume | Positive maximum retained bytes across status, media type, body, and stable headers. Every replayable result must fit before its transaction may commit. |
| `T_replay` | Endpoint/business and privacy owners, from the supported client retry/reconciliation horizon | Duration from authoritative commit during which the semantic result is replayable. |
| `T_duplicate_risk` | Endpoint/business, privacy, backup, and deployment owners, from the longest time the old intent can still arrive or remain dangerous | Duration from authoritative commit, not less than `T_replay`, during which enough evidence remains to prevent the old key from becoming new work. A finite value requires evidence that later duplication is impossible or independently harmless; otherwise this horizon is permanent. |
| `W_in_progress` | Endpoint and reliability owners, derived as a strict sub-budget of the enclosing request deadline | Maximum time to classify and reject a concurrent live owner. It is not permission to wait for that owner to finish. |
| `D_retry_after` | Endpoint/deployment reliability owner, from the client's supported backoff and recovery behavior | Positive whole seconds returned on retryable idempotency Problems. |
| `T_owner_recovery` | Database/deployment reliability owner, from session-loss detection and any selected recovery policy | Maximum time after owner death before the identity resolves to definitely rolled back, completed, or outcome unknown. Elapsed time alone never proves rollback. |
| `N_tx_retry`, `T_tx_retry` | Endpoint owner, only when server-side retry is explicitly adopted | Maximum whole-transaction attempts and total elapsed retry budget, both inside the request deadline. The repository baseline is one attempt and no hidden retry. |
| Per-authority request, concurrency, and storage admission | Endpoint/security/capacity owners, from the authenticated tenant/client workload and shared database headroom | Bounded policy charged before expensive idempotency work. Global timeout and in-flight limits remain backstops, not cross-tenant fairness. |
| Cleanup batch, cadence, concurrency, maximum lag, and capacity headroom | Deployment/capacity/privacy owners, from representative churn, autovacuum, backup, and storage evidence | Bounded values that preserve both retention horizons and reject new first executions before the store can no longer retain required evidence. |
| Writer, clock, durability, failover, replica, environment, and region policy | Deployment owner, from the actual PostgreSQL topology | One authoritative clock and writer domain whose acknowledged commits remain visible after supported failover. Any shared environment or region boundary is part of scope. |

The current repository's 16 KiB aggregate headers, 1 MiB request body, 8 second
request deadline, 10 second write timeout, 256 in-flight requests, 25 PostgreSQL
connections, 1 second acquire timeout, and 8 second statement timeout constrain
candidate values; none is copied as an idempotency default. `T_replay`,
`T_duplicate_risk`, and `B_result_bytes` have no repository or provider-derived
universal value.

## Behavior and contract delta

### R1 — Opt-in is complete and fail-closed

Actor: the owner of an authenticated OpenAPI business operation.

Rule: opt-in is valid only when the operation declares all of the following as one
contract:

- required `Idempotency-Key` with `K_key_bytes`;
- the verified authority from which tenant/client/account scope is derived;
- stable operation identity, contract/API-version boundary, resource scope, and any
  shared environment or region namespace;
- fingerprint version and exact semantic input manifest;
- replayable terminal-success statuses and bounded semantic result shape;
- stable response-header allowlist;
- `B_result_bytes`, `T_replay`, `T_duplicate_risk`, `W_in_progress`,
  `D_retry_after`, and any non-baseline transaction retry policy;
- authenticated request/concurrency/storage admission sufficient to prevent one
  authority's replay or mismatch flood from consuming every shared slot;
- current authorization rule and any endpoint-owned external-effect recovery.

Absence or ambiguity in any declaration makes contract generation or bootstrap
validation fail; it never silently falls back to unguarded execution. Opt-in adds no
generic guarantee to another operation.

Falsifier: remove any declaration from one opted operation; the generated/profile
contract is rejected before that operation can serve traffic.

### R2 — The wire key is one bounded opaque token

Actor: a caller of an opted operation.

Rule: the request carries exactly one `Idempotency-Key` field. Its value is
case-sensitive, unquoted ASCII `tchar+` under RFC 9110, and one through
`K_key_bytes` bytes inclusive. Whitespace, commas, quotes, control characters,
blank values, multiple field lines, and multiple combined values are invalid; even
identical duplicates are rejected rather than guessed or joined. The caller
generates one high-entropy key for one intent and must not put personal, credential,
or business data in it. Entropy and meaning are documented caller obligations, not
values the server attempts to infer.

Missing, malformed, duplicate, or over-bound values fail before feature work. The
Problem may identify `header.Idempotency-Key` but never echo the submitted value.
Aggregate header overflow remains the existing 431 behavior.

Falsifier: each invalid form returns the R10 validation Problem and causes no
idempotency or feature write; two distinct case variants are distinct keys.

### R3 — Scope is authenticated and reauthorized on every attempt

Actor: an authenticated caller making an initial request or retry.

Rule: durable identity is the caller key inside the operation-declared composite of
current authenticated authority, stable OpenAPI `operationId` plus declared API
version, resource scope, and any shared environment/region namespace. A raw key
alone is never global authority.
The endpoint derives tenant/client/account from a trusted authority tied to the
verified principal; a request header or body field is not identity. The current OIDC
principal supplies verified issuer, subject, and client ID but no universal tenant,
so an adopting endpoint must name its tenant authority rather than infer one from the
template.

Authentication and current authorization run before lookup, mismatch disclosure, or
replay. Revoked or newly unauthorized callers receive the current 401/403 result and
learn nothing about retained state. The same raw key in a different authority scope
is independent and cannot suppress or reveal the first authority's work.

Cheap key parsing and size classification may execute before authentication, but it
does not emit a response there. Response precedence after ordinary HTTP
framing/aggregate-header rejection is: authentication 401, authorization 403,
`Idempotency-Key` validation 400, authenticated admission 429/503, then scoped
retained-state outcomes. The first retained-state-dependent action follows this
order: verified principal, current tenant/action/resource authorization,
authenticated admission charge, scoped lookup, then replay or feature execution. An
unavailable identity, authorizer, admission owner, or tenant mapping denies; it never
falls through to a raw key or caller-sent tenant field.

Falsifier: revoke the original caller before retry and observe 401/403 rather than a
replay; submit the same key and semantic request in another tenant and observe
neither a replay nor a cross-tenant conflict.

### R4 — The fingerprint compares semantic intent across versions

Actor: the opted endpoint, which owns what one intent means.

Rule: each operation declares a versioned canonical semantic document made from its
intent-bearing typed inputs after OpenAPI decode and defaults: applicable path and
query values, decoded body fields, selected semantic headers, tenant/operation/
resource scope, and API-version semantics. Raw JSON bytes, member order, insignificant
whitespace, the idempotency key, credentials, cookies, request IDs, trace context,
`Date`, and other transport-only values are excluded. The durable representation is
a comparison digest and version, not the raw request.

The version stored by the first committed attempt remains the comparison authority
for that identity. A retry within `T_duplicate_risk` is evaluated under that recorded
version, so every canonicalizer version remains available until no retained identity
can name it. New keys use the operation's current version. Removing or changing an
in-use version is a compatibility change, not a deployment detail.

Same scope/key and equivalent semantics may replay. Same scope/key and different
semantics or an unresolvable fingerprint version is a deterministic, non-retryable
mismatch; it never reveals the earlier request.

Falsifier: reorder and reformat a JSON object without changing decoded/defaulted
meaning and obtain replay; change each declared semantic input in turn and obtain
the R10 mismatch Problem.

### R5 — One caller-owned commit is the reusable atomic boundary

Actor: the operation/application adapter that owns the feature transaction.

Rule: one authoritative commit contains the feature mutation, the matching
idempotency evidence and bounded result, and any covered outbox append. The reusable
idempotency mechanism joins that exact caller-owned PostgreSQL transaction; it does
not start or commit it. Before commit or after definite rollback, none of those
covered facts is final. After commit, all are final together.

An external effect is never described as covered merely because the local evidence
committed. Before such an endpoint opts in, it declares downstream idempotency,
transactional outbox, reconciliation, or compensation and its own final result
semantics. A broker append through the same PostgreSQL outbox transaction is covered
as durable intent; later at-least-once publication remains the outbox contract.

Falsifier: kill the process after the feature mutation is staged but before commit;
neither the mutation nor idempotency evidence is visible. Commit a mutation, result,
and outbox append; all become visible together. Insert a direct outbound effect with
no endpoint recovery declaration; opt-in validation fails.

### R6 — Concurrent and failed attempts resolve without a second live owner

The observable states and transitions are:

| State and event | Outcome | Side effects |
| --- | --- | --- |
| No retained identity; one request becomes owner | Execute | Covered effects remain uncommitted until R5 commits |
| A matching request meets a live owner | Fail fast within `W_in_progress` with the R10 in-progress Problem | No second feature execution; no wait for the owner to finish |
| Live owner definitely rolls back or dies before commit | Identity becomes eligible for exactly one successor within `T_owner_recovery` | No covered effect or completed result remains |
| Owner commits | Completed | Later same-intent attempts use R7 |
| Commit result is ambiguous | Outcome unknown | No attempt may assume rollback or become a new owner until R8 resolves it |
| PostgreSQL returns 40001 or 40P01 | Entire attempt rolls back | Baseline returns the retryable R10 unavailable Problem; an explicitly declared policy may retry the whole transaction under the same identity and `N_tx_retry`/`T_tx_retry` |
| Request deadline expires before definite resolution | 504 with unknown client outcome | Client retries only with the same key; later resolution follows rollback, replay, or R8 unknown |

No retry repeats only part of a transaction, changes scope/fingerprint, or includes an
uncovered external effect. Exhausted retry budget returns a classified failure rather
than executing outside protection. A selected design may represent in-progress work
however it chooses, but it must prove one live owner, the recovery bound, and no
permanent unclassified limbo; time alone cannot turn an ambiguous commit into
rollback.

Falsifier: hold the first owner open and observe prompt loser rejection and one
feature invocation; repeat with rollback and with process death, then observe exactly
one successor. Force retry exhaustion and observe no extra effect.

### R7 — Replay is the original bounded semantic terminal success

Actor: a currently authorized caller retrying a completed, unexpired identity with an
equivalent fingerprint.

Rule: the endpoint declares a finite subset of non-streaming terminal success
statuses and the semantic result fields for each. Only that subset is retained and
replayed. Validation, authentication, authorization, mismatch, in-progress, timeout,
and 5xx responses are never frozen as terminal results.

Replay returns the original status, media type, resource identity, and semantically
equivalent original result. It is neither byte-identical response replay, current
resource state, nor resumable execution. Business timestamps and identifiers that
are part of the declared result remain original. Wire serialization may differ while
preserving the declared schema and meaning.

Only endpoint-declared semantic headers, such as a required `Location`, are stable.
Hop-by-hop headers, authentication material, cookies, `Set-Cookie`, `Retry-After`,
and prior transport metadata are never retained. `X-Request-ID`, Problem
`request_id`, `Date`, and trace/span correlation are generated fresh for every
attempt, including replay. `Content-Type` describes the freshly rendered retained
media type.

Falsifier: lose the initial response after commit, mutate the underlying resource,
and retry. The response has the original declared result/resource identity and
status, fresh correlation data, no excluded header, and no second feature effect.

### R8 — Result bounds and commit ambiguity fail closed

Rule: the complete retained representation is measured against `B_result_bytes`
before the covered transaction may commit. If it is unrepresentable or oversized,
the whole transaction rolls back, the key remains unused, and the caller receives
the R10 non-retryable server-contract Problem. The capability never commits a
business effect it cannot later resolve after response loss.

When commit completion is unknown, the service first reconciles the stable identity
against the current authoritative writer within the remaining request budget:

- matching completed evidence means committed and yields R7;
- decisive writer-confirmed absence means not committed and permits exactly one
  whole-operation successor under the same identity;
- conflicting durable evidence is an integrity failure and permits no execution;
- unavailable, stale, or inconclusive authority remains outcome unknown and returns
  the R10 unknown Problem.

Falsifier: exceed `B_result_bytes` and observe rollback plus no durable evidence;
inject connection loss during commit and drive each authoritative reconciliation
branch without a duplicate effect.

### R9 — Retention expires replay before duplicate protection

Rule: PostgreSQL's declared authoritative clock starts both horizons at the R5
commit. Before `T_replay`, a matching retry replays R7. At and after `T_replay` but
before `T_duplicate_risk`, replay material may be removed, but enough authoritative
evidence remains to return the R10 expired Problem and prevent execution. At and
after `T_duplicate_risk`, the old identity may become new work only because the
endpoint's accepted evidence says the old intent can no longer arrive or an
independent business invariant makes duplication harmless. Without that evidence,
`T_duplicate_risk` is permanent.

Cleanup removes only eligible terminal material, in bounded work, by the declared
authoritative clock. It never removes active or unknown evidence, authorizes a new
owner, changes business truth, or makes a replica's absence authoritative. A cleanup
race with replay has the same observable outcome as one of the two serial orders.
Partial cleanup is monotonic and restartable.

Cleanup failure retains too much evidence rather than deleting early. It is
operator-visible but does not by itself make a healthy instance unready while the
writer can still preserve the contract. When declared lag or capacity headroom is
exhausted, new first executions fail with the R10 unavailable Problem before
required evidence can be lost; safe replay/conflict reads continue. A terminal
process-owned cleanup task failure, stale maintenance observation beyond its
declared safety bound, or an accepted storage-safety breach is readiness-visible
through the cached readiness owner. Cleanup batching/cadence, autovacuum, backup
deletion, and capacity remain deployment/design inputs, not guessed behavior.

Falsifier: cross both horizons with a controlled authoritative clock, interrupt and
restart cleanup, and race cleanup with retry. Observe replay, then expired rejection,
then new eligibility only at the proven duplicate-risk boundary, with no second
covered effect before it.

### R10 — Problems give one stable client action

All responses use the existing RFC 9457-shaped `Problem`; `code` is the stable
machine discriminator. New codes join the repository catalog and reuse the catalog's
status-derived type URI. No raw key, digest, tenant, fingerprint, retained result,
prior request ID, database error, or existence detail appears in `detail`,
`invalid_params`, logs, or headers.

| Trigger | HTTP status and `code` | Retry contract | Durable disposition |
| --- | --- | --- | --- |
| Missing, blank, duplicate, malformed, or over-`K_key_bytes` key | 400 `bad_request` | Correct the request; no `Retry-After` | No identity consumed |
| Authenticated authority exceeds its declared request, concurrency, or storage admission budget | 429 `too_many_requests` | Required `Retry-After`; retry under the authority's budget | No lookup, identity consumption, or feature execution |
| Same scoped key, different semantic fingerprint/version | 422 `idempotency_key_mismatch` | Do not retry that changed intent with this key; no `Retry-After` | Original identity unchanged |
| Matching live owner | 409 `idempotency_in_progress` | Required `Retry-After: D_retry_after`; retry the same key | No second execution |
| Replay expired but duplicate-risk evidence retained | 409 `idempotency_key_expired` | Do not retry the old intent; no `Retry-After` | Execution remains suppressed |
| Admission authority unavailable/ambiguous, store unavailable/saturated, 40001/40P01 exhausted, or admission closed for global cleanup capacity | 503 `idempotency_unavailable` | Required `Retry-After`; retry the same key | No lookup or unguarded execution |
| Commit cannot yet be resolved authoritatively | 503 `idempotency_outcome_unknown` | Required `Retry-After`; retry only the same key | Outcome remains unknown |
| Enclosing request budget expires | 504 `gateway_timeout` | The outcome may be unknown; retry only the same key under the client's policy | Later authority decides rollback/replay/unknown |
| Replay result is unrepresentable or exceeds `B_result_bytes` | 500 `idempotency_result_too_large` | Server contract/configuration fault; no `Retry-After` | Whole transaction rolled back; key unused |

Ordinary HTTP framing and aggregate-header rejection may answer before identity is
available; aggregate headers retain 431. Otherwise authentication and authorization
retain their existing 401/403 codes and take precedence over key validation and every
idempotency/admission disclosure. Key validation precedes authenticated admission;
admission precedes every retained-state-derived response.
An invariant conflict between durable evidence and covered business state fails as
sanitized 500 `internal_error`, permits no execution, and requires operator
reconciliation.

Falsifier: every row produces the declared status/code/header combination and no
prior-request data; a client can choose correct/request-new-key/retry-same-key/stop
from status, code, and `Retry-After` without parsing `detail`.

### R11 — The writer is the only correctness read authority

Rule: initial arbitration, an absence that could authorize execution, completed
result lookup, ambiguous-commit reconciliation, expiry transition, and cleanup
eligibility resolve against the current authoritative writer. Replica absence,
lag, read-only state, split routing, uncertain promotion, or asynchronous durability
never authorizes execution.

A promoted writer becomes authoritative only after the deployment proves one writer
and visibility/preservation of every acknowledged covered transaction. Until then,
opted requests return unavailable/unknown and no new owner is admitted. The
capability cannot advertise its atomic guarantee under uncontrolled multi-writer
topology, acknowledged-write loss, or failover that cannot preserve committed
evidence.

Falsifier: hide a committed row on a replica and route a retry through it; the
service consults the writer or fails closed, and never repeats the effect. Promote a
writer without established visibility and observe no idempotent execution.

### R12 — Privacy and telemetry preserve the boundary

Raw keys and raw requests are not retained. Retained comparison tokens,
fingerprints, tenant/resource scope, semantic results, and expiry data are treated as
sensitive, tenant-scoped data with the same access, erasure, incident, and backup
policy as the result they protect. The operation's privacy owner declares result
classification and backup/erasure horizons before opt-in. Erasure that removes
duplicate-risk evidence explicitly narrows the guarantee; it may not silently turn
an old key into new work.

Expiry or erasure immediately removes replay access even if physical cleanup or
backup extinction lags. Erasure is tenant-scoped and must leave a lawful
non-disclosing duplicate guard through `T_duplicate_risk`; without one, the endpoint
cannot promise both erasure and suppression and must reopen this contract. A backup
restore cannot become ready until expiry and erasure effective at restore time are
re-established, so restored evidence never resurrects replay access.

The closed transition-event vocabulary is:

`first_execution_started`, `replay_served`, `mismatch_rejected`,
`in_progress_rejected`, `rollback_released`, `retry_started`, `commit_unknown`,
`cleanup_failed`, `cleanup_recovered`, and `other`.

The terminal request vocabulary, recorded once for every authenticated, syntactically
valid opted request that reaches the decision, is `executed`, `replayed`, `mismatch`,
`in_progress`, `commit_unknown`, `failed`, and `other`. Unknown producer values
collapse to `other`; transition events are not misreported as terminal outcomes.

Operator evidence answers four questions without another data surface:

- Are callers being resolved, replayed, or rejected? Count the closed transition and
  terminal outcomes; the existing access record's bounded route, status, and Problem
  code identify the affected operation.
- Where is time spent? Observe first execution, lookup/reconciliation, and total
  request latency under the existing request budget.
- Can required evidence still be retained? Observe terminal/guarded volume, oldest
  expiry/cleanup observation, cleanup lag/failure, and capacity headroom with
  deployment-owned alert thresholds.
- Did an ambiguous or integrity outcome require intervention? Emit a sanitized
  bounded class correlated through the current request/trace IDs.

No raw key, comparison token, fingerprint, tenant/principal/resource identity,
request/response content, SQL, database error text, or arbitrary caller value is a
log field, span attribute, metric label, exemplar, Problem detail, or event name.
Capability metrics use only the closed vocabularies: no operation, route, tenant,
key-derived, fingerprint-derived, status, or error-text label is added. Cleanup and
storage signals expose aggregate count/bytes, oldest-expired and observation
timestamps, lag, and safety headroom without a per-row/tenant/key series; age is
derived at query time so a stalled observer cannot look fresh. Fresh request and
trace IDs remain the authorized forensic pivot already supplied by the repository.

Writer/schema absence during startup keeps startup admission closed and readiness
false; liveness remains process-only. Runtime saturation, one in-progress conflict,
one timeout, or recoverable cleanup lag is request-scoped degradation and does not
flip readiness. A terminal supervised task failure, stale maintenance observation,
accepted storage-safety breach, or loss of the authoritative PostgreSQL dependency
follows current cached-readiness behavior; no dependency I/O or cleanup runs in the
readiness handler.

Falsifier: exercise every outcome with sentinel key/tenant/fingerprint/result values
and prove those sentinels are absent from Problems, logs, spans, metric attributes,
and exported series while the bounded outcome and fresh correlation remain.

### R13 — Template initialization and deployment preserve the advertised contract

`HTTP_IDEMPOTENCY` accepts exactly `none|postgres` and defaults to `none` when
unset. An explicitly empty or unknown value fails before repository mutation.
`postgres` requires `DATABASE=postgres`; it is independent of outbox, messaging,
and transport profiles. It does not require `AUTHN=oidc-jwt` at initialization
because the health-only template opts in no business operation, but R1/R3 prevent a
future operation from opting in without real authentication and tenant authority.

`none` removes every capability-owned runtime, migration, query/generation input,
configuration, documentation, and test surface, leaving no advertised guarantee.
`postgres` retains one coherent pack, records `http_idempotency = "postgres"` in
`template.lock`, and composes with the selected PostgreSQL, authn, outbox, and
messaging profiles without opting health routes in. Repeating initialization with
the same locked choice is a no-op; a different choice is refused.

Canonical forward migrations complete before any service revision can advertise an
opted operation. The service never migrates at startup and does not become ready
against missing/incompatible schema or an unavailable writer authority. After the
first opted request reaches the activation boundary, an old service revision that
ignores this contract is not a safe rollback target for that operation; rollback
must keep opted traffic gated or use a contract-preserving revision while durable
evidence and both horizons remain intact.

Adding a required key to an already published operation is breaking unless a new or
versioned operation or an accepted client migration supplies it. Changing key
grammar or `K_key_bytes`, scope, fingerprint version/manifest, replay set/result,
stable headers, either retention horizon, expiry action, status/code, retry advice,
or writer region is a client-visible behavior change and receives explicit
compatibility treatment.

Falsifier: the generated-profile scenarios below compile and expose only their
declared surfaces; deployment of the service before its migration remains not ready;
an old unguarded revision cannot receive opted traffic after activation.

## Reusable capability versus endpoint-owned semantics

| Reusable capability | Endpoint/deployment owner must supply |
| --- | --- |
| Parse and validate one bounded `Idempotency-Key`; publish shared Problem classes | Whether a business operation opts in and its `K_key_bytes` |
| Carry generated operation identity and classify lookup/replay outcomes | Authenticated tenant/client authority, resource/API/environment/region scope, and current authorization |
| Compare a versioned semantic digest without retaining raw request | Canonical semantic input manifest and version lifecycle |
| Arbitrate one live owner and join evidence/result to caller-owned transaction | Feature mutation boundary, replayable terminal successes, semantic result, stable headers, and optional whole-transaction retry safety |
| Resolve absence/unknown/completion on the writer and render a bounded replay | Writer/failover/durability policy and `B_result_bytes` |
| Enforce retention states and cleanup eligibility | `T_replay`, `T_duplicate_risk`, cleanup/capacity/backup/erasure policy |
| Expose closed redacted outcomes and request-scoped failures | Alert thresholds, SLOs, and business reconciliation signals |
| Compose optional template profile and migration-before-service contract | Concrete endpoint adoption, deployment graph, traffic activation, and rollback route |

## Decisions, constraints, and authorities

- **D1 — Required per opted operation, not an optional best-effort header.** Two
  behaviors on one mutation make a missing header the path that bypasses protection.
  Rejected alternative: optional key. Reopen only for an explicitly different
  best-effort endpoint contract.
- **D2 — Unquoted RFC 9110 token grammar with an operation-owned byte bound.** It is
  interoperable HTTP syntax without claiming the expired IETF draft's quoted
  Structured Field grammar or copying one provider's cap. Reopen if a finalized,
  adopted standard or supported-client evidence requires another wire form.
- **D3 — Fail fast on a live owner.** Waiting by default lets one hot key occupy the
  template's scarce PostgreSQL connections under 256 request slots and a 25
  connection pool. Rejected alternative: wait for the winner. Reopen only with a
  smaller declared wait budget and representative capacity proof.
- **D4 — Frozen semantic terminal success, freshly rendered.** Rejected alternatives:
  byte replay leaks stale transport metadata; current-state replay changes the
  original outcome; resumable execution requires a workflow contract; frozen 5xx
  cannot prove whether an effect occurred.
- **D5 — Replay and duplicate-risk horizons are separate.** Deleting all evidence at
  replay expiry turns absence into permission for a late duplicate. Rejected
  alternative: provider-style TTL followed immediately by treat-as-new. Reopen when
  endpoint evidence proves both horizons identical and safe.
- **D6 — No hidden transaction retry.** The current transaction owner deliberately
  cannot know whether all effects are repeatable. An endpoint may declare bounded
  whole-operation retry only under R6; a driver-level retry is rejected.
- **D7 — Stable idempotency codes use the existing Problem envelope.** Generic 409 or
  503 alone collides with domain conflicts and other availability failures; a client
  needs the stable action-specific `code`. New ad-hoc Problem fields/type schemes are
  unnecessary.
- **D8 — Writer absence is authority; replica absence is not.** This follows the
  current outbox receipt precedent and commit-unknown taxonomy. Rejected alternative:
  execute on any missing read.
- **D9 — `HTTP_IDEMPOTENCY=postgres` is a sibling PostgreSQL profile.** It reuses
  current transaction, migration, readiness, Problem, telemetry, and profile owners
  without becoming transport-owned or part of inbox/outbox. Reopen only if design
  proves an existing pack can own the full contract without changing its current
  actor and lifetime.

Authorities: `api/openapi/service.yaml` owns published operations; `internal/problem`
owns stable HTTP codes/status/type/title; the feature owns semantic scope/result and
authorization; the exact caller-owned PostgreSQL transaction owns local finality;
the current writer owns correctness reads and clock; canonical migrations precede
service deployment; the accepted research synthesis supplies evidence and refresh
conditions but selects no architecture.

## Acceptance scenarios and proof expectations

These are behavioral acceptance boundaries, not a Test Design or command plan.

| Scenario | Pass condition and nearest faithful evidence |
| --- | --- |
| Missing/malformed/duplicate/over-bound key | Exact R10 400 result and zero feature/store calls through the generated OpenAPI path |
| Equivalent and changed semantics | Harmless JSON/default changes replay; each declared semantic change returns mismatch, against the operation's canonicalizer corpus |
| Cross-tenant retry and revoked authorization | No cross-tenant replay/existence signal; current 401/403 precedes retained state |
| Concurrent same-key requests | One live feature invocation; loser returns in-progress within `W_in_progress` with required retry advice |
| Winner rollback | No covered effect/evidence remains; exactly one successor may execute |
| Process death before, during, and after commit | Before: rollback/new eligibility; during: unknown until writer reconciliation; after: replay |
| Serialization/deadlock | Whole attempt rolls back; baseline 503 or bounded declared full retry; no partial or external effect repeats |
| Ambiguous commit | Writer-confirmed completed/absent/conflict/unavailable produce replay/new eligibility/integrity failure/unknown respectively |
| Response serialization failure or loss after commit | Retry returns semantic replay with fresh correlation and no second mutation |
| Result capacity and oversize | Maximum accepted representation fits `B_result_bytes`; one byte over rolls back everything and returns result-too-large |
| Replay and duplicate-risk expiry | Before `T_replay`: replay; between horizons: expired rejection; after the accepted duplicate-risk boundary: new eligibility only under its declared proof |
| Cleanup interruption, race, and backlog | Restart is monotonic; replay/expiry semantics do not depend on cleanup timing; capacity admission closes before required evidence is lost |
| Replica staleness and failover | Stale absence never authorizes execution; uncertain promotion returns unavailable/unknown until one writer preserves accepted commits |
| External effect | Uncovered effect cannot claim the local guarantee; outbox/downstream idempotency/reconciliation/compensation path is explicit and survives retry |
| Abuse isolation | A replay/mismatch/in-progress flood is charged to the authenticated authority before expensive work; an unrelated in-budget authority retains admission within global headroom |
| Redaction and telemetry | Sentinel sensitive values appear nowhere; closed transition/terminal outcomes, latency/capacity signal, and fresh correlation remain |
| Expiry, erasure, and restore | Expired/erased material is not replayable while physical deletion lags; backup restore stays not ready until current expiry/erasure is re-established and exposes no resurrected result |
| Migration and bootstrap | Migration-before-service succeeds; missing schema/writer never reaches readiness; service does not apply schema at startup |
| Generated-profile composition | `none` leaves no pack residue; `postgres` composes with `DATABASE=postgres`, authn on/off, and outbox/messaging on/off without changing health operations; `postgres` with non-PostgreSQL, empty, or unknown selection fails before mutation; lock replay is identical and lock mismatch fails |
| Compatibility/rollback | Existing operation is not silently made required-key; after activation no unguarded old revision receives opted traffic |

Concurrency, transaction failure, commit ambiguity, authoritative reads, expiry race,
and cleanup require real PostgreSQL because a stub cannot prove its arbitration or
visibility. Generated-profile behavior requires an initialized derived tree and
contract regeneration. Redaction requires inspecting exported Problems, logs, spans,
and metrics, not merely a redactor call. External effects require evidence at the
actual downstream/outbox/reconciliation boundary. Test Design owns the eventual
fixtures, proof levels, and commands.

## Review result

Independent whole-spec review found two client-visible forks in simultaneous
authentication/key failures and authenticated admission denial. R3 and R10 now fix
the response precedence plus 429/503 status, code, retry, and no-effect dispositions;
focused fresh re-review returned `PASS`. No other finding survived the fixed evidence
boundary.

## Risks, assumptions, and reopen conditions

- **Parameterized template, no concrete endpoint.** Safe boundary: R1 makes every
  absent business quantity a mandatory declaration, so design can build reusable
  enforcement without inventing an endpoint's tenant, retry, result, retention, or
  topology. Reopen owner: Specification for the first endpoint whose proposed values
  change observable behavior; endpoint/domain/security/deployment owners supply the
  values.
- **Single authoritative durable writer domain.** Safe boundary: every correctness
  decision uses that writer and acknowledged commits survive the supported failover.
  Invalidating evidence: multi-writer acceptance, `synchronous_commit=off`, replica
  authorization, asynchronous commit, acknowledged-write loss, or promotion without
  visibility/fencing. Reopen owners: Specification if client finality changes,
  otherwise System/Integration Design and delivery proof.
- **Bounded, replayable semantic results.** Invalidating evidence: streaming, large
  or unbounded bodies, `202`, current-state or resumable semantics, or a response
  distribution that cannot fit a justified `B_result_bytes`. Reopen owner:
  Specification for that endpoint.
- **Fail-fast concurrency.** Invalidating evidence: a supported client cannot recover
  from 409/retry advice, while measured capacity proves a bounded wait safe and
  preferable. Reopen owner: Specification.
- **Two-stage retention.** Invalidating evidence: privacy/erasure forbids keeping even
  minimized duplicate-risk evidence through the business horizon, or a domain
  invariant makes immediate safe reuse provable. Reopen owner: Specification with
  privacy/domain authority.
- **Research freshness.** The 2026-08-11 synthesis baseline matches current `main` and
  no Specification refresh trigger fired. Reopen Research if the IETF field becomes
  finalized/registered before this contract is frozen, or if a concrete endpoint
  introduces the synthesis's named external-effect, streaming/large-result,
  long-running, tenant/resource, multi-region, replica, asynchronous durability,
  failover, or out-of-range retry-window trigger.
