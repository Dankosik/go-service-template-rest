# Planning Anti-Patterns

## When To Load
Load this when reviewing a draft `plan.md` or `tasks.md` for invented decisions, duplicate authority, phase-boundary drift, vague verification, false parallelism, or artifact misuse.

Use this as a smell catalog, not a scoring rubric. Repository-local workflow rules and approved task artifacts remain authoritative.

## Good Plan Snippet

```markdown
## Blockers / Assumptions

- No rollout artifact is expected for this task because the approved scope changes only the skill reference bundle and no deployment, migration, compatibility, or runtime behavior surface is triggered.
- `tasks.md` is expected because this is non-trivial planning output with several reference files; it remains a ledger and does not restate `spec.md` decisions.
- If an example requires a new architecture, API, data, security, reliability, or rollout decision, reopen the appropriate earlier phase instead of adding it here.
```

Why it is good: it explains why a conditional artifact is absent and refuses to invent decisions.

## Bad Plan Snippet

```markdown
## Extra Safety

Add security, reliability, database, API, and rollout phases just in case. If there is no design for them, make reasonable choices during implementation.
```

Why it is bad: it creates just-in-case work and bypasses the decision/design chain.

## Good Task Ledger Example

```markdown
- [ ] T050 [Phase 1] Add the reference file explicitly required by the approved plan. Depends on: none. Proof: `rtk test -f .agents/skills/planning-and-task-breakdown/references/<required-file>.md`.
- [ ] T051 [Phase 1] Link the reference from `SKILL.md` without changing frontmatter identity or trigger semantics. Depends on: T050. Proof: manual read of `SKILL.md` frontmatter plus `rtk rg -n "references/" .agents/skills/planning-and-task-breakdown/SKILL.md`.
- [ ] T052 [Phase 1] Check that examples do not introduce unapproved architecture, API, data, security, reliability, or rollout decisions. Depends on: T050, T051. Proof: targeted `rtk rg -n "architecture|API|data|security|reliability|rollout" .agents/skills/planning-and-task-breakdown/references`.
```

Why it is good: it checks the actual anti-pattern risk without adding unrelated work.

## Bad Task Ledger Example

```markdown
- [ ] T050 [Phase 1] Add every possible planning reference.
- [ ] T051 [Phase 1] Rewrite the skill identity to make it more general.
- [ ] T052 [Phase 1] Add security and rollout examples for completeness.
```

Why it is bad: it broadens scope, violates frontmatter identity, and invents decision domains for completeness.

## Non-Findings To Avoid

- Do not flag a compact `SKILL.md` as incomplete when detailed examples live in `references/`.
- Do not flag omitted architecture/API/data/security/reliability/rollout examples when no approved decision in that domain exists.
- Do not flag direct-path or lightweight-local waivers when they are explicit, narrow, and recorded before crossing a phase boundary.
- Do not flag broad validation commands as absent when targeted proof is more appropriate for the change surface.
- Do not mistake "not expected" for "forgotten"; conditional artifacts should be absent unless triggered.

## Exa Source Links
External links found with Exa and used only to calibrate anti-pattern examples:

- Feature slicing and avoiding horizontal task dumps: https://techleadhandbook.org/agile/feature-slicing/
- Thin vertical slices and stop-anytime completeness: https://www.tomdalling.com/blog/mentoring/work-in-thin-vertical-slices/
- Martin Fowler on short decision records and linking supporting material instead of expanding the record: https://martinfowler.com/bliki/ArchitectureDecisionRecord.html
