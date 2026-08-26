// Package natsjs is the concrete NATS JetStream adapter behind typed domain
// events.
//
// Business composition declares [Route] values, builds a [Registry], registers
// typed handlers through [Registry.Handle], and publishes domain events
// values through [Publisher]. Subjects, headers, publication attempts, delivery
// metadata, acknowledgements, retries, and dead letters do not enter feature
// handlers.
//
// Publication is at-least-once. [Producer] waits for a JetStream
// acknowledgement; an unanswered request returns [ErrAmbiguous] and must be
// retried with the same logical event identity. Broker deduplication lasts only
// for the stream's configured window, so consumer effects remain durably
// idempotent by event ID across delivery, redrive, and replay.
//
// [Worker] runs one serial native consume context per configured concurrency
// slot. The NATS client owns pulls, reconnect, buffering, and callback drain.
// This package owns only the aggregate process boundary and delivery settlement:
// success uses a confirmed acknowledgement, retryable failure requests delayed
// redelivery, and permanent or exhausted work is published to the dead-letter
// stream before the source is acknowledged. A lost acknowledgement can still
// duplicate either delivery or dead-letter transfer.
//
// Source and dead-letter streams are operator-owned. [Client.NewWorker] creates
// or updates only the durable consumer settings coupled to settlement. Broker
// stream/consumer metrics belong to deployment monitoring; this adapter emits
// only publish, handler, and dead-letter operation telemetry.
//
// [ErrRejected] is a definite refusal, [ErrAmbiguous] an unknown publish
// outcome, [ErrDraining] a lifecycle refusal, and [ErrTerminal] a fault that
// stops the worker process.
package natsjs
