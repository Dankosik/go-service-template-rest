# Inbound webhooks Technical Design

status: complete candidate

Behavior authority: [`../spec.md`](../spec.md). This design selects mechanism
and ownership only. It does not define the Test Design matrix, implementation
tasks, deployment execution, or provider registration.

## Decision and drivers

The selected mechanism is one profile-owned ingress path through the existing
hardened HTTP server, one PostgreSQL receipt table, and one typed River job.
The receipt is the acceptance authority. River owns claims, attempts, retry
scheduling, recovery, telemetry, drain, and shutdown; no second queue, retry
loop, or webhook process is added.

| Driver | Selected consequence | Strongest rejected alternative | Rejection |
| --- | --- | --- | --- |
| The signature covers exact received bytes and verification must precede JSON interpretation. | Keep the generated chi route and validator, but dispatch this operation through the generated raw `ServerInterface`; its OpenAPI `application/json` media type has no schema, so the pinned validator reads and restores bytes without decoding them. | Use the generated strict request object. | oapi-codegen v2.8.0 calls `json.Decoder` before the strict handler, violating the accepted verification order and losing an honest exact-byte boundary. |
| Every `204` requires one receipt and one job in one commit. | Insert the receipt and call River v0.44.0 `InsertTx` in `postgres.InTx`. The River args contain only the receipt identity. | Put the body in River args or enqueue after committing the receipt. | The first duplicates retained payload and lifecycle authority; the second permits an acknowledged receipt with no job. |
| Duplicate and conflicting reuse must converge under concurrency. | The composite receipt key `(endpoint_id, delivery_id)` serializes `INSERT ... ON CONFLICT DO NOTHING`; a following read compares the stored SHA-256. | Process-local locks or River uniqueness. | Neither spans replicas/processes or permanently preserves the accepted identity after River retention. |
| Processing is at least once and terminal evidence outlives River cleanup. | The worker reads the receipt, invokes one registered decoder and handler, then writes `handled`, `quarantined`, or `failed` to the receipt. | Treat River job state as receipt outcome. | River job retention would eventually erase the only terminal evidence and cannot own payload erasure. |
| The capability is removable and provider-neutral. | Add one `INBOUND_WEBHOOKS=none|standard-webhooks` profile; retain the already-pinned Standard Webhooks dependency when either inbound or outbound webhooks needs it. | A protocol/provider registry or new capability generator. | V1 has one accepted protocol and the current initializer already owns profile removal. |

No new dependency, binary, queue, cache, limiter, timeout, or payload-size
setting is selected.

## Pinned compatibility and signature vector

The current tree pins:

- `github.com/standard-webhooks/standard-webhooks/libraries v0.0.1`;
- `github.com/riverqueue/river v0.44.0`;
- `github.com/getkin/kin-openapi v0.147.0`;
- `github.com/oapi-codegen/nethttp-middleware v1.2.0`; and
- generated server code from oapi-codegen v2.8.0.

Inbound verification performs strict header cardinality and syntax checks,
parses the signed Unix seconds against an injected clock with the fixed
five-minute past/future tolerance, then calls the official library's
`VerifyIgnoringTimestamp` for the active key and, when present, its predecessor.
This retains the pinned library's canonical message and constant-time
`hmac.Equal` while avoiding its process-global `time.Now` in deterministic
proof. Library errors are classified to bounded local errors and never cross a
response or telemetry boundary.

The non-secret compatibility vector is:

```text
signing material: 0123456789abcdef0123456789abcdef
Webhook-Id: msg_123
Webhook-Timestamp: 1700000000
body bytes: {"hello":"world"}
canonical bytes: msg_123.1700000000.{"hello":"world"}
Webhook-Signature: v1,jUcl6cc4RhnPU/D4RhXcoyQYBvOxqIsONY9102iBndo=
```

The vector was reproduced against the pinned library. Any dependency or
protocol change must reproduce it and the five-minute boundary before the
design remains current.

## Material flows

### Admission, verification, and durable acceptance

1. `httpx.Harden` applies request correlation, tracing, security headers,
   access logging, the existing `http.max_body_bytes`, request budget,
   max-in-flight shedding, optional rate limiting, and recovery.
2. The generated chi route binds `endpoint_id`. The OpenAPI validator enforces
   the path grammar, required header presence, `application/json`, and the
   declared public security decision. Its request-body media type deliberately
   has no schema: it restores the same bytes and does not interpret JSON.
3. The profile-owned raw operation adapter requires exactly one value for each
   Standard Webhooks header and reads the restored bounded body. It never logs
   a submitted value. The validator's restored buffer and the adapter's owned
   byte slice can coexist, so the selected path has two full-body copies per
   admitted request. The profile updates the existing startup buffer-budget
   arithmetic from one copy to two; `MaxInFlight` and `MaxBodyBytes` remain the
   admission controls and no third buffer or second size setting is added.
