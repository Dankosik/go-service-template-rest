# Lean repository validation tooling

status: ready

## Scope and non-goals

- Remove the ignored template-sync candidate bundle under
  `.artifacts/test/template-sync-77da61e-2f72764`.
- Remove the unused hard-skills `size-report` command, its implementation,
  historical `baselineSkills` mapping, and tests. Retain `targetSkills`,
  `retiredSkills`, `domainSkills`, `selectedSkills`, `evalCategories`,
  `executionSkills`, and their helpers because the repository/scoped checker
  and generic eval corpus still consume them.
- Remove the dedicated hard-skills eval emitter and runner; the existing
  workflow behavior eval runner becomes the sole baseline/candidate adapter
  harness.
- Remove the docs-drift heuristic and every Make, Docker, CI, documentation,
  and skill-reference entrypoint that exists only for that heuristic.
- Remove delivery-skill routing, references, and eval expectations that still
  teach documentation drift as a script-backed CI gate while preserving their
  generated-code, OpenAPI, and SQLC drift guidance.
- Keep one public Make target family for template initialization and one name
  for each secret scan.
- Shrink repository guardrails to derived cross-file delivery, toolchain,
  branch-protection, and import-direction invariants.
- Shrink workflow instruction checks to required owners, structural headings,
  canonical links, and the behavior-eval manifest check.
- Do not weaken generated drift, OpenAPI/SQLC validation, lint/deadcode/NilAway,
  vulnerability, gosec, secret, migration, container, branch-protection,
  worktree-transfer, setup/doctor, workflow eval mutation coverage, or Docker
  parity behavior.
- Do not add a replacement framework, dependency, compatibility wrapper, or
  archive of the removed tooling.

## Behavior and contract delta

- `hard-skills-check` supports repository and scoped checks only. It no longer
  emits a selected eval corpus or reports historical skill-size comparisons.
- `workflow-behavior-evals.sh` remains the only executable model-evaluation
  harness and continues to require explicit targets, immutable baseline,
  executable runner/judge adapters, and cost authorization.
- CI no longer treats the presence of any documentation edit as proof that an
  arbitrary implementation or workflow change was documented. Documentation
  changes remain owned by the change and review, without the removed
  path-based proxy.
- `required-guardrails-check.sh` rejects mismatched derived surfaces: Go and
  Docker toolchains, analyzer wiring across Go tools/Make/Docker/workflows,
  migration image contents, branch-protection contexts, and forbidden app
  imports. It no longer freezes same-file policy values, comments, repository
  file inventories, or the existence and exact spelling of convenience
  commands.
- `workflow-instructions-check.sh` verifies structural ownership and links. It
  no longer freezes exact prose through task-specific regular expressions or
  retired-token searches.
- `template-init*` is the public initialization family; the duplicate
  `setup*`, `bootstrap-native`, and `bootstrap-docker` Make aliases are removed.
  `secret-scan` and `docker-secret-scan` remain canonical; plural aliases and
  the shell-only `agents-check` alias are removed.

## Invariants and edge cases

- Removing an entrypoint removes its tests, docs, help text, Docker dispatch,
  Make variables/targets, CI invocation, and self-preserving guardrail
  assertions in the same change.
- Generated drift stays deterministic and fail-closed for tracked and
  untracked OpenAPI/SQLC output.
- The retained workflow routing target still runs the Go hard-skills checker,
  the structural instruction checker, and the 66-case behavior manifest check.
- Security and release gates named as non-goals remain callable under their
  canonical Make and Docker target names.
- Local artifact deletion must not touch tracked files, `.cache`, or unrelated
  `.artifacts` paths.

## Decisions, constraints, and authorities

- The accepted audit and the user's instruction to fix all findings authorize
  removal of the named tracked tooling and ignored template-sync bundle.
- Current Makefile, workflow, script, and checker callers are the source of
  truth for reachability.
- The repository's existing generated, security, branch-protection, migration,
  and container commands remain the authority for their protected gates.
- No external-platform research is needed: the change removes repository-local
  wrappers and assertions without changing an external API or platform
  contract.
- No separate technical design is needed because the retained owners and
  execution paths already exist; no runtime mechanism or placement choice
  remains.
- No separate test design is needed because each delta has an existing
  compile/test or repository-gate oracle plus exact negative reachability
  checks.

## Success criteria and proof expectations

- No executable, caller, CI/Docker, operator-documentation, or skill-consumer
  reference remains to `size-report`, `emit-selected-evals`,
  `hard-skills-evals.sh`, `docs-drift-check`, removed Make aliases, or their
  deleted files. This task-local specification and ledger may retain those
  names as intentional historical removal evidence.
- No retained delivery-skill prose or eval oracle claims that documentation
  drift is enforced by a script-backed CI gate.
- The focused hard-skills Go tests, workflow routing, retained guardrails,
  behavior manifest, instruction-eval mutation harness, shell syntax, lint, Go
  security, and diff checks pass.
- A no-side-effect `make -n` wiring check proves each retained
  `template-init*` target dispatches `scripts/dev/setup.sh` with its exact
  current flag set.
- CI and Docker tooling contain no docs-drift or dedicated hard-skills-eval
  branches.
- The named ignored template-sync directory no longer exists and the tracked
  worktree remains free of unrelated changes.

## Risks, assumptions, and reopen conditions

- Risk: a private operator may invoke an undocumented removed command. Reopen
  only with concrete current usage; do not keep compatibility aliases
  speculatively.
- Risk: shrinking exact-prose checks can expose instruction drift that only
  review or behavior evals detect. Reopen the structural checker only when a
  concrete regression has a stable machine-verifiable invariant.
- Reopen delivery policy if a retained protected gate loses its canonical
  Make/Docker/CI path or its focused proof cannot run.
