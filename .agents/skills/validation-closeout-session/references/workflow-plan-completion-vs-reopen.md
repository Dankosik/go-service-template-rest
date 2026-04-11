# Workflow Plan Completion Vs Reopen

## When To Load
Load this when updating existing `workflow-plan.md` or an existing `workflow-plans/validation-phase-<n>.md` to reflect final completion, blocked closeout, or explicit reopen routing.

Authoritative closeout sources:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `.agents/skills/go-verification-before-completion/SKILL.md`

## Good Closeout Snippets

Completion:

```markdown
Current phase: validation-phase-1
Phase status: complete
spec.md status: Validation and Outcome refreshed from fresh proof in this session
tasks.md status: existing ledger updated for T001-T006 from fresh proof
workflow-plans/validation-phase-1.md status: complete
Blockers: none
Session boundary reached: yes
Ready for next session: no
Next session starts with: N/A
Task state: done
```

Reopen:

```markdown
Current phase: validation-phase-1
Phase status: blocked
spec.md status: Validation and Outcome refreshed with failing proof
tasks.md status: T003 remains unchecked because migration validation failed
workflow-plans/validation-phase-1.md status: blocked
Blockers: `make migrate-check` failed in this session
Session boundary reached: yes
Ready for next session: yes
Next session starts with: implementation-phase-1
Task state: reopened
```

No dedicated validation phase:

```markdown
Validation phase file: not used by approved direct-path waiver
Routing: update existing `workflow-plan.md` and `spec.md` only; do not create `workflow-plans/validation-phase-1.md`.
```

## Bad Closeout Snippets

```markdown
Current phase: mostly done
Ready for next session: maybe
Next session starts with: TBD
Task state: complete enough
```

```markdown
Validation failed, but the workflow is done because all code has been written.
```

```markdown
Missing validation phase file created during closeout; status complete.
```

## Sufficient Vs Insufficient Proof Examples

Sufficient for completion:
- all positive closeout claims have fresh passing proof
- `spec.md`, `workflow-plan.md`, existing `tasks.md`, and existing validation phase file agree on completion
- next session is `N/A` because no planned follow-up remains

Sufficient for reopen:
- at least one required proof command failed, was missing, stale, skipped, or too narrow
- the workflow plan names the narrowest reopen target and why it blocks closeout
- session boundary is still marked so the next session resumes the earlier phase intentionally

Insufficient:
- ambiguous `mostly done` language
- completion state in `workflow-plan.md` while `spec.md` says proof failed
- `tasks.md` checkboxes marked complete while the workflow is reopened
- inventing a missing validation phase file during closeout

## Exa Source Links
Exa MCP was attempted before these examples were authored on 2026-04-11:
- `web_search_exa` query: "official Anthropic Claude Skills documentation SKILL.md references progressive disclosure best practices"
- `web_fetch_exa` URLs: `https://docs.claude.com/en/docs/agents-and-tools/agent-skills/best-practices`, `https://docs.claude.com/en/docs/agents-and-tools/agent-skills`, and `https://www.anthropic.com/engineering/equipping-agents-for-the-real-world-with-agent-skills`
- Result: Exa returned `402` credit-limit errors, so no Exa-returned source links were available.

Fallback source links used only for skill-packaging context:
- [Anthropic Agent Skills quickstart](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/quickstart)
- [Anthropic skill authoring best practices](https://platform.claude.com/docs/en/agents-and-tools/agent-skills/best-practices)
- [Building Agents with Skills](https://claude.com/blog/building-agents-with-skills-equipping-agents-for-specialized-work)
