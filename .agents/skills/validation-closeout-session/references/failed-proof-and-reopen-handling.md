# Failed Proof And Reopen Handling

## When To Load
Load this whenever required proof fails, is missing, is stale, is skipped, or is too narrow for the closeout claim.

Authoritative closeout sources:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/go-verification-before-completion/SKILL.md`

## Good Closeout Snippets

```markdown
Claim: task done.
Scope: repository-wide task closeout.
Verification Commands:
- `go test ./... -count=1`
- `make openapi-check`
Observed Result:
- `go test ./... -count=1` failed in `internal/export`.
- `make openapi-check` was not run because the first required proof already blocks closeout and the next session must reopen implementation.
Conclusion: not verified.
Next Action: reopen `implementation-phase-1` to fix the failing export package test, then rerun validation.
Boundary: no code changes in this closeout session.
```

```markdown
Claim: validation-phase-1 complete.
Scope: T001-T004 in existing `tasks.md`.
Observed Result: required `tasks.md` is missing even though `workflow-plan.md` says the non-trivial task uses a ledger.
Conclusion: not verified.
Next Action: reopen `planning` to repair the missing task ledger and phase routing. Do not create `tasks.md` during validation.
```

## Bad Closeout Snippets

```markdown
One test failed. I fixed the code and reran it during closeout, so the task is complete.
```

```markdown
The required validation file is missing. I created it and marked it complete.
```

```markdown
OpenAPI drift failed, but the implementation is otherwise done, so Outcome: complete with minor follow-up.
```

```markdown
The command was skipped because it is slow. Outcome: ready.
```

## Sufficient Vs Insufficient Proof Examples

Sufficient failed-proof handling:
- names the failed, missing, skipped, stale, or too-narrow proof
- says why it blocks the claim now
- records the narrowest reopen target
- updates `spec.md` and workflow routing to `not verified` or `reopened`
- stops without implementing fixes or creating missing process artifacts

Insufficient failed-proof handling:
- "balances" a failing command with other passing checks and still says done
- changes code or tests during validation to make proof pass
- creates missing `tasks.md` or `workflow-plans/validation-phase-<n>.md`
- leaves `Next session starts with` as `TBD`
- downgrades a required command to a weaker substitute without recording the proof gap

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11:
- `web_search_exa` query: "official Anthropic Claude Skills documentation SKILL.md references progressive disclosure best practices"
- `web_fetch_exa` URLs: `https://docs.claude.com/en/docs/agents-and-tools/agent-skills/best-practices`, `https://docs.claude.com/en/docs/agents-and-tools/agent-skills`, and `https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills`
- Result: Exa returned `402` credit-limit errors, so no Exa-returned source links were available.

Fallback source links used only for skill-packaging context:
- [Anthropic Agent Skills quickstart](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/quickstart)
- [Anthropic skill authoring best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)
- [Building Agents with Skills](https://claude.com/blog/building-agents-with-skills-equipping-agents-for-specialized-work)
