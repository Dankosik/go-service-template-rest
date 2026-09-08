# Prompt Composition

Read this owner when writing a prompt for another agent, session, phase, or
repository-native entry skill.

Write as if sending a work message to a capable colleague who will carry out
the task. Explain what needs to happen, why it matters when useful, and what a
good result means. Trust the receiver to choose how to investigate, implement,
and check the work.

## The Message

Recover the intended request from rough notes, dictation, repetition, or mixed
languages. Keep the user's meaning and level of commitment. Use the requested
language; otherwise follow the conversation. Return a message ready to send.

Use ordinary sentences and familiar words. Add a short list when it makes
distinct requirements easier to read. Let the task determine the length and
shape; neither a fixed template nor the fewest possible words is the goal.

Include context the receiver needs to understand the request, especially facts
that would otherwise remain in this conversation. Link relevant files or
accepted work with a brief explanation of their purpose. Repository
instructions and accepted artifacts supply the detailed working context;
a short restatement is useful when it makes the request understandable.

Preserve exact user-supplied values, identifiers, behavior, scope, and authority
limits. Carry explicit phase stops and external-effect boundaries across a
handoff. Distinguish a suspected cause from a verified fact. Use
[Intake](spec-first-workflow/phases/intake.md) when a missing user-owned decision
would materially change the request; technical choices belong to the receiver.

## Leave Room For Judgment

Describe the desired outcome and real constraints. Leave investigation order,
tools, implementation details, and routine validation to the executor unless
the user has chosen them or an established requirement makes them necessary.
An uncertain starting point can be a useful lead without becoming an instruction
to follow a particular path.

Avoid role play, motivational language, prompt-engineering formulas, prescribed
reasoning steps, and reminders to be competent or thorough. Do not invent scope,
acceptance gates, approval rounds, or stop conditions while polishing a request.
The receiver discovers repository instructions through its normal environment.

## Entry Points And Handoffs

A ready artifact can make the message very short: ask the receiver to continue
the work it describes and include any new context or limit. Use a native skill
invocation when it selects the requested workflow, without reducing every
message to an invocation and a path.

Use `$<skill>` in Codex and `/<skill>` in Claude Code, Qwen Code, Grok Build,
Cursor, or OpenCode. The syntax selects the entry point; it does not change the
task. In Grok Build, the primary-session message launches the work. OpenCode
discovers `.agents/skills` through the `skill` tool; `/orchestrator` binds the
Orchestrator carrier.

Use the [Subagent Brief Template](subagent-brief-template.md) for information
needed by a delegated lane and [Transition](spec-first-workflow/shared/transition.md)
for boundary state. Required machine fields and exact protocol tokens keep
their existing contracts; the accompanying request can still use natural prose.

## Completion

The message is ready when a capable colleague could understand the task,
recognize the result and real limits, and begin without reconstructing the
conversation. Remove wording that manages their every move or adds no useful
meaning; keep the context that makes the request clear.
