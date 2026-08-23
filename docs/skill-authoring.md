# Skill Authoring

This file owns the machine and information structure of repository skills.
[Prompt Maintenance](prompt-maintenance.md), loaded first for skill edits, owns
wording, pruning, examples, prohibitions, and behavior-proof claims.

## Invocation

Use a model-invoked skill when the agent or another skill must discover it.
Front-load a likely domain word in the description and keep neighboring skills
distinguishable. Encode the machine contract as a `Use` trigger, owned outcome,
and, only when evidence requires it, one decisive exclusion in at most two
sentences.

Treat the description as a routing discriminator, not a body summary: name the
observable pressure and the decision it owns. Prefer positive discriminants.
Add a negative exclusion only when a concrete observed collision demonstrates
material over-trigger without it.

Use a user-invoked skill when human judgment should select it and autonomous
discovery has no value. Material with no independent trigger or steps is a
canonical non-triggerable reference, not a skill.

Every split spends context or recall budget. Split only when the new invocation
boundary changes a predictable process.

## Machine Contract

The canonical `.agents/skills` entry keeps non-empty `name` and `description`
plus exactly one invocation class and kind:

```yaml
metadata:
  invocation: model | user | role
  kind: method | workflow | carrier
```

`model/method` entries remain autonomously discoverable. `user/workflow` and
`role/carrier` entries set `disable-model-invocation: true` for Claude, Qwen,
Grok, Cursor, and OpenCode and Codex `agents/openai.yaml` policy `allow_implicit_invocation:
false`; they remain available through the harness's explicit skill syntax or
bound carrier. The generated Claude/Qwen views and skill sync checks enforce
these projections. Grok, Cursor, and OpenCode read the canonical `.agents/skills`
set directly and need no generated skill symlink. OpenCode ignores
`disable-model-invocation`; keep those entries behind `/orchestrator` or the
bound agent file rather than relying on implicit `skill` loading.

Codex starts with names, descriptions, and paths and may shorten a crowded
catalog. Put the leading word and decisive trigger first. The repository gate
keeps local metadata below its vendor-context ceiling; that is structural
headroom, not invocation proof. Rules that must always apply belong in
bootstrapped authority, not only a skill.

## Body And References

Keep steps shared by every invocation branch in `SKILL.md`. End at the smallest
checkable outcome that proves the skill's job. Put branch-only material behind a
pointer whose text says when to load it; load one matching reference by default
and another only for an independent pressure.

Co-locate a concept's rule, consequences, review signals, and proof. Keep
`SKILL.md` below 500 lines. Before splitting a skill, disclose a reference and
verify that a real trigger boundary remains.

## Method Skills: Behavioral Compression V2

A `model/method` skill exists to change one technical judgment the base model
otherwise makes inconsistently.

A promoted method skill binds four elements to the same domain judgment:

1. **Operator** — one pretrained technical term repeated in the description and
   opening.
2. **Materialized story** — finite typed domain records the agent must build
   before packing the decision or findings into a shared result interface.
3. **Falsifier** — one plausible wrong default paired with the correct
   replacement behavior.
4. **Done** — one local, checkable, exhaustive completion criterion.

Every `every affected X` instruction names its traversal start and terminal
closure. Done means every enumerated record has a disposition and rejecting
proof or an exact gap; a lexical claim that all paths were considered is not a
materialized story.

The canonical [specialist neighbor map](../.agents/contracts/specialist-neighbors.json)
owns catalog collisions. Keep neighboring triggers distinct and route concrete
wrong defaults through the matching method or reference.

## Structural Budget

Use these per-artifact review heuristics:

- 50–150 words for a session or index `SKILL.md` body;
- 100–250 words for a flat decision/reference method;
- 250–600 words for a sequential method with hard gates where premature
  completion is a real risk;
- 250–500 words for one non-obvious reference.

Past 600 words, disclose branch-only detail in a reference without moving a
mandatory sequence gate out of the body.

The skill entry owns domain judgment only. Shared finding, completion, proof,
handoff, workflow, and harness mechanics stay with their existing shared
owners.
