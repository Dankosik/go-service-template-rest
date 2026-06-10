# Go Code / Ownership Design Phase

Detailed phase companion for `docs/spec-first-workflow.md`. Read this after system/integration design is complete or explicitly not expected, when the next design checkpoint must decide how the approved behavior belongs in idiomatic, maintainable Go code before technical design review and planning.

## Read When

- System/integration design is complete, compactly covered in reviewed `spec.md`, or explicitly not expected for the task.
- Separate design depth is triggered and planning would otherwise need to choose packages, files, responsibilities, dependency direction, local interfaces, composition roots, test ownership, helper placement, or cleanup of replaced code.
- The change risks growing large hand-written files, creating generic helper buckets, duplicating logic, introducing pattern-shaped scaffolding, or scattering one responsibility across multiple owners.
- The next phase is recorded as `go-code-ownership-design`, or the workflow is repairing Go code ownership design after review.

## Inputs

- Specification-review-approved `spec.md` and specification-review obligations.
- Completed system/integration design record or an explicit rationale that system/integration design is not expected.
- Current `workflow-plan.md` and `workflow-plans/go-code-ownership-design.md` when phase-local routing exists.
- `docs/repo-architecture.md` when package boundaries, dependency direction, bootstrap/app/infra edges, generated sources, or runtime ownership matter.
- Current source responsibility audit for affected packages: current owner files, sibling files, generated/manual authority, existing tests/proof owners, retained legacy surfaces, and approximate hand-written line counts when size affects placement.
- Approved Dependency/OSS Due Diligence and Pattern Fit decisions when they affect local abstractions, helpers, adapters, or package ownership.

## Outputs

- A Go code ownership design record in `design/overview.md`, `design/go-code-ownership.md`, `design/component-map.md`, `design/ownership-map.md`, or `design/dependency-graph.md`.
- A concrete owner package/file placement decision for each meaningful changed responsibility, or a blocker/reopen target when placement would require a missing system/spec decision.
- Recorded `Design fan-out (go-code/ownership): complete | scoped_down | local_only | blocked` with candidate code seams, lane table or rationale, fan-in result, and readiness consequence.
- Workflow-control updates that route next to `technical-design-review`, or to the smallest reopen target when Go code ownership design is blocked.

## Stop Rule

Stop after the Go code ownership design checkpoint is complete or blocked. Do not draft `tasks.md`, approve implementation readiness, start implementation, or perform technical design review in this phase.

## Ownership

Go code ownership design owns maintainability of the implementation shape inside the approved behavior. It translates system decisions into idiomatic Go package, file, responsibility, abstraction, and test boundaries.

Boundary with system/integration design: system/integration design owns component-level behavior, contracts, source-of-truth, runtime sequence, failure behavior, validation, and rollout. Go code ownership design owns import/package/file-level placement inside those approved components. If a package or file placement decision would change observable behavior, source of truth, runtime sequence, failure behavior, validation, rollout, or generated/manual contract authority, stop and reopen system/integration design or specification instead of deciding it here.

It owns:

- Owner package and owner file decisions for each changed responsibility.
- Current source responsibility evidence and rejected owner locations for each meaningful placement decision.
- Whether code belongs in an existing focused file, a new same-package seam file, a generated source, an adapter package, an application/service package, a repository package, or a composition root.
- Responsibility boundaries for hand-written files, including split-or-keep rationale for large or mixed-responsibility files.
- Local interfaces, dependency direction, composition, helper placement, test ownership, fixture ownership, and generated/manual boundaries.
- Removal, refactor, or retained-surface decisions for replaced code, tests, fixtures, scripts, docs, generated artifacts, skills, agents, or mirrors inside the accepted scope.
- Code-level Go pattern choices when they reduce duplication, branching, allocation, test burden, or change radius without creating class-oriented scaffolding or a mini-framework.

It does not own:

- Observable API behavior, external provider contracts, data consistency, rollout policy, retry semantics, or security policy changes. Reopen system/integration design or specification when package placement would require those decisions to change.
- Task order, implementation slices, checkpoint placement, command transcripts, or validation execution sequence.
- Broad style preferences that do not change ownership, reviewability, proof, or future change safety.

## Go-Native Design Principles

Use Go-native maintainability rules rather than class-oriented ceremony:

- Prefer explicit control flow, concrete types, small functions, and narrow packages over indirection layers.
- Use interfaces only where the consumer owns a behavior boundary, a test seam, or a stable cross-package dependency; do not create interface-per-struct scaffolding.
- Keep one source of truth for a policy, mapping, state transition, telemetry vocabulary, validation rule, or generated/manual boundary.
- Add a helper only when it removes meaningful duplication or encodes a stable policy boundary; avoid generic `utils`, `common`, or catch-all files.
- Prefer table-driven tests, guard clauses, first-class function strategies, map-driven dispatch, and narrow adapters only when they make the code smaller, easier to test, or safer to change.
- Treat SOLID as a pressure test, not a design pattern import: single responsibility, explicit dependencies, and substitutable boundaries matter; class hierarchy vocabulary does not.

## Artifact Shape

Use the smallest durable artifact shape that lets technical design review and planning verify code ownership without rediscovering the repository:

- Keep code ownership in `spec.md` `Compact Design` only for lean-local work where package/file ownership is concise and uncontested.
- Use `design/overview.md` as the entrypoint whenever separate design exists; it should link to the code ownership record and name whether system/integration design is complete or not expected.
- Use `design/go-code-ownership.md` when owner file/package decisions need their own artifact.
- Use or update `design/component-map.md` when affected packages, modules, generated surfaces, adapters, and components need mapping.
- Use or update `design/ownership-map.md` when source-of-truth ownership, dependency direction, generated-code authority, adapter responsibility, and non-owners need explicit mapping.
- Use `design/dependency-graph.md` when package or module dependency shape, coupling risk, or generated-code dependency flow needs reviewable proof.

