# Amazon SQS delivery decisions

Use this reference only for Amazon SQS choices. Select Standard or FIFO from the business ordering and throughput contract, then verify current regional quotas.

## Standard versus FIFO

- Standard queues provide at-least-once delivery and best-effort ordering. Design every business effect for duplicates and reordering.
- FIFO deduplication suppresses sends with the same `MessageDeduplicationId` only within a five-minute interval. A retry after that window is a new accepted queue message, so payments and other non-repeatable effects need a durable business idempotency key beyond SQS.
- FIFO ordering is per `MessageGroupId`. One in-flight message blocks later messages in that group until deletion or visibility expiry. Choose a group key that matches the business ordering scope without creating an avoidable hot lane.
- Content-based deduplication hashes the body, not message attributes. Prefer an explicit stable deduplication ID when attributes or business identity matter.

## Acceptance, visibility, and deletion

- A successful `SendMessage` response means SQS accepted and redundantly stored the message. A request timeout can be ambiguous; retry with the same logical identity and, for FIFO within its window, the same deduplication ID.
- Receiving leaves the message in the queue and returns a receipt handle. Visibility timeout is a lease-like delay, not a lock or processing acknowledgement; SQS can still deliver duplicates.
- Delete with the current receipt handle only after the durable effect and inbox record commit. A crash, visibility expiry, or lost delete response can cause another delivery.
- Set visibility from a measured processing bound and extend it while the worker owns useful progress. Stop extending when the effect cannot complete; permit a bounded retry or DLQ transition.
- `ReceiveRequestAttemptId` deduplicates a FIFO receive request inside its documented window. It is not message identity or business-effect idempotency.

## Retention, DLQ, and redrive

- Queue retention defaults to four days and can be configured up to fourteen days. Set it from detection plus recovery time, not from the default.
- A redrive policy moves a message after `maxReceiveCount`; choose the count from the transient retry budget. A low value can quarantine healthy work during a short outage.
- A FIFO DLQ can break the original operation order and a poison message can block its message group until it moves. Make that tradeoff explicit.
- DLQ redrive creates new message IDs and enqueue times and can interleave moved messages with live publishes. It cannot filter or modify messages during the move. Use a bounded velocity and retain business identity in the payload/envelope.
- For Standard queues, DLQ expiry uses the original enqueue timestamp; configure DLQ retention longer than the source. FIFO resets the enqueue timestamp on DLQ transfer.

## Signals and security that change the design

- Observe oldest-message age, visible, delayed, and not-visible counts, receive count, empty receives, send/delete errors, visibility extensions and expiries, DLQ depth/age, redrive rate, and per-group head-of-line symptoms. SQS count metrics are approximate.
- Separate send, receive/delete/change-visibility, redrive, purge, and administration IAM roles. Restrict queue and KMS policies to intended principals and sources, enforce TLS, and use SSE-SQS or SSE-KMS as classification requires. SSE protects message bodies, not all queue or message metadata.

## Primary sources

- [Amazon SQS Standard queues](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/standard-queues.html)
- [Amazon SQS FIFO deduplication terms](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/FIFO-key-terms.html)
- [Amazon SQS visibility timeout](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html)
- [Amazon SQS dead-letter queues](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-dead-letter-queues.html)
- [Amazon SQS DLQ redrive](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-configure-dead-letter-queue-redrive.html)
- [Amazon SQS encryption and access](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-least-privilege-policy.html)

Sources checked 2026-08-02. Recheck quotas and target-region behavior before implementation.