4. The adapter calls the neutral `inboundwebhook.Receiver` port and maps only
   its closed accepted, duplicate, unknown-endpoint, rejected, conflict, and
   unavailable categories through the existing Problem writer.
   `postgresinboundwebhook.Receiver` implements that port, resolves the endpoint
   before cryptography or SQL (`unknown` becomes `404` with no database call),
   and validates the delivery ID, canonical Unix
   timestamp, active/predecessor secret binding, and signature. It computes
   SHA-256 only after verification. Authentication failure is one sanitized
   `400` class and performs no SQL.
5. `postgres.InTx` attempts to insert the receipt with a stdlib
   `crypto/rand.Text` internal receipt ID. A new row is followed by River
   `InsertTx` for job kind `inbound_webhook_receipt`; any error rolls both back.
   The job args contain only that internal receipt ID.
6. The adapter returns `204` only after `postgres.InTx` reports a definite
   commit. `postgres.ErrCommitUnknown`, dependency failure, or saturation maps
   to retryable `503` with `Retry-After`; an expired request context maps to
   `504`. No unknown outcome becomes success.

### Duplicate and conflicting reuse

Every duplicate is revalidated against the current active/predecessor secret
and timestamp before storage lookup. `INSERT ... ON CONFLICT DO NOTHING` waits
for a concurrent owner of the composite key. Under the next read-committed
statement, the stored hash is visible:

- equal hash: create nothing and return `204`;
- different hash: preserve the first row and job, create nothing, and return
  `409`.

The same delivery ID under another endpoint uses another composite key.

### Processing and terminalization

The jobs worker loads the receipt by the internal receipt ID carried in River
args. The loaded row supplies the authoritative composite
`(endpoint_id, delivery_id)` identity. A non-`pending` receipt is a successful
no-op, making a duplicate or recovered job harmless. A pending receipt is
copied into `inboundwebhook.VerifiedDelivery` and passed to the one binding
registered for that endpoint.

Registration is typed: `Bind[T]` accepts exactly one pure decoder
`func(json.RawMessage) (T, error)` and one handler
`func(context.Context, VerifiedDelivery, T) error`. The registry rejects nil or
duplicate bindings and jobs-worker startup requires exact set equality with the
configured endpoint manifest. A typed, bounded decoder rejection marks
`quarantined` and retains the body without invoking the handler.

The worker invokes decoder and handler behind a local recovery/sanitization
boundary. It recovers every panic before River can observe the panic value and
maps unexpected decoder failures, handler errors, store/SQL failures, and
panics to closed package-owned sentinels whose text contains no wrapped cause,
provider value, or receipt content. Only those sentinels are returned to River,
so River logs, persisted attempt errors, and otelriver trace status never
receive feature or provider text. The worker records only internal receipt ID
and a bounded failure class through capability-owned telemetry. Feature
bindings inherit the same rule for any logs or spans they create; canary proof
must cross the complete binding call, not only inspect the outer worker error.
Sanitized unexpected failures remain retryable under River's existing job
policy.

The River-facing error texts are fixed to `inbound webhook decoder failed`,
`inbound webhook handler failed`, `inbound webhook storage unavailable`,
`inbound webhook panic recovered`, and
`inbound webhook terminalization unavailable`. Corresponding log classes are
`decoder_internal`, `handler_retryable`, `storage_retryable`,
`panic_recovered`, and `terminalization_retryable`. No constructor accepts a
cause or caller-provided string.

After handler success, one conditional update changes `pending` to `handled`,
sets the terminal time, and erases the payload bytes while retaining identity,
hash, signed/received times, and outcome. If this update is lost, a later
attempt may call the handler again; the feature handler therefore owns an
idempotent effect keyed by `(endpoint_id, delivery_id)` or a stronger provider
identity.

The job reserves one River attempt beyond the ordinary handler-attempt budget.
That last attempt invokes no business code; it only persists `failed` after
handler exhaustion. If the terminal update cannot be stored, the worker uses
River's native `JobSnooze` so terminalization does not consume another handler
attempt or fall into discarded-with-pending state. This is recovery inside the
existing engine, not another retry scheduler.

### Startup, readiness, drain, and release order

The service process constructs the immutable endpoint and secret manifests,
the River producer, and the receiver after the existing PostgreSQL dependency.
Missing or inconsistent manifest data fails startup. HTTP readiness continues
to use the existing PostgreSQL probe; it does not call the worker.

The jobs-worker bootstrap keeps its current nil-builder check and calls
`WorkersBuilder` before database I/O. The builder returns a small
`WorkersRuntime` containing the already-validated workers, cleanup, and an
optional profile-owned `Bind` function. Only after every builder error has been
closed does bootstrap open its single PostgreSQL pool and call `Bind` with that
pool and the existing meter provider. The inbound binder constructs its exact
feature bindings, attaches runtime dependencies to its worker, and registers
that worker before River client construction. Outbound and test workers have
no binder. The same River client still owns polling, queue concurrency, drain,
and shutdown.