Do not create all code design artifacts for completeness. Create the ones that carry planning-critical answers and record not-expected rationale for plausible omissions.

## Required Code Ownership Questions

Every completed Go code ownership design should answer these questions or mark them not applicable with evidence:

```text
Source responsibility audit:
Path/package | Current responsibility | Size/mixed-responsibility signal | Sibling files inspected | Generated/manual authority | Existing tests/proof owner | Placement consequence

Owner map:
Surface | Selected owner package/file or approved placement rule | Rejected owner locations | Responsibility | Keep/split rationale | Source/sibling evidence | Dependency direction | Cleanup/removal | Tests/proof owner | Reopen condition
```

Required questions:

- Can every changed responsibility be mapped to a selected system mechanism, source-of-truth owner, proof obligation, rejected-alternative rule, and system reopen trigger from system/integration design? If not, block to `system-integration-design` before owner mapping.
- Which existing files and packages currently own the affected responsibilities?
- Which affected source files, sibling files, generated surfaces, and tests were inspected before selecting ownership?
- Which files are large, mixed-responsibility, generated, or inappropriate owners for new code?
- Which new file or package, if any, is the narrowest owner, and why is it not a catch-all helper?
- Which plausible owner locations were rejected, and why?
- If the exact file is intentionally deferred to coding, what owner package or artifact surface and placement rule bound that choice?
- Which dependency direction is allowed, and which imports or ownership inversions are forbidden?
- Which local interfaces, adapters, constructors, or composition-root changes are justified by current needs?
- Which duplicate or replaced surfaces are removed, refactored into the active path, or retained with owner, reason, proof, and exit condition?
- Which tests own the behavior, edge cases, generated drift, and negative proof?
- Which Go-native pattern or straightforward shape was selected, which viable alternatives were rejected, and why no class-oriented scaffolding or mini-framework is warranted?

For large hand-written files, line count is a smoke signal, not the decision. Around 600+ lines, default toward a focused same-package seam file unless the file has one cohesive responsibility and adding the new code keeps that responsibility clearer than a split. Mixed abstraction levels, unrelated concerns, or hard-to-review growth require a focused owner file or package boundary regardless of line count; cohesive files may stay together when the audit explains why a split would make ownership less clear.

## Fan-Out

Before marking the checkpoint complete, identify code ownership seams that could change design correctness or planning readiness. Use read-only lanes when independent code-quality domains can change owner placement, package boundaries, abstraction choice, or proof.

Each lane must name one live-fork question, the inspect-first source targets, and the artifact section it could change. Do not spawn a lane for a domain that can only produce a constraint or proof obligation without changing package/file ownership, dependency direction, abstraction choice, cleanup, or test ownership.

Typical lanes:

- Architecture/package ownership and dependency direction.
- Go idiom, language simplification, interface ownership, and abstraction cost.
- Domain invariant placement, state-transition ownership, and side-effect boundaries.
- Data/repository/cache access placement and transaction boundaries.
- Security/authz/session/trust-boundary enforcement placement.
- Observability/log/metric/trace vocabulary ownership.
- QA/test ownership, deterministic fixtures, negative proof, and generated drift.
- Performance/concurrency ownership when hot paths, goroutines, locks, or allocation-sensitive code are involved.

Record the gate in this shape:

```text
Design fan-out (go-code/ownership): complete | scoped_down | local_only | blocked
Candidate seams: <code ownership seams considered>
Lane table: <lane id, lens/domain, live-fork question, artifact section it could change, planning blocker if unanswered, skill/no-skill, inspect-first target, read-only enforcement, status>
Collapsed seams: <duplicate or consequence-only seams folded into the integrated pass>
Escalation seams: <seams that require another lane, system design reopen, specification reopen, research, or user decision>
Fan-in outcome: <orchestrator reconciliation that changes or confirms the code ownership design>
Reviewer falsification handle: <how review can verify omitted lanes and collapsed seams>
Readiness consequence: <ready for technical-design-review | blocked | reopen system-integration-design/specification/research>
```

`local_only` is valid only with candidate-lane analysis proving no omitted lane can change code ownership correctness or planning readiness. For full-orchestrated, protected-domain, high-impact, or user-requested agent-backed design, `local_only` is not eligible.

## Handoff To Technical Design Review

This checkpoint is ready to hand off only when technical design review can test planning readiness without choosing package/file ownership.

Missing `Source responsibility audit` keeps this checkpoint blocked. It is not a planning concern and cannot be converted into a proof-only obligation.

The handoff must include:

- reviewed `spec.md` and specification-review result;
- completed system/integration design record or not-expected rationale;
- code ownership artifact inventory and trigger decisions;
- source responsibility audit, rejected owner locations, owner package/file map or approved placement rules, split-or-keep rationale, dependency direction, abstraction/pattern choices, generated/manual boundaries, cleanup decisions, and test ownership;
- accepted risks and proof obligations that planning must carry;
- explicit reopen conditions if planning or implementation discovers a missing system/spec/code ownership decision.

If planning would still need to decide where meaningful code belongs, whether to split a file, how responsibilities are divided, which dependency direction is legal, or which tests own proof, the handoff is not complete.

The handoff must let planning fill `Files:`, `Owner package/file:`, and `Placement evidence:` from approved design. If exact file choice is intentionally deferred, the handoff must already name the owning package or artifact surface, the allowed placement rule, and the first-task inspection bounds; unknown owner package, package boundary, generated/manual authority, cleanup owner, or test owner blocks handoff.
