---
name: production-diagnosis
description: "Evidence-first diagnosis of production backend incidents and regressions. Use when latency, errors, throughput, saturation, or correctness degrades in a running system and the cause is unknown: quantifying the symptom, mitigation-versus-diagnosis decisions, correlating deploys, config, flags, traffic, and data growth, localizing the failing hop with traces, metrics, and logs, falsifying causes, and proving recovery. After localization route the cause to postgres-performance, cache-engineering, reliable-messaging, durable-background-jobs, or external-api-integration; own the loop end to end when no dedicated skill applies."
---

# Evidence-First Production Diagnosis

Localize before explaining; falsify before fixing:

`symptom contract -> impact and mitigation -> change timeline -> localize the hop -> falsifiable cause -> smallest fix -> verify -> prevent`

The unit of progress is an eliminated hypothesis, not a promising story. Every conclusion states the evidence that produced it and the observation that would have refuted it.

## Choose the branch

Run only the branch the request needs. Record missing evidence as a gap instead of broadening the task.

- **Live incident:** user impact is ongoing; mitigation outranks understanding, and the investigation stays open after impact stops.
- **Post-hoc diagnosis:** a regression or degradation without an active fire. Run the full loop to a falsified cause and a verified fix.
- **Review a diagnosis:** audit someone else's causal chain read-only: does evidence, not narrative, carry each link from symptom to claimed cause?
- **Operate:** rollbacks, restarts, flag flips, scaling, failover, and cache flushes are production actions requiring explicit authorization for the exact action and target. Preflight, execute only that action, verify with fresh readback.

After localization, hand the cause to its owner (see the route table below). This skill retains the symptom contract, localization method, cause falsification, and recovery verification, and it owns fixes on surfaces without a dedicated skill: service code, configuration, resource sizing, network behavior, and stores not covered elsewhere.

**Complete when:** the branch, authority boundary, requested artifact, and excluded concerns are explicit.

## Contract the symptom

Quantify before touching anything: the metric and unit; current value against a baseline with its comparison window; the cohort (endpoints, tenants, regions, instances, versions, request classes); onset time as precise as the data allows; the shape (step, ramp, periodic, spikes). "Slow" and "some errors" are not symptoms; a delta you cannot state is a delta you cannot verify fixing.

Record the negative space with the same care: what is *not* affected — writes but not reads, one region, one tenant, only large payloads. Every unaffected cohort eliminates whole families of causes for free.

**Complete when:** the symptom is metric + baseline delta + cohort + onset + shape, plus the unaffected complement.

## Bound impact and decide: mitigate or diagnose

Severity comes from user-visible impact and its trend, not from how interesting the anomaly is. While impact is ongoing, prefer a reversible mitigation over understanding: roll back the last change, disable the flag, shed load, scale out, fail over. Mitigation is not diagnosis — record what it masks and keep the investigation open.

Before any state-destroying action (restart, scale event, failover), capture what dies with the process: dumps, in-memory queue depths, connection states, recent logs. Ninety seconds of capture usually beats hours of chasing an unreproducible mystery.

**Complete when:** impact and trend are bounded, the mitigate-or-diagnose decision is recorded, and perishable evidence is captured before destructive actions.

## Build the change timeline

Most incidents are changes. Assemble what changed around onset: deploys of this service and its dependencies, config and flag changes, infrastructure events, certificate, quota, and key expirations, scheduled jobs, traffic volume and mix shifts, and data crossing thresholds — table sizes, working sets, partition counts. Ramp-shaped regressions correlate with growth curves rather than events.

Correlation nominates candidates; it convicts none. Each candidate needs a mechanism that would produce this symptom shape in this cohort. Rolling back the top candidate can be justified as mitigation before its mechanism is proven — label it as that.

**Complete when:** the timeline exists and every candidate has a plausible mechanism or is explicitly parked.

## Localize the failing hop

Walk the request path from the edge inward and ask one question per hop: is the degradation produced here or inherited from below?

- **Latency:** compare self-time to downstream time per hop — direct with traces, otherwise per-hop latency deltas at onset. The hop whose self-time grew owns the symptom.
- **Errors:** find the hop where the error class originates rather than propagates; timeout wrappers and generic 5xx relabeling hide origin — match error fingerprints across hops at onset time.
- **Saturation:** check utilization, queue depth, and errors per resource — CPU, memory, connection pools, worker pools, disk, network — at the suspect hops.
- **Skew:** compare across instances, shards, partitions, tenants. One hot instance or key is a different cause family than uniform degradation.

Distinguish cause from victim: when a downstream slows, upstreams saturate from queueing and alert loudest. The loudest component is usually a victim — order degradations by onset time and follow the earliest mover. Where telemetry is missing, run the smallest safe probe (one traced canary request, one targeted query) instead of tuning blind.

**Complete when:** one hop or boundary owns the symptom with evidence, victims are marked as victims, and inherited hops are cleared.

## Route the located cause

| Localized at | Hand to |
| --- | --- |
| PostgreSQL latency, locks, capacity | `postgres-performance` |
| Cache staleness, stampede, hot key, eviction | `cache-engineering` |
| Broker lag, loss, duplicates, redrive | `reliable-messaging` |
| Job backlog, stuck or poison jobs, schedules | `durable-background-jobs` |
| Third-party provider boundary | `external-api-integration` |
| Corruption from concurrent writers | `concurrency-control` |
| Authentication or authorization failures | `auth-access-control` |
| Design flaw spanning services | `distributed-system-design` |

Hand over the evidence package — symptom contract, timeline, localization — not a bare suspicion. Continue in this skill when the cause sits in service code, configuration, resource sizing, network behavior, or a store without a dedicated skill.

**Complete when:** ownership is handed with evidence or retained explicitly.

## Falsify the cause

State the hypothesis as a mechanism with a prediction: X causes the symptom via Y; if true, Z must be observable. Seek the observation most likely to refute it first — the cheapest kill. Write the prediction down before looking; evidence found afterward fits stories too easily. Change or examine one variable at a time.

The bar for "cause": it explains onset time, shape, cohort, and the negative space. A candidate that explains only part is a co-symptom or a contributor — keep going. When safe, reproduce at the smallest scale (a staging replay, one canary instance); a reproduced mechanism ends the argument.

**Complete when:** the cause survived an attempted refutation and explains all four dimensions, or is demoted with the observation that killed it.

## Fix smallest and verify against the baseline

Apply the smallest change that removes the mechanism; prefer reversible; one change per observation window, or attribution is lost. Verification is the symptom contract rerun: the same metric on the same cohort over a comparable window returns to baseline, and adjacent metrics show no displacement — a fixed latency that reappears as queue depth elsewhere is displacement, not recovery. Remove temporary mitigations deliberately and observe; a fix proven only under mitigation is unproven.

**Complete when:** the before/after comparison on the contracted metric and cohort shows recovery, displacement was checked, and mitigation debt is cleared or ticketed.

## Prevent recurrence

Name the invariant that broke and guard it: an alert on the leading indicator (answer explicitly: why did users notice before the alerts did?), a regression falsifier where the mechanism reproduces, a runbook delta for the next responder, and — when a change caused it — the gate that should have caught that change. Keep artifacts actionable and owned; ceremony without an owner is decoration.

**Complete when:** the detection gap and each guard have a named owner.

## Report

Lead with the verdict: the cause — or the current best hypothesis with confidence and open alternatives — the evidence chain, fix state, and the verification delta. Separate observed facts from inference at every step. For a live incident, include mitigation state and its debt. For a review, give a verdict per causal link: carried by evidence, plausible but unproven, or refuted.