The mixed-version order is additive migration; stop and drain every old jobs
worker; start only jobs workers with the new kind and complete bindings; prove
their immutable image/build identity and readiness; then expose the new HTTP
service. An old River worker is never allowed to overlap accepted jobs of the
new kind because River claims queue-wide before discovering an unknown kind.

Rollback first blocks new public ingress and drains the new HTTP service. While
the new worker remains the only jobs-worker version, deployment runs this
canonical read against the writable primary:

```sql
SELECT
    NOT pg_is_in_recovery()
        AND current_setting('transaction_read_only') = 'off' AS writer_primary,
    count(*) FILTER (WHERE outcome = 'pending') AS pending_receipts
FROM inbound_webhook_receipts;
```

Only `writer_primary = true AND pending_receipts = 0`, observed after HTTP
drain, permits replacement by an old worker. Any nonzero/unknown result is a
roll-forward-only stop: keep the new worker until it terminalizes the backlog.
The receipt table remains across binary rollback, and a down migration may drop
it only when the whole table is empty. The capability documentation owns this
query and order; the deployment owner must supply primary read authority,
immutable worker identity/readiness readback, and the actual stop/start gates.
Provider registration, ingress/TLS, secret rotation, capacity, and execution
remain external inputs.

## Data authority and schema

Migration `000010_postgres_inbound_webhooks.sql` adds one relation. The
composite provider identity remains its primary key; one random internal ID
exists only for logs and River lookup. There is no payload projection, attempt
table, endpoint table, or foreign key into River internals.

```text
inbound_webhook_receipts
  receipt_id        text COLLATE "C"  NOT NULL UNIQUE
  endpoint_id       text COLLATE "C"  NOT NULL
  delivery_id       text COLLATE "C"  NOT NULL
  body_sha256       bytea             NOT NULL
  signed_at         timestamptz       NOT NULL
  received_at       timestamptz       NOT NULL DEFAULT clock_timestamp()
  payload           bytea
  outcome           text COLLATE "C"  NOT NULL DEFAULT 'pending'
  terminal_reason   text COLLATE "C"
  terminal_at       timestamptz
  PRIMARY KEY (endpoint_id, delivery_id)
```

Constraints enforce a bounded non-empty internal receipt ID, endpoint bytes
`1..64`, delivery bytes `1..256`, a 32-byte hash, and this closed lifecycle:

| outcome | payload | terminal reason | terminal time |
| --- | --- | --- | --- |
| `pending` | present | absent | absent |
| `handled` | absent | absent | present |
| `quarantined` | present | bounded decoder reason | present |
| `failed` | present | `attempts_exhausted` | present |

PostgreSQL `text` owns valid UTF-8; the receiver owns the stronger whitespace
and control-character rule before SQL. Payload is `bytea`, not `jsonb`, because
the exact signed bytes are authoritative and JSON interpretation is
asynchronous. The primary key owns duplicate arbitration. One partial index on
`(received_at, receipt_id) WHERE outcome = 'pending'` supports the canonical
rollback drain read without indexing terminal evidence or provider IDs; River
still owns runnable selection.

SQLC owns six query shapes in
`internal/infra/postgres/queries/postgres_inbound_webhooks.sql`: claim, read by
provider identity, read by internal receipt ID, mark handled with payload
erasure, mark quarantined, and mark failed. State updates are conditional on
`pending`; replay of an already-terminal receipt is a no-op, not a second
transition.

## OpenAPI and response composition

`api/openapi/service.yaml` owns `POST /webhooks/{endpoint_id}` inside the
inbound profile. It declares `security: []` and an `x-security-decision`
explaining signature authentication, no CORS, all reachable statuses, required
`Retry-After` on `429` and `503`, and an empty `204` response.

The generated standard server method exposes `*http.Request`, whereas its
strict sibling decodes JSON first. The existing `Handlers.API` field remains
the complete generated `openapi.StrictServerInterface`, so the selected design
also fixes how that interface is composed:

- the existing strict adapter continues to serve health and ordinary
  application operations;
- `cmd/service/internal/bootstrap/service_api.go` owns the concrete service API
  composite. It embeds/delegates ordinary feature handlers and implements the
  inbound strict method as a fail-closed unreachable fallback, so an ordinary
  feature handler is not forced to implement an operation it does not own;
- the generated standard-server wrapper delegates only `ReceiveWebhook` to
  `inbound_webhook.go`, which owns header cardinality, raw body read, and mapping
  typed receiver results through the existing RFC 9457 writer, shadowing that
  strict fallback before dispatch.

This keeps the generated route, request validator, hardening chain, route
labels, and Problem catalog. A manual chi route or second HTTP stack is not
added.

## Configuration and secret ownership

The retained profile adds `InboundWebhooksConfig` with two immutable JSON
strings:

- `inbound_webhooks.endpoints`: non-secret endpoint IDs plus active and optional
  predecessor key references;
- `inbound_webhooks.static_secrets`: environment-only endpoint/key-reference to
  `whsec_` secret material.

