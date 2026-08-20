# Direct Work

Use this method for a clear, local, reversible, single-owner outcome with
bounded proof and no unresolved protected decision.

## Method

Reconstruct the accepted outcome from current repository authority. State only
a non-obvious assumption that would change the result and its reopen condition.
Inspect the current diff and affected callers, change the narrowest causal
owner, and preserve unrelated work and generated/manual authority.

Self-review the bounded diff and observable path. Use [Validation
Routing](../validation-routing.md) to run the smallest current check that would
fail if the outcome were absent or wrong. Load [Implementation / Validation /
Closeout](phases/implementation-validation-closeout.md) only when validation,
deployment, integration, independent review, or blocked closeout is non-obvious.

## Stop Rule

Return the changed outcome, proof actually run, and any unverified remainder.
Switch to the workflow router when durable decisions, coordination, protected
domains, or proof no longer fit one local owner.
