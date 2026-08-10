// Package postgresoutbox publishes domain events through PostgreSQL instead of
// through the request path. A feature stores its event in the same transaction as
// the domain mutation that produced it, and a separate relay process later hands
// that durable intent to a broker. The API process therefore never calls a
// broker, and a broker outage appears as observable backlog rather than as a
// dual-write failure on the request path.
//
// Delivery is at-least-once. A crash between broker acknowledgement and the
// PostgreSQL update republishes the same event ID and bytes after the lease
// expires, so consumers deduplicate whatever is not naturally idempotent.
//
// Three audiences share one [Store]. The write path calls [Store.Append],
// inside the transaction that owns its mutation, and [Store.ReconcileCommit]
// afterward when that commit's own result was lost — that path is a PostgreSQL
// repository adapter rather than a feature package, because depguard denies a
// feature package both internal/infra and pgx; see the ownership split in
// docs/postgres-transactional-outbox.md. The relay process owns [Store.Claim],
// the mark and retry family, [Store.CleanupPublished], and [Store.Observe].
// [Store.Redrive] is operator tooling, and so is [Store.Get] — but the relay
// reads it too, to resolve a finalization its batch statement did not report, so
// Get's error semantics belong to both.
//
// # The relay cycle
//
// [Relay.Run] repeats one cycle in relay.go, and each step has one owner in this
// package — one file on each side of the store boundary, so a step can be read
// end to end without the rest of the cycle:
//
//   - claim (relay.go, store_claim.go) — [Store.Claim] leases up to BatchSize
//     eligible rows under one fresh token; that token fences the batch against
//     another relay replica for the whole cycle, and expiry is what recovers a
//     crashed relay's work.
//   - publish (relay_publish.go) — the batch goes to [Publisher] through at most
//     PublishConcurrency concurrent calls sharing one deadline, derived in
//     publishAll from the earlier of PublishTimeout and the lease. At most one
//     event per ordering key is ever claimable, so concurrency cannot reorder a
//     key. Because that budget belongs to the batch, its tail can find it spent;
//     such an event is reported as errPublicationNotAttempted rather than handed
//     a dead context.
//   - classify (relay_finalize.go) — classify turns each outcome into exactly one
//     durable transition: published, retried under full-jitter backoff, poisoned
//     for operator redrive, or — for the events publication never reached —
//     released immediately with the claim's attempt given back. It is the single
//     place the delivery policy lives.
//   - finalize (relay_finalize.go, store_finalize.go) — one statement per
//     transition covers the whole lease, so a backlog costs round trips per batch
//     rather than per event. Finalization is detached from process cancellation,
//     because an acknowledged event left unmarked becomes a duplicate.
//   - maintain (relay_maintain.go, store_maintenance.go) — periodic
//     [Store.Observe] feeds readiness and the backlog gauges; periodic
//     [Store.CleanupPublished] deletes retained published rows in bounded batches.
//
// The budget every step reads is relay_config.go, and store_operator.go holds
// what runs outside the cycle: [Store.Get] and [Store.Redrive].
// store_legacy_classification.go holds the other thing outside it,
// [Store.ClassifyLegacyUncertainty], which a service runs once at bootstrap to
// resolve rows an older schema left unclassified.
//
// Beside the cycle: event.go owns the [Event] envelope and its size limits,
// publisher.go the broker contract, notify.go the append listener that turns a
// commit into a wake-up, store_receipt.go the append receipt and the
// [Store.ReconcileCommit] answer a writer needs when its own commit response was
// lost, relay_ready.go the readiness policy the diagnostics probe and the metric
// callback share, errors.go the sentinel set, vocabulary.go the closed metric
// and log vocabularies with the functions that bound them. telemetry.go owns
// metrics and scrape state, telemetry_publish.go publish spans, and
// telemetry_log.go operator records.
//
// # Trace continuity
//
// An event carries the trace context that was active when it was appended, so a
// publication minutes or days later can still name the operation that produced
// it. tracecontext.go owns both directions: [Store.Append] captures it, and
// [Store.Claim] restores it into [Event.CreationContext] for the adapter to put
// on broker headers.
//
// It is stored in its own column rather than in [Event.Metadata], because
// metadata is the caller's bytes and merging into them would break that. The
// relay's publish span links to the restored context instead of descending from
// it — see Telemetry.StartPublish for why a child would be wrong.
//
// Nothing about it can fail a delivery. A context that cannot be captured is
// stored as absent and counted; one that cannot be decoded reads as absent. A
// telemetry field must never be the reason a business event is rejected or a
// publication fails.
//
// # Ordering-key lifetime
//
// outbox_ordering_heads keeps one row per ordering key ever appended, and
// cleanup never removes one: the retained high-water mark is what rejects a
// replayed sequence, so it has to outlive the events it came from.
// store_retire.go is the only way a row leaves — [Store.RetireOrderingKeys], an
// assertion by the feature that owns the aggregate that its key is terminal.
// Nothing infers that from idleness, because a quiet key is not a finished one.
//
// # Extending it
//
// Three extensions are expected. The first two do not require reading the relay
// loop; the third is the relay loop, and its cost is stated last.
//
// To emit a new event type, the repository adapter declares the narrow local
// interface it consumes — normally just [Store.Append] — and accepts *[Store]
// through that interface. It appends inside the transaction that owns the
// mutation, so an append error rolls back both. The worked wiring and batching
// guidance live in docs/postgres-transactional-outbox.md.
//
// With the NATS profile selected, cmd/outbox-relay registers the concrete
// natsjs adapter and supervises its client lifecycle. An outbox-only generated
// service deliberately keeps the builder nil and fails before claiming work.
// To publish through a different broker, implement [Publisher] and register a
// complete runtime in cmd/outbox-relay/main.go. [Publisher] owns the closed
// result vocabulary, concurrency contract, and shared batch deadline.
//
// To add a stage to the cycle itself — a verification step, a second publish
// attempt, an audit write — expect to edit relay.go's runLoop or runCycle, the
// new stage's own relay_*.go file, boundedOperation in vocabulary.go, doc.go's
// cycle list above, and, if the stage can stop the relay, both errors.go and
// failureClass in cmd/outbox-relay/main.go. Four rules are not visible from the
// place a stage is added, and each is owned somewhere else:
//
//   - Stop-reason precedence. A cancelled batch reports an ordinary drain even
//     when something else also failed in it. A reason added below that check in
//     publishBatch inherits that precedence; one that must outrank a shutdown in
//     progress goes above it. relay_publish.go owns this.
//   - Which deadline applies. Everything after publication runs under the lease
//     the batch was claimed under, on a context detached from process
//     cancellation, because an acknowledged event left unmarked becomes a
//     duplicate. A stage that writes durable state belongs on that side of
//     publishBatch, not before it.
//   - Whether the stage carries a duration. boundedOperation in vocabulary.go
//     states the unit of every operation label; a stage measuring one span end to
//     end uses Telemetry.RecordOperation, and one reporting a verdict reached
//     partway through uses CountOperation. An operation missing from that switch
//     reports "other" rather than failing.
//   - Whether its failure stops the process. Returning an error from a stage ends
//     the cycle; only the two faults finalizationFaultTolerance names are absorbed
//     and retried, and relay_finalize.go owns why those two and no others.
//
// The schema, the claim and finalization statements, and this package's Go code
// are one unit: a change to delivery state belongs in
// migrations/000001_postgres_outbox.sql and
// internal/infra/postgres/queries/postgres_outbox.sql first, then in the
// regenerated sqlc output, then here.
//
// # Where the driver may appear
//
// File names here are a lint contract, not a matter of taste. .golangci.yml
// exempts store*.go from the postgres_driver_boundary and sqlc_boundary rules and
// notify*.go from the first, because those are the PostgreSQL adapters; the relay
// loop and the envelope stay driver-free. That glob is why the statements split
// as store.go, store_append.go, store_claim.go, store_finalize.go,
// store_legacy_classification.go, store_maintenance.go, store_operator.go,
// store_retire.go, and store_rows.go: a file named anything
// else moves pgx and
// the generated types outside the exemption, and depguard then reports the import
// rather than the rename that caused it. Keep the prefix, or move the exemption
// with the code. The same holds for the test files that hold the driver doubles,
// which is why they are store_fixtures_test.go and store_*_test.go rather than a
// plain fixtures_test.go.
//
// The contract binds this package only. `make lint` passes no build tags, so
// files behind `//go:build integration` — including the outbox suite in test/,
// which imports pgx directly — are not analyzed by depguard or any other linter.
// Do not read a green lint run as proof that an integration test respects these
// boundaries.
//
// See docs/postgres-transactional-outbox.md for the operator-facing contract and
// docs/repo-architecture.md for the repository's extension seams.
package postgresoutbox
