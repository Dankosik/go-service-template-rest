# Selected services deliver immutable business events durably to authorized webhook destinations

status: ready
Problem: The template has no supported boundary for accepting one business
event for one or more subscriber destinations and delivering it recoverably
without making feature code own destination snapshots, signatures, outbound
network safety, retries, duplicate and ambiguity handling, response evidence,
operator recovery, or worker lifecycle. Adopters would otherwise either build
those mechanics repeatedly or treat the current broker-publication outbox as a
per-destination HTTP ledger even though it does not own that behavior.

## Scope and non-goals

In scope is an optional, template-init-selected capability pack,
`WEBHOOKS=durable`, with a reusable delivery-engine boundary. A business-event
owner supplies immutable event identity and bytes. A subscriber-management
owner supplies an authorized destination snapshot. The delivery engine durably
accepts one delivery occurrence per selected destination generation and owns
bounded signed HTTP attempts, outcome classification, retry scheduling,
destination isolation, terminal and unknown state, attempt and operator audit,
telemetry, retention enforcement, redrive mechanics, and independently
operable lifecycle.

The delivery guarantee is at least once. A receiver can observe the same
delivery more than once and can observe deliveries out of event order. A
receiver-visible delivery ID makes deduplication possible; it does not make an
arbitrary receiver idempotent or make a remote side effect exactly once.

The three authorities remain separate:

| Authority | Owns | Does not own |
| --- | --- | --- |
| Business-event semantics | Event type and meaning, occurrence identity, exact payload bytes, business schema/version, content type, and any business partition/sequence | Subscriber authorization or lifecycle, network attempts, signatures, retries, or delivery truth |
| Subscriber management | Subscriber ownership and scope, event selection, payload-version preference, destination identity and generation, endpoint authorization and verification, administrative state, signing-key provisioning and rotation policy | Business truth, delivery attempts, HTTP outcome, or a claim that a receiver processed an event |
| Reusable delivery engine | Durable event and destination snapshots, delivery and attempt identity, signed bounded HTTP execution, scheduling, outcome and ambiguity state, isolation, audit, retention enforcement, redrive, telemetry, and lifecycle | Event meaning, who may subscribe, subscriber-facing CRUD or portal behavior, receiver business processing, or exactly-once effects |

Out of scope, with the condition that reopens each boundary:

- **Subscriber administration.** No public registration, verification,
  update, delete, list, portal, or secret-management API is created. A named
  product and security owner must specify those client-visible contracts,
  authorization, disclosure, and lifecycle rules before one is exposed.
- **Business-event catalog and payload design.** The pack defines the
  immutable handoff required from a feature; it does not select events, fields,
  schemas, compatibility policy, or subscriber-specific transformations. The
  first event type requires its feature-owned Specification and compatibility
  authority.
- **Receiver implementation.** The pack documents the delivery ID, signature,
  timestamp, duplicate, and ordering contract a receiver must consume. It does
  not ship receiver deduplication or prove receiver business processing.
- **Implementation-family selection.** A distinct local delivery store, a
  deliberate generalized durable owner, a self-hosted system, and a managed
  system remain candidate families for System / Integration Design. The
  PostgreSQL-native family is the leading Research fit, not accepted
  architecture.
- **Private-network destinations or extra transport trust.** Plain HTTP,
  private or special-use addresses, ports other than 443, custom roots, mutual
  TLS, proxies, private resolvers, and service-mesh-only routes require a
  separate trust-boundary Specification. They are not dormant configuration
  flags in this pack.
- **Ordered delivery.** The baseline deliberately offers no FIFO or partition
  ordering mode. A business owner requiring order must specify partition,
  sequence, head-of-line, poison, replay, and receiver-processing semantics
  before this rule is reopened.
- **Automatic destination disable policy.** The engine supports the observable
  paused/disabled behavior below, but no failure threshold, window, cooldown,
  or automatic re-enable is enabled without adopter SRE/product policy.
- **Public operator API.** Authorized locate, inspect, suspend, redrive, and
  risk-closure behavior is specified, but its transport and exposure remain
  later design decisions and are not public by implication.
- **Technical representation and placement.** Package names, interfaces,
  schemas, SQL, binaries, queue topology, connection-pool layout, concrete
  storage framing, configuration keys, metric names, and vendor/library choices
  belong to Technical Design or later phases. The receiver-visible HTTP and
  signature fragment is fixed by W3-W4 and is not a design choice.
- **System Design, Go ownership, Test Design, Planning, migrations, rollout,
  and implementation.** No architecture, package placement, scenario matrix,
  command, task ledger, schema, or code is selected here.

## Behavior and contract delta

### Terms and authoritative facts

The identities below remain distinct even if a later realization derives one
from another. Opaque identity values are scoped to the authorized subscriber
owner; guessing one must not reveal or mutate another owner's state.

| Identity | Required meaning and lifetime | Conflict rule |
| --- | --- | --- |
| Business-event ID | One domain occurrence, stable across all selected destinations and every retry or redrive | Reuse with different event type, schema/version, content type, or exact payload is an integrity conflict |
| Destination ID and generation | One authorized endpoint configuration snapshot; URL text is mutable configuration, not identity | A configuration change creates a new generation and never redirects an already accepted delivery |
| Fan-out snapshot ID | The complete subscriber-selection result for one accepted business event and policy revision | A repeated acceptance cannot add or remove members silently; a changed set is a conflict or a new explicit event-selection action |
| Delivery ID | One business event directed to one destination generation; stable through automated retry and manual redrive | A second logical delivery for the same accepted event/destination generation is a duplicate-acceptance defect |
| Attempt ID and timestamp | One bounded network execution; each repeat has a fresh identity and signing time | One recorded attempt may cause at most one network send; any repeat is a new durable attempt |
| Operator-action ID | One authenticated mutation or risk decision, replay-safe within the subscriber scope | Repeating an identical action returns its original disposition; reuse for different intent is rejected |
| Signing-key generation | One secret generation selected by the subscriber secret authority; only its opaque reference is durable delivery evidence | Secret bytes, raw signatures, and a retired generation cannot be substituted or exposed as identity |

The feature's business state remains authoritative for whether the event is
true. The subscriber authority is authoritative for who owns a destination,
which events and payload versions it accepts, its generation, and whether
sending is currently allowed. The delivery engine's durable state is
authoritative for accepted fan-out, delivery, attempt, scheduling, terminal,
ambiguity, and operator-action facts.

For an accepted delivery, the engine's stored event and destination snapshots
win over later business serialization or endpoint edits. A later security or
administrative disable may prevent another send but cannot rewrite the URL,
payload, identity, or evidence of an earlier attempt. A receiver-owned receipt
or query is the only possible external authority for receiver processing; none
is part of the baseline.

Authoritative absence means a successful read from the current delivery writer
under the same owner scope. Absence from a replica, cache, stale observation,
or unavailable store is `unknown`, not proof of non-acceptance. A retained
`privacy_deleted` tombstone is authoritative presence with content unavailable,
not absence. A terminal local delivery state is final only for automatic
scheduling. It never upgrades HTTP acceptance into receiver business success.

### W1 — Profile selection is independent, complete, and fail-closed

Actor: a developer initializing the template.

Rule: `WEBHOOKS=none` is the default and removes the whole capability.
`WEBHOOKS=durable` retains exactly one delivery-engine realization and every
dependency that realization requires. The selector is independent of `OUTBOX`,
`MESSAGING`, `JOBS`, and `OUTBOUND_HTTP`; selecting any sibling neither selects
webhooks nor merges its identity, retry, readiness, audit, or recovery policy.

| Selection | Observable outcome |
| --- | --- |
| Webhooks not selected | No webhook runtime, worker, schema/migration, dependency, config, secret reference, documentation, CI, generated authority, or test residue remains; existing service behavior is unchanged |
| `WEBHOOKS=durable` with its selected family's complete prerequisites | The event-handoff and independently operable delivery boundaries are retained |
| Unknown value, incomplete dependency combination, or more than one engine family | Initialization is rejected before mutating the destination checkout |
| Webhooks combined with retained sibling profiles | Every pack remains independently buildable, configurable, observable, and removable without source edits |

