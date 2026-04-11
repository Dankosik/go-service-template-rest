# Ownership Boundary And Failure Seams

## When To Load
Load this when the candidate path touches source-of-truth ownership, actor permissions, cross-domain side effects, async handoffs, cache/state propagation, or failure semantics that would otherwise be decided during implementation.

## Strong Vs Weak Questions
| Strong | Weak |
| --- | --- |
| "Which component owns the durable state transition if the handler succeeds but the async side effect fails?" | "Who owns this?" |
| "If cache invalidation fails after the DB commit, which source of truth wins and how is stale state bounded?" | "What about caching?" |
| "Which actor is allowed to reverse this state, and does the candidate path preserve auditability after rollback?" | "Is auth okay?" |
| "If a downstream call times out after performing the side effect, what exactly may be retried?" | "How do failures work?" |

## Blocker Classifications
- `blocks_planning`: ownership is ambiguous enough that task breakdown would assign the same invariant to multiple components, or no component owns recovery.
- `blocks_specific_domain`: one domain seam needs reopening, such as data source-of-truth, API idempotency, or authorization boundary.
- `non_blocking`: ownership is clear but a lower-level retry or observability detail can be carried into design as an explicit obligation.

## Next-Action Examples
- `answer`: "The orchestrator can resolve from `docs/repo-architecture.md` or existing package ownership if the boundary is already documented."
- `re-research`: "Reopen a data/cache lane if source-of-truth and invalidation behavior are not evidenced."
- `ask_user`: "Ask only if actor authority is a product policy choice not inferable from the repository."
- `defer`: "Leave a non-critical metric label or log wording to design when ownership and correctness are settled."
- `accept_risk`: "Accept a manual recovery seam only if the plan names the owner, trigger, and validation evidence."

## Exa Sources
- [Michael Nygard: Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions.html) for capturing forces and consequences instead of only the chosen action.
- [ProductTalk: Assumption Testing](https://www.producttalk.org/2023/10/assumption-testing/) for feasibility and ethical assumptions that can expose ownership or harm risks.
- [Google Cloud: Reliable releases and rollbacks](https://cloudplatform.googleblog.com/2017/03/reliable-releases-and-rollbacks-CRE-life-lessons.html) for asking whether rollback and recovery are easy, trusted, and low-risk.
