# Skill Authoring

A skill exists to make a stochastic agent follow a predictable process. Judge every instruction by whether it changes behavior toward that process; cost and maintainability follow from predictability.

## Invocation

Use a model-invoked skill when the agent or another skill must discover it. Front-load its domain leading word in the description; choose a pretrained word or compact phrase already present in likely prompts or repository language, and give neighboring skills distinguishable anchors rather than one generic workflow noun. Then give one trigger per real branch — a distinct way the skill is invoked, not a topic it mentions — plus the owned outcome and decisive exclusion. Preserve the machine contract — a `Use` trigger clause, `Own`, and `Skip` — in at most two sentences.

Use a user-invoked skill when human judgment should select it and autonomous discovery has no value. A growing user-only catalog may justify one user-invoked index, but model-invoked domain skills should remain independently discoverable instead of hiding behind a router.

Every split spends one of two budgets: a model-invoked skill spends context load, because its description stays resident in every session; a user-invoked skill spends cognitive load, because a human must recall that it exists. Divide a skill only when the split buys more predictability than the budget it spends.

Material with no independent trigger or steps is not a skill: keep it in a canonical non-triggerable external reference rather than spending model context or human recall on it.

The canonical `.agents/skills` set is cross-harness and keeps a non-empty
`name` and `description` for portable discovery. Do not emulate a glossary
user-only skill by deleting required metadata; keep genuinely non-triggerable
material as an external reference, or use a harness-native invocation policy
only when that skill is intentionally harness-specific.

Codex starts with skill names, descriptions, and paths, then loads `SKILL.md`
only after selection; its initial catalog has a bounded context budget and may
shorten descriptions or omit skills when crowded ([OpenAI Skills](https://learn.chatgpt.com/docs/build-skills)).
Front-load the leading word and decisive trigger so shortening preserves
invocation. The repository structural gate keeps its own name/description bytes
at or below 8,000; global skills and paths still consume catalog space, so that
budget is headroom, not proof of invocation. A rule that must always apply
belongs in bootstrapped `AGENTS.md`, not only in a skill.

## Information Hierarchy

Keep ordered actions and material shared by every branch in `SKILL.md`. End on the smallest checkable outcome that proves the skill's job; enumerate items only when omitting one can change correctness, safety, or the downstream decision.

Split a sequence only across a real context boundary; [Steering](#steering) owns whether the remaining steps should be hidden at all.

Put branch-only rules behind a context pointer whose wording states exactly when to load them. Load one matching reference by default and another only for an independent pressure. When behavioral evidence shows that a must-have pointer is missed, sharpen its loading condition; inline the material only if the miss persists. Co-locate a concept’s rule, consequences, review signals, and proof so one read brings the whole judgment into context. Keep `SKILL.md` below 500 lines; disclose live reference before splitting an invocation boundary.

## Steering

Steering decides how much work happens inside a step, not which steps exist.

A completion criterion has two independent dimensions: clarity — whether done and undone are distinguishable — and demand — how much must hold before it is satisfiable. Clarity resists premature completion; demand drives legwork. A criterion that is checkable but undemanding produces a confident, thin result. Write both: the observable that proves the step, and the coverage that observable must span.

Legwork is the reading, searching, and probing the agent performs inside a step. The skill never executes it and can only raise or lower it. It goes thin when the criterion is satisfiable early, when later steps are visible, or when the step's leading word carries no demand. Raise it by naming the evidence surfaces a satisfying answer must have touched; adjectives do not raise it.

Visible later steps pull attention forward and end the current one early. Sharpen the criterion first; hide the remaining sequence only when premature completion is observed and the criterion is irreducibly fuzzy.

Prohibitions steer poorly, because naming a behavior activates it. [Prompt Maintenance](spec-first-workflow.md#prompt-maintenance) owns that phrasing rule for every instruction in this repository.

## Pruning

Give every meaning one source of truth. Link the canonical workflow or shared contract instead of restating it. Remove duplication, stale sediment, and default-behavior no-ops. Repeat a leading word only where it sharpens invocation, the decisive action or attention lens, or the completion criterion, and let it replace the longer explanation; a generic or coined word that changes none of them is a no-op.

Treat invocation and no-op claims as model-relative. Compare realistic should-trigger and near-miss prompts against the previous version; structural checks prove shape, not behavior. When live evaluation is unavailable, report the claim as unproven.

Preserve accepted behavior and the realistic examples that explain it when
changing a skill. If an externally owned live evaluation system exists, change
its oracle only for an accepted behavior reason, never to make a gate green.
Otherwise report invocation and no-op claims as unproven. Budgets remain review
heuristics, not gates, and bind one artifact each: 50--150 words for a session
or index `SKILL.md` body, 100--250 for a specialist `SKILL.md` body, and
250--500 for a reference file carrying one named non-obvious method or failure
mode.
