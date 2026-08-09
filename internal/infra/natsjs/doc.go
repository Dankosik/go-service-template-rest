// Package natsjs is the concrete NATS JetStream durable-messaging pack: one
// broker connection, an admission-bounded producer, and a durable pull consumer
// that runs feature handlers and settles each message with the broker.
//
// Delivery is at-least-once in both directions. A publish whose acknowledgement
// is lost returns [ErrAmbiguous] and is safe to retry, because the broker
// deduplicates on Event.PublicationID; a handler that runs but whose
// acknowledgement is lost sees the message again. Handlers must tolerate
// duplicates.
//
// Three audiences share the package. Feature code touches only [Event],
// [Message], [Handler], and [Permanent] — all in message.go. Everything else is
// the composition root's: [Connect], [Client.Producer], [Client.NewWorker], and
// the lifecycle methods the two cmd packages drive. The outbox relay uses only
// [NewOutboxPublisher], which converts the broker-neutral event, forwards its
// stored W3C creation context, and maps publication evidence onto the outbox's
// permanent, not-accepted, and ambiguous outcomes.
//
// # The delivery cycle
//
// [Worker.Run] repeats one cycle, and each step has one owner:
//
//   - fetch (worker.go) — up to one message per free handler slot from the
//     durable consumer, so a round trip covers a batch rather than a message,
//     with at most MaxConcurrency handlers in flight either way. The pull is
//     bounded by operationTimeout and an expired pull that delivered nothing is
//     ordinary, not a failure.
//   - admit (worker_delivery.go) — the delivery bounds this worker was configured
//     with, envelope decoding, and the attempt budget. A message that fails any of
//     them is dead-lettered without the handler ever seeing it.
//   - handle (worker_delivery.go) — the feature [Handler] runs under its own span
//     and HandlerTimeout. A panic is caught, reported with sanitized frames, and
//     stops the worker.
//   - settle (worker_delivery.go) — acknowledge, dead-letter, or request a delayed
//     redelivery. This is the single place the delivery policy lives.
//
// Around that cycle: worker_admission.go proves the broker topology before a
// Worker exists, client.go owns the connection and the reconnect state machine,
// producer.go owns publish admission, message_wire.go owns the header contract
// — including the dead-letter reasons, which travel on the wire — config.go
// owns the settings and their validation, errors.go holds the five sentinels
// described below, vocabulary.go holds the metric and log labels with the
// functions that bound them, and telemetry.go holds the instruments, the
// recorders, and the two spans.
//
// # Extending it
//
// To consume a new subject, write a [Handler] and register it from
// cmd/worker/main.go, which passes a HandlerBuilder into
// cmd/worker/internal/bootstrap. The template ships that builder unregistered:
// there is no noop fallback, and a worker without one fails startup.
//
// One worker consumes one durable consumer under one WorkerConfig.FilterSubject,
// and a client owns at most one worker, so a second subject is a second worker
// process — or one handler that switches on [Message.Subject] behind a single
// filter wide enough to cover both. Neither is more correct; a separate process
// gets its own attempt budget, concurrency, and dead-letter stream, and a shared
// handler gets one deployment.
//
// There is deliberately no seam for a second broker. [Producer], [Message], and
// [Handler] are concrete throughout, including in the composition roots'
// signatures, so replacing NATS means editing those roots and every handler
// rather than implementing an interface. Broker choice is abstracted one layer
// out instead: postgresoutbox.Publisher is the seam, and this package is one
// implementation behind it. Add an interface here only when a second
// implementation actually exists.
//
// Three things about a handler are not visible from its signature. It gets
// 1+len(RetryDelays) deliveries in total — the first, plus one per configured
// delay — and then goes to the dead-letter stream whatever it returns. Wrapping
// its error in [Permanent] spends that budget immediately instead, for a
// rejection retrying the same bytes cannot fix. And its context is cancelled
// only by forced shutdown, so returning promptly on cancellation is what lets a
// drain finish inside its budget.
//
// To publish, take the [Producer] the same builder receives, or in an API
// process the one cmd/service/internal/bootstrap wires. Publishing is bounded
// rather than queued: beyond MaxPendingPublishes a call is rejected with
// [ErrCapacity], so a stalled broker becomes back pressure on the caller.
//
// Both roles connect through [Connect] with the matching [Role], which decides
// only what the readiness metric reports. A client owns at most one [Worker].
//
// # Errors
//
// Five sentinels cover every failure this package produces, and the composition
// root routes on them: [ErrRejected] for anything the broker or this package
// refused, [ErrAmbiguous] for a publish whose outcome is unknown, [ErrCapacity]
// and [ErrDraining] for the two admission refusals, and [ErrTerminal] for a
// fault that must stop the process.
//
// See docs/durable-messaging.md for the operator-facing contract and
// docs/repo-architecture.md for the repository's extension seams.
package natsjs
