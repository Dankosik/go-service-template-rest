# Optional Durable Outbound Webhook Delivery Research

status: research complete; independent review PASS
valid as of: 2026-08-12

## Accepted outcome and boundary

Research the evidence needed for a template-init-selected capability pack in
which feature code registers a business event and destination while reusable
module code owns durable outbound delivery, request signing, bounded HTTP
execution, retry and ambiguity handling, telemetry, and operator recovery with
minimal configuration.

This note owns evidence, candidate-family coverage, downstream constraints,
proof obligations, and reopen conditions. It does not select architecture,
specify behavior, place Go code, plan work, or authorize provider or production
actions. Subscriber management and business-event semantics remain separate
from the reusable delivery-engine question.

## Open-item map

| ID | Decision-changing question | Method | Downstream owner | Closure target |
| --- | --- | --- | --- | --- |
| RQ-1 | Which current repository authorities and lifecycles can be reused without changing their guarantees, and where does a webhook capability require a new owner? | Current-state semantic baseline | Specification, then System / Integration Design | Trace event identity, PostgreSQL outbox, HTTP client, workers/relay, config/secrets, telemetry, and template-init composition through their first unsupported edge. |
| RQ-2 | What delivery identity, destination, payload/version, signature/rotation, egress, bounded-I/O, retry, ambiguity, duplicate, ordering, fan-out, health-disable, retention, redrive, and audit constraints are established by current primary authorities and production counter-evidence? | Current external contract; conflict/freshness | Specification and Test Design | Separate established facts from policy/design choices; retain falsifiers, unsafe defaults, and proving surfaces. |
| RQ-3 | Which viable candidate families occupy the delivery-engine slot, how do they relate to subscriber management and event semantics, and what local-fit evidence eliminates or carries each family? | Solution discovery evidence | System / Integration Design | Compare current-outbox reuse, a per-destination delivery store, mature OSS, and managed delivery without ranking or selecting target architecture. |

## Question closure and lens map

### RQ-1 — repository baseline

- Leading hypothesis: the current transactional outbox can remain an upstream
  atomic publication/handoff option, but its publication row and relay state
  are not automatically a per-destination delivery ledger; feature state
  remains the business-event authority.
- Falsifier: current code or contract already represents destination-scoped
  attempts, response evidence, independent retry/disable state, and fan-out
  completion without weakening its broker-publication guarantees.
- Smallest evidence: authoritative schema/queries, package contracts and call
  graph, executable bootstrap/lifecycle wiring, config and template-init
  manifests, and telemetry owners.
- Coverage: repository authority and generated authority `researched`; deployed
  adopter/runtime state `not triggered` because the requested deliverable is a
  template research baseline, not rollout readiness.
- Stop: every named current surface reaches either a reusable contract or its
  first unsupported edge.

### RQ-2 — delivery boundary contract

- Leading hypothesis: registered destinations plus dial-time address policy,
  redirect refusal, bounded attempts/responses, timestamped signatures with
  overlapping keys, stable delivery identity, and at-least-once recovery are
  necessary; exact retry and disable policy remains for Specification/Design.
- Falsifier: a current standard or widely operated provider proves a materially
  safer or simpler contract for arbitrary URLs, exactly-once delivery,
  unbounded responses, response-code-only retry, or single-key cutover.
- Smallest evidence: HTTP and signature specifications, current provider docs,
  maintained source code, vulnerability guidance, and concrete production
  failure evidence that challenges provider happy-path prose.
- Coverage: standards/provider authority, implementation source,
  local-applicability, and operational counter-evidence `researched`; live
  credential/provider mutation `not triggered` because Research is read-only.
- Stop: each requested concern has an authoritative disposition or an explicit
  policy/proof owner and refresh condition.

### RQ-3 — candidate families

- Neutral decision slot: the durable per-destination delivery engine after a
  business event is durably accepted and before subscriber administration or
  business interpretation.
- Live substitutes: extend/reuse the current outbox, add a distinct
  destination-delivery store/worker, adopt mature OSS, or use a managed webhook
  delivery system. An event bus or outbox may be a prerequisite rather than a
  substitute; subscriber management may be a complement.
- Falsifier: a family cannot preserve current transaction/event identity,
  egress/security policy, operator evidence, profile optionality, or acceptable
  data/control ownership without an unowned integration gap.
- Candidate rungs: repository reuse, PostgreSQL-native queue/store, already
  installed dependencies, mature OSS, managed service, and minimum custom code.
- Stop: searches by durable webhook, fan-out delivery, endpoint retry,
  dead-letter/redrive, webhook gateway, and eventing aliases yield no materially
  different candidate family; reappearing products only refine an existing
  family.

## Evidence synthesis

### Three authorities must not be collapsed

The reusable capability has three different authorities. Similar data shape
does not make them one subsystem.

| Authority | Owns | Must not silently own |
| --- | --- | --- |
| Business-event semantics | Event type, domain meaning, occurrence identity, schema/version, exact payload content, and which transaction makes the event true | Subscriber lifecycle, HTTP policy, retry classification, signing, or operator delivery claims |
| Subscriber management | Destination ownership and authorization; registration/update/delete/disable; secret provisioning and rotation; event selection; payload-version preference; endpoint verification; and any customer-facing portal/API | Business truth, network-attempt state, or a claim of exactly-once processing |
| Reusable delivery engine | A durable delivery occurrence; immutable accepted payload/application envelope plus attempt-header evidence; bounded attempts; address-policy enforcement; signing; outcome classification; scheduling; terminal/unknown state; attempt/operator audit; telemetry; retention and redrive mechanics | The business meaning of an event, whether a subscriber processed it, or the product policy for who may subscribe |

