# Research Reopen And Next-Action Examples

## When To Load
Load this after strong questions survive filtering and you need to recommend whether the orchestrator should answer locally, reopen research, ask the user, defer, or accept risk. Keep the recommendation advisory; do not approve `spec.md` or produce final decisions.

## Strong Vs Weak Questions
| Strong | Weak |
| --- | --- |
| "What exact fact would let planning continue, and which specialist lane can retrieve it?" | "Research this more." |
| "Can the orchestrator answer from existing evidence, or does the question require repository/runtime proof?" | "Who knows?" |
| "If this remains unresolved, is the plan still coherent with a named accepted risk?" | "Can we move on?" |
| "Should the challenge be rerun after this research because a material decision may change?" | "Do another pass later." |

## Blocker Classifications
- `blocks_planning`: reopen research or specification when the answer can change scope, ownership, contract, migration order, or validation strategy.
- `blocks_specific_domain`: reopen only the missing specialist lane when the remaining plan is stable.
- `non_blocking`: answer, defer, or accept risk when the point is real but not planning-critical.

## Next-Action Examples
- `answer`: "The orchestrator should answer from existing research and record the resolution in candidate synthesis."
- `re-research`: "Reopen a reliability lane to inspect retry and timeout behavior; rerun challenge only if idempotency semantics change."
- `re-research`: "Reopen a delivery lane to identify rollback guardrails; no rerun needed if it only adds metrics to an already chosen rollout path."
- `ask_user`: "Ask the user whether partial beta exposure is acceptable because repository evidence cannot decide product appetite."
- `defer`: "Carry low-impact flag cleanup wording into technical design; it does not change task order."
- `accept_risk`: "Proceed with no extra research when the risk is low blast radius, reversible, and paired with a validation check."

## Exa Sources
- [ProductTalk: Assumption Testing](https://www.producttalk.org/2023/10/assumption-testing/) for choosing the smallest test or evidence path for high-risk assumptions.
- [Gary Klein: Premortem](https://www.gary-klein.com/premortem) for identifying plausible failure causes before committing to a plan.
- [Martin Fowler: Feature Toggles](https://martinfowler.com/articles/feature-toggles.html) for matching feature-flag questions to toggle category and lifespan.
- [ACM Queue: Canary Analysis Service](https://queue.acm.org/detail.cfm?id=3194655) for reopening rollout research when canary metrics or populations are not meaningful.
