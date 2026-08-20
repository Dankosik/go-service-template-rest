# Go Code / Ownership Design

Use when package or file ownership is not mechanically forced. Own the exact
responsibility map and inverse file map without writing implementation.

## Inputs

Consume the ready Specification and System / Integration Design when present,
repository architecture, current packages/files/symbols/callers/composition
roots/generated sources/tests, and replacement paths.

## Method

1. Decompose every selected system decision entering Go into changed
   responsibilities across normal, failure/recovery, operator, lifecycle,
   cleanup, composition, and proof paths.
2. Map each responsibility exactly once to its semantic owner and exact
   repository location. Map each added or materially changed file back to one
   present responsibility.
3. Rehearse composition -> caller -> owner -> wiring -> proof far enough to fix
   package/file action, visibility, import direction, generated/manual authority,
   lifecycle/resource/error ownership, cleanup, and proof placement.
4. Reuse current owners, stdlib, native capabilities, and installed dependencies
   before custom code. When implementation source remains a real choice without
   changing the accepted mechanism, load Research's [Solution Discovery](research-branches.md#solution-discovery-evidence).
5. Validate both maps against the actual acyclic import graph, `internal`
   visibility, composition roots, generated boundaries, and one semantic edit
   site plus executable parity for any forced representation.

Do not specify function bodies, local variables, statement control flow, or
private helpers without independent policy/state/lifecycle ownership. A touched
file with unrelated reasons to change requires the smallest behavior-preserving
split or deletion needed by the accepted responsibilities.

## Output

Return [Ownership Map V1](../interfaces/ownership-map-v1.md). It fixes exact
repository-relative placement or an evidence-bounded deterministic
implementation-local rule, implementation-source authority/version when
non-mechanical, replacement cleanup, proof ownership, and the code-shape
condition that would invalidate the map.

## Review

Use root self-review for unambiguous placement. Load [Go Ownership
Review](../rubrics/go-ownership-review.md) only when its conditional trigger is
present. A broader shared [Review](../shared/review.md) routes through Technical
Design Review and consumes any current panel receipts.

## Exit And Reopen

Exit when every responsibility has one owner, every changed file one present
reason, the import graph is acyclic, and Planning can preserve placement without
choosing ownership, dependency/composition, generated authority, lifecycle,
cleanup, proof location, or exported surface. Reopen System Design when
placement cannot preserve mechanism/truth; reopen Specification when it cannot
preserve observable behavior.