The selected profile must declare whether it owns local database/migrations,
an external service handoff, a worker/executable, outbound HTTP, secret
resolution, and telemetry. It may reuse accepted repository primitives, but it
cannot make selecting broker messaging a hidden prerequisite for HTTP delivery.

Falsifier: initialize omitted, webhook-only, invalid, sibling-pair, and
all-retained combinations twice. Any invalid mutation, removed-profile
reference, undeclared prerequisite, or shared behavioral switch fails the rule.

### W2 — Durable acceptance preserves one event, one fan-out snapshot, and one delivery per destination generation

Actor: feature code after it has decided that an event is part of an accepted
business outcome.

Preconditions: the caller supplies a stable business-event ID, event type,
business schema/version, content type, exact payload bytes, and an authorized
fan-out snapshot containing owner-scoped destination generations. Every member
has passed W4-W5 and W16 admission. The caller also supplies one replay-safe
acceptance identity.

Rule: delivery acceptance must have no unprotected gap after business truth.
A realization may share an atomic authority with the business mutation or may
use a durable idempotent handoff plus reconciliation, but it must expose the
same behavior:

| Acceptance outcome | Meaning and allowed caller action |
| --- | --- |
| Accepted new | The exact event and complete fan-out snapshot are durable. Every member already has, or will be reconciled to, exactly one stable delivery ID without caller resubmission |
| Accepted existing | The same acceptance identity names byte-for-byte identical event intent and the same fan-out snapshot; return the original identities and create nothing |
| Conflict | The acceptance identity, event ID, fan-out snapshot, or derived delivery identity already names different immutable intent; change nothing and report integrity failure |
| Privacy deleted | A retained tombstone matches the acceptance, event, fan-out, or delivery identity. Return the tombstoned identities and deletion disposition without comparing absent content, creating work, or treating it as accepted-existing |
| Rejected | Admission or durable handoff definitely failed before acceptance; create no partially authoritative fan-out |
| Unknown | The engine cannot prove accepted or rejected. The caller may read back or retry only with the same acceptance identity and immutable intent; it must not mint new event or delivery identities |

If physical delivery-row expansion occurs after event acceptance, the durable
event plus fan-out snapshot is the reconciliation authority. Partial expansion
is visible unfinished work, not a successful partial fan-out. Reconciliation
is idempotent and cannot select a newer subscriber set.

No database transaction or durable claim remains open across network I/O or a
retry delay.

Falsifier: interrupt acceptance before and after each durable boundary and
repeat it with identical and conflicting intent. Any lost snapshot member,
duplicate logical delivery, silent partial success, or need for the feature to
recompute current subscribers fails the rule.

### W3 — Payload, version, envelope, and destination evidence are immutable

Actor: the delivery engine accepting and later attempting a delivery.

Rule: acceptance freezes the exact request payload bytes, content type,
business event type and schema/version, delivery-envelope version, signature
profile version, business-event ID, fan-out snapshot, destination ID and
generation, and URL snapshot. Every automated retry and manual redrive sends
those same payload bytes and semantic versions. A deployment, current feature
serializer, destination update, or version-preference change cannot regenerate
or transform an accepted delivery.

The exact request body is the feature-owned, versioned business-event envelope;
before admission its schema must carry or unambiguously bind the business-event
ID, event type, and business schema/version. The reusable delivery envelope
adds only the three HTTP fields fixed by W4: stable delivery identity, attempt
timestamp, and signature set. It does not add unsigned mutable event metadata.
Request bodies are sent without content transformation or compression.

A payload-version migration applies only to fan-out snapshots accepted after
the subscriber authority activates the new preference. A live old version
remains readable and deliverable through its full active and redrive horizon.
Unsupported or undecodable accepted data is retained and made operator-visible;
it is never coerced through the current serializer or discarded as poison.

Falsifier: accept a delivery, deploy a changed serializer and destination
preference, then retry and redrive. Any changed payload byte, content type,
version, destination generation, or delivery ID fails the rule.

### W4 — Every network attempt is timestamped, signed, and rotation-safe

Actor: the delivery engine and the subscriber secret authority.

Rule: the baseline `v1` signature profile uses HMAC-SHA256 with 32 to 64 bytes
of high-entropy per-destination key material. Subscriber management provisions
that key to the receiver as `whsec_` followed by the padded standard-Base64
encoding of the raw key bytes; the prefix and encoding are presentation, not
HMAC input. HTTP field names are
case-insensitive; their canonical spellings and values are:

| Field | Exact `v1` value |
| --- | --- |
| `Webhook-Id` | The stable delivery ID as its admitted printable ASCII representation, with no `.`, control, or whitespace bytes |
| `Webhook-Timestamp` | The attempt instant as canonical unsigned base-10 Unix seconds, with no sign, fraction, or leading zero except the single value `0` |
| `Webhook-Signature` | One or two space-separated entries `v1,<signature>` ordered newest key generation first |

For each entry, `<signature>` is the padded standard-Base64 encoding of the
32-byte HMAC-SHA256 digest over these exact bytes, with no newline or Unicode
normalization:

```text
<Webhook-Id bytes> "." <Webhook-Timestamp bytes> "." <exact request-body bytes>
```

The two literal separators are ASCII `0x2e`. A verifier reconstructs the input
from the field values and raw body, Base64-decodes each `v1` value, compares the
digest in constant time, and accepts the signature set when at least one entry
validates under a key generation that the subscriber authority currently
accepts for that timestamp. Unknown versions and malformed entries do not
validate; one malformed or unknown entry does not invalidate another valid
`v1` entry. Timestamp replay tolerance is receiver policy, but it is checked
before business processing and never replaces delivery-ID deduplication. The
per-destination secret provides destination binding.

Each attempt obtains the currently active key generation for the authorized
destination. During an explicitly active rotation overlap it emits signatures
for the new generation and its immediate predecessor over the same signed
input, in that order. Outside overlap it emits exactly one signature for the
active generation. After retirement it emits no predecessor signature.
Delivery ID, payload, destination generation, and signature profile stay fixed
while the attempt timestamp, signature value, and signing-key generation
evidence may change.

Secret bytes are resolved only inside the signing boundary and are never
stored in event, delivery, attempt, operator, config-file, log, trace, metric,
or response evidence. Audit records contain only key-generation references and
a non-replayable digest or equivalent evidence of the emitted signature set.
Raw signatures are not retained or logged.

If no valid active key can be resolved, the engine emits no unsigned request.
A transient secret-authority failure is bounded retryable local work; a
missing, revoked, or invalid generation suspends the delivery until the
subscriber/security owner repairs the authority or the delivery becomes
ineligible under retention. Retirement cannot delete a key reference still
needed to interpret retained attempt evidence.

Receiver replay-window validation and receiver deduplication are separate.
Every retry has a fresh valid timestamp while retaining the delivery ID; the
receiver's dedup horizon must cover the declared delivery plus redrive horizon,
not only its signature clock tolerance.

Falsifier: verify exact raw bytes with current, overlapping, retired, wrong,
missing, and cross-destination secrets. Any unsigned send, single-key gap
during declared overlap, old-key use after retirement, signature over
regenerated bytes, or secret/signature disclosure fails the rule.

### W5 — Destination admission and every new connection fail closed against SSRF and rebinding

Actor: subscriber management at registration and the delivery engine before
each attempt and connection.

Rule: an admitted destination snapshot contains an authorized owner scope,
destination ID and generation, endpoint-ownership verification receipt, exact
event/version selection, signing policy and key authority, and complete bounded
delivery policy. Registration proves authorization; the engine independently
enforces transport safety at use time.