A thin registration call can cross these authorities only through explicit
identities and snapshots. It cannot make the delivery engine the source of
business truth or make an endpoint URL the subscriber identity.

### Current repository boundary

The current PostgreSQL outbox provides useful durable-flow primitives but has
a narrower contract:

- [`postgresoutbox.Event`](../../../internal/infra/postgresoutbox/event.go) is
  one immutable broker-publication occurrence. Its `ID` is the publication
  identity across retry/redrive, and it contains one adapter-interpreted
  `Destination`, exact payload/metadata, schema, and captured origin trace.
- [`Store.Append`](../../../internal/infra/postgresoutbox/store_append.go) uses
  the caller-owned PostgreSQL transaction. One business mutation can therefore
  atomically append multiple rows, but each row is independently identified.
- [migration `000001`](../../../migrations/000001_postgres_outbox.sql) gives one
  row one lease, attempt count, uncertainty flag, terminal state, and retained
  receipt keyed by `event_id`. The relay invokes one `Publisher.Publish` for a
  claimed event and finalizes that one acknowledgement in
  [`relay_publish.go`](../../../internal/infra/postgresoutbox/relay_publish.go)
  and [`relay_finalize.go`](../../../internal/infra/postgresoutbox/relay_finalize.go).
- The publisher contract distinguishes acknowledged, definitely-not-accepted,
  permanent, and ambiguous outcomes. The relay preserves sticky ambiguity,
  bounded attempts, poison/unknown terminal states, audited redrive and confirm
  actions, retention, readiness, backlog/age telemetry, and controlled drain.
  Those are reusable concepts and lifecycle evidence, not evidence that a
  broker acknowledgement and an HTTP response have the same policy.

Consequently, an unchanged row cannot represent one logical event delivered to
N destinations with independent claims, responses, retries, disable state, and
operator actions. Appending N current outbox events is representable only when
feature code deliberately mints N independent publication identities. That
moves fan-out snapshots and subscriber policy into feature code, loses a common
business-event/delivery identity model, and still lacks destination-scoped
attempt evidence and health controls. Adding those semantics to the current
schema would make it a per-destination delivery store in substance, regardless
of package name.

The separate outbox process nevertheless establishes repository-native
lifecycle patterns worth carrying: validated construction; fail-closed missing
adapter; readiness only from fresh observations; lease/fence-based work;
bounded drain followed by forced cancellation and join; and telemetry shutdown
after workers. It does not authorize running arbitrary webhook work inside the
API process.

The current NATS [`cmd/worker` composition](../../../cmd/worker/internal/bootstrap/run.go)
adds the sibling consumer-lifecycle evidence: it rejects a missing feature
handler or disabled transport, validates shutdown budgets, requires diagnostics,
admits readiness before service, supervises the broker connection/readiness and
worker, then marks draining, shuts down the durable consumer inside its own
budget, stops diagnostics, joins background work, and only cleans handler-owned
resources after the worker is known stopped; see
[`runWorkerLifecycle`](../../../cmd/worker/internal/bootstrap/lifecycle.go).
That cleanup ordering and fail-closed composition are reusable patterns. Its
broker consumer acknowledgement/DLQ semantics are not outbound HTTP delivery
semantics and do not provide a webhook worker for free.

The current [`httpclient`](../../../internal/infra/httpclient/client.go) is also
a pattern, not yet the dynamic-destination solution. It is built around one
configured scheme and authority, refuses redirects, disables proxies, checks
the actual dial address after DNS resolution, bounds headers/body/connections,
and gives the full request including body consumption one timeout. Retries are
outside those per-attempt bounds and require a safe method or usable
`Idempotency-Key`. Dynamic subscriber authorities, destination admission,
destination-specific policy, and connection-pool ownership are its first
unsupported edge. A webhook delivery ID can help a receiver deduplicate; merely
putting it in `Idempotency-Key` cannot prove an arbitrary receiver implements
idempotent POST semantics. Go's transport may itself replay a request on some
reused-connection failures when it recognizes an idempotency header, so a
webhook engine must also prove that one recorded attempt cannot hide two
network sends.

Configuration currently allows non-secret YAML defaults, rejects unknown keys,
and routes secret-like keys through `APP__...` environment variables; see the
[configuration source policy](../../../docs/configuration-source-policy.md).
That is suitable for a small static endpoint set, but it cannot by itself own
arbitrary per-subscriber signing secrets or overlapping rotations. Dynamic
registration therefore requires a concrete secret authority, or its scope must
remain statically configured. The initializer has independently optional
database, outbox, messaging, and outbound-HTTP selectors, but no webhook
profile. A future profile must be independently removable and must not assume
that selecting the broker outbox also selects webhook semantics.

There is current uncommitted durable-jobs work in the checkout. It is neither
stable template authority nor a complete worker/profile path, and its research
already rejects treating broker publication as a general job engine. This note
uses it only as a collision/ownership warning, not as proof of a reusable
webhook runtime.

### Identity and immutable evidence

Provider practice exposes four identities rather than one:

1. A **business-event ID** identifies the domain occurrence and remains stable
   across all subscribers.
2. A **subscription/destination identity plus generation** identifies the
   authorized endpoint configuration whose snapshot was selected. URL text is
   mutable configuration, not identity.
3. A **delivery ID** identifies one business event directed to one destination
   generation. It remains stable through automated retries and manual redrive
   and is the receiver's deduplication key.
