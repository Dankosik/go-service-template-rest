# Design: outbox trace continuity and ordering-key lifecycle

status: ready
Realizes `../spec.md`. Records only the decisions implementation would otherwise
invent, plus the one that reopened Specification.

## Scoping down

System / Integration Design is not run as a separate step. Concrete reason: the
outcome adds no deployable, no integration adapter, no outbound target, and no
cross-service contract. Every change lands inside one existing package, its
already-owned schema, and its already-owned operator document, so the only open
questions are representation, placement, and Go ownership.

## D-A — Trace context is a column, not metadata

`outbox_events` gains `trace_context bytea NOT NULL DEFAULT '\x7b7d'`, checked
`IS JSON OBJECT` and bounded to 1024 bytes. It carries the W3C propagator's own
carrier shape — a flat JSON object of header name to value — so the stored bytes
are whatever the configured propagator produced, not a shape this package
invented.

The migration is edited in place rather than added as `000002`. `migrations/` is
absent from `template-owned.paths`, so it is never mirrored into an already
running derived service; only a newly initialized one consumes it, which is the
pack's existing "one migration holds the whole schema" rollout decision.

## D-B — The new field gets its own allowance (reopened by the spec)

`spec.md` required this choice to be made explicitly because it decides whether
a previously valid event becomes rejected.

Decision: `trace_context` is **not** charged against `maxEnvelopeBytes`. It has
its own `maxTraceContextBytes` bound. Charging it against the shared budget
would reject events that a service is appending successfully today, purely
because the outbox started storing something of its own.

Consequence, and it must be documented: the stored row can exceed the
caller-facing envelope budget by at most the trace-context allowance. The
caller-facing contract — what a caller must keep under 288 KiB — is unchanged.

## D-C — Go surface: unexported field, exported accessor

`Event` gains an unexported `traceContext propagation.MapCarrier` and an
exported `CreationContext() propagation.MapCarrier`.

Unexported is what makes R1's "outbox-owned" real: a caller constructing an
`Event` literal cannot set or forge it, and `Append` captures it from the
calling context instead. `Claim` restores it so the adapter can read it off the
event it is handed.

`propagation.MapCarrier` rather than `map[string]string` so an adapter closes
the loop in one call — `otel.GetTextMapPropagator().Extract(ctx, event.CreationContext())`
— without this package choosing the broker's header encoding. The returned map
is live rather than copied, for the same reason `http.Header` is: copying it per
event would allocate on the publish path for a value every adapter only reads.

## D-D — The publish span links to the creation context; it is not its child

**This reopened `spec.md` success criterion 1**, which asked for one shared trace
identity. Amended there.

An outbox publication can happen minutes, or after a redrive days, after the
append. Parenting the publish span to the producing request would keep that
request's trace open for the whole backlog and redrive horizon, past the trace
assembly window of every backend that has one, and would make one slow ordering
key extend an unrelated request's trace. The link carries the same join without
that lifetime coupling, and it is what the OpenTelemetry messaging convention
prescribes for a send that has a separate creation context.

Shape: one span per publication attempt, `SpanKindProducer`, named
`publish {destination}`, a root of its own trace, linked to the extracted
creation context.

Attributes are only the bounded ones this package can honestly own:
`messaging.operation.name` and `messaging.destination.name`.
`messaging.system` is deliberately absent — the relay is broker-neutral and
only the adapter knows the system; an adapter that wants it sets it on its own
span. The event ID and ordering key never reach the span. The stricter identity
privacy rule in the later ready inbox and production-closure specifications
supersedes this design's earlier `messaging.message.id` choice without changing
the trace-parent/link decision.

A failed publication records the error and sets the span status with the same
bounded error class the publish metric carries, so a trace and a dashboard name
one condition.

## D-E — No batch span

Rejected. A span per cycle would add a trace whose only content is fan-out that
`outbox.relay.inflight` and the claim metric already report. Reopen if operators
need the publish fan-out shape itself.

## D-F — The tracer comes from the global provider

`NewTelemetry` keeps its signature and takes its tracer from
`otel.GetTracerProvider()`, mirroring the meter fallback it already has. The
relay bootstrap already installs a real provider and the global W3C propagator,
so this reaches a configured pipeline without a new constructor argument or a
new configuration key.

## D-G — Ownership and file map

| File | Change | Why here |
| --- | --- | --- |
| `migrations/000001_postgres_outbox.sql` | `trace_context` column, check | Canonical schema authority |
| `internal/infra/postgres/queries/postgres_outbox.sql` | Column in both inserts, claim projection, `Get`; new retire statement | Canonical statement authority |
| `internal/infra/postgresoutbox/event.go` | Field, accessor, bound, carrier codec | Owns the envelope and its limits |
| `internal/infra/postgresoutbox/store_append.go` | Capture from `ctx` into the column arrays | Owns the append statement |
| `internal/infra/postgresoutbox/store_rows.go` | Restore on claim and `Get` | Owns row mapping |
| `internal/infra/postgresoutbox/store_retire.go` | **New.** `RetireOrderingKeys` | Ordering-key lifecycle is an independently changing responsibility, and the `store` prefix is the lint contract for driver access |
| `internal/infra/postgresoutbox/relay_publish.go` | The publish span | Owns one publication attempt |
| `internal/infra/postgresoutbox/telemetry.go` | Tracer, span helper | Owns instruments |
| `internal/infra/postgresoutbox/errors.go`, `vocabulary.go` | `ErrOrderingKeyActive`, `retire_ordering_key` | Own the sentinel set and the closed label vocabularies |

`RetireOrderingKeys` does **not** join `failureClass` in
`cmd/outbox-relay/main.go`: it is feature-facing and cannot stop the relay.

## D-H — Retirement is one statement under the append's own lock

The precondition and the delete are one statement. It locks the head rows
`FOR UPDATE` in key order — the same rows and the same order the ordered append
takes — which is what makes R4's concurrency rule hold: a racing append for the
same key serializes against the head lock either before the retirement, in which
case its unpublished event blocks it, or after, in which case it establishes a
fresh mark.

The statement returns the keys it refused rather than the keys it removed, the
same rejection-report shape the ordered append already uses. A key with no head
is neither refused nor removed, which is R4's idempotent case.
