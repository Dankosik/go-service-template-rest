# Tasks Progress Update Examples

## When To Load
Load this when a task-local `tasks.md` already exists and closeout must align checkbox or progress state with fresh proof.

Authoritative closeout sources:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/go-verification-before-completion/SKILL.md`

## Good Closeout Snippets

```markdown
- [x] T001 Phase 1: Implement export job API handler
  - Closeout proof: `go test ./internal/httpapi/export -count=1` passed in this session.
- [x] T002 Phase 1: Keep generated OpenAPI output current
  - Closeout proof: `make openapi-check` passed in this session.
- [ ] T003 Phase 1: Validate migration compatibility
  - Closeout status: blocked. `make migrate-check` failed; reopen `implementation-phase-1`.
```

```markdown
Ledger update: skipped.
Reason: this direct-path task explicitly waived `tasks.md` in the approved workflow; do not invent a task ledger during closeout.
```

## Bad Closeout Snippets

```markdown
- [x] T001-T006 All tasks done because implementation is complete.
```

```markdown
- [x] T004 Migration validation
- [ ] T007 Fix the migration validation failure during closeout
```

```markdown
Created a new `tasks.md` so the validation closeout has somewhere to record progress.
```

## Sufficient Vs Insufficient Proof Examples

Sufficient:
- the existing task item has a proof expectation and the matching fresh command passed
- the checkbox state is changed only for the item covered by that fresh proof
- failed proof leaves the item unchecked and records the reopen target in the existing task note only if the local ledger format already supports such notes

Insufficient:
- checking every item because one broad command passed when some tasks required separate proof
- checking an item from a review-agent summary without rerunning or otherwise verifying the proof in the current workspace
- adding new tasks, splitting old tasks, or rewriting the ledger during validation
- creating `tasks.md` after implementation began because closeout wants a ledger

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11:
- `web_search_exa` query: "official Anthropic Claude Skills documentation SKILL.md references progressive disclosure best practices"
- `web_fetch_exa` URLs: `https://docs.claude.com/en/docs/agents-and-tools/agent-skills/best-practices`, `https://docs.claude.com/en/docs/agents-and-tools/agent-skills`, and `https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills`
- Result: Exa returned `402` credit-limit errors, so no Exa-returned source links were available.

Fallback source links used only for skill-packaging context:
- [Anthropic Agent Skills quickstart](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/quickstart)
- [Anthropic skill authoring best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)
- [Building Agents with Skills](https://claude.com/blog/building-agents-with-skills-equipping-agents-for-specialized-work)
