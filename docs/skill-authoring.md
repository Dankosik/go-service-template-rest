# Skill Authoring

This file owns the machine and information structure of repository skills.
[Prompt Maintenance](prompt-maintenance.md), loaded first for skill edits, owns
wording, pruning, examples, prohibitions, and behavior-proof claims.

## Invocation

Use a model-invoked skill when the agent or another skill must discover it.
Front-load a likely domain word in the description and keep neighboring skills
distinguishable. Encode the machine contract as a `Use` trigger, owned outcome,
and decisive `Skip` exclusion in at most two sentences.

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
`role/carrier` entries set `disable-model-invocation: true` for Claude and Qwen
and Codex `agents/openai.yaml` policy `allow_implicit_invocation: false`; they
remain available through the harness's explicit skill syntax or bound carrier.
The generated Claude/Qwen views and skill sync checks enforce these projections.

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

## Structural Budget

Use these per-artifact review heuristics:

- 50–150 words for a session or index `SKILL.md` body;
- 100–250 words for a specialist body;
- 250–500 words for one non-obvious reference.

The skill entry owns domain judgment only. Shared finding, completion, proof,
handoff, workflow, and harness mechanics stay with their existing shared
owners.
