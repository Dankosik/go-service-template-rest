# Implementation, Validation, And Closeout Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this when implementing from an approved ledger, running post-code review/reconciliation, validating, or closing out.

## Read When

- Implementation, review, validation, or closeout is next from an approved `tasks.md`.
- Post-code proof, negative legacy-surface proof, generated or mirror drift proof, or ledger-owned closeout updates are required.
- Validation exposes a missing decision, missing proof path, or old surface that may require reopening an earlier phase.

## Inputs

- Approved `tasks.md` first, then artifacts named by that ledger.
- Existing `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` only when the ledger explicitly names that pre-created phase file.
- Repository-owned validation commands and current workspace state.

## Outputs

- Code, tests, config, generated output, or docs required by the approved ledger.
- Updated existing `tasks.md` progress/evidence and ledger-owned closeout surfaces.
- Fresh validation evidence, or a recorded blocker with exact reopen target.

## Stop Rule

Do not create or approve missing pre-code workflow artifacts after implementation starts. If proof or validation exposes a missing decision or artifact, record the blocker in the allowed surface and reopen the owning earlier phase.

## Coding, Review, Reconciliation, And Validation

Coding consumes the approved task handoff. It may create or edit code, tests, migrations, configs, generation inputs, and generated output required by the task ledger.

Before adding substantial code to an existing hand-written source file, inspect its current responsibility, sibling files in the package, and package owner. If the new code is a distinct concern, abstraction level, mapping, validation, lifecycle, adapter, or test-helper policy, place it in a focused same-package seam file or the correct owner package instead of enlarging a catch-all file. If that split would change approved architecture, public contract, dependency direction, generated-source ownership, or another protected decision, stop and reopen the owning phase.

Coding may use the selected dependency or custom approach recorded in approved artifacts. If implementation discovers that the chosen approach needs a new runtime dependency, custom infrastructure, or a material helper/abstraction not covered by dependency/OSS due diligence, stop and reopen specification, technical design, or planning according to where the decision belongs. Do not add the dependency or build the custom substitute silently inside coding.

Coding may implement the selected design or system-design pattern recorded in approved artifacts. If implementation discovers that the chosen shape needs a different pattern, a previously rejected pattern, or a custom design not covered by Pattern Fit Diligence, stop and reopen research, specification, technical design, or planning according to where the decision belongs. Do not introduce a new pattern or pattern-like abstraction during coding just because it seems cleaner locally.

Coding may use local code-level patterns only to simplify approved behavior. Good candidates include table-driven tests for several meaningful cases, guard clauses, same-package policy seams, first-class function strategy, narrow consumer-owned interfaces, map-driven dispatch, middleware or decorator only at an existing composition seam, and functional options only when optional construction has real combinatorial pressure. If the pattern adds files, interfaces, callbacks, option bags, or indirection without reducing duplication, branch complexity, ownership ambiguity, or test burden, inline the code or use the stdlib/repo idiom instead.

Cleanup made necessary by the approved task is part of implementation scope. Coding removes stale old-path code and adjacent artifacts, refactors old code into the active path when that is the approved target state, or stops at the smallest reopen target when retention/removal would change public contract, data behavior, security, reliability, rollout, generated contracts, or another protected domain.

If implementation discovers an old surface not named by the approved spec or ledger, classify it before editing: in-scope and safe to remove/refactor, intentionally retained by an existing approved artifact, or requiring reopen because removal or retention changes contract, data, security, reliability, rollout, generated-source, or another protected-domain behavior.

Implementation sessions may continue across the approved `tasks.md` items and the ledger's named proof checks. They must not use implementation momentum to create or approve missing specification, design, planning, review, or validation-phase artifacts.

Post-code work is ledger-driven. It may update only:

- existing `tasks.md` checkbox/progress state;
- `spec.md` `Validation` and `Outcome`.
- existing `workflow-plans/review-phase-N.md` or `workflow-plans/validation-phase-N.md` only when the approved `tasks.md` explicitly names that pre-created phase file as part of the post-code checkpoint.

Do not update `workflow-plan.md` or phase-control files merely because they exist. After `tasks.md` is approved, those files are not the implementation source of truth.

Do not create new workflow/process artifacts after implementation starts. Reopen the earlier phase that owns the missing artifact instead.

Review is read-only and risk-driven. Review findings are advisory until the orchestrator reconciles them. Review should flag unexplained surviving replaced or unused code, tests, fixtures, configs, docs, generated artifacts, skills, agents, or mirrors as merge-risk findings unless an approved artifact records why the surface remains with owner, reason, proof, and exit condition.

Review should also flag custom implementations, newly added dependencies, or meaningful helper abstractions that lack approved stdlib, repository-pattern, and OSS due diligence. Severity depends on ownership risk, security/license exposure, transitive dependency cost, and whether a mature maintained library or standard-library path appears to satisfy the same contract.

Review should also flag architecture, workflow, integration, resilience, data-flow, or abstraction shapes that lack approved Pattern Fit Diligence when they appear invented, cargo-culted, or inconsistent with the selected pattern. Severity depends on whether the missing comparison could change ownership, interfaces, failure behavior, validation, rollout, or idiomatic Go implementation shape.

Review should also flag verbose local code that missed an obvious Go-native simplification or small code-level pattern, and pattern-shaped code that adds indirection without reducing duplication, branch complexity, ownership ambiguity, or test burden.

Review should also flag hand-written source files that grew into mixed-responsibility, multi-abstraction-level, or hard-to-review catch-all modules when the approved artifacts did not justify that placement. Severity depends on whether the file now hides ownership, couples unrelated concerns, blocks focused tests, or makes future changes likely to land in the wrong owner.

Severity for unexplained surviving replaced paths is risk-based: `high` when the old path can still execute, import, generate, or validate; `medium` for test, fixture, doc, config, skill, agent, or mirror drift; `low` only when the surface is clearly unreachable, non-authoritative, and unlikely to mislead future work.

Validation uses fresh evidence. A closeout claim is valid only when the commands or manual proof actually cover that claim, including targeted negative searches or reads for retired identifiers and references where text proof is reliable, retained-surface proof when old artifacts remain, generated or mirror drift proof when owning sources changed, and whitespace/drift checks for changed docs or tooling.

Negative proof must name the retired identifiers, paths, commands, config keys, generated files, fixtures, docs, skills, agents, or mirrors searched. A generic search such as `rg legacy` is not sufficient unless the retired surface is literally named `legacy`.

If implementation or validation discovers legacy cleanup that cannot be completed inside the approved scope, record the blocker in the allowed ledger or closeout surface and reopen the smallest owning phase: planning for missing tasking/proof, technical design for new ownership/tooling semantics, or specification for changed scope, public contract, protected-domain behavior, or retention policy.
