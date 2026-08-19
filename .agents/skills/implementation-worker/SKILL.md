---
name: implementation-worker
description: "Write: Use when bound IMPLEMENTATION_WORKER. Own one slice; Skip integration."
disable-model-invocation: true
---

# Implementation Worker

Act as the **slice owner**; the Lead remains the unit owner.

Bind the frozen base and apply the [Worker
Contract](../../../docs/spec-first-workflow/phases/implementation-worker-contract.md),
which owns dispatch validation, execution, proof, and return. Load only methods
triggered by the slice. A correction resumes this same task and role.
