# Required Checks And Change Scope

## Behavior Change Thesis
When loaded for symptom "which checks must block merge," this file makes the model require the single `ci-required` context instead of the likely mistake of enumerating individual job names — which leaves change-scope-skipped jobs pending forever and makes the pull request permanently unmergeable.

## When To Load
Load for required status checks, branch protection or ruleset contexts, merge queue readiness, change-scope routing, or a claim that a check "passed" when it was skipped.

## Decision Rubric
- Require exactly one context from `ci.yml`: `ci-required`. It runs `if: ${{ always() }}`, `needs` every other job, and asserts through `jq` that each job either succeeded or was legitimately skipped for the current change scope. Renaming or adding a job then cannot silently drop a gate.
- Job execution is routed by `repo-integrity` outputs `expensive_required` and `template_required`, not by trigger path filters — `ci.yml` deliberately has none. A docs-only change legitimately skips most jobs, and `ci-required` still evaluates them. Keep new scope rules in those outputs.
- A job skipped by its own `if:` reports success and satisfies a required context; a *workflow* skipped by a path or branch filter never reports at all and leaves the context pending. This asymmetry is why scope lives in job conditions here.
- `openapi-breaking` runs its comparison only when the base commit already contains `api/openapi/service.yaml`; otherwise the step is skipped and the job is green. On a branch that introduces the spec, green means "not compared," not "not breaking."
- `migration-validate` is a step inside `container-security`, not a job, and cannot be named as a status context.
- CodeQL reports per-language contexts named `Analyze (<language>)`. Require the aggregate `CodeQL`, which survives a matrix change.
- Merge queue needs the `merge_group` trigger on `ci.yml` before a ruleset requires those checks for queued merges.

## Imitate
- "Branch protection requires `ci-required` and `CodeQL`; the job inventory stays owned by `ci.yml`." Copy the single-stable-context habit.

## Reject
- "Require every job in `ci.yml`." Jobs skipped by change scope never report, so the ruleset blocks the merge with no way to satisfy it.
- "Add path filters to the workflow trigger to save minutes." That converts a reported skip into a pending required context.

## Agent Traps
- `ci-required`'s `jq` expression names jobs as literal strings, so a rename compiles fine and fails closed at merge time until the expression is updated in the same change.
- Nightly `reliability` and the `scorecard` workflow report separately and are not reachable through `ci-required`; treating either as merge evidence skips the gate that actually blocks.

## Validation Shape
Use the Actions run URL, the `ci-required` conclusion on the merge SHA, and the `repo-integrity` scope outputs that explain each skip.