There is no runtime `enabled` flag: selecting the initializer profile makes the
route and startup obligations present. The parser rejects unknown fields,
duplicates, invalid identifiers, more than 4096 endpoints, missing referenced
keys, unused keys, decoded keys outside 32..64 bytes, active/predecessor
equality, and secret-byte reuse across endpoints. Secrets remain byte slices in
the immutable process snapshot and never enter YAML, generated files,
responses, logs, traces, metrics, River args, or receipt rows.

The service config loader retains and validates both leaves and is the only
process that decodes or holds verification keys. The jobs-worker projection
selects only the exact non-secret `inbound_webhooks.endpoints` leaf, validates
its endpoint IDs, and requires exact configured-endpoint to decoder/handler
binding equality; it neither selects nor validates
`inbound_webhooks.static_secrets`. Worker deployment must not inject the
service-only secret variable, and worker startup rejects it when present rather
than silently accepting a broadened secret distribution. Key references may
remain in the non-secret endpoint document for one configuration authority,
but the worker never resolves or stores their secret bytes.

## Profile removal and retained dependency

The initializer adds `INBOUND_WEBHOOKS=none|standard-webhooks`, defaults to
`none`, requires `DATABASE=postgres JOBS=postgres` for the selected value, and
records `inbound_webhooks = "..."` in `template.lock`.

`none` removes the following complete pack before regeneration and `go mod
tidy`:

- the OpenAPI path and generated operation;
- `internal/inboundwebhook/`;
- `internal/infra/postgresinboundwebhook/`;
- `internal/infra/http/inbound_webhook.go` and its raw-composition blocks;
- `internal/config/inbound_webhooks_config.go` and all marked config/snapshot
  surfaces;
- `cmd/service/internal/bootstrap/startup_inbound_webhooks.go` and marked
  service wiring, including the profile-owned `service_api.go` composite;
- the second-copy increment in
  `cmd/service/internal/bootstrap/runtime_request_buffer_budget.go` and its
  selected-profile proof;
- `cmd/jobs-worker/inbound_webhook_bindings.go` and marked worker registration;
- migration `000010_postgres_inbound_webhooks.sql`, its SQLC source/generated
  file, and inbound PostgreSQL/process proofs;
- inbound webhook documentation, Makefile/check surfaces, examples, and
  profile-marked architecture text.

`cmd/jobs-worker/builder_webhooks.go` remains when either inbound or outbound
webhooks is selected and is removed only when both are `none`. Migration
`000008_river.sql` remains whenever jobs, outbound webhooks, outbox, or inbound
webhooks needs River. The Standard Webhooks module remains when either
`WEBHOOKS=durable` or `INBOUND_WEBHOOKS=standard-webhooks` imports it and is
otherwise removed by `go mod tidy`. The selected tree contains no unresolved
profile markers after initialization.

The jobs pack changes its internal builder return tuple to `WorkersRuntime` so
bootstrap can preserve the current pre-database builder error order and still
offer one post-pool binding point. The `Bind` field, call, imports, and inbound
registration are profile-marked and disappear when inbound is `none`; the
remaining runtime contains only the existing workers and cleanup values.
`JOBS=none` still removes the entire binary. All shipped builders adapt
mechanically to the result type, and
`INBOUND_WEBHOOKS=none JOBS=postgres` preserves the current nil-builder,
builder-error, database-open, River-start, and cleanup order without an inbound
artifact.

## Observability and proof boundaries

The adapter constructs two counters from the existing meter provider: ingress
outcomes (`accepted`, `duplicate`, `rejected`) and processing outcomes
(`quarantined`, `retrying`, `handled`, `failed`). HTTP status metrics retain
transport classes such as conflict, overload, and timeout; no second HTTP
counter restates them. Logs correlate by request ID before acceptance and by
the random internal receipt ID afterward. Endpoint IDs, delivery IDs, body,
headers, signatures, secrets, decoder errors, SQL errors, and provider text are
absent from logs, traces, and metric attributes.

Technical Design fixes these proving surfaces without defining Test Design:

- pinned-library vector and active/predecessor verification at a controllable
  clock;
- generated raw-route byte identity and proof that JSON decoding does not run
  before verification;
- real PostgreSQL receipt/job atomicity, concurrent duplicate/conflict, rollback,
  and commit-unknown readback by sender retry;
- River process-loss recovery, at-least-once handler invocation, payload erasure,
  quarantine, exhaustion finalization, and snoozed terminal-write recovery;
- a provider-text canary through decoder error, handler error, SQL/store error,
  and panic, proving absence from River logs, persisted attempt errors, spans,
  adapter logs, and metric attributes;
- selected two-copy and unselected one-copy request-buffer budget arithmetic;
- worker config proof that endpoint IDs are present while verification secrets
  are neither selected nor accepted; and
- release readback against a writable primary plus proof that no old worker
  overlaps the new job kind;
