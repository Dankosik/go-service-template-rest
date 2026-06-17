# Implementation, Validation, And Closeout Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when implementing from an approved ledger, running post-code review/reconciliation, validating, or closing out.

## Read When

- Implementation, review, validation, or closeout is next from an approved `tasks.md`.
- Post-code proof, negative legacy-surface proof, generated or mirror drift proof, or ledger-owned closeout updates are required.
- Validation exposes a missing decision, missing proof path, or old surface that may require reopening an earlier phase.

## Inputs

- Approved `tasks.md` first, then artifacts named by that ledger, including `test-plan.md` when the ledger references scenario IDs.
- Existing `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` only when the ledger explicitly names that pre-created phase file.
- Repository-owned validation commands and current workspace state.

## Outputs

- Code, tests, config, generated output, or docs required by the approved ledger.
- Updated existing `tasks.md` progress/evidence and ledger-owned closeout surfaces.
- Fresh validation evidence, or a recorded blocker with exact reopen target.

## Stop Rule

Do not create or approve missing pre-code workflow artifacts after implementation starts. If proof or validation exposes a missing decision, missing `test-plan.md` scenario, or missing artifact, record the blocker in the allowed surface and reopen the owning earlier phase.

## Ledger-First Execution

Implementation starts from an approved, reviewed `tasks.md` or it does not start. Read `tasks.md` before `workflow-plan.md`, phase-control files, prior chat, or broad repository search. Then read only the artifacts the ledger names under `Read before coding` and the task-specific artifacts it names under `Read before relevant tasks`.

Before the first edit, read current workspace state and separate pre-existing unrelated changes from this session's work. Do not use unrelated dirty files as task evidence, do not clean them up without an explicit request, and do not let their presence widen the ledger's approved scope.

Before the first edit, set or verify the Codex Goal from the `tasks.md` Goal Contract and implementation handoff. The goal must cover every required ledger task through final validation, not only the first task or checkpoint. Also confirm the ledger records implementation readiness as `PASS`, eligible `CONCERNS`, or eligible `WAIVED`; identifies the first executable task or checkpoint; and separates successful completion from blocked-stop behavior. If readiness, the Goal Contract, `test-plan.md` scenario mapping when referenced, or the completion condition is missing, stale after repair, `FAIL`, or too vague to determine the next executable task and final proof, stop and reopen planning, test design, specification review, technical design review, or the owning earlier phase named by the gap.

At the start of each task, bind the task ID to its source anchor, dependencies, owner package/file, test-design scenario ID when present, proof, evidence fields, and stop or reopen condition. Do not infer hidden architecture, ownership, dependency, rollout, scenario class, proof level, or validation choices from chat memory. If a required field is absent or contradicts approved artifacts, leave the task unchecked and reopen the owning phase instead of deciding during implementation.

Execute tasks in dependency order through the ledger's final proof unless blocked. After each task or checkpoint, update the existing `tasks.md` checkbox and evidence fields with the command or read performed, result, key output or evidence reference, changed proof files when relevant, and any residual blocker or narrower claim. A task remains unchecked when proof is skipped, unavailable, stale, failing, cached in a way that does not prove the claim, or narrower than the task's stated behavior.

On resume, read current workspace state and `tasks.md` first, then continue at the first unchecked task whose dependencies and checkpoint gates are satisfied. Re-run only the proof needed to detect drift unless the ledger, changed surface, or failing evidence requires broader validation.

## Coding, Review, Reconciliation, And Validation

Coding consumes the approved task handoff. It may create or edit code, tests, migrations, configs, generation inputs, generated output, or docs required by the task ledger.

Before adding substantial code to an existing hand-written source file, inspect its current responsibility, sibling files in the package, and package owner. Record or satisfy the ledger's owner-file/package placement decision before editing. If the new code is a distinct concern, abstraction level, mapping, validation, lifecycle, adapter, or test-helper policy, place it in a focused same-package seam file or the correct owner package instead of enlarging a catch-all file. If that split would change approved architecture, public contract, dependency direction, generated-source ownership, or another protected decision, stop and reopen the owning phase.

