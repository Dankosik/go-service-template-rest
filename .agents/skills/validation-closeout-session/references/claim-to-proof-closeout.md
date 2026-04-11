# Claim-To-Proof Closeout

## When To Load
Load this when the closeout claim is explicit but the fresh proof set still needs to be chosen or narrowed. Use it to keep success wording proportional to current evidence.

Authoritative closeout sources:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/go-verification-before-completion/SKILL.md`

## Good Closeout Snippets

```markdown
Claim: `phase complete` for the tenant export API handler changes only.
Scope: handler validation, generated OpenAPI drift, and package tests listed in Phase 1 proof obligations.
Verification commands:
- `go test ./internal/httpapi/export -count=1`
- `make openapi-check`
Conclusion: verified for Phase 1 only if both commands pass. Do not call the repository fully green from this focused proof.
```

```markdown
Claim: `ready for handoff` for a task that changed API, SQL migrations, and cache invalidation.
Scope: all changed surfaces in the approved plan.
Verification commands:
- scoped package tests for changed API and cache packages
- repository-owned OpenAPI drift check
- repository-owned migration validation command
Conclusion: handoff-ready only if every required surface passes fresh verification.
```

## Bad Closeout Snippets

```markdown
Claim: task done.
Proof: `go test ./internal/httpapi/export -run TestCreateExport -count=1` passed.
Conclusion: the whole repository is done and ready.
```

```markdown
Claim: ready for handoff.
Proof: the review agent said the code looked safe.
Conclusion: verified.
```

```markdown
Claim: all tests pass.
Proof: did not run tests because the diff is small.
Conclusion: tests pass by inspection.
```

## Sufficient Vs Insufficient Proof Examples

Sufficient for a scoped claim:
- Claim says "Phase 1 export handler behavior is verified."
- Commands cover the changed handler package and any generated contract check required by the approved plan.
- Observed result is a fresh pass in the current workspace.

Insufficient for a broad claim:
- Claim says "task done" but only one focused test ran.
- Claim says "OpenAPI contract is current" but no generation or drift command ran.
- Claim says "migration safe" but no migration validation or relevant integration check ran.
- Claim says "ready for handoff" but delegated reports were not rechecked against current workspace state.

Closeout wording rule:
- broad proof may support broad wording
- focused proof supports focused wording
- failed or missing proof supports only `not verified` plus a reopen target

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11:
- `web_search_exa` query: "official Anthropic Claude Skills documentation SKILL.md references progressive disclosure best practices"
- `web_fetch_exa` URLs: `https://docs.claude.com/en/docs/agents-and-tools/agent-skills/best-practices`, `https://docs.claude.com/en/docs/agents-and-tools/agent-skills`, and `https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills`
- Result: Exa returned `402` credit-limit errors, so no Exa-returned source links were available.

Fallback source links used only for skill-packaging context:
- [Anthropic Agent Skills quickstart](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/quickstart)
- [Anthropic skill authoring best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)
- [Building Agents with Skills](https://claude.com/blog/building-agents-with-skills-equipping-agents-for-specialized-work)