The baseline admits only an absolute `https` URL with a non-empty host, port
443 (explicit or implicit), an optional path, and no user information, query,
fragment, embedded credential, or alternate authority. Standard certificate
and hostname verification is mandatory. Endpoint changes create a new
destination generation.

Before every attempt and every new connection, the engine resolves and checks
all resulting IPv4 and IPv6 addresses. If any answer is loopback, private,
link-local, multicast, unspecified, metadata, documentation, benchmarking, or
otherwise non-public/special-use under the current IANA registries, the entire
answer set is denied. The actual dial address is checked again immediately
before connection. CNAMEs, mixed answers, DNS changes between registration and
send, and pooled-connection replacement cannot bypass the rule.

Environment proxies are disabled. A resolver, dialer, proxy, or transport that
cannot preserve the same final-address check is not admitted. Deployment-owned
network egress policy independently denies internal and metadata destinations;
application validation is not its substitute.

Any 3xx response is recorded without following it and is classified under W7.
No second request is sent, and no payload, signature, authorization, or
correlation header is transferred to the redirect target. Endpoint migration
requires an audited new destination generation.

No caller-supplied authorization, cookie, forwarding, request-ID, trace,
baggage, or idempotency header is forwarded. The attempt emits only the closed
webhook envelope, content headers, a bounded constant user agent if selected by
Design, and its signatures. Local spans may link to event origin, but the
baseline emits no remote trace or baggage context to an arbitrary receiver.

Falsifier: exercise A/AAAA and mixed public/special answers, rebinding between
registration/resolution/dial, redirects, proxy and custom-resolver paths, TLS
name/certificate failure, and cross-owner ID guessing. Any forbidden dial,
redirected send, leaked header, or cross-owner observation fails the rule.

### W6 — One attempt is one bounded POST with bounded non-content response evidence

Actor: a worker executing one due attempt.

Preconditions: it holds current authority for one delivery attempt, the
destination is send-enabled, the policy snapshot is valid, and the signing key
is available. The feature, caller, and durable store hold no transaction open
for the network call.

Rule: the worker sends one POST of the exact payload. Transport-level automatic
replay is disabled; the dedicated delivery ID is not emitted as a generic
`Idempotency-Key`. One durable attempt can therefore cause at most one network
send. A repeat always requires a new durable attempt ID and timestamp.

Every policy declares positive finite bounds for total attempt time, response
header time and bytes, response body bytes, connections per destination,
global in-flight attempts, and worker drain. DNS, connect, TLS, request write,
response headers, and response reading all fit inside the total attempt budget;
no stage or cleanup can outlive it without producing a bounded failure signal.
The response requests identity encoding; transparent decompression is disabled.
The body is read or discarded only up to its raw-byte cap and is then closed.

The durable protected attempt evidence contains the attempt and delivery IDs,
destination generation, attempt time, signature profile and key-generation
references, request payload digest/size, actual admitted authority/address
evidence, timing, whether any request bytes may have been sent, HTTP status when
known, bounded header/body byte counts, normalized `Retry-After` disposition,
semantic network/outcome class, and finalization time. It stores no arbitrary
response header, response body or excerpt, URL, payload copy, secret, or raw
signature. The immutable payload and URL remain in their protected owning
snapshots, not duplicated into attempt audit.

A 2xx can be accepted as soon as its status is safely observed; consuming a
receiver body is not part of success. A response-body overflow cannot disclose
content or turn a known 2xx into failure. A status/header/protocol failure after
the request might have reached the receiver is ambiguous under W7-W9.

Falsifier: stall each network stage, overflow headers and bodies, return
compressed data, reset reused connections, and cancel during drain. Any
unbounded resource, hidden second send, retained receiver content, or attempt
record claiming stronger send evidence than observed fails the rule.

### W7 — Outcomes and retry eligibility use one closed classification

The engine classifies the strongest fact it observed; it does not infer
receiver business processing from a status code.

| Observation | Delivery outcome class | Automatic action |
| --- | --- | --- |
| HTTP 200-299 safely observed | `http_accepted` | Finalize accepted; no automatic retry |
| Failure proven before any request byte could be sent, including a transient DNS/connect/capacity failure | `definitely_not_sent_retryable` | Schedule under W8 while budget remains |
| HTTP 408, 425, 429, or 500-599 except 501 and 505 | `retryable_http_ambiguous` | Record that receiver effect is unknown; schedule under W8 while budget remains |
| Timeout, cancellation, connection loss, write/read/protocol/header failure, or process loss after any request byte may have been sent | `transport_ambiguous` | Preserve ambiguity and schedule under W8 while budget remains |
| HTTP 100-399 outside 200-299, HTTP 400-499 outside 408/425/429, or HTTP 501/505 | `http_rejected` | Do not retry automatically; 3xx is never followed |
| URL/address/TLS/proxy/resolver policy denial, invalid immutable envelope, retired key with no active replacement, or another deterministic local policy violation | `locally_denied` | Send nothing; suspend until the same generation becomes admissible or reaches terminal retention policy |

A destination-specific receiver contract may narrow retry eligibility only in
that destination generation's approved policy. It cannot turn a local security
denial or redirect into a send, turn non-2xx into HTTP acceptance, or erase
ambiguity after bytes may have been sent. Widening the portable status table or
claiming a stronger outcome reopens Specification.

Any previous ambiguous attempt remains material audit evidence. A later 2xx
may finalize `http_accepted`, but a later rejection or budget exhaustion cannot
prove that the receiver did not accept an earlier attempt.

Falsifier: vary response codes, failures before/after write, process loss after
remote effect and before finalization, and a later contradictory response. Any
exactly-once claim, erased ambiguity, retried redirect/security denial, or
status-only business-success inference fails the rule.

### W8 — Retry scheduling, `Retry-After`, and exhaustion are durable and bounded

Actor: the delivery scheduler after a retry-eligible outcome.

Rule: every admitted destination policy declares finite maximum attempts,
maximum delivery age, backoff base and cap, `Retry-After` cap, and
per-destination/global concurrency. Missing, zero, contradictory, or unbounded
values reject destination admission; no library, provider, or outbox default
silently becomes webhook policy.

The authoritative engine records `accepted_at` when the event and fan-out
snapshot first become durably accepted and fixes each delivery's automatic
deadline as `accepted_at + maximum delivery age`. Delayed delivery-row
materialization, first attempt, restart, failover, clock correction, retry,
automatic pause, administrative disable, secret outage, or local policy denial
cannot move that deadline later. Due times and deadline comparisons use the
selected durable authority's UTC clock rather than a worker-local wall clock;
the recorded absolute deadline is the cross-process truth. An observed clock
rollback cannot decrease already recorded elapsed age or extend a deadline.

Retries are future durable work with decorrelated jitter. A worker releases its
claim and resources instead of sleeping through backoff. Restart, failover, or
worker absence does not erase the due time or reset attempts/age. The next due
time cannot exceed the delivery-age or retention boundary.

For a retry-eligible response, `Retry-After` is considered only as a
non-negative delay-seconds value or valid HTTP date. A valid response `Date` is
the date reference when present; otherwise the attempt's local wall clock is
used. Malformed or past values are ignored. A valid future value is capped by
the declared `Retry-After` cap, combined as the later of the capped hint and
the local jittered backoff, and then bounded by remaining delivery age. The
attempt evidence records parsed, ignored, capped, and age-exhausting outcomes.
No hint can create unbounded retention or bypass local concurrency and age
budgets.

When no ambiguity has ever occurred and every attempt was proven not sent,
budget exhaustion becomes `attempts_exhausted`. When any attempt may have
reached the receiver, exhaustion becomes `outcome_unknown`. A non-retryable
HTTP rejection with no earlier ambiguity becomes `http_rejected`. These are
terminal for automatic scheduling and eligible for operator policy under W13.
Age continues to elapse while a delivery is paused, disabled, locally denied,
or waiting for a secret. Reaching the fixed automatic deadline applies the same
definite-versus-ambiguous terminal rule even if no attempt can currently start;
re-enable after that point requires W13 redrive.