Coding may use the selected dependency or custom approach recorded in approved artifacts. If implementation discovers that the chosen approach needs a new runtime dependency, custom infrastructure, or a material helper/abstraction not covered by dependency/OSS due diligence, stop and reopen specification, technical design, or planning according to where the decision belongs. Do not add the dependency or build the custom substitute silently inside coding.

Coding may implement the selected design or system-design pattern recorded in approved artifacts. If implementation discovers that the chosen shape needs a different pattern, a previously rejected pattern, or a custom design not covered by Pattern Fit Diligence, stop and reopen research, specification, technical design, or planning according to where the decision belongs. Do not introduce a new pattern or pattern-like abstraction during coding just because it seems cleaner locally.

Coding may use local code-level patterns only to simplify approved behavior. Good candidates include table-driven tests for several meaningful cases, guard clauses, same-package policy seams, first-class function strategy, narrow consumer-owned interfaces, map-driven dispatch, middleware or decorator only at an existing composition seam, and functional options only when optional construction has real combinatorial pressure. If the pattern adds files, interfaces, callbacks, option bags, or indirection without reducing duplication, branch complexity, ownership ambiguity, or test burden, inline the code or use the stdlib/repo idiom instead.

Cleanup made necessary by the approved task is part of implementation scope. Coding removes stale old-path code and adjacent artifacts, refactors old code into the active path when that is the approved target state, or stops at the smallest reopen target when retention/removal would change public contract, data behavior, security, reliability, rollout, generated contracts, or another protected domain.

If implementation discovers an old surface not named by the approved spec or ledger, classify it before editing: in-scope and safe to remove/refactor, intentionally retained by an existing approved artifact, or requiring reopen because removal or retention changes contract, data, security, reliability, rollout, generated-source, or another protected-domain behavior.

Implementation sessions may continue across the approved `tasks.md` items and the ledger's named proof checks. They must not use implementation momentum to create or approve missing specification, design, test-design, planning, review, or validation-phase artifacts.

Post-code work is ledger-driven. It may update only:

- existing `tasks.md` checkbox/progress state;
- ledger-owned `spec.md` `Validation` and `Outcome` when the approved `tasks.md` requires closeout and the task has a spec;
- existing `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` only when the approved `tasks.md` explicitly names that pre-created phase file as part of the post-code checkpoint.

Do not update `workflow-plan.md` or phase-control files merely because they exist. After `tasks.md` is approved, those files are not the implementation source of truth.

Do not create new workflow/process artifacts after implementation starts. Reopen the earlier phase that owns the missing artifact instead.

Review is read-only and risk-driven. Review findings are advisory until the orchestrator reconciles them. During orchestrator reconciliation or implementation, fix findings that are inside the approved ledger and proof path. If a finding requires a new decision, missing test-design scenario, missing artifact, broader validation policy, changed dependency choice, generated-source authority change, or retention/removal decision outside the approved ledger, record the blocker in the allowed surface and reopen the owning earlier phase instead of creating a new pre-code artifact after coding starts.

Review should flag unexplained surviving replaced or unused code, tests, fixtures, configs, docs, generated artifacts, skills, agents, or mirrors as merge-risk findings unless an approved artifact records why the surface remains with owner, reason, proof, and exit condition.

Review should also flag custom implementations, newly added dependencies, or meaningful helper abstractions that lack approved stdlib, repository-pattern, and OSS due diligence. Severity depends on ownership risk, security/license exposure, transitive dependency cost, and whether a mature maintained library or standard-library path appears to satisfy the same contract.

Review should also flag architecture, workflow, integration, resilience, data-flow, or abstraction shapes that lack approved Pattern Fit Diligence when they appear invented, cargo-culted, or inconsistent with the selected pattern. Severity depends on whether the missing comparison could change ownership, interfaces, failure behavior, validation, rollout, or idiomatic Go implementation shape.

