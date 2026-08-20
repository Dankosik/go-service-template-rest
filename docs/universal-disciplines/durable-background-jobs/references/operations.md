# Job Operations

Load only for fairness/backpressure, civil schedules, checkpoints/cancellation,
deploy/drain, retention, or operational signals.

Set global, job-type, and tenant concurrency from downstream capacity. Define
admission under saturation, queue-age SLO, priority without starvation, and the
signal that slows claims. Civil schedules retain IANA zone, intended occurrence
identity, DST gap/fold, misfire/catch-up, overlap, jitter, bounds, and
cancellation; a daily civil schedule is not a fixed 24 hours.

For long work, commit a replay-safe chunk effect before the next cursor. A
heartbeat proves liveness, not checkpoint/effect commit. Poll cancellation at
safe boundaries, stop new effects, and resume the same job/effect identities
from durable position.

Drain by stopping claims, then finishing bounded work or checkpointing and
releasing it under a hard deadline. Keep compatible workers/routing until no
payload, checkpoint, schedule, or history requires them. Manual retry creates an
audited attempt against the same effect key.

Observe ready depth/age, claims/renewals, runtime/checkpoint age, attempts and
retry budget, terminal/quarantine/cancellation, replay suppression,
reconciliation drift, fairness saturation, and drainage. Retain effect and
deduplication identity beyond the longest retry, redrive, misfire, and late
delivery window.

Proof covers the one selected operational branch: continuous high-priority
fairness, civil-time folds/gaps, crash/cancel between chunk effect and cursor,
or mixed-version drain and hard shutdown.
