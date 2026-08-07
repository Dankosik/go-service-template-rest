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
// Three audiences share one [Store]. A feature calls only [Store.Append], inside
// the transaction that owns its mutation. The relay process owns [Store.Claim],
// the mark and retry family, [Store.CleanupPublished], and [Store.Observe].
// [Store.Redrive] is operator tooling, and so is [Store.Get] — but the relay
// reads it too, to resolve a finalization its batch statement did not report, so
// Get's error semantics belong to both.
//
// # The relay cycle
//
// [Relay.Run] repeats one cycle, and each step has one owner in this package:
//
//   - claim — [Store.Claim] leases up to BatchSize eligible rows under one fresh
//     token; that token fences the batch against another relay replica for the
//     whole cycle, and expiry is what recovers a crashed relay's work.
//   - publish — the batch goes to [Publisher] through at most PublishConcurrency
//     concurrent calls sharing one deadline, derived in publishAll from the
//     earlier of PublishTimeout and the lease. At most one event per ordering key
//     is ever claimable, so concurrency cannot reorder a key.
//   - classify — classify in relay.go turns each outcome into exactly one durable
//     transition: published, retried under full-jitter backoff, or poisoned for
//     operator redrive. It is the single place the delivery policy lives.
//   - finalize — one statement per transition covers the whole lease, so a backlog
//     costs round trips per batch rather than per event. Finalization is detached
//     from process cancellation, because an acknowledged event left unmarked
//     becomes a duplicate.
//   - maintain — periodic [Store.Observe] feeds readiness and the backlog gauges;
//     periodic [Store.CleanupPublished] deletes retained published rows in bounded
//     batches.
//
// # Extending it
//
// Two extensions are expected, and neither requires reading the relay loop.
//
// To emit a new event type, build an [Event] and pass it to [Store.Append] inside
// the transaction that owns the mutation. Append never begins or commits a
// transaction, so returning its error rolls back the mutation and the event
// together. Pass every event of one business transaction in a single variadic
// call: each column travels as one array, so the call costs one statement
// whatever mix of ordered and unordered events it carries. A rejected field
// returns [ErrInvalidEvent] before any statement is sent; a replayed ordering
// sequence returns [ErrOrderingSequence] from the append statement itself,
// because only PostgreSQL holds the retained high-water mark. Either way the call
// stores nothing.
//
// To publish through a different broker, implement [Publisher] and register the
// builder in cmd/outbox-relay/main.go, which ships deliberately unregistered:
// there is no production noop fallback. [Publisher] documents the whole
// acceptance contract, including concurrency safety, the shared batch deadline,
// and when to return [ErrPermanentPublication] versus [ErrPublicationNotAccepted].
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
// loop and the envelope stay driver-free. Splitting store.go into a file named
// anything else moves pgx and the generated types outside the exemption, and
// depguard then reports the import rather than the rename that caused it. Keep
// the prefix, or move the exemption with the code.
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
