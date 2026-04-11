# Blocker Classification Examples

## When To Load
Load this when too many questions are being marked as blockers, or when a real risk is being under-classified as ordinary polish. The goal is to route the next step, not to win an argument about severity.

## Strong Vs Weak Questions
| Strong | Weak |
| --- | --- |
| "Does this unanswered question change the implementation order, ownership, contract, or validation proof?" | "Is this important?" |
| "Can planning proceed if this is recorded as an accepted risk, or would the task ledger mislead implementers?" | "Should we block?" |
| "Which single domain must reopen to answer this, and what evidence would be enough?" | "Who should look at this?" |

## Blocker Classifications
- `blocks_planning`: "If old clients may still send the previous payload, the API contract and task ordering change." Planning should stop until compatibility is answered.
- `blocks_planning`: "If no component owns async recovery, task breakdown would invent ownership in code." Reopen design or domain ownership.
- `blocks_specific_domain`: "Only cache invalidation evidence is missing; reopen data/cache research, not the whole spec."
- `blocks_specific_domain`: "Only rollout guardrails are unclear; reopen delivery/reliability, not product framing."
- `non_blocking`: "The log field name is unsettled but the observability obligation is clear."
- `non_blocking`: "The exact cohort percentage can be deferred when the release path and rollback trigger are already explicit."

## Next-Action Examples
- `answer`: "Resolve in orchestration when the artifact already contains enough evidence; do not send it back to research for ritual."
- `re-research`: "Use when missing evidence is factual and specialist: mixed-version behavior, auth boundary, cache invalidation, SLO metric, or migration precedent."
- `ask_user`: "Use only for external policy: launch cohort, acceptable data-loss risk, compliance stance, or product semantics."
- `defer`: "Use for a real but non-planning-critical point that belongs in technical design or validation detail."
- `accept_risk`: "Use when the risk is known, bounded, reversible, and proof obligations are explicit."

## Exa Sources
- [ProductTalk: Assumption Testing](https://www.producttalk.org/2023/10/assumption-testing/) for prioritizing assumptions by importance and evidence instead of testing everything.
- [HBR: Performing a Project Premortem](https://hbr.org/2007/09/performing-a-project-premortem) for surfacing reservations early enough to improve the plan.
- [Michael Nygard: Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions.html) for separating consequences that matter from generic documentation bulk.
