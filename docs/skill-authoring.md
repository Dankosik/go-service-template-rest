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
`role/carrier` entries set `disable-model-invocation: true` for Claude, Qwen,
Grok, Cursor, and OpenCode and Codex `agents/openai.yaml` policy `allow_implicit_invocation:
false`; they remain available through the harness's explicit skill syntax or
bound carrier. The generated Claude/Qwen views and skill sync checks enforce
these projections. Grok, Cursor, and OpenCode read the canonical `.agents/skills`
set directly and need no generated skill symlink. OpenCode ignores
`disable-model-invocation`; `opencode.json` denies `user/workflow` skill names
on the built-in `build` and `plan` agents. The `orchestrator` carrier stays
loadable on `build` so a user request to orchestrate a ledger can dispatch
without a slash command. Keep other `role/carrier` entries behind Task
`subagent_type` rather than implicit `skill` loading.

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

## Method Skills: Behavioral Compression

A `model/method` skill exists to change one technical judgment the base model
otherwise makes inconsistently. Before editing, state the ablation: what
observable decision becomes worse when the body is absent?

A promoted method skill binds four elements to the same domain judgment:

1. **Operator** — one pretrained technical term repeated in the description and
   opening.
2. **Story** — one concrete path, lifecycle, matrix, table, graph, or ownership
   map the agent must build.
3. **Falsifier** — one plausible wrong default paired with the correct
   replacement behavior.
4. **Done** — one local, checkable, exhaustive completion criterion.

Prove promotion with trigger, non-trigger, neighbor-collision, decision, and
completion evals. Compare current and ablated baselines with the candidate
before claiming changed model behavior.

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
