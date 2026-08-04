# Visibility and acknowledgement queue mechanics

Read this reference only for a broker or cloud queue whose receive/claim contract uses visibility, acknowledgement, or receipt tokens. The concrete example below is Amazon SQS; map another engine only from its official documentation.

## SQS claim, renewal, and acknowledgement

Receiving an SQS message leaves it in the queue, returns a receipt handle, and makes it temporarily invisible. Processing succeeds at the queue layer only when the consumer deletes the message with the current receipt handle. If it is not deleted before visibility expires, it becomes visible and another consumer may receive it.

`ChangeMessageVisibility` replaces the remaining visibility interval from the time of the call; it does not persist as the next receive's default. Set initial visibility above measured runtime tails with headroom; renew before expiry when an attempt can cross that window. SQS caps visibility at 12 hours from the initial receive, so work beyond the cap needs chunking or another engine.

The receipt handle maps to attempt identity, not business identity; every receive gets a new handle. `DeleteMessage` is the queue completion/acknowledgement after the [common effect contract](../SKILL.md#make-effects-reentrant-and-failures-bounded) succeeds.

## Delivery and ambiguous completion

SQS standard queues are at-least-once: a message can be delivered more than once, including rare delivery after a successful delete. A visibility timeout also does not prevent a duplicate from being delivered within the interval. FIFO send deduplication suppresses producer duplicates only within its documented deduplication window; it does not make an external business effect exactly once.

Treat these boundaries as ambiguous:

- send returned an error after the queue may have accepted the message;
- effect committed but delete was not sent or its response was lost;
- visibility renewal timed out while the original worker kept running;
- an old receipt handle was used after redelivery.

Feed these boundaries into the [common crash matrix](../SKILL.md#define-the-lease-and-crash-matrix). A failed API response does not prove non-delivery.

## Retry, poison, and scheduling

Redelivery after visibility expiry is the retry mechanism unless the application explicitly changes visibility. `maxReceiveCount` maps the common retry budget to a DLQ/quarantine transition. Preserve payload version, receive count, and error evidence for diagnosis.

SQS delay queues and per-message timers schedule delivery at most 15 minutes ahead; FIFO queues do not support individual message timers. AWS recommends EventBridge Scheduler for longer or recurring schedules. Keep schedule occurrence identity and DST/misfire/overlap policy in the scheduling owner rather than attributing them to SQS visibility.

## Capacity and observability

SQS-specific signals are oldest-message age, visible/not-visible depth, receive count, visibility-extension and delete failures, DLQ depth, and redrive age. Short visibility raises overlap risk; long visibility delays recovery. Use the [capacity branch](operations.md#capacity-fairness-and-backpressure) to set budgets.

## Executable proof

Use the real queue API or an emulator whose visibility behavior is explicitly known. Run two consumers and force:

1. runtime beyond initial visibility with and without renewal;
2. a crash after the business effect but before delete;
3. lost renewal and lost delete responses;
4. repeated poison failures through the configured receive budget;
5. an old receipt-handle delete after redelivery;
6. delayed delivery and the handoff to the external scheduler beyond 15 minutes.

Assert duplicate attempts, old-handle rejection behavior, the configured DLQ transition, and observable recovery through the [core effect invariant](../SKILL.md#make-effects-reentrant-and-failures-bounded). A mock that invokes the handler once cannot prove visibility behavior.

For the narrow duplicate-after-expiry case, the minimum faithful check waits past the initial visibility, obtains a second receipt handle with another consumer, attempts completion with the stale first handle, and asserts one durable business effect by the stable effect key. Counting provider calls without observing the second receive and receipt-handle transition is not a visibility test.

**Complete when:** the chosen initial visibility and any renewal cadence have measured headroom, renewal uncertainty has an explicit stopping boundary, and work that can exceed 12 hours has a chunking or engine-change disposition.

## Primary sources

- [Amazon SQS visibility timeout](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-visibility-timeout.html)
- [Amazon SQS `ChangeMessageVisibility`](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/APIReference/API_ChangeMessageVisibility.html)
- [Amazon SQS `DeleteMessage`](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/APIReference/API_DeleteMessage.html)
- [Amazon SQS dead-letter queues and `maxReceiveCount`](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-dead-letter-queues.html)
- [Amazon SQS message timers and extended scheduling](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/sqs-message-timers.html)
- [Amazon SQS FIFO send deduplication](https://docs.aws.amazon.com/AWSSimpleQueueService/latest/SQSDeveloperGuide/FIFO-queues-exactly-once-processing.html)
