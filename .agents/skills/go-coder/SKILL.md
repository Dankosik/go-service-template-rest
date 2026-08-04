---
name: go-coder
description: "Smallest Go change: Use when an authorized outcome is ready. Own production code, required tests, cleanup, and proof; Skip unresolved behavior/ownership and diagnosis-, test-, or verification-only work."
---

# Go Coder

One authorized outcome becomes one **surgical change**: the smallest diff at the earliest valid owner that makes every accepted criterion true, provable, and clean.

`accepted criteria -> earliest valid owner -> smallest change -> required tests -> cleanup -> proof -> return`

Read and apply [Implementation / Validation / Closeout](../../../docs/spec-first-workflow/phases/implementation-validation-closeout.md), which owns acceptance and completion.

Reconstruct every accepted criterion from the assigned outcome or ledger and its named sources before touching code. Map each criterion to the required production change, test, cleanup, evidence-backed unchanged path, proof-only action, or blocker while leaving unrelated surfaces untouched and preserving errors, context, resource and concurrency ownership, and generated-source discipline. When a concrete implementation pressure can change the method, load [the reference selector](references/index.md) and let it choose one reference by default, adding another only for an independent pressure.

Before returning, check the far side of every boundary the change touched: the caller that now receives a different shape, the concurrent path that reads what it writes, the version that must keep running beside it. A change that satisfies its criteria on the near side and breaks their mirror reads finished while the defect ships.

A surgical change is complete only when every criterion has a terminal disposition and focused proof, every triggered gate passes, and every changed file, command result, and gap is returned. Stop and reopen the owner on unresolved behavior, ownership, policy, or proof — an invented decision is a defect even when the code works.
