# RabbitMQ

Load for RabbitMQ routing/confirms, acknowledgement/requeue, prefetch, quorum
queue, poison handling, or recovery; verify queue and server/client versions.

Reliable publish combines a durable/replicated queue, persistent message, and
publisher confirm. A confirm proves broker responsibility under the queue type,
not routing or consumer effect. Use mandatory returns or monitored alternate
exchange when routing is part of success. Track bounded outstanding sequence
numbers because confirms can batch/reorder. Lost confirm is ambiguous; republish
the same logical identity.

Use manual acknowledgement on the same channel only after durable effect and
inbox commit. Connection loss requeues unacknowledged delivery. Redelivered flag
is neither full attempt count nor idempotency key. Bound delivered unfinished
work with per-consumer prefetch; concurrent handlers acknowledge individually or
through a completed frontier so multiple-ack cannot settle unfinished work.

FIFO enqueue does not imply effect order under priority, multiple consumers,
requeue, or variable processing. Use single-active consumer or an application
ordering/fence when required. Immediate requeue loops are overload; bounded
delayed retry and intentional quorum delivery limit lead to quarantine.
Dead-lettering is another publish boundary and may duplicate or reorder.
Redrive preserves original logical identity and uses a bounded canary.

Observe unroutable returns, confirm latency/nacks/timeouts, ready and
unacknowledged age, redelivery, prefetch saturation, connection churn, delivery
limit, DLX failure, quarantine, and reconciliation.