Falsifier: exercise delay seconds, dates with and without `Date`, malformed,
past, oversized, and beyond-age hints; restart while scheduled; and exhaust
definite versus ambiguous attempts. Any sleeping claim, reset budget,
uncapped hint, retry after age, or collapsed terminal meaning fails the rule.

### W9 — Duplicate and ambiguous outcomes remain explicit through finalization and recovery

Rule: the stable delivery ID is the receiver deduplication key on every
attempt. The engine openly documents that duplicates can arise from retry,
transport ambiguity, process loss after remote effect, finalization failure,
manual redrive, or receiver response loss. It never claims exactly once.

The local finalization transaction cannot be atomic with the remote effect. If
a worker observes a response but cannot durably finalize it, recovery treats
the attempt according to the strongest durable evidence; missing response
evidence after a possible send is ambiguous. A stale worker cannot finalize or
overwrite a newer attempt. Divergence between current delivery state and
append-only attempt evidence is quarantined for reconciliation rather than
silently repaired toward success or failure.

`http_accepted` means only that one 2xx was durably observed for the delivery.
`http_rejected` and `attempts_exhausted` mean only that the sender observed no
HTTP acceptance and no ambiguous send. `outcome_unknown` means receiver
acceptance cannot be determined. None states that receiver business effects
were applied, absent, unique, or durable.

The cumulative delivery summary follows strongest historical evidence across
the automatic cycle and every redrive. Once any durable 2xx exists, the summary
remains `http_accepted` even if a later authorized redrive fails or is
ambiguous. Without a 2xx, once any historical attempt may have been sent, the
summary remains `outcome_unknown`; a later definitely-not-sent attempt or cycle
cannot downgrade it to `attempts_exhausted` or `http_rejected`. Each automatic
or redrive cycle also retains its own disposition, so an operator can see the
latest recovery result without overwriting the cumulative fact.

Falsifier: crash before send, after effect/before response, after response/
before finalization, and after finalization/before telemetry; overlap stale and
new workers. Any lost durable acceptance, stale overwrite, unmarked possible
duplicate, or remote-business truth inferred from local finality fails the
rule.

### W10 — Fan-out snapshots are complete, destination progress is independent, and order is unspecified

Actor: event acceptance, scheduler, and subscriber management.

Rule: one event acceptance freezes the complete selected destination-generation
set. A subscriber added afterward receives no implicit historical delivery; an
endpoint update creates a new generation and does not retarget existing work;
disable/delete affects attempts as W11 defines but does not erase accepted
evidence. Every fan-out member has an independent delivery ID, schedule,
attempts, terminal state, redrive, and retention eligibility.

No global, event, or per-destination FIFO is promised. Workers may attempt
later events before earlier events, retries may overtake new work, and receiver
processing may differ from sender completion order. Business publication order
alone creates no ordering contract.

The scheduler enforces finite global and per-destination concurrency and makes
progress across ready destinations. A slow, poisoned, high-volume, suspended,
or repeatedly retrying destination cannot consume all claims, connections, or
worker slots while another eligible destination has work. The contract claims
isolation and eventual selection under admitted load, not a latency or
throughput SLO without adopter measurements.

Falsifier: fan out one event to N destinations, update/delete one generation,
stall and flood one destination, and retry older work beside new work. Any
retargeting, cross-destination terminal coupling, starvation under the declared
capacity envelope, or implied order fails the rule.

### W11 — Destination pause and disable stop new sends without rewriting accepted truth

The engine observes four administrative/health dispositions independent of
individual delivery state:

| Disposition | New fan-out materialization | Existing scheduled/ready work | In-flight work | Recovery |
| --- | --- | --- | --- | --- |
| Active | Allowed by current subscriber snapshot | Eligible | May start/finish | Normal |
| Automatically paused | Still durably materialized so accepted events are not silently lost | No new attempt starts; backlog remains visible | May finish; cancellation follows the bounded drain rule and may be ambiguous | Explicit or policy-owned re-enable resumes nonterminal work; terminal work needs W13 redrive |
| Administratively disabled | Excluded from fan-out snapshots accepted after the authoritative disable revision | No new attempt starts | May finish if already sent; no action can unsend it | Authorized re-enable affects only then-current nonterminal work; terminal work needs W13 redrive |
| Deleted/retired | Excluded from new snapshots; identity remains for retention/audit | No new attempt starts | Same bounded in-flight rule | Cannot resume until subscriber management explicitly provisions an authorized new generation; old deliveries are not silently retargeted |

Changing disposition is owner-scoped, authenticated, replay-safe, and audited.
A stop request first prevents new claims; an in-flight request that may have
sent bytes is allowed to record its observed outcome or becomes ambiguous on
cancellation. Disable never rewrites an accepted delivery as rejected,
delivered, or deleted.

Automatic pause is off by default. Enabling it requires a named SRE/product
owner to provide the eligible failure classes, counting window, threshold,
minimum traffic, pause duration or manual-only recovery, effect on retention,
and alert. Local infrastructure and policy-denial failures do not consume a
receiver-health budget unless that policy explicitly assigns them.

Falsifier: race each disposition transition with acceptance, claim, send,
retry, and redrive. Any new send after the claim barrier, lost fan-out member,
automatic threshold inherited from an implementation, or endpoint update that
mutates an old generation fails the rule.

### W12 — Retention has separate horizons and never deletes live recovery authority

Actor: the adopter's data/privacy, business recovery, subscriber, and SRE
owners.

Rule: before a concrete destination is admitted, policy declares finite
horizons for immutable event payloads, active/retryable deliveries, terminal
delivery summaries, attempt evidence, operator actions, destination generations,
and redrive eligibility. The subscriber integration owner separately publishes
the minimum receiver-dedup horizon. One TTL cannot stand in for all of them.

The following precedence holds:

1. Payload and destination evidence required by an active, scheduled,
   suspended, in-flight, ambiguous, or redrive-eligible delivery is retained.
2. Delivery, attempt, operator, destination-generation, and key-reference
   evidence needed to interpret a retained fact is retained at least as long as
   that fact.
3. A delivery cannot remain redrive-eligible after its payload, destination
   generation, signing authority, or required audit evidence expires.
4. Cleanup is bounded and cannot claim deletion until authoritative deletion
   succeeds. Unknown cleanup outcome is retried idempotently.
5. A legal/privacy deletion that must outrank recovery makes affected delivery
   work non-sendable and non-redrivable, records the tombstone defined below,
   and never fabricates a delivery outcome.

Privacy deletion is an explicit terminal content-lifecycle transition named
`privacy_deleted`; it is orthogonal to the last delivery outcome. Before
removing content, the engine durably records the minimum owner-scoped
idempotency tombstone the data/privacy authority permits: acceptance identity,
business-event ID, fan-out snapshot ID, delivery IDs and destination-generation
identities, last semantic delivery/ambiguity class, deletion authority and
time, and no payload, URL, secret, signature, response content, or reversible
content digest. The deletion is not complete until the tombstone is durable and
the governed content is authoritatively absent.

An authoritative read of any retained identity returns `privacy_deleted`, the
non-content identities the caller is authorized to see, and the prior semantic
class when policy permits it. Repeating the old acceptance identity or reusing
any tombstoned event/fan-out/delivery identity returns the `Privacy deleted`
outcome in W2. It never returns `Accepted existing`, attempts byte comparison,
creates a new delivery, or resurrects the content. A genuinely new event must
use identities outside the retained tombstone set and pass ordinary admission.
Diagnostics expose deletion disposition without content.

If law or policy forbids retaining even those stable identities, safe
idempotent reuse cannot be proved. The owner scope and its current identity
namespace are then retired before full erasure; no new or repeated acceptance
is admitted in that namespace. Subscriber management must provision a new
non-colliding namespace/generation before future work. The data/privacy owner
must choose between the minimal tombstone and namespace retirement before
deletion is enabled; implementation cannot silently treat full erasure as
authoritative absence.

