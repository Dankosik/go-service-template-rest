# Input Sufficiency And Challenge Readiness

## When To Load
Load this when the candidate synthesis might be too thin to challenge, when the orchestrator did not provide evidence links, or when the pass risks becoming blank-page design. In this repository, keep phase placement from `docs/spec-first-workflow.md`: this is a synthesis checkpoint before `spec.md`, not the `spec.md` approval clarification gate.

## Strong Vs Weak Questions
| Strong | Weak |
| --- | --- |
| "Which candidate decision would change if the retry assumption is false, and what evidence currently supports it?" | "Do we have enough detail?" |
| "Is the missing constraint product policy, repository fact, or specialist evidence?" | "What are all the open questions?" |
| "If no candidate decision is present, should this return to research or framing instead of challenge?" | "Can we write a better spec?" |

## Blocker Classifications
- `blocks_planning`: no candidate decision is named, the problem frame is absent, or the known contradiction could change scope or ownership.
- `blocks_specific_domain`: one lane is missing enough evidence, such as data migration ownership or API compatibility, while the rest of the synthesis is challengeable.
- `non_blocking`: an evidence link is thin but the assumption is already low-impact or can be recorded as accepted risk.

## Next-Action Examples
- `answer`: "The orchestrator can answer from existing research because the candidate decision already cites the repository package boundary."
- `re-research`: "Reopen a data or API lane to verify whether mixed-version clients exist; local inference is not enough."
- `ask_user`: "Ask only if the missing fact is external product policy, such as whether beta customers may see divergent behavior."
- `defer`: "Carry a non-critical naming uncertainty as an assumption; it does not affect planning."
- `accept_risk`: "Proceed with a known weak evidence link when the plan is reversible and validation will prove it early."

## Exa Sources
- [ProductTalk: Assumption Testing](https://www.producttalk.org/2023/10/assumption-testing/) for testing the riskiest assumptions rather than whole ideas.
- [Gary Klein: Premortem](https://www.gary-klein.com/premortem) and [HBR: Performing a Project Premortem](https://hbr.org/2007/09/performing-a-project-premortem) for looking from future failure back to plausible causes.
- [Michael Nygard: Documenting Architecture Decisions](https://cognitect.com/blog/2011/11/15/documenting-architecture-decisions.html) for keeping context, decision, and consequences distinct.
