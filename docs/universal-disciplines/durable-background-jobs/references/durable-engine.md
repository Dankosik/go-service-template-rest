# Durable Execution Engines

Load when a durable workflow engine is selected; the concrete mapping below is
Temporal-like.

Workflow code is deterministic coordination replayed from durable history;
external I/O belongs in activities. Map logical job to workflow/business
identity, attempt to activity task/token, liveness to heartbeat timeout,
checkpoint to heartbeat details or application state, completion to recorded
history, retry to an explicit policy bounded by schedule-to-close, and schedule
occurrence to schedule identity plus intended start.

Replay is not exactly-once execution. An activity can commit an external effect
and crash before completion is recorded, then run again; retain the business
effect key. Heartbeats are asynchronous and not atomic with effects. Long
activities need start-to-close, schedule-to-close, heartbeat, cooperative
cancellation, and replay-safe checkpoints. Make retryable/non-retryable classes
explicit rather than inheriting unlimited defaults.

Worker versioning must keep old histories replayable or pinned until they
close. Test payload, result, checkpoint, workflow-code, and history
compatibility before draining an old build. Continue-as-new is a deliberate
lifecycle, not cleanup for uncontrolled history.

Use deterministic time skipping for timers, retries, schedules, and
cancellation, then a real local service for worker loss/routing. Inject worker
termination, effect-before-recorded-completion, missed heartbeat, cancellation
between checkpoints, replay of old histories, DST/overlap/catch-up, and old
worker drainage.