Secret bytes remain under the secret authority's lifecycle, not payload
retention. Key-generation references remain long enough to interpret attempts.
No production retention, storage, privacy, recovery, or receiver-dedup claim is
made until the named owners supply the actual horizons and payload class.

Falsifier: run cleanup against active, suspended, unknown, terminal,
redrive-eligible, privacy-deleted, and already-cleaned records. Any orphaned
fact, live payload deletion, recovered secret, indefinite undeclared horizon,
or redrive after required evidence deletion fails the rule.

### W13 — Redrive and reconciliation preserve identity, serialize attempts, and never invent receiver truth

Actor: an authenticated operator acting within one subscriber owner scope.

Rule: a redrive is one replay-safe operator action on the original delivery.
It preserves delivery ID, event ID, exact payload and versions, destination
generation, and signature profile, and creates a fresh attempt ID, timestamp,
and current valid key-generation evidence. It never changes the target to a
new destination generation.

Redrive is admitted only when the delivery is terminal or suspended, no
automated attempt or other redrive is active, required evidence remains, the
same destination generation is currently authorized and transport-admissible,
and an owner-approved finite redrive attempt/age budget is supplied. Admission
atomically prevents a scheduled automated retry from racing the action.
Repeating the same operator-action ID returns the original result.

A successful redrive admission records `redrive_accepted_at` from the same
durable UTC clock and fixes that recovery cycle's deadline as
`redrive_accepted_at + redrive maximum age`, additionally capped by the
delivery's redrive-eligibility and evidence-retention horizon. Repeating the
operator-action ID does not reset it. Pause, disable, restart, secret outage,
or local denial consumes redrive age and cannot move the deadline. On expiry,
the redrive cycle records `attempts_exhausted` when every attempt in that cycle
is proven not sent, or `outcome_unknown` when any attempt in that cycle may
have reached the receiver. W9 then derives the separate cumulative delivery
summary: any historical 2xx preserves `http_accepted`; otherwise any historical
possible send preserves `outcome_unknown`; only a delivery whose complete
automatic and redrive history is definitely not sent may have cumulative
`attempts_exhausted`. Cycle-local status never erases an earlier response or
ambiguity.

By default, `http_rejected`, `attempts_exhausted`, `outcome_unknown`, and
eligible `locally_denied` deliveries may be redriven after the responsible
owner records remediation or accepts the duplicate/ambiguity risk.
`http_accepted` redrive is disabled; enabling it requires a business and
receiver-integration policy explicitly accepting a duplicate after observed
HTTP acceptance. A new endpoint generation requires a new explicitly accepted
delivery, not redrive.

An operator may close an unknown delivery to stop recovery, but without a
receiver-owned receipt the durable outcome remains `closed_unknown`, not
`http_accepted` or receiver-processed. The action records actor, scope,
bounded reason code, optional protected bounded note, prior state, decision,
and duplicate-risk acknowledgement. No manual action deletes prior attempts.

Reconciliation compares delivery state, fan-out membership, attempt evidence,
and operator actions. It may idempotently materialize missing deliveries,
complete a provable local transition, or quarantine a conflict. It may not
infer remote acceptance from absence, metrics, a worker claim, or operator
belief.

Falsifier: race redrive with scheduled retry, repeat the action ID, redrive
after key/payload/destination expiry, target a new generation, and reconcile
missing or contradictory evidence. Any concurrent send, new delivery identity,
changed bytes, false acceptance, or erased audit fails the rule.

### W14 — Authorization, audit, telemetry, and diagnostics expose bounded facts without creating a data leak

Every destination lookup, event/delivery inspection, attempt, disable, redrive,
secret resolution, rotation, retention action, and reconciliation is scoped at
the data-access boundary to the authorized subscriber owner. An opaque ID is
never sufficient authorization. Operator mutations require an authenticated
principal, replay-safe action ID, allowed action, and audit evidence.

Metrics use only closed low-cardinality values: delivery state/outcome class,
error class, attempt bucket, profile/worker identity, and bounded age/backlog
aggregates. Destination URLs, hostnames, owner/subscriber IDs, event/delivery/
attempt/action IDs, payload/schema values, status-code values, response
content, secrets, key references, and signatures are not metric labels.
Per-destination health is aggregated into bounded cohorts rather than one
series per endpoint.

Logs and traces contain no payloads, response bodies, URLs, secrets, raw
signatures, arbitrary headers, or unbounded remote errors. When the adopter's
data policy permits it, protected logs/traces may carry event, delivery,
attempt, and operator-action IDs plus semantic classes. Delivery attempt spans
link locally to the business-event origin; they are not children of a stale
request context and emit no remote correlation by default.

An authenticated diagnostic surface can answer: whether an event and fan-out
were accepted; each destination generation and delivery state; current due/
in-flight/suspended/terminal/unknown counts and oldest age; freshness of worker
observation; bounded outcome/error trends; retry/redrive budget consumption;
and the append-only event -> delivery -> attempt -> operator-action chain. It
returns no secret or raw response content and applies subscriber scope before
lookup.

Readiness and alert signals distinguish store/worker observation freshness,
backlog age, destination health cohorts, retry exhaustion, outcome unknown,
SSRF denial, secret-rotation failure, response bounds, and reconciliation
conflict. Receiver outage and backlog are operational degradation signals, not
by themselves process liveness or worker-readiness failures.

Falsifier: inject secrets, URLs, payload fragments, arbitrary receiver
headers/bodies/errors, and cross-owner IDs into every failure path and telemetry
sink. Any unsafe value, unbounded label, synchronous metric-collection I/O, or
cross-owner diagnostic result fails the rule.

### W15 — Acceptance and delivery have independently observable lifecycle and bounded drain

The event-acceptance boundary fails closed when it cannot durably accept or
authoritatively resolve a required delivery intent. A worker outage does not
rewrite already accepted work; if durable acceptance remains available, work
may accumulate as visible backlog.

The delivery runtime has independently operable startup, readiness, liveness,
drain, and cleanup regardless of its later process placement:

- construction validates immutable config, engine/store registration, secret
  authority, network policy, policy bounds, and profile prerequisites before
  work is claimed; a missing concrete adapter or authority blocks startup;
- liveness is process-only; receiver availability, backlog, and destination
  disable state never make a live process dead;
- readiness becomes true only after fresh successful observations of every
  dependency needed to claim, sign, schedule, and finalize work, and becomes
  false when those observations are stale or drain starts; readiness performs
  no destination probe or side-effecting webhook request;
- one receiver or destination outage does not make the whole worker unready
  while the engine can safely progress or schedule other work;
- drain marks not-ready, stops new claims, allows in-flight attempts only
  within the declared attempt/drain budgets, cancels the remainder, records
  definite or ambiguous outcomes from strongest evidence, releases recoverable
  leases/claims, and joins all workers before dependency or telemetry cleanup;
- partial startup failure and forced shutdown close HTTP resources, secret
  clients, stores, diagnostics, and telemetry in their owned order without a
  surviving goroutine or hidden claim.

Freshness has one closed rule. Configuration supplies one positive finite
observation interval. A process starts not ready and records its last successful
complete required-dependency observation using process-local monotonic elapsed
time. It is fresh through two observation intervals, tolerating one missed
sample; it becomes stale when elapsed time is greater than two intervals. A
failed or partial observation never advances the timestamp. Restart loses the
process-local observation and returns to not ready until a new complete sample;
drain and a stopped worker make readiness false immediately regardless of the
last sample. Metrics and the readiness probe derive from the same observation
and stale-after rule. The interval value belongs to the adopter SRE/platform
owner; missing, zero, unbounded, or overflow-prone values block startup.

No startup/readiness check sends to a subscriber destination. No shutdown
claim states that cancellation undid a remote effect.

