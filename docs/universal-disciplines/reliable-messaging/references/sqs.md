# Amazon SQS

Load for SQS Standard/FIFO delivery, visibility/deletion, message groups,
deduplication, DLQ, or redrive; verify current regional contract and quotas.

Standard delivery is at-least-once and best-effort ordered. FIFO send
deduplication is bounded by the documented window and grouping order is per
`MessageGroupId`; neither replaces durable business idempotency. Choose group
scope from the required business order without creating a hot lane. A successful
send proves redundant queue acceptance; timeout can be ambiguous and retains the
same logical identity/deduplication ID.

Receive returns an attempt receipt and starts visibility; it is not a lock or
effect acknowledgement and duplicates can still occur. Delete with the current
receipt only after durable effect/inbox commit. Size/extend visibility from
measured processing while useful ownership remains; crash, expiry, lost delete,
or stale receipt causes safe redelivery.

Retention covers detection plus recovery. `maxReceiveCount` maps bounded retry
to DLQ; low values can quarantine healthy work during outages. FIFO poison work
can block a group, while moving it can break order. Redrive creates new broker
IDs/timestamps and can interleave with live traffic, so preserve business
identity, bound velocity, and observe the invariant. DLQ expiry semantics differ
between Standard and FIFO and must be pinned.

Observe oldest age, visible/not-visible/delayed counts, receive attempts,
visibility/delete failures, DLQ age/depth, redrive, and group head-of-line delay.
