---
name: external-api-integration
description: "External provider request identity, bounded attempts, ambiguity, callbacks, reconciliation, and migration depth reference."
---

# External API Integration

Use only after the active phase identifies an external-provider boundary.
Inherit its authority, artifact, review, proof, output, and completion contract;
do not select a work mode or authorize provider actions here.

Core invariant: persist one local operation identity before side-effecting I/O,
reuse it across attempts and observation paths, and retain an ambiguous outcome
until authoritative lookup, callback, or reconciliation resolves it.

Load one branch:

- outbound HTTP deadlines, retries, rate limits, pagination, or response bounds
  -> [http-resilience.md](references/http-resilience.md);
- OAuth, token lifecycle, scope, or credential selection ->
  [authentication.md](references/authentication.md);
- callback/webhook verification, identity, ordering, or replay ->
  [webhooks.md](references/webhooks.md);
- side-effect convergence, reconciliation, migration, or proof ->
  [convergence-and-proof.md](references/convergence-and-proof.md).
