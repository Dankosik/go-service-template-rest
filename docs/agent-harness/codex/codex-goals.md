# Codex Goals

## Read When

Read only before starting, inspecting, resuming, pausing, or clearing a Codex
Goal.

Vendor authority: [Follow a
goal](https://learn.chatgpt.com/use-cases/follow-goals), [Using Goals in
Codex](https://developers.openai.com/cookbook/examples/codex/using_goals_in_codex),
and the [configuration
reference](https://learn.chatgpt.com/docs/config-file/config-reference#configtoml).

`/goal <objective>` starts work. The objective is non-empty, at most 4,000
characters. The evaluator can inspect files, tests, logs, and artifacts, so name
the evidence rather than restating its expected transcript. Use `/goal` to
inspect and native pause, resume, or clear controls when available. A disabled
Goal is a capability blocker, not permission to emulate it.
