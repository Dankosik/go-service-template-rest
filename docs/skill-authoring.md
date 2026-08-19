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
only when that skill is intentionally harness-specific. Claude discovery links
use `make claude-skills-sync`; Qwen reads the canonical directory directly and
needs no `.qwen/skills/` mirror.

Codex starts with skill names, descriptions, and paths, then loads `SKILL.md`
only after selection; its initial catalog has a bounded context budget and may
shorten descriptions or omit skills when crowded ([OpenAI Skills](https://learn.chatgpt.com/docs/build-skills)).
Front-load the leading word and decisive trigger so shortening preserves
invocation. The repository structural gate keeps its own name/description bytes
at or below 7,000 so global skills and paths retain space under the 8,000-character
vendor ceiling. That budget is headroom, not proof of invocation. A rule that must always apply
belongs in bootstrapped `AGENTS.md`, not only in a skill.

## Information Hierarchy

Keep ordered actions and material shared by every branch in `SKILL.md`. End on the smallest checkable outcome that proves the skill's job; enumerate items only when omitting one can change correctness, safety, or the downstream decision.

Split a sequence only across a real context boundary; [Steering](#steering) owns whether the remaining steps should be hidden at all.

Put branch-only rules behind a context pointer whose wording states exactly when to load them. Load one matching reference by default and another only for an independent pressure. When behavioral evidence shows that a must-have pointer is missed, sharpen its loading condition; inline the material only if the miss persists. Co-locate a concept’s rule, consequences, review signals, and proof so one read brings the whole judgment into context. Keep `SKILL.md` below 500 lines; disclose live reference before splitting an invocation boundary.

## Steering

Write each completion criterion with the observable that proves the step and the
coverage that observable must span. When work stops too early, name the evidence
surfaces a satisfying answer must touch. Hide later steps only after observed
premature completion survives a sharper criterion.

Prohibitions steer poorly, because naming a behavior activates it. [Prompt Maintenance](prompt-maintenance.md) owns that phrasing rule for every instruction in this repository.

## Pruning

Apply [Prompt Maintenance](prompt-maintenance.md#change-and-proof); it owns
removal-first changes, examples, model-relative no-op claims, and proof
language. Skill-specific pruning repeats a leading word only when it sharpens
invocation, the decisive action or attention lens, or completion.

Budgets are per-artifact review heuristics, not gates: 50--150 words for a
session or index `SKILL.md` body, 100--250 for a specialist `SKILL.md` body, and
250--500 for a reference carrying one named non-obvious method or failure mode.
