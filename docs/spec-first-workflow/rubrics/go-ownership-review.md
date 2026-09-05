# Go Ownership Review

Load this complementary panel only when a real responsibility/placement fork
survives, several packages or ownership boundaries change, generated/manual
containment is non-obvious, the restructure is broad or hard to reverse, or
current evidence/reviewers materially disagree. Ordinary unambiguous placement
uses root self-review.

Partition one fixed [Ownership Map V1](../interfaces/ownership-map-v1.md) into:

1. responsibility and execution-path ownership;
2. package placement, import direction, composition, visibility, and
   generated/manual containment;
3. file cohesion, naming, declaration grouping, and fixture placement.

Each read-only lane returns only its lens result through shared
[Read-Only Delegation](../shared/read-only-delegation.md) and
[Review](../shared/review.md). The root
synthesizes compatibility. The panel passes only when every triggered lens is
`PASS` on the same candidate; repair or reopen findings, then re-run only each
materially affected lens in fresh context. A broader triggered Technical Design
Review consumes these receipts without repeating them.
