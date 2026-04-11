# Spec Validation And Outcome Updates

## When To Load
Load this when updating task-local `spec.md` `Validation` and `Outcome` from a final closeout pass.

Authoritative closeout sources:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/go-verification-before-completion/SKILL.md`

## Good Closeout Snippets

```markdown
## Validation

Claim: Phase 1 is complete for tenant export job API and migration scope.
Scope: approved Phase 1 surfaces in `plan.md` and T001-T004 in existing `tasks.md`.
Verification Commands:
- `go test ./internal/httpapi/export ./internal/export -count=1`
- `make openapi-check`
- `make migrate-check`
Observed Result: all three commands passed in this session.
Conclusion: verified for Phase 1 scope.
Next Action: close `validation-phase-1` and mark the workflow complete unless later planned phases remain.

## Outcome

Phase 1 closed with fresh proof for API behavior, generated contract drift, and migration validation. No implementation work was performed during closeout.
```

```markdown
## Validation

Claim: task done.
Scope: repository-wide task closeout.
Verification Commands:
- `go test ./... -count=1`
- `make openapi-check`
Observed Result: `go test ./... -count=1` failed in `internal/export`.
Conclusion: not verified.
Next Action: reopen `implementation-phase-1` to address the failing export package test, then return to validation.

## Outcome

Closeout blocked. Fresh proof failed, so the task is reopened to `implementation-phase-1`.
```

## Bad Closeout Snippets

```markdown
## Validation

Tests looked fine.

## Outcome

Done.
```

```markdown
## Validation

`go test ./...` failed, but only in an unrelated-looking area.

## Outcome

Task complete except for a minor follow-up.
```

```markdown
## Outcome

Implementation completed and spec decisions were adjusted during validation.
```

## Sufficient Vs Insufficient Proof Examples

Sufficient:
- `Validation` names the claim, scope, exact commands, observed pass or fail result, conclusion, and next action.
- `Outcome` is no broader than the fresh proof.
- failing proof produces `not verified` and a reopen target rather than "complete with caveats."

Insufficient:
- `Validation` only says "tests pass" without command names.
- `Outcome` says "done" when at least one required proof command failed or was skipped.
- `Outcome` includes new decisions or implementation notes that belong in a reopened earlier phase.
- prior chat output is pasted as if it were fresh command evidence.

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11:
- `web_search_exa` query: "official Anthropic Claude Skills documentation SKILL.md references progressive disclosure best practices"
- `web_fetch_exa` URLs: `https://docs.claude.com/en/docs/agents-and-tools/agent-skills/best-practices`, `https://docs.claude.com/en/docs/agents-and-tools/agent-skills`, and `https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills`
- Result: Exa returned `402` credit-limit errors, so no Exa-returned source links were available.

Fallback source links used only for skill-packaging context:
- [Anthropic Agent Skills quickstart](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/quickstart)
- [Anthropic skill authoring best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)
- [Building Agents with Skills](https://claude.com/blog/building-agents-with-skills-equipping-agents-for-specialized-work)