Falsifier: start with missing/invalid dependencies, stale observations, one
unhealthy endpoint, and partial initialization; then signal graceful and forced
drain during every network stage. Any side-effecting probe, readiness based on
stale success, leaked worker/resource, lost claim, or false no-effect outcome
fails the rule.

### W16 — Configuration and secret inputs are immutable, minimal, and complete before use

Non-secret global engine bounds and profile wiring follow the repository's one
validated immutable config snapshot and unknown-key rejection. Secret-like
values never enter YAML or flags. A static destination may receive an
environment-backed secret through the repository secret policy; dynamic or
externally managed destinations require one named secret authority and opaque
key-generation references. This Specification selects no dynamic secret store.

A concrete destination is admitted only with this complete declaration:

- authorized subscriber owner scope, destination ID/generation, ownership
  verification, event and payload-version selection, exact public-HTTPS URL,
  signature profile, active key authority, and administrative state;
- maximum payload size and accepted content types/business schema versions;
- finite attempt, response-header, response-body, connection, global and
  per-destination concurrency bounds;
- finite attempts, delivery age, backoff, `Retry-After`, redrive, retention,
  and receiver-dedup guidance owned by the named policy authorities;
- one finite worker observation interval whose readiness freshness is exactly
  the two-interval rule in W15;
- telemetry/privacy classification, authorized operator actions, network egress
  owner, and any automatic-pause policy.

The empty template admits no destination and makes no provider, workload,
retention, SLO, or production-readiness claim. Missing optional policy keeps
its behavior off; missing required identity, safety, secret, I/O, retry,
retention, or authorization input rejects the destination or startup boundary
that would otherwise use it. Runtime reload is not implied. A requirement for
live destination/secret/config mutation reopens the subscriber and lifecycle
boundary before design.

Falsifier: start with absent, partial, zero, unbounded, contradictory, unknown,
or secret-bearing config and register cross-owner or incomplete destinations.
Any best-effort default, secret in non-secret config, partially admitted
destination, or runtime mutation without an accepted authority fails the rule.

### W17 — Current outbox concepts may be reused, but its unchanged contract cannot satisfy general delivery

The current PostgreSQL outbox remains one broker-publication occurrence with
one acknowledgement policy, lease, attempt budget, ambiguity state, and
receipt. It does not represent a shared business event plus N independently
authorized destination generations, signed HTTP attempts, bounded response
evidence, per-destination scheduling/health, or fan-out completion.

Appending N unchanged outbox rows is permitted only when feature code
intentionally owns N independent publication events; it does not satisfy this
general capability. Evolving its schema and relay to own W1-W16 would be a
per-destination delivery store in substance and must be evaluated as such.

System / Integration Design may reuse transaction, lease/fence, sticky
ambiguity, audit, retention, telemetry, or lifecycle patterns; place a durable
handoff upstream of another engine; deliberately generalize an owner while
keeping broker and webhook policy independent; or select an external family.
No family is selected by this rule. Reopen the research finding only if scope
is reduced to the fixed low-fan-out/static-policy boundary named in Research or
new evidence shows the unchanged outbox satisfies every independent
destination behavior above.

## Decisions, constraints, and authorities

- **D1 — The reusable engine consumes snapshots; it does not become subscriber
  management or business truth.** This keeps authorization, event meaning, and
  delivery mechanics independently replaceable and prevents an endpoint URL or
  delivery row from becoming a subscriber or event identity.
- **D2 — At-least-once and unordered are the only portable claims.** Stable
  delivery identity, attempt evidence, and receiver guidance make duplicates
  diagnosable; no local state can make the remote effect atomic or ordered.
- **D3 — HMAC-SHA256 is the minimal baseline signature.** It has the strongest
  interoperable evidence in the Research set and the smallest receiver and
  secret-distribution surface. A named adopter requiring asymmetric
  non-forgeability or RFC 9421 semantics reopens W4 before design; no dormant
  multi-algorithm abstraction is added.
- **D4 — Public HTTPS:443, whole-answer denial, and redirect refusal are the
  baseline trust boundary.** Availability loss from a mixed DNS answer is
  accepted over a path that could select a forbidden address. A named private
  route, proxy, custom trust, or alternate port reopens W5 as a separate
  security capability.
- **D5 — No response content is retained.** Status, normalized timing/bounds,
  semantic class, and `Retry-After` disposition answer the reusable operator
  questions without creating a receiver-controlled data store. A concrete
  support requirement for content reopens W6/W14 with data classification,
  allowlist, access, retention, and disclosure proof.
- **D6 — Portable retry classes are fixed; quantities are adopter policy.** The
  engine inherits no provider or outbox attempts, age, delay, concurrency,
  pause, or retention default. Absence rejects destination admission rather
  than leaving behavior unbounded.
- **D7 — Automatic pause is off; administrative disable remains required.**
  Isolation already prevents one failed endpoint from consuming all work.
  Automation begins only when an owner supplies the failure budget and recovery
  semantics W11 requires.
- **D8 — `WEBHOOKS=durable` selects capability, not an implementation family.**
  System / Integration Design must keep, revise, or reject the local
  PostgreSQL-native leading hypothesis against the exact profile, durability,
  capacity, privacy, cost, deployment, and exit constraints.
- **D9 — Current outbox reuse is conceptual, not behavioral authority.** W17
  preserves its narrower broker-publication contract while allowing later
  design to reuse evidence-backed primitives or an upstream handoff.
- **D10 — Age is anchored at durable acceptance and consumes every pause.** A
  retry or operational stop cannot extend business recovery or retention by
  moving a deadline. Redrive has one separately anchored bounded recovery cycle.
- **D11 — Privacy erasure never turns prior identity into reusable absence.**
  The data/privacy owner must permit the minimal non-content tombstone or retire
  the whole identity namespace before erasure; silent resurrection is not an
  available implementation choice.

Current authorities are [Research synthesis](research/synthesis.md), valid as
of 2026-08-12; the repository architecture baseline; current config-secret,
bounded outbound HTTP, bootstrap/worker lifecycle, template-init, and
PostgreSQL outbox contracts named by that synthesis; RFC 9110 for HTTP status
and `Retry-After`; IANA special-purpose address registries; OWASP SSRF guidance;
and the Standard Webhooks/provider evidence summarized in Research. Research
remains evidence, candidate coverage, and reopen authority; this document owns
the behavioral choices above.

## External-owner inputs and checkpoints

Unavailable adopter values do not block this reusable contract because each
has a fail-closed absence rule. They do block the concrete action named below.

