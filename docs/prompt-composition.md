# Prompt Composition

Read this owner when writing a prompt for another agent, session, phase, or
repository-native entry skill.

A prompt is a locator plus the smallest missing decision delta. Repository
instructions own how to work; accepted artifacts own behavior, mechanism,
proof, and current state. Point to those owners instead of copying them.

## Native Entry Fast Path

When the target has a native skill or default prompt and its artifact is ready,
return only:

```text
<native-skill-entry>
Use <artifact path>.          # only when discovery is ambiguous
<new authority or stop delta> # only when absent from the artifact
```

Use `$<skill>` in Codex and `/<skill>` in Claude Code, Qwen Code, Grok Build, or Cursor.
The syntax selects a carrier; it does not change the skill's semantic
contract. In Grok Build the current primary session is the Orchestrator
carrier; do not require a prepared CLI launch.

This is normally one to three lines. Omit any line already supplied by the
native entrypoint, current repository, or named artifact.

## General Prompt

Retain only information unavailable to the receiving agent that can change its
accepted outcome, business meaning, scope, authority, target, first owner, or
stop/reopen condition. Preserve exact user-supplied values and identifiers. Use
[Intake](spec-first-workflow/phases/intake.md) only when one of those decisions
is unresolved.

Do not restate role duties, repository workflow, phase methods, read order,
artifact contents, accepted architecture, review history, proof matrix,
validation commands, model/effort/isolation fields, or generic quality language.
Do not tell the receiver to read `AGENTS.md`; the harness and repository own
instruction discovery. The receiving agent loads its current authorities.

Use the [Subagent Brief Template](subagent-brief-template.md) for a delegated
lane and [Transition](spec-first-workflow/shared/transition.md) for boundary state;
neither changes this no-duplication rule.

## Completion

The receiver can start from the locator and authoritative state without chat
reconstruction, and removing any remaining sentence would lose a decision that
no named owner supplies.
