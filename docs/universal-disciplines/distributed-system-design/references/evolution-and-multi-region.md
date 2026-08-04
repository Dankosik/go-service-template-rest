# Evolution and Multi-Region Design

## Make change part of the architecture

A target diagram is incomplete without a path from the current state. Define the transitional states, their invariants, and how each exits. Prefer small reversible steps with observable convergence.

For every changed contract specify:

- producer and consumer compatibility across the oldest and newest live versions;
- additive or expand-before-contract changes;
- deployment and rollback order;
- ownership of adapters, flags, routing, and temporary state;
- the condition that removes the old path;
- validation that no old reader, writer, job, message, or restore path remains.

Rollback is safe only while the old version can interpret state produced by the new one. Once that boundary is crossed, define roll-forward and data repair instead.

## Data migration loop

Use an explicit sequence:

`prepare -> dual-compatible release -> bounded backfill -> reconcile -> shadow/compare -> cutover -> observe -> contract`

Pin the source-of-truth boundary and snapshot or change-capture position. Make backfill resumable and idempotent, bound batches and resource use, record checkpoints, and reconcile counts plus domain invariants. Define concurrent-write handling and late arrivals.

Dual writes create another distributed consistency problem. If unavoidable, name the commit order, ambiguous outcome, repair log, reconciliation, and authority during divergence. Prefer one authoritative write plus change propagation when it meets the freshness contract.

### Strangler migration

Use when a large system can move behind a stable interception or routing boundary one capability at a time. Define route ownership, legacy/new data access, semantic translation, cross-system calls, comparison, fallback, and the deletion condition for the facade and old capability. Treat the facade as transitional architecture with its own capacity and failure mode.

## Earn multiple regions

Use multiple regions only when a quantified latency, residency, regional-loss, or disaster-recovery force pays for cross-region data, routing, deployment, observability, security, and testing complexity.

For each critical operation define:

- traffic home and routing during normal, degraded, and recovery states;
- active/passive, active/active, cell/stamp, or read-local/write-home topology;
- data authority, replication direction, acknowledged commit, lag, and partition behavior;
- consistency and user-visible semantics during regional isolation;
- failover trigger and authority, fencing, split-brain prevention, and failback;
- regional dependency independence and any global control-plane dependency;
- capacity in the surviving region and backlog catch-up after recovery;
- data residency, keys, secrets, audit, and operator access by region.

### Active/passive

Fits a clear write authority and recovery target that tolerates promotion. Define replication data-loss window, promotion and fencing, traffic convergence, warm capacity, dependency readiness, and failback. A standby that has never served representative traffic is unproven.

### Active/active

Fits latency or availability requirements that need multiple serving regions and operations whose conflicts can be prevented or resolved. Define writer placement, conflict semantics, global invariants, replication lag, partition behavior, routing stickiness, and convergence. Multiple active compute regions do not make a single-region state authority active/active.

### Cell or deployment stamp

Fits tenant or cohort placement, repeatable capacity units, residency, and bounded blast radius. Define placement authority, routing metadata, evacuation, spare capacity, fleet version skew, shared services, and cross-cell operations. Avoid a global router or control plane that cannot meet the aggregate availability target.

## RPO, RTO, and recovery

Set RPO and RTO per business capability, not only per database. Trace all required state and dependencies: primary data, object storage, messages, search indexes, caches, configuration, secrets, identity, DNS, and external providers.

Define backup and replication separately. A replica can copy corruption; a backup without a tested restore does not establish recovery. Specify restore order, integrity checks, point-in-time selection, replay or rebuild, capacity, user-visible mode, and reconciliation. Exercise failover, restoration, and failback with evidence against the target.

## Evolution proof

Before cutover, test:

- oldest/newest version interoperability;
- backfill restart, duplicate batch, concurrent write, and late arrival;
- shadow comparison and domain-level reconciliation;
- abort before and after the rollback boundary;
- regional dependency and routing failure;
- failover at representative load, split-brain fencing, and surviving capacity;
- restore from an independently retained backup and integrity verification;
- catch-up and safe return to normal topology.

Record which evidence is local, staged, deployed, or verified live. A tabletop exercise proves reasoning and ownership; it does not prove the runtime path.

## Primary sources

- [Azure Architecture Center: Strangler Fig pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/strangler-fig)
- [Azure Architecture Center: Deployment Stamps pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/deployment-stamp)
- [Azure Architecture Center: Geode pattern](https://learn.microsoft.com/en-us/azure/architecture/patterns/geodes)
- [AWS Prescriptive Guidance: Multi-Region Fundamentals](https://docs.aws.amazon.com/prescriptive-guidance/latest/aws-multi-region-fundamentals/introduction.html)
- [Google SRE: Data Integrity](https://sre.google/sre-book/data-integrity/)
- [Google Research: Transparent Migration of Datastore to Firestore](https://research.google/pubs/transparent-migration-of-datastore-to-firestore/)