| Input | Owner | Required before | Safe absence and reopen condition |
| --- | --- | --- | --- |
| Who may create/update/disable/delete destinations; owner/tenant scope; endpoint verification | Subscriber-management product and security owner | First destination admission or any management API Specification | Admit no destination and expose no API; reopen W2/W5/W11/W14 when the owner supplies the lifecycle and authorization contract |
| First event type, exact payload authority, schema/version compatibility, content type and size | Business-event feature/domain owner | First event-type admission | Accept no event of that type; reopen W2-W3/W16 with a canonical contract and live-version horizon |
| Payload/URL classification, retention/deletion, tombstone-versus-namespace-retirement choice, and data residency | Data/privacy/legal owner with business recovery owner | Persisting the first real payload/destination, enabling privacy deletion, and production approval | Admit no destination/event, retain no real data, and keep privacy deletion disabled; reopen W12/W14/W16 with finite horizons, deletion precedence, and legally permitted idempotency evidence |
| Attempt/age/backoff/`Retry-After`/redrive budgets and delivery SLO | Product/SRE owner plus receiver integration owner | First destination admission and any production delivery claim | Reject incomplete policy and publish no SLO; reopen W7-W8/W13 on accepted values or a receiver-specific status contract |
| Global/per-destination concurrency, arrival/burst/fan-out, database/network budgets and fairness target | Adopter SRE/platform/data owner with measurement | Production topology/capacity approval | Require finite safety caps but make no throughput/latency/starvation-under-overload claim; reopen W6/W10/W15 with representative evidence |
| Worker observation interval and acceptable readiness staleness | Adopter SRE/platform owner | Delivery-runtime startup and any readiness claim | Block startup when absent/invalid and use no inherited engine default; W15 fixes freshness at exactly two accepted intervals, and a different tolerance reopens W15 |
| Administrative roles, allowed recovery actions and reason/audit policy | Security/SRE/business owner | Enabling disable, redrive, accepted-redrive, or unknown closure | Permit no operator mutation beyond fail-safe stop; reopen W11/W13/W14 with roles, scopes, state eligibility and duplicate-risk acceptance |
| Automatic pause classes, window, threshold, cooldown/re-enable and alert | SRE/product owner | Enabling automatic pause | Keep it off; reopen W11 only with the complete policy and attribution proof |
| Signing-secret issuance, authority, rotation overlap, retirement, emergency revocation and recovery | Security/platform owner plus subscriber integration owner | First destination admission | Resolve no key and send nothing; reopen W4/W12/W16 when one named authority and overlap/retirement contract exists |
| Public egress controls, DNS path, TLS roots and deployment route | Network/security/platform owner | Deployment and production-readiness approval | Make no deployed SSRF or reachability claim; a private/proxy/custom-trust/alternate-port need reopens W5 and the capability boundary |
| Receiver signature tolerance and delivery-ID dedup retention | Receiver integration owner | Receiver conformance and any duplicate-recovery claim | State at-least-once only; reopen W4/W9/W12 if the receiver cannot retain dedup through delivery plus redrive |
| Event ordering partition/sequence and poison recovery | Business-event and receiver owner | Any ordered-delivery claim | Remain explicitly unordered; reopen W10 in Specification before design |
| Static versus dynamic destination/secret mutation and customer-facing portal/API | Subscriber product/security/platform owner | Any runtime management surface | Keep destinations pre-authorized through an external/static owner and expose no generic CRUD; reopen subscriber-management and lifecycle Specifications |
| Engine family, external product/version, region, license, data custody, cost, provider quotas and exit | Technical Design with adopter platform/security/legal/procurement owners | System / Integration Design selection and provider/deployment readiness | Keep all families unselected and make no provider claim; refresh Research when candidate or freshness conditions change |

## Invariants and edge cases

- One accepted business event and fan-out snapshot never silently lose, add, or
  retarget a destination member.
- Business-event, destination-generation, delivery, attempt, operator-action,
  and signing-key identities retain their separate meanings through retry,
  redrive, rotation, deletion, and cleanup.
- Every send is authorized and transport-admissible at use time; a historical
  registration receipt cannot bypass current disable, DNS/address, TLS, egress,
  or key policy.
- Every retry/redrive keeps the exact payload and stable delivery ID and creates
  a fresh attempt identity, timestamp, and valid signature evidence.
- A delivery may be HTTP-accepted, HTTP-rejected, attempts-exhausted,
  locally-denied, or outcome-unknown; none is receiver business-processing
  truth.
- A possible remote effect is never rewritten as definitely absent because of
  cancellation, timeout, crash, stale worker, finalization failure, disable, or
  operator belief.
- Destination progress, budgets, administrative state, retention, and redrive
  remain independent across fan-out members.
- No response body, payload, URL, secret, raw signature, or arbitrary remote
  error reaches general telemetry.
- No active or recoverable delivery loses the snapshots, key-reference,
  attempt, or operator evidence required to interpret or redrive it.
- Profile composition reuses sibling capabilities only through explicit
  boundaries; webhook selection does not inherit broker ACK/DLQ, outbox
  publication, job scheduling, or fixed-target HTTP behavior by resemblance.

## Success criteria and proof expectations

These are behavioral proof boundaries for later Test Design. They select no
test layer, fixture topology, command, schema, or implementation.

| ID | Observable pass/fail boundary |
| --- | --- |
| SC-01 Profile composition | Omitted, selected, invalid, and mixed profile initialization is deterministic, repeatable, non-mutating on rejection, dependency-clean, and free of removed-profile residue |
| SC-02 Durable acceptance | A business event and complete fan-out become durably recoverable with exactly one delivery per destination generation, or the caller receives rejected/unknown without minting new identities; interruption exposes no silent partial fan-out |
| SC-03 Immutable identity and bytes | Retry, deploy, serializer/version change, endpoint update, rotation, and redrive preserve event/delivery identity, exact payload, versions and destination generation while changing only attempt/time/current key evidence |
| SC-04 Signing and receiver contract | The receiver verifies the exact `Webhook-Id`, canonical Unix-seconds `Webhook-Timestamp`, ordered one/two-entry `Webhook-Signature`, signed-byte grammar and Base64 HMAC contract; any currently accepted overlap key suffices, wrong/missing/cross-destination/expired signatures fail, and receiver dedup observes stable delivery IDs across distinct attempts and out-of-order events |
| SC-05 Destination security | Registration/use-time parsing, A/AAAA/mixed/rebinding resolution, actual dial checks, redirects, proxies/resolvers, TLS, egress and owner scope deny every forbidden path before disclosure |
| SC-06 Bounded execution and classification | DNS/connect/TLS/write/header/body stalls and overflow, compression, connection reuse, cancellation, every HTTP class and before/after-send failure stay within bounds; one attempt never hides two sends and stores no response content |
| SC-07 Retry and ambiguity | Durable jittered scheduling, acceptance-anchored deadlines, pause/disable/local-deny time consumption, separately anchored redrive age, every `Retry-After` form/cap, restart, stale worker, crash around remote effect/response/finalization, and exhaustion preserve cycle-local results plus cumulative precedence for historical 2xx/ambiguity, the exact age/attempt budget, duplicate warning and final unknown distinction |
| SC-08 Fan-out, isolation and disable | N destinations progress independently; one slow/poisoned/flooded/paused endpoint cannot consume all declared capacity; update/delete/pause/re-enable races obey W10-W11 and create no ordering claim |
| SC-09 Retention, redrive and audit | Cleanup preserves every live dependency; privacy deletion atomically leaves the permitted tombstone or retires the namespace, repeated old acceptance returns `privacy_deleted` without resurrection, and authorized replay-safe redrive serializes with retry, preserves identity/bytes, uses a fresh signature, and leaves append-only evidence |
| SC-10 Lifecycle and observability | Invalid startup fails closed; no sample, one missed interval, more than two intervals, restart and drain exercise the one readiness-freshness rule; process-only liveness, receiver degradation, graceful/forced drain, join and cleanup obey W15; bounded telemetry and protected diagnostics answer W14 without leakage or cross-owner access |

Proof obligations retained for downstream ownership:

- Durability, fan-out materialization, claim/fence, finalization, retention,
  reconciliation, retry/redrive races, stale workers, and crash recovery require
  the selected real durable authority. If PostgreSQL is selected, concurrency
  and visibility claims require real PostgreSQL and independent workers; mocks
  cannot establish them.
- DNS rebinding, actual dial-address enforcement, proxy/resolver bypass, TLS,
  network egress, and remote-effect ambiguity require real network boundaries
  and controlled receivers. Parser-only or mock-transport proof is insufficient.
- Process lifecycle requires real startup, signal, drain, forced cancellation,
  worker join, and cleanup observations. Profile proof requires initialized
  generated checkouts, not only the template source tree.
- Security proof covers every owner-scoped lookup/mutation, signature/key
  transition, and disclosure sink. Receiver conformance separately covers raw
  bytes, replay tolerance, dedup horizon, duplicates, and out-of-order events.
- Fairness, capacity, SLO, retention, privacy, provider, cost, and production
  readiness claims require the adopter inputs and actual deployment path named
  above. Local/template evidence proves only the reusable pack surface.
