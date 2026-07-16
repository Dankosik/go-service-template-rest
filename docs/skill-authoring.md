# Skill Authoring

A skill exists to make a stochastic agent follow a predictable process. Judge every instruction by whether it changes behavior toward that process; cost and maintainability follow from predictability.

## Invocation

Use a model-invoked skill when the agent or another skill must discover it. Front-load its domain leading word in the description, then give one trigger per real branch plus the owned outcome and decisive exclusion. Preserve the machine contract `Use when`, `Own`, and `Skip when` in at most two sentences.

Use a user-invoked skill when human judgment should select it and autonomous discovery has no value. A growing user-only catalog may justify one user-invoked index, but model-invoked domain skills should remain independently discoverable instead of hiding behind a router.

## Information Hierarchy

Keep ordered actions and material shared by every branch in `SKILL.md`. Every step or branch ends on a checkable completion criterion; prefer exhaustive bounds such as “every affected contract dispositioned” over deliverable-shaped bounds such as “write a report.”

Put branch-only rules behind a context pointer whose wording states exactly when to load them. Load one matching reference by default and another only for an independent pressure. Co-locate a concept’s rule, consequences, review signals, and proof so one read brings the whole judgment into context. Keep `SKILL.md` below 500 lines; disclose live reference before splitting an invocation boundary.

## Pruning

Give every meaning one source of truth. Link the canonical workflow or shared contract instead of restating it. Remove duplication, stale sediment, default-behavior no-ops, and prohibitions that can be stated as a positive target. A leading word earns repetition only when it sharpens invocation or execution.

Preserve existing eval prompts, fixtures, assertions, and accepted behavior when changing a skill. Rename or rehome them with the skill; change an oracle only for an accepted behavior reason, never to make a gate green. Budgets remain review heuristics: 50--150 words for session/index skills, 100--250 for specialists, and 250--500 only for a named non-obvious method or failure mode.