- OpenAPI/runtime response and public-signature-auth parity; and
- selected/unselected initializer trees, shared-dependency retention, generated
  drift, migration history, and profile-marker absence.

Local proof cannot establish public ingress, TLS, secret rotation, provider
registration, database/worker capacity, or a provider-specific decoder and
idempotent effect.

## Ownership Map V1

### Responsibilities

| responsibility | affected path | current evidence | semantic owner | exact package/file action | dependency/composition/generated boundary | cleanup | proof owner | reopen condition |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| Public route and responses | OpenAPI -> generated chi router | `api/openapi/service.yaml`, oapi-codegen v2.8.0 | OpenAPI contract | add marked path and regenerate | YAML is authority; generated Go is derived | profile removes path/output | OpenAPI/runtime contract | observable wire rule changes |
| Exact-byte HTTP dispatch | Harden -> validator -> generated standard method | validator v0.147.0 restores bytes; strict generator decodes | HTTP transport | add profile-scoped raw override in `internal/infra/http` | ordinary operations remain strict; no manual route | profile removes override | byte-identity transport proof | generator/validator stops preserving bytes |
| Request-buffer capacity accounting | validator buffer + raw adapter buffer -> startup warning | current runtime reports one full copy | service runtime limits | profile-mark `runtime_request_buffer_budget.go` to count two copies for selected inbound | existing `MaxInFlight`, `MaxBodyBytes`, GC limit, warning, and no-reject policy remain authoritative | selected increment/test leaves with profile; unselected remains one | arithmetic/log-boundary proof | raw path copy count or admission model changes |
| Synchronous acceptance contract | raw HTTP adapter -> durable receiver | HTTP must not depend sideways on a PostgreSQL implementation | inbound capability | add `internal/inboundwebhook/receiver.go` with raw delivery input, closed result/error categories, and receiver port | stdlib-only contract; PostgreSQL implements it and HTTP consumes it; bootstrap wires concrete value | file leaves with profile | port/category and HTTP mapping proof | another transport or persistence mechanism needs a new category |
| Endpoint processing contract | River worker -> decoder -> handler | no current inbound owner | inbound capability | add `internal/inboundwebhook/processor.go` | no PostgreSQL, River, HTTP, config, or secret imports | package leaves with profile | typed registry/handler proof | provider protocol or handler contract changes |
| Secret and endpoint snapshot | service config -> verification manifest; worker projection -> endpoint IDs only | configuration source policy and outbound manifest precedent | config projection plus concrete inbound adapter | add config section/manifest; service selects both leaves; worker selects endpoints and rejects supplied static secrets | service alone decodes keys; worker owns binding equality without secret bytes | all leaves/branches leave with profile | process-specific config/secret negative proof | secret source, process distribution, or rotation behavior changes |
| Verification and receipt acceptance | neutral receiver port -> PostgreSQL/River transaction | Standard Webhooks v0.0.1; `postgres.InTx`; River `InsertTx` | concrete inbound adapter | add `postgresinboundwebhook/receiver.go` implementing the neutral port | returns only capability-owned categories; depends inward on capability contract and existing adapters | leaves with profile | signature and real-PostgreSQL proof | protocol, tolerance, or commit semantics change |
| Durable processing and terminalization | River claim -> receipt -> sanitized binding boundary -> receipt outcome | jobs pack v0.44.0 logs/persists returned errors and panic values | concrete inbound adapter | add `postgresinboundwebhook/worker.go` with local panic recovery and cause-free sentinels | River owns scheduling; feature owns effect idempotency; raw feature/store errors never cross to River | leaves with profile after drain | River/process-loss plus error-canary proof | processing effect or dependency needs a new authority |
| Bounded operator evidence | receiver/worker outcomes -> existing meter provider and logs | HTTP and River telemetry providers already exist | concrete inbound adapter | add `postgresinboundwebhook/telemetry.go`; record only closed outcome/failure classes | service/worker roots supply the provider; request/internal receipt IDs own log correlation | instruments leave with profile | metric/log/trace/persisted-error redaction and cardinality proof | an operator needs a new dimension, SLO, or sink |
| Receipt schema and access | migration -> SQLC -> adapter | canonical migrations and SQLC package | PostgreSQL schema/access owners | add migration and named query source; regenerate SQLC | no hand-written SQL beside SQLC | down only when empty; profile removes pre-init | migration/constraint/concurrency proof | retention, erasure, operator query, or new access path appears |
| Service composition | config/dependencies -> receiver -> raw handler and full strict API | bootstrap owns startup, pool lifecycle, and concrete handler composition | service bootstrap | add `startup_inbound_webhooks.go` and `service_api.go`, wire after PostgreSQL | bootstrap composite supplies the inbound strict fallback and embeds/delegates ordinary feature handlers | files/blocks leave with profile | startup/readiness and interface-composition proof | readiness, dependency criticality, or generated method set changes |
| Worker composition | config -> workers runtime -> pool/meter bind -> River client | current builder errors precede pool open | jobs-worker bootstrap and binary root | return `WorkersRuntime`; keep builder before DB; profile-owned binder registers inbound after pool and before River | nil/builder errors preserve current order; River client remains lifecycle owner | `Bind` field/call/imports and inbound registration leave together | worker startup/error-order/drain proof | another process/resource lifecycle is required |
| Feature binding | configured endpoint -> one decoder/handler | no provider-specific feature in template | derived service composition | add fail-closed `cmd/jobs-worker/inbound_webhook_bindings.go` | imports feature-owned decoders/handlers only after adoption | leaves with profile | startup exact-set proof | first adopter contract reopens Specification |
| Initializer removal | selector -> retained/removed tree -> lock | `scripts/init-module.sh` profile mechanism | initializer | add selector, dependency checks, markers, exact deletions, lock field | regenerate OpenAPI/SQLC after profile resolution | unselected tree has zero residue | template-init matrix | initializer ownership changes |
| Release compatibility | migration -> exclusive new worker set -> service; rollback HTTP drain -> primary pending read -> worker rollback | River claims queue-wide before unknown-kind rejection | deployment design constraint | preserve canonical primary query, immutable-worker readback, and stop/roll-forward gates in capability docs | no external action authorized here; deployment supplies primary read and instance identity | old worker only after writable-primary zero-pending receipt | integrated deployment/readback proof | actual target cannot honor the order or primary readback |