- If System Design selects OSS or managed delivery, exact version/license/
  advisory, durable ingress idempotency, local-to-provider reconciliation,
  quotas, outage/ambiguity, dependency HA/backups/migrations, data/secret
  custody, load/cost, export, and exit remain mandatory proving surfaces.

Success means every applicable SC boundary passes for the exact selected
profile and engine, every concrete event and destination has its owner inputs,
and no claim exceeds the durable, network, receiver, generated-profile, or
deployed surfaces actually exercised.

## Risks, assumptions, and reopen conditions

- **Assumption — destinations are pre-authorized snapshots, not a generic
  public subscription product.** Affected rules: W2, W5, W11, W14, W16. Safe
  boundary: feature/bootstrap code receives snapshots from a named external or
  static subscriber authority and exposes no CRUD. Owner: subscriber product
  and security. Reopen when a runtime/customer management surface, endpoint
  challenge, tenant policy, or live destination mutation is required.
- **Assumption — public HTTPS:443 with system trust is sufficient.** Affected
  rule: W5. Safe boundary: arbitrary external receivers reachable through
  deployment-controlled public egress. Owner: network/security/platform.
  Reopen on a named private address, proxy, custom CA, mTLS, service mesh,
  alternate port, or mandatory custom resolver.
- **Assumption — HMAC verifier forgery risk is acceptable for the portable
  baseline.** Affected rule: W4. Safe boundary: a high-entropy per-destination
  secret shared only with its authorized receiver and controlled overlap.
  Owner: security and receiver integration. Reopen when asymmetric signing,
  public verification, non-forgeability by receivers, RFC 9421, or a different
  mandated profile is required.
- **Assumption — retry quantities are destination policy, not template
  defaults.** Affected rules: W6-W8, W10-W13, W15-W16. Safe boundary: no
  destination is admitted until finite values are supplied. Owner:
  product/SRE/receiver integration. Reopen when repository-wide evidence
  supports one portable default set or a named receiver contract conflicts
  with the status table.
- **Assumption — default unordered delivery is acceptable.** Affected rule:
  W10. Safe boundary: receiver uses event timestamps/versioned business state
  and delivery-ID dedup without relying on arrival order. Owner: business-event
  and receiver integration. Reopen on a real partition/sequence invariant,
  then specify head-of-line, poison, late, and replay behavior before design.
- **Assumption — automatic pause is unnecessary without a failure-budget
  owner.** Affected rule: W11. Safe boundary: finite per-destination
  concurrency, backoff, age and fairness isolate failures while operators can
  disable explicitly. Owner: SRE/product. Reopen on measured retry storms,
  operator load, or accepted threshold/cooldown policy.
- **Assumption — no receiver response content is operationally required.**
  Affected rules: W6, W14. Safe boundary: status, semantic class, timing,
  byte counts and normalized `Retry-After` are sufficient. Owner: support,
  security and data/privacy. Reopen only with a concrete diagnostic need and
  bounded allowlist/access/retention proof.
- **Assumption — the receiver can deduplicate for the full delivery and redrive
  horizon.** Affected rules: W4, W8-W9, W12-W13. Safe boundary: the receiver
  stores stable delivery IDs long enough and accepts fresh attempt timestamps.
  Owner: receiver integration. Reopen when it cannot; the honest outcome may
  narrow retry/redrive rather than claim exactly once.
- **Risk — payload, attempt, and audit retention can conflict with privacy.**
  W12 gives privacy deletion precedence by stopping send/redrive and retaining
  only a non-sensitive tombstone. Actual values remain blocked on the
  data/legal and business-recovery owners; storage growth never authorizes
  silent cleanup.
- **Risk — retries and large fan-out can amplify an outage.** No fairness,
  capacity, SLO, or shared-datastore readiness claim exists without measured
  arrival, burst, fan-out, duration, connection, database, WAL/storage,
  failover, egress, and recovery-age evidence. Failed budgets reopen topology,
  engine family, admission caps, or product policy rather than weakening
  durability or safety.
- **Risk — local evidence cannot close a remote side effect.** Persistent
  ambiguity and `closed_unknown` are required when no receiver authority can
  decide. A need for stronger finality reopens the receiver receipt/
  reconciliation contract, not the meaning of 2xx.
- **Leading implementation hypothesis remains unselected.** Research found a
  distinct PostgreSQL event/delivery/attempt owner to have the fewest current
  integration gaps. System / Integration Design must test it first and keep,
  revise, or reject it. Workload scale, a mandated external control plane,
  unacceptable OLTP impact, provider/data/cost constraints, or a smaller
  proven candidate overturn it.
- **Refresh Research** when a concrete OSS/managed product/version enters
  design; repository outbox, HTTP, config, lifecycle, or profile authority
  changes materially; address/signature standards change; or evidence adds a
  candidate family or makes unchanged outbox reuse satisfy W1-W16.

No unavailable owner input blocks this vendor-neutral Specification. It blocks
the first concrete event, destination, operator control, or production claim at
the exact checkpoint above and fails closed there.

## Specification review

Independent whole-artifact review initially returned `FAIL` on four
Specification-owned gaps: receiver-visible signature interoperability,
privacy-deletion replay/authority, delivery and redrive age anchors across
pause, and readiness freshness. W2-W4, W8, W12-W13, W15-W16, the owner-input
table, and SC-04/07/09/10 now close those behaviors.

Focused fresh review found one repair interaction: a redrive-cycle expiry could
overwrite stronger historical acceptance or ambiguity. W9 and W13 now keep
cycle-local disposition separate from the cumulative delivery summary and
apply monotone strongest-evidence precedence. Focused re-review of behavioral
candidate SHA-256 `cf68230736dec57b1a7aec2817b3ed94ded53ec4f39b234ad6157a41bd4ce76c`
returned **PASS** with no surviving finding. The review covered Specification
behavior only; it provides no System / Integration Design, Go ownership, Test
Design, implementation, provider, deployment, or production evidence.

## Standalone prompt for Technical Design

```text
Work in /Users/daniil/Projects/Opensource/go-service-template-rest using the structured spec-first workflow.

Continue with the Technical Design macro phase only, starting in System / Integration Design. Specification is ready with independent review PASS: specs/outbound-webhook-delivery/spec.md owns the behavioral contract, specs/outbound-webhook-delivery/research/synthesis.md owns the evidence and candidate-family limits, and docs/repo-architecture.md owns current repository boundaries. Read the current router-selected System / Integration Design owner before acting; continue to Go Code / Ownership Design and the required Technical Design review only within this macro phase.

Use a distinct PostgreSQL event/delivery/attempt owner with an independently operable webhook worker as the leading hypothesis because Research found it has the fewest current unowned integration boundaries. Test it first against W1-W17: protected business-commit-to-delivery acceptance or reconciled idempotent handoff, complete fan-out identity and immutable evidence, the fixed HMAC `v1` receiver contract, dynamic-destination SSRF/DNS/dial enforcement, one-send attempts and cumulative ambiguity, durable retry/redrive clocks, destination fairness/disable, tombstone/retention authority, readiness/drain, secret resolution, and independently removable profile composition. Compare it with a deliberate generalized durable owner and concrete self-hosted/managed families; keep, revise, or reject the hypothesis from current workload, data/control ownership, dependency, capacity, cost, license, provider-ingress, outage/reconciliation, and exit evidence rather than treating Research fit as architecture selection.

First, reconstruct the affected deployment graph and the business event -> fan-out snapshot -> delivery -> attempt/operator authority flow, then select the smallest coherent engine family and durable acceptance boundary. Preserve subscriber management and business-event semantics as external authorities, keep the current outbox's unchanged contract insufficient for general per-destination delivery, and retain every external-owner checkpoint and downstream proof obligation. Reopen Specification if design would change receiver wire behavior, identity/finality, retry/disable/redrive, retention/privacy, lifecycle, or profile meaning; reopen Research only if its named candidate/outbox/security evidence condition changes. Do not enter Test Design, Planning, tasks, migrations, or implementation in this session.
```
