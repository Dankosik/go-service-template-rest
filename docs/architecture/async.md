# Async Architecture

Load for a queue, job, outbox, webhook, event, or background-process change.

<!-- profile:outbox-postgres:start -->
The PostgreSQL outbox appends a typed event as a River job in the same `pgx.Tx`
as the mutation. `internal/infra/natsjs` maps event versions to subjects and
publishes stored jobs; `cmd/outbox-relay` owns the worker process. Profile
initialization requires both outbox and NATS.
<!-- profile:outbox-postgres:end -->

<!-- profile:jobs-postgres:start -->
The PostgreSQL jobs pack delegates queue state and lifecycle to River, retains a
separate worker binary, stays inert without a business worker builder, and uses
`InsertTx` for caller-owned transactions.
<!-- profile:jobs-postgres:end -->

<!-- profile:webhooks-durable:start -->
Durable webhooks prepare immutable fan-out before the feature transaction and
insert one River job per receiver inside it. `cmd/jobs-worker` runs the worker;
the adapter signs one bounded public-HTTPS request and maps the outcome into
River's completion/retry contract.
<!-- profile:webhooks-durable:end -->

<!-- profile:messaging-nats-jetstream:start -->
NATS JetStream uses a separate `cmd/worker` root and concrete
`internal/infra/natsjs` producer/consumer. The service remains producer-only;
the worker requires a duplicate-safe feature handler. With outbox enabled, the
same package restores stored W3C creation context and publishes the stored event
without inventing consumer ordering.
<!-- profile:messaging-nats-jetstream:end -->

Business behavior stays in `internal/<feature>`, mechanics in
`internal/infra/<integration>`, and lifecycle/config/telemetry in an explicit
`cmd/<binary>` root. A distinct lifecycle or scaling model earns a separate
binary instead of a durable loop hidden in an HTTP handler.
