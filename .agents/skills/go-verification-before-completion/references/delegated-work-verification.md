# Delegated Work Verification

## When To Load
Load this when another agent, worker, prior session, CI snippet, or tool report says work is done, tests passed, findings were fixed, or code is ready for handoff.

## Trust Rule
Treat delegated reports as leads, not proof. Completion language must be based on the current workspace and commands you just inspected or ran. If delegated output includes command logs, check whether they match the current files, scope, and claim.

## Example Claims

| Claim | Sufficient proof | Insufficient proof |
|---|---|---|
| "The worker fixed it" | current diff inspected for the affected paths plus the focused reproducer passing now | worker final answer says fixed |
| "The reviewer finding is resolved" | changed code addresses the finding and the relevant command passes now | reviewer says no more findings, but no current command evidence |
| "The delegated tests passed" | you rerun the named command, or verify current CI output for the same commit and scope | pasted log from before later edits |
| "Ready to hand off delegated work" | current workspace status understood plus claim-scoped commands passing now | subagent summary plus uninspected local modifications |

## Exact Command Patterns

```bash
git status --short
git diff --stat
git diff -- .agents/skills/go-verification-before-completion
go test ./path/to/pkg -run '^TestName$' -count=1
go test ./path/to/pkg/...
make test
make lint
make openapi-check
make migration-validate
```

Use `git diff` to understand what the delegated work changed, but do not treat diff inspection as behavioral proof. Use the changed surface to choose the command set, then report only what those commands prove.

## Exa Source Links
- [Go command documentation](https://pkg.go.dev/cmd/go): command evidence is scoped to the package patterns and command target that ran.
- [testing package documentation](https://pkg.go.dev/testing): focused tests and subtests can be selected by name with `-run`.
- [Data Race Detector](https://go.dev/doc/articles/race_detector.html): race detector proof requires executing the relevant concurrent paths.

## Repo-Local Sources
- `docs/build-test-and-development-commands.md`: repository command targets for validating changed surfaces.
- `Makefile`: exact command recipes behind repository targets.
