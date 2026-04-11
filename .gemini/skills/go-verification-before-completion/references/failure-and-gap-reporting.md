# Failure And Gap Reporting

## Behavior Change Thesis
When loaded for failed, skipped, missing, stale, cached, or weaker-than-claim proof, this file makes the model say "not verified" or "partially verified" with the blocking signal instead of burying the problem under a positive completion summary.

## When To Load
Load this when proof failed, was not run, was skipped, used cached results unexpectedly, required tools or services were unavailable, or the available evidence is weaker than the requested completion claim.

## Decision Rubric
- If a required command failed, lead with `Not verified:` and the failed command.
- If a command skipped the behavior it was meant to prove, treat it as a proof gap even when the exit code is 0.
- If evidence is narrower than the claim, use `Partially verified:` and state what is not proven.
- If cached results are acceptable for the claim, say that explicitly; if executed test bodies are required, rerun with `-count=1` or report the gap.
- Give the smallest next verification action that would turn the current gap into real proof.
- Do not end with success wording unless the evidence supports the whole claim or the remaining risk was explicitly accepted.

## Imitate
Use this shape for a failed command:

```text
Not verified: `make test` failed.
Signal: `FAIL ./internal/app/user`, with `TestCreateUser` failing on duplicate key handling.
Next verification action: fix the failing path, then rerun `go test ./internal/app/user -run '^TestCreateUser$' -count=1` and `make test`.
```

Use this shape for a missing or skipped command:

```text
Not verified: migration rehearsal did not run.
Signal: `make migration-validate` reported that `MIGRATION_DSN` was empty and Docker was unavailable, so it skipped migration validation.
Next verification action: provide `MIGRATION_DSN` or start Docker, then rerun `make migration-validate`.
```

Use this shape for weaker evidence:

```text
Partially verified: focused parser test passed, but repository tests were not run.
Evidence: `go test ./internal/app/parser -run '^TestParserRejectsTrailingJSON$' -count=1` passed.
Not proven: "all tests pass" or "ready for merge".
Next verification action: run `make test` and any changed-surface checks.
```

## Reject
| Plausible bad conclusion | Why it fails |
|---|---|
| "Done; only `make lint` failed" | A failed required command blocks the positive claim. |
| "Migration validated" after a skip message | The command did not rehearse the migration. |
| "Race detector clean" when the race command was not run because it is slow | Skipping a costly check is not proof. |
| "Ready" while omitting a failed optional-looking surface check that the change actually triggered | Triggered checks are part of the claim unless risk is explicitly accepted. |

## Agent Traps
- Exit status alone can be misleading for commands that intentionally skip when prerequisites are missing.
- `make check-full` can be narrower than full Docker-backed CI if Docker is unavailable and local targets skip integration, migration, or container checks.
- `test-fuzz-smoke` can exit successfully because no fuzz targets exist; that proves no fuzz smoke ran.
- Cached `go test` output is not bad by itself, but it changes what freshness claim you can honestly make.
- Do not phrase the final sentence as "ready" after a `Not verified` block.

## Validation Shape
Use one of three labels: `Verified`, `Partially verified`, or `Not verified`. Then state evidence, missing or failed proof, and the next verification action.
