# Visibility And Acknowledgement Queues

Load for a queue whose receive contract uses visibility, acknowledgement, or
attempt receipt tokens; map concrete behavior from its current official
contract.

For SQS-like semantics, receive leaves the message durable and returns an
attempt-specific receipt while hiding it temporarily. Delete/acknowledge only
after the durable business effect commits. Renewal replaces the remaining
visibility window from the renewal call; size initial visibility from measured
runtime tails and renew with headroom. Work exceeding the provider's maximum
visibility needs chunking or another engine.

Receipt handles are attempt identity, not job or business identity. Standard
delivery may duplicate even before expiry or after successful delete. FIFO send
deduplication is producer-side and window-bounded; neither provides exactly-once
business effects.

Treat send-after-possible-acceptance errors, effect-before-delete, lost renewal
or delete responses, and stale handles after redelivery as ambiguous. Preserve
the logical identity and converge from durable effect state. Map receive budget
to terminal/DLQ disposition without changing identity.

Proof uses two consumers and the real queue contract: run beyond initial
visibility, obtain a second receipt, attempt stale-handle completion, lose
renewal/delete responses, exhaust poison retries, and observe one durable effect
plus the configured terminal transition. A single handler mock cannot prove
visibility behavior.
