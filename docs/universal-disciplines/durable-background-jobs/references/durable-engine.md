# Durable execution engine mechanics

Read this reference only when a durable execution engine is selected. The concrete model is Temporal. Keep a single-job capability small; do not turn it into multi-process business orchestration without a demonstrated need.

## Map the lease to Temporal

Temporal Workflow code is deterministic coordination replayed from durable Event History. External I/O and other failure-prone or non-deterministic work belongs in Activities. Map the common contract as follows:

| Common concept | Temporal mechanism |
| --- | --- |
| logical job identity | Workflow ID and explicit business identity |
| attempt/lease | Activity Task attempt and task token, bounded by Activity timeouts |
| lease liveness | Activity Heartbeat plus Heartbeat Timeout |
| checkpoint | Heartbeat details for resumable Activity progress, or explicit durable application checkpoint |
| durable completion | Activity completion recorded in Workflow Event History |
| retry | Activity Retry Policy within Schedule-To-Close and business budgets |
| schedule occurrence | Schedule ID plus scheduled start time / explicit occurrence key |

Workflow replay is not exactly-once execution. An Activity can commit an external effect and crash before Temporal records completion, then execute again; this is the Temporal instance of the [common effect contract](../SKILL.md#make-effects-reentrant-and-failures-bounded).

## Timeouts, heartbeats, retries, and cancellation

Set Start-To-Close to detect a crashed Activity attempt and Schedule-To-Close to bound the whole Activity including retries. For long Activities, set a shorter Heartbeat Timeout and heartbeat from definite progress points. Heartbeat details can seed the next attempt, but a heartbeat is delivered asynchronously and is not a transaction with an external effect.

Temporal delivers Activity cancellation through heartbeat. A long Activity that does not heartbeat cannot receive cancellation promptly. Make cancellation cooperative, checkpoint before stopping where safe, and record whether cleanup itself is idempotent.

Activities retry by default; Temporal's documented default maximum attempts is unlimited, while Workflow Executions do not retry by default. Map explicit Retry Policies and non-retryable error types to the [common failure dispositions](../SKILL.md#make-effects-reentrant-and-failures-bounded).

## Scheduling and recovery

Temporal Schedules have independent identity, calendar or interval specs, named time zones, jitter, overlap policies, catch-up windows, pause-on-failure, and backfill. UTC is recommended by Temporal; map civil-time requirements to the [schedule branch](operations.md#civil-schedules-and-misfires).

Workflow state recovers by replaying durable Event History. Activity attempts restart from initial state unless checkpoint details or application state provide a resume point. Keep Workflow code deterministic and verify replay compatibility before deploying changes.

## Worker versions and drain

Temporal Worker Versioning can pin a Workflow to one Worker Deployment Version or allow auto-upgrade with replay-safe code. A version remains draining while open pinned Workflows depend on it and becomes drained after they close. Map this state to the [deploy/drain branch](operations.md#deploy-drain-and-versioning).

Choose pinned versus auto-upgrade from job duration and rollout needs. Test payloads, Activity inputs/results, checkpoint formats, Workflow replay, and old histories against the new build. Long-running jobs may use bounded Continue-As-New checkpoints only when that lifecycle is part of the chosen contract.

## Executable proof

Use the SDK's deterministic time-skipping environment for schedule, timer, retry, and cancellation logic, then run integration tests against a real local Temporal service for worker loss and routing. Inject:

- worker termination during an Activity;
- effect commit before Activity completion is recorded;
- missed heartbeat and retry from checkpoint details;
- cancellation while the Activity is between checkpoints;
- Workflow replay with old histories on the new build;
- DST, catch-up, overlap, pause, backfill, and worker-drain cases.

Assert deterministic replay and the requested Temporal mappings; the common proof owns effect, retry, checkpoint, schedule-occurrence, and terminal-state invariants.

## Primary sources

- [Temporal Activities and Activity completion](https://docs.temporal.io/activities)
- [Temporal Activity failure detection, timeouts, and heartbeats](https://docs.temporal.io/encyclopedia/detecting-activity-failures)
- [Temporal Retry Policies](https://docs.temporal.io/encyclopedia/retry-policies)
- [Temporal Schedules, overlap, catch-up, and backfill](https://docs.temporal.io/schedule)
- [Temporal Worker Versioning and drainage](https://docs.temporal.io/production-deployment/worker-deployments/worker-versioning)
