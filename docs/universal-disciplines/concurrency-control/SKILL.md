---
name: concurrency-control
description: "Durable-state interleaving, arbitration, conflict, fencing, and singleton depth reference reached from go-concurrency."
---

# Concurrency Control

Use only after the active phase identifies a shared durable-state race. Inherit
its authority, artifact, review, proof, output, and completion contract; do not
select a work or operational mode here.

Core invariant: name the contested state and breaking schedule, then choose the
weakest mechanism that closes it at the real isolation boundary. Any expiring
exclusive holder is stale-capable and must be fenced or made idempotent.

Load one branch:

- contested state, writers, or breaking schedule ->
  [interleavings.md](references/interleavings.md);
- conditional write, constraint, optimistic/pessimistic lock, or serializable
  arbitration -> [arbitration.md](references/arbitration.md);
- conflict retry, lease, fencing, leader, or singleton overlap ->
  [conflicts-and-fencing.md](references/conflicts-and-fencing.md).