Review should also flag verbose local code that missed an obvious Go-native simplification or small code-level pattern, and pattern-shaped code that adds indirection without reducing duplication, branch complexity, ownership ambiguity, or test burden.

Review should also flag hand-written source files that grew into mixed-responsibility, multi-abstraction-level, or hard-to-review catch-all modules when the approved artifacts did not justify that placement. Severity depends on whether the file now hides ownership, couples unrelated concerns, blocks focused tests, or makes future changes likely to land in the wrong owner.

Severity for unexplained surviving replaced paths is risk-based: `high` when the old path can still execute, import, generate, or validate; `medium` for test, fixture, doc, config, skill, agent, or mirror drift; `low` only when the surface is clearly unreachable, non-authoritative, and unlikely to mislead future work.

Post-code review findings close only when the task evidence names the finding, the action taken, the proof that covers it, and the residual risk or narrower claim. A reviewer report, subagent summary, cached command, or green unrelated check is not proof by itself.

Validation uses fresh evidence matched to the changed surface and the ledger's completion condition. A closeout claim is valid only when the commands or manual proof actually cover that claim, including targeted behavior proof, repository-owned validation commands, targeted negative searches or reads for retired identifiers and references where text proof is reliable, retained-surface proof when old artifacts remain, generated or mirror drift proof when owning sources changed, and whitespace/drift checks for changed docs or tooling.

Negative proof must name the retired identifiers, paths, commands, config keys, generated files, fixtures, docs, skills, agents, or mirrors searched. A generic search such as `rg legacy` is not sufficient unless the retired surface is literally named `legacy`.

Generated or mirrored-surface proof must name the authoritative source, generator or sync command, expected derived paths, and drift check. If the source changed but the generated or mirrored output cannot be regenerated or proven current, leave the relevant task unchecked and record the missing command, failing output, or unavailable environment as a blocker. Do not hand-edit generated or mirrored output unless the approved ledger explicitly authorizes that path.

Validation evidence should be no broader or narrower than the claim. A package-level test can prove a package claim; repository readiness needs the repository-owned command set for the changed surfaces; generated API, migration, sqlc, docs, agent, skill, or mirror changes need their drift checks when those surfaces are in scope. When a required command is unavailable, record the unavailable command, why it could not run, the narrower evidence that did run, and the residual unverified claim.

Before closeout, map every changed file from the final diff to a `tasks.md` task, checkpoint, or ledger-owned closeout surface, and verify that the task evidence names the proof covering that file. Any changed file that cannot be mapped is either unrelated pre-existing work to leave alone, an accidental edit to remove, or a blocker/reopen signal because the approved ledger did not cover it.

Closeout is complete only when all required ledger tasks and checkpoint gates are checked with current evidence, required validation passes, ledger-owned `spec.md` `Validation` and `Outcome` updates are current when the task has a spec, and any pre-created review or validation phase file explicitly named by `tasks.md` is updated. Do not claim completion from a recorded blocker, unchecked task, stale proof, or proof that validates a neighboring surface instead of the changed one.

Final responses must clamp the claim to the recorded evidence. Use `done`, `complete`, `passed`, or equivalent success language only when the ledger completion condition is satisfied with fresh proof; otherwise report `blocked`, `partially verified`, or `not verified` with the exact failing or missing proof and reopen target.

If implementation or validation discovers legacy cleanup that cannot be completed inside the approved scope, record the blocker in the allowed ledger or closeout surface and reopen the smallest owning phase: planning for missing tasking/proof, technical design for new ownership/tooling semantics, or specification for changed scope, public contract, protected-domain behavior, or retention policy.

Blocker records must be specific enough for the reopened phase to act without chat history. Include the task ID or checkpoint, failing or missing command/read, exact missing decision or artifact, affected surface, evidence already gathered, tasks left unchecked, narrower claim if any, and the owning reopen target. A blocker is a valid stop, not a successful closeout claim.
