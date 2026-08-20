# Evolution And Multi-Region

Load when migration, mixed versions, region topology, disaster recovery, RPO,
or RTO can change the design.

For every contract change define oldest/newest compatibility, expand-before-
contract states, deploy/rollback order, temporary owners, deletion condition,
and proof no old reader, writer, job, message, or restore path remains. Rollback
ends when the old version cannot interpret new state; after that boundary own
roll-forward and repair.

Use `prepare -> dual-compatible -> bounded backfill -> reconcile ->
shadow/compare -> cutover -> observe -> contract`. Pin source authority and
snapshot/change position. Backfills are resumable and idempotent with bounded
batches, checkpoints, concurrent-write handling, late arrivals, and domain
reconciliation. Dual writes need explicit commit order, ambiguity log, repair,
and authority during divergence; prefer one writer plus propagation.

Earn multiple regions only from quantified latency, residency, regional-loss,
or recovery forces. Per critical operation fix traffic home, active/passive or
active/active shape, data authority/replication/acknowledgement, isolation
semantics, failover authority and fencing, surviving capacity, catch-up,
failback, global control dependencies, and regional security/access.
Active/active compute over one regional authority is not active/active state.

Set RPO/RTO per capability across primary data, objects, messages, indexes,
configuration, secrets, identity, DNS, and providers. Replication can copy
corruption; backup proves nothing without restore. Define restore order,
integrity checks, point-in-time choice, replay/rebuild, user-visible mode, and
reconciliation.

Proof covers mixed-version interoperability, backfill restart/duplicate/late
write, abort on both sides of rollback boundary, representative regional
failure, split-brain fencing, restore from independent backup, catch-up, and
safe return.
