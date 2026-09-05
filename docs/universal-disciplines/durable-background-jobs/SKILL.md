---
name: durable-background-jobs
description: "Durable acceptance, identity, leases, effects, retries, schedules, and recovery depth reference."
---

# Durable Background Jobs

Use only after the active phase identifies a job-specific decision. Inherit its
authority, artifact, review, proof, output, and completion contract; do not
select design, implementation, diagnosis, or operation mode here.

Core invariant: a lease is expiring permission to attempt work, never proof of
exclusive execution. One logical job and business-effect identity must converge
across overlapping attempts, crashes, and lost acknowledgements.

Load one branch:

- acceptance, identities, lease, transition, or effect contract ->
  [contract.md](references/contract.md);
- database-backed claims and transitions ->
  [database-backed.md](references/database-backed.md);
- visibility timeout and acknowledgement queues ->
  [visibility-queue.md](references/visibility-queue.md);
- durable workflow engine history and replay ->
  [durable-engine.md](references/durable-engine.md);
- fairness, schedules, checkpoints, cancellation, drain, or retention ->
  [operations.md](references/operations.md).
