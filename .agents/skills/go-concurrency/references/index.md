# Reference Selector

State how the selected reference changes the review judgment.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| Fire-and-forget work, lost context, channel close or ownership, blocked send/receive, `errgroup`, early-return abandonment, or shutdown join. | [goroutine-lifetime-and-blocking-sites.md](goroutine-lifetime-and-blocking-sites.md) | Enumerate every blocking site and name what unblocks it, instead of accepting a context parameter or an assumed close as the lifetime story. |
| Visibility, publication, readiness flags, atomics, `sync.Map`, immutable snapshots, lock scope, or work done under a lock. | [shared-state-publication-and-locking.md](shared-state-publication-and-locking.md) | Check what the named edge covers and for how long, instead of treating an atomic store or a held lock as protection for everything reachable from it. |
| Fan-out, pools, semaphores, `SetLimit`, queues, async send wrappers, or producer/consumer pressure. | [bounded-work-and-backpressure.md](bounded-work-and-backpressure.md) | Bound goroutines and queued work, not just the critical section, and treat the submission path as a blocking site. |
| Timer/ticker stop or reset, `time.After` loops, `AfterFunc`, sleep-based coordination, or shutdown timing. | [timers-and-time-driven-shutdown.md](timers-and-time-driven-shutdown.md) | Apply this module's Go 1.23+ timer semantics and report the shutdown-latency or lost-signal defect instead of a retracted leak. |