4. An **attempt ID/time** identifies one bounded network execution. It may have
   a fresh signature timestamp and key generation while retaining the delivery
   ID. Operator actions need their own replay-safe audit identity.

Shopify's [delivery structure](https://shopify.dev/docs/apps/build/webhooks/delivery-structure)
explicitly distinguishes a shared event ID from a delivery/webhook ID, while
GitHub keeps its [delivery ID stable across redelivery](https://docs.github.com/en/webhooks/testing-and-troubleshooting-webhooks/redelivering-webhooks).
The
[Standard Webhooks specification](https://github.com/standard-webhooks/standard-webhooks/blob/main/spec/standard-webhooks.md)
likewise keeps the message ID stable across retries but generates a new attempt
timestamp. These practices support the decomposition; they do not establish a
universal identifier format.

The exact payload bytes, business schema/version, delivery-envelope/signature
version, and destination generation must be reconstructible for every retry and
redrive. Regenerating old payloads through the current serializer would let a
deployment silently change bytes, signatures, or semantics. Stripe freezes an
event's payload structure at event creation and treats destination version
migration as an explicit workflow; see its [webhook delivery](https://docs.stripe.com/webhooks)
and [versioning](https://docs.stripe.com/webhooks/versioning) documentation.
This supports immutable acceptance evidence while leaving event content and
version policy with feature/subscriber authorities.

### Signatures and secret rotation

The strongest interoperable candidate convention in the evidence set is the
[Standard Webhooks specification](https://github.com/standard-webhooks/standard-webhooks/blob/main/spec/standard-webhooks.md):
sign the exact bytes `message-id.timestamp.payload`, use HMAC-SHA256 or Ed25519,
carry the ID/timestamp/signature in dedicated headers, compare in constant time,
and allow multiple signatures during key overlap. Stripe independently signs a
timestamped payload and emits signatures for overlapping old/new endpoint
secrets during rotation. GitHub and Shopify likewise require verification over
the unmodified raw request body; see their [signature](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries)
and [delivery](https://shopify.dev/docs/apps/build/webhooks/delivery-structure)
contracts.

This establishes requirements to preserve exact bytes, use high-entropy
per-destination key material, authenticate the timestamp and delivery ID, and
support an overlap interval without logging secrets or raw signature material.
It does not select HMAC versus Ed25519, a replay tolerance, secret residence,
rotation duration, or retirement policy. [RFC 9421 HTTP Message
Signatures](https://www.rfc-editor.org/rfc/rfc9421.html) and [RFC 9530 Digest
Fields](https://www.rfc-editor.org/rfc/rfc9530.html) are primary alternatives
for a versioned profile with explicit covered components, declared signing-key
identifier, creation and expiry, and an authenticated content digest; they do
not themselves define a webhook profile or replay window. A symmetric verifier
can forge messages,
while asymmetric verification adds key-distribution/lifecycle work. The chosen
profile must name exact signed bytes/components and destination binding rather
than claim that an algorithm name alone provides compatibility.

Replay-window enforcement and durable business deduplication are separate: a
five-minute signature tolerance cannot be used as the dedup retention horizon
for a delivery retried or redriven days later. Each retry can be freshly
timestamped and signed while keeping the same delivery ID.

### Destination admission, SSRF, DNS, and redirects

An arbitrary destination URL is an outbound SSRF capability. The
[OWASP SSRF prevention guidance](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
names webhooks as a case where an allowlist may be impossible and therefore
requires defense in depth: strict URL validation, DNS/address checks, redirect
control, and network-layer egress policy. Registration-time resolution is not
enough because records can change between validation and connection. The
actual dial address for every new connection must be checked after resolution,
for both A and AAAA results, against loopback, link-local, private, multicast,
unspecified, metadata, and other special-use ranges. Proxy or custom resolver
paths must not bypass that decision. Egress filtering is the independent final
barrier. IANA's current [IPv4](https://www.iana.org/assignments/iana-ipv4-special-registry/iana-ipv4-special-registry.xhtml)
and [IPv6](https://www.iana.org/assignments/iana-ipv6-special-registry/iana-ipv6-special-registry.xhtml)
special-purpose registries are the address authority; “global unicast” alone
is not a sufficient public-destination predicate. Whether a mixed DNS answer
set rejects the destination wholesale or admits only a checked selected
address remains an explicit security/availability policy.

Both registries were rechecked on 2026-08-15 and still report `2025-10-09` as
their latest revision. The implementation corpus and its documented revision
therefore remain aligned; a later registry revision must reopen the corpus and
tests together.

Redirect following is unsafe because it permits a validated public authority to
transfer the request, secret-bearing headers, or method semantics to another
authority. The current local client already refuses redirects, and Shopify
treats every 3xx as failure. The delivery boundary should therefore carry
redirect refusal as a downstream constraint; endpoint migration belongs to an
audited subscription update. HTTPS/TLS verification is the general external
default. [BCP 195 / RFC 9325](https://www.rfc-editor.org/rfc/rfc9325.html)
requires TLS 1.2 support and recommends TLS 1.3 for existing application
protocols; the newer [RFC 9852](https://www.rfc-editor.org/rfc/rfc9852.html)
requires TLS 1.3 for new protocols and permits TLS 1.2 as compatibility. This
supports a materialized TLS 1.3 default with one explicit immutable TLS 1.2
exception, never a fallback below 1.2. Any private-network, custom-CA,
mutual-TLS, proxy, or non-HTTPS route is a separate trust-boundary capability,
not a harmless endpoint option.

Destination ownership verification, allowed domains/ports, and who may
register are subscriber-management policy inputs. No surveyed standard proves
one universal rule. Leaving those owners undefined would leave the engine as a
generic authenticated port scanner even if private addresses are blocked.

### Bounded HTTP execution and response evidence

Go's [`net/http.Client`](https://pkg.go.dev/net/http#Client) defines a client
timeout across connection, redirects, and response-body reading, and requires
body closure for connection reuse. The repository adds header/body and
connection bounds. Webhook work also needs independent outer delivery age and
worker-drain bounds, because a retry schedule must not hold a goroutine or
transaction open.

Every stage can consume a budget: DNS, connect, TLS, request write, response
headers, bounded body read, retry delay, and total delivery age. The module
must not read an unbounded success or error body, decompress an unbounded body,
or store arbitrary receiver content as audit evidence. Go enables transparent
compression by default, so disabling it or enforcing the cap after decoding is
part of the response-bound proof. Status, bounded size,
latency, a bounded/redacted diagnostic, and network error class are sufficient
research-level evidence; whether any response excerpt is retained is a privacy
and operations policy for Specification.

Provider deadlines are evidence against generous defaults, not a common
constant. [Shopify](https://shopify.dev/docs/apps/build/webhooks/delivery-structure)
documents a one-second connection budget and five-second total request; AWS
EventBridge API Destinations impose a five-second timeout;
GitHub asks receivers to acknowledge quickly. A template default needs an
explicit downstream latency/load proof rather than copying the largest value.

### Outcomes, retries, duplicates, and `Retry-After`

The durable boundary is at-least-once. A 2xx proves that the receiver accepted
the HTTP request according to its own contract; it does not prove downstream
business processing. A timeout or disconnect after any bytes may have reached
the receiver is ambiguous. Retrying the same delivery ID can recover the event
but can duplicate effects. Even a 5xx can follow an applied effect. The receiver
therefore needs delivery-ID deduplication, and the sender must expose ambiguity
rather than claim exactly once.

[RFC 9110 section 9.2.2](https://www.rfc-editor.org/rfc/rfc9110.html#section-9.2.2)
warns against automatically retrying a non-idempotent request unless its
semantics are known to be idempotent or the client knows the request was not
applied. Arbitrary webhook receivers do not provide that knowledge. The safe
recovery contract is consequently stable identity plus duplicate tolerance,
not inference from status alone.

There is no universal provider retry table:

- [Stripe](https://docs.stripe.com/webhooks) retries live-mode deliveries for
  up to three days and allows manual resend without cancelling automatic
  retries; it documents duplicates and no ordering guarantee.
- [GitHub](https://docs.github.com/en/webhooks/using-webhooks/handling-failed-webhook-deliveries)
  documents manual redelivery but does not automatically redeliver failed
  deliveries.
- Shopify treats non-2xx, including 3xx, as failure and documents eight retries
  and removal of unhealthy dynamically created subscriptions, while its
  current [delivery](https://shopify.dev/docs/apps/build/webhooks/delivery-structure)
  and [troubleshooting](https://shopify.dev/docs/apps/build/webhooks/troubleshoot)
  pages conflict on the exact retry window.
- [AWS EventBridge API Destinations](https://docs.aws.amazon.com/eventbridge/latest/userguide/eb-api-destinations.html)
  retry 401, 407, 409, 429, and 5xx, do not retry 1xx/2xx/3xx/other 4xx, and are
  bounded by the event-bus retry age/attempt policy and optional DLQ.

Those differences make retryable statuses, maximum attempts/age, endpoint
budgets, and automatic disable product policy—not standards to inherit. The
engine still needs closed outcome classes: accepted; definitely rejected and
non-retryable; retryable rejection; locally denied; ambiguous transport; and
exhausted/unknown. A retry must be persisted as future work rather than sleep a
claiming worker.

[RFC 9110 section 10.2.3](https://www.rfc-editor.org/rfc/rfc9110.html#section-10.2.3)
defines `Retry-After` as either an HTTP date or non-negative delay seconds. It
does not make the hint unbounded authority. The downstream policy must define
where it is honored, parsing/date reference, a cap inside delivery retention,
malformed and past-date behavior, and interaction with jitter and endpoint
budgets. AWS documents a conservative combination with its own backoff, which
is evidence that blindly replacing local scheduling is not required.

[RFC 6585](https://www.rfc-editor.org/rfc/rfc6585.html#section-4) permits a 429
response to carry `Retry-After`; [RFC 8470](https://www.rfc-editor.org/rfc/rfc8470.html#section-5.2)
defines 425 as a retry signal for early-data requests. Neither status proves a
webhook effect was absent. Classifying them as ambiguous retryable responses
inside the fixed attempt/deadline/dedup contract is therefore conservative;
the client sends no TLS early data and grants neither response an unbounded
retry mandate.

### Fan-out, ordering, and unhealthy destinations

Fan-out needs an immutable subscription snapshot or generation. Otherwise an
endpoint update could redirect an already accepted event, and deleting one
subscriber could erase evidence required by another. One business event can
produce N independent delivery occurrences; one slow or poisoned endpoint must
not monopolize claims or global concurrency.

Retries make cross-event completion order unstable. Stripe and
[GitHub](https://docs.github.com/en/webhooks/testing-and-troubleshooting-webhooks/troubleshooting-webhooks)
both document that consumers must tolerate out-of-order delivery. Strict per-endpoint
FIFO would head-of-line block every later event behind a slow/poisoned one and
therefore cannot be an implicit engine default. If a business event family
requires order, it must supply a partition/sequence and an explicit recovery
contract; publication order alone is not receiver processing order.

Endpoint health and automatic disable are likewise policy. A destination may
need independent concurrency/rate limits, retry budgets, backpressure, and a
pause/disable state so it cannot starve healthy endpoints. Specification must
distinguish whether disabling blocks only new materialization, also cancels
scheduled retries, or freezes all claims, and what re-enable/redrive does.
Shopify's subscription removal, provider pause controls, and GitHub's lack of
automatic retries demonstrate that no single behavior is portable.

A recent [Paddle engineering interview](https://paddle.engineering/blog/hookdeck-podcast-event-delivery-scale/)
describes production traffic as extremely spiky and moves failed-destination
retries to a lower-priority queue to contain noisy neighbors. A January 2026
[webhooks-at-scale talk](https://noti.st/leggetter/RqpGVK/slides) argues that
vendor retry is not a guarantee and carries reconciliation, DLQ, backpressure
and observability as operator obligations. Both are practitioner/vendor
counter-evidence for fairness and recovery proof, not authority for this
template's queue topology or retry policy.

### Retention, redrive, audit, and ambiguous recovery

Retention is not one TTL. Separate horizons exist for immutable event bytes,
active/retryable deliveries, terminal outcomes, per-attempt diagnostic/audit
records, operator actions, destination generations, key references, and
receiver deduplication guidance. Payload privacy/deletion obligations may
shorten one horizon while operational redrive requires another. No repository
or surveyed standard supplies the business values, so Research does not invent
them.

Redrive must retain the delivery ID and exact accepted payload/version while
creating a fresh attempt timestamp/signature and an audited operator action.
It must be serialized with automated retry or explicitly coexist; Stripe's
manual resend not cancelling automatic retry is direct evidence that otherwise
an operator action can create concurrent duplicates. A manual “confirm
delivered” action cannot establish receiver truth unless a receiver-owned
receipt or lookup exists. Without one, it records an operator risk decision,
not delivery fact.

Recent failure reports counter the assumption that buying or adopting a system
removes these problems:

- A [Hookdeck incident](https://status.hookdeck.com/incident/613300?mp=true)
  reported attempts that were actually delivered but remained pending because
  completion was not recorded. Its incident history also includes workers not
  recovering after a database switchover and scheduled retries not being
  respected.
- A [Svix self-host issue](https://github.com/svix/svix-webhooks/issues/1511)
  reports accepted/enqueued messages for which workers produced no attempts.
  This is evidence about one self-hosted failure, not hosted-service incidence
  or comparative availability.
- GitHub's [July 2026 incident report](https://www.githubstatus.com/history)
  describes a rollout that skipped persistence for millions of delivery
  records: most requests were still delivered, but missing records made the UI,
  API evidence, and redelivery unavailable. Other 2026 reports tie webhook
  delays to queue capacity and connection/rebalance failures.
- [Hookdeck Outpost release notes](https://github.com/hookdeck/outpost/releases)
  include fixes so proxy infrastructure failures do not consume endpoint
  failure budgets, plus worker-recovery and migration-order repairs. These are
  useful source-level counter-evidence that failure attribution, dependencies,
  and schema/lifecycle operations remain part of a self-hosted adoption.
- A 2026 [Papra webhook SSRF advisory](https://github.com/papra-hq/papra/security/advisories/GHSA-5g86-85rp-f9hx)
  documents registered webhook redirects reaching internal addresses. It is
  concrete counter-evidence against registration-time URL validation without
  redirect/dial-time enforcement.

The durable claim therefore needs coherent delivery state plus append-only
attempt/operator evidence and reconciliation. No family can promise that a
remote side effect and local finalization commit atomically.

### Observability without a new data leak

The operator questions are: how many deliveries are ready/scheduled/in-flight,
retryable/ambiguous/terminal/disabled; how old is the oldest in each bounded
state; are claims and observations fresh; which bounded outcome/error class is
changing; are healthy endpoints progressing; are retries/redrives exhausting
budgets; and are SSRF, timeout, response-bound, or rotation failures occurring.
Delivery spans should link to the business-event origin and attempt, while an
authenticated diagnostic path can locate one event/delivery/operator action.

Destination URLs, tenant/subscriber IDs, delivery IDs, status codes with
unbounded values, payloads, response bodies, secrets, and signatures do not
belong in metric labels. IDs belong in bounded logs/traces or protected query
surfaces; payloads and secrets should not be logged. Per-endpoint health metrics
need bounded aggregation rather than a label per customer endpoint. The current
outbox's bounded state/error vocabulary, backlog age, freshness, linked spans,
and shutdown ordering are the local pattern.

## Candidate map

The comparison concerns only the delivery-engine slot. A portal, destination
API, business-event catalog, or broker may complement a family but is not proof
that the slot is closed.

| Candidate family | Durable acceptance and identity fit | What it reuses or buys | Material gaps / ownership cost | Research disposition and reopen condition |
| --- | --- | --- | --- | --- |
| Current outbox unchanged; one row per concrete destination | Caller transaction can append N rows, but each current ID is an independent broker-publication identity. There is no parent business event or destination generation. | Existing PostgreSQL durability, claims/leases, sticky ambiguity, relay lifecycle, bounded telemetry, audit/redrive concepts | Feature code owns fan-out and N identities; publisher acknowledgement is not HTTP policy; no endpoint/secret authority, attempt/response ledger, per-delivery `Retry-After`, health isolation, or destination generation | **Insufficient for the intended general capability.** Reopen only if scope is reduced to a fixed, low-fan-out set where feature-owned fan-out, static secrets, and current common retry policy are accepted constraints. |
| Evolve/reuse the outbox schema and relay for destination deliveries | Can add business-event, destination-generation, delivery and attempt identities, but those alter the outbox's current one-publication contract | Maximum code/lifecycle reuse and one PostgreSQL transaction | Mixed broker/HTTP policy risks a generic queue with incompatible acknowledgements, retention, response evidence, and health scheduling; migration and compatibility surface increase | **Not a literal reuse; equivalent in substance to adding a per-destination store.** System / Integration Design may consider shared primitives or a deliberate generalized owner only if both policies retain independent contracts. |
| Distinct PostgreSQL event/delivery/attempt store and webhook worker | Can represent one immutable event snapshot and N independent delivery rows with separate attempt evidence; durable fan-out expansion must be same-transaction or an idempotent handoff with reconciliation | Existing PostgreSQL, transaction boundary, relay lifecycle/fencing/telemetry patterns, and bounded-HTTP building blocks without importing broker semantics | New schema/migrations, choice and proof of the expansion boundary, dynamic destination and secret authority, HTTP-specific scheduler/classifier, retention/reconciliation, worker/profile lifecycle, and fairness/capacity proof | **Viable local-native family with the fewest unowned integration boundaries.** This is a fit finding, not architecture selection. Reopen against measured scale, a mandated external control plane, or a requirement to avoid PostgreSQL delivery load. |
| Mature self-hosted webhook system | Product normally owns event/message, endpoint and attempt identities; integration must still make business commit -> product acceptance durable without an unprotected dual write | Fan-out, signing, retries, endpoint controls, portals, and operator UI may already exist | Another deployable and data/control plane; PostgreSQL/Redis/broker dependencies; upgrades, backups, HA, schema/license/security response; profile is no longer a small in-process pack; event handoff/reconciliation remains | **Viable only as a system integration, not a library substitution.** A concrete product/version must prove transactional handoff, exact feature coverage, dependency and license policy, operational ownership, and exit path. |
| Managed webhook delivery service | Provider owns deliveries/attempts after its durable acceptance; local transaction still needs an outbox/idempotent provider-ingress handoff and reconciliation | Lowest local worker/operator implementation; provider may supply portals, signing, retry, analytics and support | Availability/control/data residency, secret custody, egress, quotas/cost, payload retention/privacy, provider identity/version semantics, outage visibility, export/exit, and inability to atomically couple provider acceptance to local DB commit | **Viable only with product and external-service inputs absent from the repository.** Reopen with workload/SLO, regions/compliance, budget, payload classification, provider-ingress idempotency and outage/reconciliation evidence. |

Representative systems refine rather than add families:

- [Svix](https://github.com/svix/svix-webhooks) is an actively maintained MIT
  self-hosted webhook service with PostgreSQL and Redis-related operational
  requirements, and also has a managed offering. Its endpoint/fan-out/signing,
  retry and portal breadth reduce feature code but expand the deployment and
  subscriber-management boundary. Its source keeps [endpoint](https://github.com/svix/svix-webhooks/blob/21edf261f0971d927765e69212c7e5b466ef7598/server/svix-server/src/db/models/endpoint.rs#L18-L39)
  and [attempt](https://github.com/svix/svix-webhooks/blob/21edf261f0971d927765e69212c7e5b466ef7598/server/svix-server/src/db/models/messageattempt.rs#L12-L32)
  state distinct and places signing, HTTP execution, retry and disablement in
  its [worker](https://github.com/svix/svix-webhooks/blob/21edf261f0971d927765e69212c7e5b466ef7598/server/svix-server/src/worker.rs#L174-L245).
- [Hookdeck Outpost](https://github.com/hookdeck/outpost) is an Apache-2.0 Go
  system aimed at multi-tenant event destinations with at-least-once delivery,
  retries, signing and OpenTelemetry. It is newer and brings PostgreSQL plus a
  queue/broker/control-plane surface, so production maturity and exact-version
  proof remain open. Its [delivery handler](https://github.com/hookdeck/outpost/blob/d1a3309cb4f71ac8839a2249bfaaca7e5056903e/internal/deliverymq/messagehandler.go#L67-L186)
  and [retry queue](https://github.com/hookdeck/outpost/blob/d1a3309cb4f71ac8839a2249bfaaca7e5056903e/internal/deliverymq/retry.go#L44-L166)
  confirm that these mechanics require queue acknowledgements and an attempt
  source of truth rather than an HTTP helper alone.
- [Convoy](https://github.com/frain-dev/convoy) remains an active self-hosted
  gateway but its [Elastic License 2.0](https://github.com/frain-dev/convoy/blob/d597a15e433fadf61b3548b6afdcc1124ff98e4d/LICENSE#L1-L38)
  makes it source-available rather than an unqualified OSS dependency. Its
  [worker](https://github.com/frain-dev/convoy/blob/d597a15e433fadf61b3548b6afdcc1124ff98e4d/internal/dataplane/worker.go#L57-L140)
  composes PostgreSQL, Redis-backed jobs, attempt storage, retry, circuit
  breaking and retention, adding both an operational and legal/support gate.
- AWS EventBridge API Destinations are a managed generic eventing family, not a
  complete subscriber-management system. Their public-certificate, timeout,
  retry-age/attempt, rate, connection/secret and DLQ constraints would become
  product behavior if adopted.

A broker/event bus by itself is not a fifth delivery-engine family. It can
durably carry accepted work to a consumer, but the consumer still needs every
per-destination identity, HTTP, ambiguity, retry, audit and recovery semantic
above. Likewise, “no capability” remains the correct unselected template
profile but does not meet the accepted enabled-profile outcome.

The current evidence eliminates unchanged outbox reuse as a general answer.
It leaves a distinct PostgreSQL-native store and concrete self-hosted/managed
systems as viable families. The local-native family has the fewest presently
unowned boundaries; adopting an external system can dominate only after the
missing deployment, cost, data, scale, and operational inputs are supplied.
Specification should therefore describe the behavioral delivery-engine
boundary independently of which viable family later implements it.

## Downstream constraints and proof obligations

### Constraints carried into Specification

These are research conclusions that constrain the next phase; they are not a
completed behavioral contract:

- Preserve separate business-event, destination-generation, delivery, attempt,
  and operator-action identities. Delivery identity is stable through retry
  and redrive; attempt identity/time is not.
- Freeze the exact accepted payload and its business schema/version. Treat
  delivery-envelope and signature versions independently. Never silently
  rebuild old bytes through current business serialization.
- Make subscriber authorization, event selection, endpoint lifecycle,
  destination ownership verification, version preference, and secret
  provisioning explicit external authorities. The engine consumes their
  snapshot; it does not invent them.
- Admit arbitrary destinations as an SSRF trust boundary: validate syntax and
  policy at registration/use, re-check the actual dial address on every new
  connection for IPv4 and IPv6, refuse redirects, prevent proxy/resolver
  bypass, and retain network egress defense in depth.
- Bound DNS/connect/TLS/write/header/body, decompression, response evidence,
  concurrency, retries, total age, and shutdown. Never hold a PostgreSQL
  transaction during network I/O or sleep a claimed worker for backoff.
- Model at-least-once delivery and ambiguous outcomes. A 2xx is receiver HTTP
  acceptance, not business processing; no response after possible send is not
  proof of failure. Stable delivery identity enables receiver deduplication but
  does not provide exactly once.
- Refuse redirects and treat endpoint change as subscription state. Define
  retryable/permanent/local-deny/ambiguous classes, bounded `Retry-After`,
  endpoint isolation, and retry/disable/redrive interaction explicitly rather
  than inheriting a provider table.
- Do not imply ordering. Any ordered event family must own its partition,
  sequence, head-of-line and poison recovery contract. Fan-out must snapshot
  subscribers and isolate destination progress.
- Separate payload, active delivery, terminal/attempt audit, operator action,
  destination-generation, key-reference, and receiver-dedup retention horizons.
  Do not invent numerical values without business/privacy/recovery inputs.
- Redrive uses the same delivery and exact payload with a fresh attempt and
  signature, is safe against concurrent automated retry, and is audited.
  Manual confirmation without receiver evidence is a risk decision, not truth.
- Keep metrics bounded and secret/payload/URL safe; provide protected
  delivery-level diagnosis and link event origin -> delivery -> attempt.
- Scope every destination lookup, send, disable, redrive, secret read and
  rotation to its authorized owner at the data-access boundary; opaque-ID
  guessing must not cross that boundary.
- Make the capability independently selectable and removable at template init.
  Profile selection must express its database, worker, HTTP, secret, telemetry,
  migration and optional external-system dependencies without coupling
  webhook selection to broker messaging.

### Policy inputs still absent

Specification cannot honestly choose numerical or business policy without
owners for: authorized registrants and endpoint verification; static versus
dynamic subscriptions; payload privacy/classification and deletion; replay and
dedup horizons; delivery SLO and retry age/attempt budget; response-evidence
retention; automatic pause/disable and re-enable semantics; event ordering
families; secret residence/rotation authority; per-endpoint and global rate/
concurrency budgets; and whether a customer-facing portal/API is in scope.

These are downstream decision inputs, not reasons to reopen Research unless a
new source changes the candidate set. Specification may use explicit bounded
assumptions with reopen conditions where repository authority is absent.

### Proof obligations carried downstream

| Surface | Required proving cases |
| --- | --- |
| PostgreSQL durability and concurrency | Real PostgreSQL proof of the selected local acceptance boundary: either business mutation plus event/delivery acceptance is atomic, or event-to-delivery expansion is idempotent and reconciled; concurrent claim/fence behavior; crash before send, after remote effect/before response, after 2xx/before finalize, after finalize/before evidence publication; lease expiry/restart; stale worker; N-destination independence; retry/redrive and disable/claim races; destination-generation update; retention versus active/redrive rows. |
| Receiver-visible identity and signing | Exact raw-byte signatures; stable delivery ID with distinct attempts; wrong/missing/expired signature; old+new overlap and retirement; key-generation audit without secret leakage; replay-window behavior distinct from durable dedup; receiver fixture proving duplicate detection and observable out-of-order delivery. |
| Destination and network security | Registration and use-time URL cases; A/AAAA public/private/special ranges and mixed answers; rebinding between validation and retry; actual dial-address enforcement; redirects; proxy/custom-resolver bypass; TLS name/cert failure; owner/tenant-scoped access; credential/URL/log redaction; egress-policy integration. |
| Bounded HTTP and scheduling | Deterministic DNS/connect/TLS/write/header/body stalls; cancellation and drain; header/body/post-decompression overflow; connection reuse and caps; proof that one durable attempt cannot hide a transport replay; status/error classes; ambiguous disconnects; `Retry-After` seconds/date/malformed/past/oversized; jitter/cap/age; scheduled retry survives restart without occupying a worker. |
| Fan-out, fairness, health and order | Subscription snapshot and generation semantics; endpoint update/delete/disable/re-enable; one poisoned/slow/high-volume endpoint cannot starve another; per-endpoint/global budgets; default unordered progress; any selected ordered lane's head-of-line and poison recovery; large-fan-out amplification/capacity. |
| Operator recovery and audit | Authenticated locate/inspect; automated retry versus redrive serialization; replay-safe operator action; same delivery/exact payload on redrive; unknown versus terminal distinction; confirmation semantics; retention/deletion; reconciliation when delivery state and attempt evidence diverge. |
| Lifecycle, telemetry and privacy | Startup/readiness/freshness, graceful drain then forced join, DB/HTTP/telemetry cleanup order, bounded metric cardinality, event-to-attempt trace links, oldest-backlog/unknown signals, alert recovery, and absence of payloads, secrets, signatures, response bodies, URLs and subscriber identifiers from unsafe telemetry. |
| Template composition | Contract matrix for webhook unselected and selected profiles across database/outbox/messaging/outbound-HTTP choices; exact owned-path removal; no orphan config/dependency/migration/binary/docs/generated authority; selected profile builds, starts, drains and fails closed when its required secret/store/adapter is absent. |
| Concrete OSS/managed adoption, if selected | Exact version/license/advisory and upgrade proof; feature-contract conformance; provider-ingress idempotency; transactional handoff and reconciliation; quotas/rate/timeout/retry/DLQ behavior; outage and ambiguous-state drills; dependency HA/backups/migrations; data residence/privacy/secret custody; load/cost; export and provider-exit recovery. |

No mock-only proof can establish PostgreSQL arbitration, actual DNS/dial policy,
or remote side-effect ambiguity. The smallest deterministic local fixtures
should be used first, with real PostgreSQL and real loopback/network controls;
any selected external system additionally requires its sandbox and deployed
path before a readiness claim.

## Limits, refresh, and stop rationale

### Source posture and conflicts

Primary authorities in this note are current repository source/schema/docs,
RFC 9110, Go `net/http`, OWASP's security guidance, maintained project source
and licenses, and provider-owned contracts/status reports. Provider docs are
authoritative only for that provider. The Standard Webhooks document is a
useful community interoperability specification, not an IETF standard and not
authority for business retention or retry policy.

Recent provider release and incident reports are counter-evidence, not a
statistical availability comparison. They show possible failure modes—missing
attempt persistence, stale pending state after actual delivery, queue/database
recovery and capacity failure—but do not establish candidate reliability ranks.
Recent articles and talks are used the same way. The
[SREcon25 EMEA retry-avalanche talk](https://www.usenix.org/conference/srecon25emea/presentation/zhen)
connects retries and blocked queues to cascading overload and recommends
end-to-end retry budgets, degradation and queue shedding; it strengthens the
retry-amplification proving obligation but is not webhook-specific policy.
Product versions, licenses, pricing, quotas, regions and incident histories are
freshness-sensitive and must be refreshed when a concrete system enters System
/ Integration Design or procurement. No credentials, provider mutation, load
test, adopter inventory, vulnerability scan, or production path was exercised.

The repository was inspected in a dirty checkout. Unrelated changes were
preserved. Current-tree durable-jobs artifacts were treated as uncommitted
collision evidence, not accepted capability authority. This research does not
claim deployed-adopter state.

### Research stop

RQ-1 closes because every named local surface reaches a reusable contract or
its first unsupported edge. RQ-2 closes because every requested delivery
concern has a primary authority, a policy owner, and failure counter-evidence.
RQ-3 closes because repeated product and alias searches refine the four
families rather than reveal a materially different engine boundary.

The macro phase stops here. Architecture selection, behavioral requirements,
package/schema/process placement, test design, task decomposition, code,
provider choice, and production proof are intentionally absent. New evidence
reopens Research only if it changes the identity/authority boundary, makes
unchanged outbox reuse satisfy independent destination delivery, adds a
materially different candidate family, or invalidates a security/failure
constraint. Otherwise the next authorized macro phase is Specification.

## Standalone prompt for Specification

```text
Work in /Users/daniil/Projects/Opensource/go-service-template-rest using the structured spec-first workflow.

Resume the optional durable outbound-webhook delivery capability at the Specification macro phase. Read AGENTS.md, docs/spec-first-workflow.md, the current Specification phase instructions selected by the router, docs/repo-architecture.md, and specs/outbound-webhook-delivery/research/synthesis.md. Treat the research synthesis as evidence and constraints, not as an already selected architecture.

Produce only the behavioral Specification artifact(s) for a template-init-selected capability pack. Specify the reusable delivery-engine boundary separately from subscriber management and business-event semantics. Resolve or bound the behavior for identity, immutable payload/version evidence, signatures and secret rotation, destination admission and SSRF/DNS-rebinding defense, redirects, bounded HTTP execution and response evidence, retry classification and bounded Retry-After, duplicates and ambiguous outcomes, fan-out and ordering, unhealthy-destination isolation/disable semantics, retention, redrive, auditability, telemetry/privacy, lifecycle, configuration/secrets, and profile composition. Preserve the research finding that unchanged current-outbox reuse is insufficient for general per-destination delivery, while leaving implementation-family selection to the appropriate later design phase. Carry every policy assumption with its owner and reopen condition, and retain the downstream proof obligations without writing Test Design.

Do not write Technical Design, System / Integration Design, Test Design, Planning, tasks, or code. Run the required Specification review/repair loop, then stop at the Specification macro-phase boundary and finish with the standalone prompt for the next routed phase.
```
