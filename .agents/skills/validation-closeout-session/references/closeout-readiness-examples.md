# Closeout Readiness Examples

## When To Load
Load this when deciding whether a dedicated `validation-closeout-session` may proceed, should be skipped as ceremony, or must reopen an earlier phase before validation.

Authoritative closeout sources:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/go-verification-before-completion/SKILL.md`

## Good Closeout Snippets

```markdown
Closeout readiness: proceed.
Claim: task done for the approved Phase 1 scope.
Routing: `workflow-plan.md` says current phase is `validation-phase-1`; the existing validation phase file is present.
Inputs: `spec.md`, `plan.md`, existing `tasks.md`, and `workflow-plans/validation-phase-1.md` list the same proof obligations.
Proof action: run fresh scoped package tests, API drift check, and migration validation now.
Boundary: no code, test, migration, or workflow-file creation in this session.
```

```markdown
Closeout readiness: not ready.
Claim is broader than the available artifacts: the user asks for task-wide completion, but `workflow-plan.md` still says `implementation-phase-2` is in progress.
Next action: stop validation and route the next session to `implementation-phase-2`; do not run closeout by momentum.
```

## Bad Closeout Snippets

```markdown
Implementation is probably done. I will run tests and patch anything small that fails so we can still close it.
```

```markdown
The validation phase file is missing, so I will create `workflow-plans/validation-phase-1.md` and continue closeout.
```

```markdown
Yesterday's CI was green, so the task is closeout-ready without another run.
```

## Sufficient Vs Insufficient Proof Examples

Sufficient:
- `workflow-plan.md` and the active phase file both route to validation or closeout.
- `spec.md` has explicit validation obligations or a narrow enough scope to derive them from existing approved artifacts.
- expected `tasks.md` and validation phase file already exist, or an explicit direct-path waiver says they are not used.
- fresh commands can be run now against the current workspace without creating new process artifacts.

Insufficient:
- chat memory says implementation is done but master workflow routing says an earlier phase is still active.
- a required `tasks.md` is missing for a non-trivial workflow that expected a ledger.
- proof would require first adding tests, code, migrations, or a missing validation phase file.
- the only positive evidence is stale output, a delegated summary, or an unverified CI snippet.

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11:
- `web_search_exa` query: "official Anthropic Claude Skills documentation SKILL.md references progressive disclosure best practices"
- `web_fetch_exa` URLs: `https://docs.claude.com/en/docs/agents-and-tools/agent-skills/best-practices`, `https://docs.claude.com/en/docs/agents-and-tools/agent-skills`, and `https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills`
- Result: Exa returned `402` credit-limit errors, so no Exa-returned source links were available.

Fallback source links used only for skill-packaging context:
- [Anthropic Agent Skills quickstart](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/quickstart)
- [Anthropic skill authoring best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)
- [Building Agents with Skills](https://claude.com/blog/building-agents-with-skills-equipping-agents-for-specialized-work)
