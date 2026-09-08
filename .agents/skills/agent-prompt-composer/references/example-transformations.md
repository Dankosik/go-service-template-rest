# Example Transformations

These examples calibrate tone and useful detail. They are work messages, not
formats to copy. Technical precision belongs where it defines the requested
result; the executor chooses the working method.

## Example 1: HTTP OPTIONS / CORS Policy Bug

Raw fixture: `files/http-options-cors.md`

```md
Please fix OPTIONS handling for existing routes. A normal OPTIONS request should return 204 with the correct Allow header. CORS preflight must still be rejected when CORS is disabled.

Keep the problem+json behavior stable, and leave OpenAPI unchanged unless the public contract actually changes. There may be relevant tests in router_contract_test.go.
```

## Example 2: Repo-Local Skill / Prompt Tooling Request

Raw fixture: `files/skill-tooling.md` (explicitly requests English output)

```md
Please update this repository's agent-prompt-composer skill so it can turn rough notes, dictation, and mixed-language input into a clear English task message for someone already working here. It should capture what the user wants and include useful repository context so the recipient can get started.

Keep the change local to this repository and update the examples to reflect it.
```

## Example 3: Flaky Shutdown / Drain Investigation

Raw fixture: `files/flaky-shutdown.md` (mixed language; this version uses English)

```md
Please investigate and fix the intermittent shutdown/drain hang. It looks like context canceled may be getting swallowed, or a worker may not be stopping. Bootstrap and health/readiness are possible starting points.

We need shutdown to finish reliably while preserving graceful drain and readiness behavior. Don't just increase the timeout to hide the hang. Check for a race or integration issue if the investigation points there.
```

## Example 4: Ready Native Orchestrator Entry

The user wants to continue an accepted ledger and stop before live rollout.
For Codex:

```text
$orchestrator
Please continue the work in specs/category-mapping-knn-first/tasks.md. Stop before live rollout.
```

Use the target environment's invocation syntax, such as `/orchestrator` in
Claude Code. The ledger supplies the detailed task state; the message explains
the request and preserves the user's rollout boundary.