Reuse rung: existing hardened HTTP, PostgreSQL, SQLC, River, configuration,
Problem, initializer, and Standard Webhooks owners. Strongest viable rejected
source: a new ingress framework or queue. Upgrade condition: only a measured or
provider-owned requirement that an existing owner cannot preserve.

### Files

| path | responsibilities | one present reason to exist | declarations/visibility | call-path role | lifecycle/error ownership | allowed dependencies | forbidden responsibilities |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `internal/inboundwebhook/receiver.go` | raw delivery input, acceptance result/errors, receiver port | neutral seam between HTTP and durable implementation | exported immutable input/result and narrow interface | HTTP-to-durable boundary | capability owns closed categories; caller context owns cancellation | stdlib only | HTTP, SQL, River, config, secrets, JSON decoding |
| `internal/inboundwebhook/processor.go` | verified delivery and typed binding registry | provider-neutral decoder/handler contract | exported delivery/binding constructors; internals hidden | worker-to-feature boundary | feature handler owns effects/errors | stdlib only | HTTP, SQL, River, config, secrets |
| `internal/infra/postgresinboundwebhook/manifest.go` | endpoint/key parsing and lookup | immutable trust snapshot | exported constructors/lookup; secret fields hidden | startup and verification input | adapter owns validation/redaction | stdlib | HTTP response, DB, worker lifecycle |
| `internal/infra/postgresinboundwebhook/receiver.go` | signature verification, hash, receipt/job transaction | concrete synchronous acceptance implementation | exported constructor; neutral receiver method | capability port-to-PostgreSQL/River boundary | request context and `postgres.InTx` own cancellation/commit | inbound contract, postgres, River, official library | HTTP mapping, JSON decoding, handler effects, retries |
| `internal/infra/postgresinboundwebhook/worker.go` | receipt load, recovered/sanitized decode/handle, terminal transitions | asynchronous durable path and River disclosure boundary | exported worker registration; package-private cause-free sentinels | River-to-feature boundary | River owns attempts/drain; receipt owns terminal outcome | inbound contract, postgres/SQLC, River | signature verification, HTTP, wrapped causes, provider text, custom scheduler |
| `internal/infra/postgresinboundwebhook/telemetry.go` | bounded ingress/processing counters and sanitized log classes | both service and worker use one evidence vocabulary | package-private instruments/classes | receiver/worker outcome recording | existing meter provider/logger owns export/lifecycle | OpenTelemetry metric API, slog | provider identifiers, raw errors, bodies, HTTP status duplication |
| `internal/infra/http/inbound_webhook.go` | raw generated method and Problem mapping | signature-sensitive transport exception | package-scoped handler behind one constructor | generated route-to-receiver | Harden/request writer own budgets and response commitment | openapi, problem, inbound contract | cryptography, SQL, JSON schema, business effects |
| `internal/infra/http/handlers.go` | raw standard-server override composition | combine one raw operation with the existing strict server | existing public `Handlers`; internal wrapper | router assembly | existing router error owners | openapi and feature interface only | strict fallback, receipt behavior, or provider policy |
| `internal/infra/http/router.go` | apply the inbound raw override to the generated standard server | current router is the sole strict/standard assembly point | one profile-marked assembly call | strict server -> raw override -> generated chi router | existing request/response error handlers | http/openapi package owners | inbound behavior or manual routing |
| `internal/config/inbound_webhooks_config.go` | typed config shape/default/validation | removable config section | exported immutable section type | loader-to-bootstrap input | config validation owns startup rejection | config package only | secret parsing, runtime lookup |
| `internal/config/types.go` | add inbound section to immutable `Config` | aggregate config shape owner | existing exported `Config` field only | decoded snapshot shape | config owns zero value | config package only | defaults, validation, parsing |
| `internal/config/defaults.go` | merge inbound defaults | aggregate default-map owner | existing package-private merge only | source-to-snapshot defaults | config owns default precedence | config package only | validation or behavior |
| `internal/config/validate.go` | invoke inbound validation in service snapshot | aggregate validation order owner | existing package-private call only | snapshot admission | config owns fail-fast error order | config package only | manifest parsing or startup wiring |
| `internal/config/jobs_worker_config.go` | select endpoint leaf only, validate IDs, reject supplied static secret | worker-specific config projection owner | existing package-private key/validation branches | jobs-worker config admission | config owns missing/forbidden-key failure | config package only | secret decoding, worker registration, DB access |
| `cmd/service/internal/bootstrap/startup_inbound_webhooks.go` | receiver construction | profile-scoped service composition | package-private constructor | dependencies-to-HTTP binding | bootstrap startup/cleanup | config, inbound adapter, existing pool | request behavior or worker loop |
| `cmd/service/internal/bootstrap/service_api.go` | full generated strict-interface composite and inbound fail-closed fallback | ordinary handlers must coexist without owning the raw operation | package-private concrete composite | bootstrap-to-`Handlers.API` | bootstrap owns missing delegate/fallback errors | openapi and ordinary feature HTTP handlers | raw dispatch, receipt behavior, dependency lifecycle |
| `cmd/service/internal/bootstrap/run.go` | insert inbound construction into established order | one composition root | marked local variables/wiring only | startup sequence | existing shutdown owner | bootstrap owners | adapter logic |
| `cmd/service/internal/bootstrap/runtime_request_buffer_budget.go` | count the second full-body copy when inbound is retained | startup warning otherwise under-reports admitted heap | profile-marked copy-count increment in existing arithmetic owner | config/memory limit-to-startup warning | existing no-reject policy and log owner | config/logging stdlib | request handling or a new admission control |
| `cmd/jobs-worker/internal/bootstrap/run.go` | two-stage workers runtime while preserving pre-DB builder errors | inbound needs the existing pool/meter without changing failure precedence | `WorkersRuntime` result plus profile-marked `Bind` field/call | builder -> pool -> bind -> River client | existing cleanup/pool/River teardown | config, postgres, telemetry, River | feature behavior or profile registration |
| `cmd/jobs-worker/builder_webhooks.go` | build outbound workers and profile-owned inbound runtime binder | both webhook directions share the shipped worker bundle without sharing behavior | package-private builder/runtime result | builder validation then pre-River inbound bind | bootstrap owns pool/client lifecycle | selected adapters | feature decoding or transport behavior |
| `cmd/jobs-worker/inbound_webhook_bindings.go` | adopter-owned decoder/handler composition | fail closed until one binding exists per endpoint | package-private builder hook | config/pool to binding registry | startup owns failure | feature packages after adoption | generic verification, queue mechanics |
| `cmd/jobs-worker/builder_testworker.go` | preserve test-worker build under `WorkersRuntime` result | mechanical interface compatibility | existing test-build declarations | process harness | existing test-worker lifecycle | bootstrap and River | inbound behavior or runtime binding |
| `internal/openapi/openapi.gen.go` | generated models/routes/interfaces | derived OpenAPI output | generated only | runtime route source | generator owns errors | generated dependencies | manual edits |
| `internal/infra/postgres/sqlcgen/models.go` | generated receipt row | derived schema output | generated only | repository mapping | SQLC owns | pgx/stdlib generated set | manual policy |
| `internal/infra/postgres/sqlcgen/postgres_inbound_webhooks.sql.go` | generated receipt queries | derived SQL source output | generated only | adapter-to-PostgreSQL | SQLC owns | pgx/stdlib generated set | hand-written query changes |
| `internal/inboundwebhook/processor_test.go` | processor binding mechanics | proof stays beside provider-neutral owner | package test | registry boundary | test owns no runtime lifecycle | stdlib/testing | HTTP, PostgreSQL, River fixtures |
| `internal/inboundwebhook/receiver_test.go` | closed receiver category and input-copy mechanics | proof stays beside neutral acceptance owner | package test | HTTP/durable port boundary | test owns no runtime lifecycle | stdlib/testing | concrete HTTP, PostgreSQL, River fixtures |
| `internal/infra/postgresinboundwebhook/manifest_test.go` | manifest and secret parsing mechanics | proof stays beside manifest owner | package test | startup input boundary | test owns fake non-secret keys only | package/testing | HTTP or database fixtures |
| `internal/infra/postgresinboundwebhook/receiver_test.go` | signature/vector and receiver classification mechanics | proof stays beside synchronous adapter | package test | verification-to-transaction seam with substituted store | test owns deterministic clock | package/testing | real PostgreSQL concurrency claims |
| `internal/infra/postgresinboundwebhook/worker_test.go` | worker outcome, panic recovery, and cause-free sentinel mechanics | proof stays beside worker owner | package test | River work method with substituted store/binding | test owns deterministic attempt/error inputs | package, River test support | real process-loss or external sink claims |
| `internal/infra/postgresinboundwebhook/telemetry_test.go` | bounded attributes and secret/provider-data absence | proof stays beside instrument owner | package test | receiver/worker metric emission | test meter owns capture | package, OTel test support | business outcomes |
| `internal/infra/http/inbound_webhook_test.go` | raw-byte preservation and typed Problem mapping mechanics | proof stays beside raw transport owner | package test | generated raw method-to-receiver | HTTP test request owns body/header fixtures | package/testing | PostgreSQL or worker behavior |
| `internal/infra/http/openapi_contract_test.go` | add the operation's reachable response/security declarations | existing hand-maintained OpenAPI/runtime parity owner | existing package test cases | contract-to-runtime catalog | existing test harness | openapi/http package | receipt or provider mechanics |
| `internal/infra/http/router_contract_test.go` | prove raw override and ordinary strict API coexist under one generated router | existing router-assembly proof owner | existing package test case | strict/raw composition | HTTP harness owns request lifecycle | http package | receiver persistence or worker behavior |
| `internal/config/inbound_webhooks_config_test.go` | inbound section decode/validation | proof stays beside new config leaf | package test | source-to-snapshot admission | config test env owns fake values | config test support | manifest cryptography |
| `internal/config/snapshot_contract_test.go` | add every inbound leaf to snapshot sentinels | reflection-enforced aggregate mapping owner | existing package test maps | full snapshot coverage | existing test harness | config package | feature validation semantics |
| `internal/config/secret_policy_test.go` | prove secret material is env-only and absent from files/errors | existing secret-source proof owner | existing package test cases | source admission boundary | secret policy owns rejection | config package | manifest behavior |
| `internal/config/jobs_worker_config_test.go` | endpoint-only worker snapshot plus static-secret rejection | existing worker-config proof owner | existing package test cases | jobs-worker config admission | config owns rejection | config package | secret parsing or River registration |
| `cmd/service/internal/bootstrap/startup_inbound_webhooks_test.go` | startup manifest/receiver/service-API composition | proof stays beside new startup stage | package test | dependencies-to-handler construction | bootstrap test owns substitutions | bootstrap/package tests | HTTP scenarios or real DB claims |
| `cmd/service/internal/bootstrap/run_test.go` | preserve startup and shutdown order with inbound stage | existing lifecycle proof owner | existing package test events | process composition | existing harness | bootstrap package | adapter semantics |
| `cmd/service/internal/bootstrap/runtime_limits_test.go` | selected two-copy and unselected one-copy budget arithmetic | existing runtime-limit proof owner | existing package test cases | config-to-warning calculation | existing log capture | bootstrap package | HTTP byte identity or persistence |
| `cmd/jobs-worker/builder_webhooks_test.go` | selected inbound/outbound worker registration and exact binding set | existing shipped-builder proof owner | existing package test cases | builder-to-workers bundle | builder owns startup errors | package/River tests | worker execution outcomes |
| `cmd/jobs-worker/internal/bootstrap/run_test.go` | builder-error-before-pool, binder-before-River, and unchanged nil-builder/teardown behavior | existing jobs process proof owner | existing package test cases | process startup/drain | bootstrap harness owns resources | bootstrap/package tests | inbound processing semantics |
| `test/postgres_inbound_webhook_integration_test.go` | real PostgreSQL receipt/job atomicity and concurrent identity invariant | external integration package owns real DB proof | integration-tagged external test | HTTP-independent adapter-to-PostgreSQL/River seam | pgtest owns database cleanup | public package APIs and pgtest | process deployment claims |
| `test/inbound_webhook_process_integration_test.go` | accepted-job survival, restart, and provider-text canary through River logs/persisted errors/traces | external process package owns kill/restart and real sink proof | integration-tagged external test | service/worker durable and disclosure boundary | process harness owns logs, trace capture, database, cleanup | built binaries, PostgreSQL/telemetry harness | provider or deployment registration |

These rows fix proof placement only. Test Design still owns the final oracle,
scenario matrix, proving levels, determinism, and command selection.

## Reopen conditions

- Reopen Specification for a non-Standard-Webhooks sender, another response or
  replay contract, legal retention/erasure, provider ordering/reconciliation,
  per-sender fairness, or payload classification incompatible with retention.
- Reopen System Design if the pinned generator/validator cannot preserve raw
  bytes, receipt and River job cannot share one PostgreSQL transaction, or the
  target cannot preserve migration -> worker -> service order.
- Reopen Go Ownership if a real adopter cannot place exactly one typed binding
  without an import cycle or if another raw OpenAPI operation creates a shared
  composition requirement.
- Reopen the external deployment owner for ingress/TLS, secret delivery and
  rotation timing, provider registration, capacity, or rollout authority.
