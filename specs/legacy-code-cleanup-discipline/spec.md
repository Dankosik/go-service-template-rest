# Legacy Code Cleanup Discipline

Mode: lean local
Status: approved for planning

## Intent

Make removal of replaced or unused legacy code an explicit repository discipline instead of an optional cleanup note. When a change replaces an old path, implementation should either remove the old code and its dependent artifacts inside the accepted scope or record why the legacy surface remains intentionally active with a bounded owner, proof, and exit condition.

The goal is to make cleanup visible in the same artifact chain that already controls spec-first work: repository invariants, task-local specs, task ledgers, coding guidance, review guidance, and validation evidence.

## Scope / Non-goals

In:
- repository workflow authority and companion docs for spec-first work;
- task-local artifact obligations for specs, design context when present, tasks, review, and closeout;
- source-managed skills that shape specification, planning, coding, review, and validation behavior;
- validation command documentation and any repo-owned guardrail surface needed to make the proof expectation discoverable;
- generated skill mirrors produced by the existing sync flow when source-managed skills change;
- task-local closeout evidence for this amendment after implementation.

Out:
- changing Go runtime behavior, public APIs, database schema, migrations, deployment policy, or CI job topology for this specification phase;
- implementing the docs, skill, or tooling changes in this session;
- creating `tasks.md` in this session;
- requiring blind deletion of code whose current runtime, compatibility, generated-source, fixture, or documentation role is not yet understood;
- promising a universal static dead-code detector for all possible Go or non-Go artifacts.

## Behavior / Contract Delta

ADDED:
- A repository invariant: replaced or unused legacy code is not acceptable as remembered-later cleanup. It must be removed, refactored into the active path, or explicitly retained with a current owner, reason, proof, and exit condition.
- Task-local specs that replace behavior must name known legacy surfaces, expected removal or retention semantics, and the proof that will show the old surface is gone or intentionally still used.
- Task ledgers for replacement work must include cleanup audit/removal tasks for code, tests, fixtures, generated artifacts, configs, docs, scripts, examples, and mirrors that belong to the replaced path.
- Coding guidance must treat cleanup made necessary by the approved change as in scope for that change, not as speculative side work.
- Review guidance must flag surviving replaced or unused code unless an approved artifact records why it remains.
- Validation must include targeted negative proof for retired identifiers, routes, configs, commands, generated files, fixtures, or documentation references, in addition to normal build/test/lint proof.

MODIFIED:
- Existing "do not leave cleanup for later" workflow language becomes an enforceable artifact, coding, review, and proof rule.
- Planning readiness must fail or reopen earlier phases when the implementation ledger omits known in-scope legacy removal.
- Implementation readiness must not pass when coding would need to decide whether legacy code is deleted, kept, or refactored.
- Completion claims must cover both the replacement behavior and the removal or explicit retention of the replaced surface.
- Informational cleanup tools such as `modernize-check` remain advisory; they do not replace artifact-declared cleanup proof for a specific replacement.

REMOVED:
- The implicit permission to ship a replacement while leaving the replaced implementation, tests, fixtures, configs, docs, or generated artifacts in place without a recorded keep decision.
- Review closeouts that prove only the new path works while ignoring stale old-path references.
- Planning handoffs that turn known target-state cleanup into follow-up work when it is part of the accepted scope.

## Decisions

- D1: `AGENTS.md` owns the compact repository invariant that in-scope replacement work includes removal or explicit retention of replaced and unused legacy code.
- D2: `docs/spec-first-workflow.md` owns the detailed artifact mechanics: where specs name legacy surfaces, where tasks carry removal work, where review checks it, and where validation records proof.
- D3: A retained legacy surface is allowed only when it has a recorded reason, current owner, proof of continued use or compatibility need, and an exit condition or reopen trigger. "Might be useful later" is not a valid retention reason.
- D4: The invariant applies to source code and adjacent artifacts that keep a replaced path alive or confusing: tests, fixtures, mocks, generated output, config, scripts, examples, docs, skills, agents, and mirrors.
- D5: Planning must represent legacy cleanup as executable tasking or explicit non-task rationale. A ready `tasks.md` must not leave deletion/refactor/keep decisions for implementation to infer.
- D6: Coding must remove stale code made obsolete by the approved task, unless the approved artifact chain says to retain it. Cleanup outside the accepted change remains out of scope.
- D7: Reviews should classify unexplained surviving legacy code as a merge-risk finding, not a style preference, because stale paths can hide source-of-truth drift and false proof.
- D8: Validation proof must be targeted and auditable. It should name the old surfaces or identifiers checked, the command or manual read used, and whether each surface was removed, refactored into the active path, or intentionally retained.
- D9: The repository should not pretend existing Go tooling can prove every dead-code case. Existing compile, lint, test, drift, mirror, and search checks are proof ingredients; artifact-declared negative checks close the task-specific gap.
- D10: Generated and mirrored artifacts follow source-of-truth discipline. Remove or update the owning source first, regenerate through repo commands, and verify drift checks instead of hand-editing derived files as the primary cleanup.
- D11: If a task discovers legacy code whose removal would change public contract, data behavior, security, reliability, rollout, or another protected domain, the task reopens the owning earlier phase instead of deleting silently.
- D12: This work remains lean local because it is a bounded repository workflow/coding-discipline change with no direct runtime API, data, security, money, reliability, concurrency, or deployment behavior.

## Subagent Gate Decision

Gate type: lean-local specification gate.

Required lane policy: local-only rationale.

Local-only rationale:
- The approval question is concentrated in one repository policy seam: how future specs, plans, coding, review, and validation must treat replaced or unused legacy code.
- No public API, persisted data, security, money, runtime reliability, concurrency, or deployment behavior changes in this specification phase.
- The available multi-agent execution tool is restricted to explicit user requests for sub-agents, and this request asked for specification only rather than delegated lane work.
- Repository authority docs and the requested skill reads provide enough source evidence to write a planning-ready lean spec without inventing downstream implementation details.

Result: PASS for lean-local specification. Planning may start from this spec with the proof obligations below. No subagent-derived blockers remain unresolved.

## Compact Design

Affected surfaces:
- `AGENTS.md`
- `docs/spec-first-workflow.md`
- `docs/build-test-and-development-commands.md`
- `docs/subagent-contract.md`
- `docs/subagent-brief-template.md`
- `.agents/skills/spec-document-designer/SKILL.md`
- `.agents/skills/go-design-spec/SKILL.md`
- `.agents/skills/planning-and-task-breakdown/SKILL.md`
- `.agents/skills/go-coder/SKILL.md`
- `.agents/skills/go-design-review/SKILL.md`
- `.agents/skills/go-language-simplifier-review/SKILL.md`
- `.agents/skills/go-qa-review/SKILL.md`
- `.agents/skills/go-verification-before-completion/SKILL.md`
- `.agents/skills/validation-closeout-session/SKILL.md`
- existing skill mirrors generated by `make skills-sync`
- existing repository validation or guardrail scripts only where planning identifies a deterministic, low-noise check that supports this invariant without pretending to solve global dead-code detection

Ownership / source of truth:
- `AGENTS.md` owns the hard invariant and scope boundary.
- `docs/spec-first-workflow.md` owns phase mechanics, artifact placement, readiness gates, and reopen behavior.
- `docs/build-test-and-development-commands.md` owns discoverable validation commands and when to use them.
- `spec-document-designer` owns spec-level cleanup decisions and assumptions.
- `go-design-spec` owns design-level source-of-truth, dependency, generated-artifact, and retention consequences when separate design depth is triggered.
- `planning-and-task-breakdown` owns task-ledger coverage and implementation-readiness enforcement.
- `go-coder` owns implementation-time cleanup discipline within the approved scope.
- Review skills own read-only findings for unexplained surviving legacy code in their domains.
- Validation skills own closeout proof wording and fresh-evidence discipline.
- Source-managed `.agents/skills/*` files remain canonical; runtime mirrors are derived.

Sequence / failure behavior:
- Specification names replaced or unused legacy surfaces when they are known at approval time, or records a bounded assumption and reopen trigger when discovery belongs to planning or design.
- Planning converts the cleanup invariant into concrete tasking: identify old surfaces, delete/refactor/retain them, update dependent tests/docs/generated/mirrors, and name the proof for each meaningful surface.
- Coding implements the active replacement and cleanup in the same accepted target-state change. If removal is unsafe or changes a protected behavior, coding stops and routes to the smallest owning reopen target.
- Review checks both directions: the new path satisfies the approved behavior, and the old path is gone or intentionally retained.
- Validation records fresh evidence for the changed surface, including targeted negative searches or reads, relevant build/lint/test/drift checks, and explicit retained-surface proof when applicable.

Separate design depth:
- Not expected for this amendment. The compact design above is enough for planning because the work is a workflow/coding-discipline update with stable source-of-truth ownership.
- If planning finds that deterministic tooling enforcement needs a new CI contract, generated artifact model, or broad guardrail redesign, reopen technical design before approving `tasks.md`.

## Assumptions / Reopen Conditions

- [assumption] This amendment can be implemented by updating repository authority docs, source-managed skills, validation command docs, and only narrowly scoped guardrail tooling if it proves useful.
- [assumption] The cleanup proof requirement can stay task-specific and evidence-based rather than relying on a universal dead-code detector.
- [reopen_spec_if_false] If implementation would make legacy removal mandatory across public compatibility windows, persisted data migrations, deployment rollouts, or generated contracts, reopen specification and reclassify the affected protected domain.
- [reopen_technical_design_if_needed] If planning cannot define deterministic validation or guardrail behavior without designing new tooling semantics, stop planning and open technical design.

No implementation-blocking open questions remain for planning.

## Risk Challenge

1. What irreversible or externally visible decision could be wrong?
   Answer: Over-aggressive deletion could remove code that still carries compatibility, generated-source, fixture, documentation, or optional-adapter value. The spec avoids blind deletion by requiring explicit remove/refactor/retain classification with owner, reason, proof, and exit condition for retained legacy.
2. What hidden invariant or owner could this break?
   Answer: Generated and mirrored artifacts have source-of-truth ownership. The cleanup rule must update owning sources and then regenerate or verify drift, not hand-edit derived files or treat mirrors as independent authorities.
3. What fresh proof will make the completion claim trustworthy?
   Answer: Targeted legacy-surface search/read evidence, repo-owned build/lint/test/drift checks matched to the changed surface, skill mirror sync/check proof when skills change, and `rtk git diff --check`.

Gate: PASS

## Task Handoff / Workflow State

Current phase: specification complete.

Next phase: planning only.

Planning output:
- create `specs/legacy-code-cleanup-discipline/tasks.md`;
- run and record the post-ledger task review/readiness gate;
- stop after planning with updated workflow state and the next-session prompt for implementation if readiness is `PASS` or eligible `CONCERNS`.

Planning constraints:
- Do not implement docs, skill, tooling, or generated mirror changes during planning.
- Do not create separate technical design artifacts unless planning proves compact design is insufficient; if so, reopen the correct design phase instead of filling the gap inside `tasks.md`.
- The ledger must cover the accepted target state, including source docs, source-managed skills, generated mirrors, validation docs, and any narrow guardrail/tooling work planning determines is required.
- The ledger must carry proof obligations for cleanup classification, mirror sync/checks, changed-file scope audit, and whitespace/drift checks.

## Validation

Specification-phase proof:
- This session created only `specs/legacy-code-cleanup-discipline/spec.md`.
- `rtk git diff --check -- specs/legacy-code-cleanup-discipline/spec.md` passed.
- Placeholder scan over the spec returned no unresolved placeholder markers before this validation note was recorded.
- `rtk sed -n '1,260p' specs/legacy-code-cleanup-discipline/spec.md` reread the full spec.
- `rtk git status --short --branch` confirmed the new task-local spec directory and showed pre-existing unrelated dirty workflow/skill files still present.

Implementation-phase proof:
- Scope: repository authority docs, spec-first workflow mechanics, subagent docs/templates, validation command docs, narrow guardrail script checks, canonical `.agents/skills`, generated skill mirrors, and task-local closeout evidence. No Go runtime behavior, public API, database schema, migrations, deployment policy, or CI job topology was changed.
- Changed-file audit: `rtk git status --short --branch` showed the expected changed authority docs, canonical `.agents/skills`, generated skill mirrors, `scripts/ci/required-guardrails-check.sh`, and this task bundle, plus unrelated pre-existing `SOUL.md` / `specs/orchestrator-soul-md/` work that was not reverted.
- Targeted invariant proof: `rtk rg -n` checks found the cleanup invariant, retention requirements, spec/planning/coding/review/validation/reopen mechanics, subagent legacy-surface reporting, validation-command docs, and source-managed skill instructions in the changed surfaces.
- Legacy-surface audit: the prior permissive "cleanup later" style wording was refactored into the active remove/refactor/retain invariant; `modernize-check` remains intentionally retained as informational only and is documented as not replacing task-specific cleanup proof; `optional cleanup` / `record_only` language remains only as taste/local-style review classification in `go-design-spec` and mirrors, bounded by the new invariant if it becomes ambiguous.
- Negative proof: `rtk bash -lc 'if rg -n "Might be useful later|cleanup is optional|optionally clean up|leave cleanup for later|remember cleanup later" ...; then ...; else ...; fi'` reported retired permissive cleanup wording absent from authority docs, workflow docs, canonical skills, and generated skill mirrors.
- Source and mirror proof: `rtk make skills-sync` passed; `rtk make skills-check` passed. `scripts/dev/sync-agents.sh` was read, and `rtk git diff --name-only -- .codex/agents .claude/agents` returned no changed agent source or mirror files, so `agents-sync` / `agents-check` were not applicable.
- Guardrail proof: `rtk bash -n scripts/ci/required-guardrails-check.sh` passed, and `rtk make guardrails-check` passed.
- Drift and whitespace proof: `BASE_REF=origin/main HEAD_REF=HEAD rtk make docs-drift-check` passed with "no files changed, docs drift check passed"; `rtk git diff --check` passed.
- Runtime proof: not applicable. `rtk bash -lc 'changed_go=$(git diff --name-only -- "*.go" "*.mod" "*.sum" ...); ...'` reported no runtime Go/module files changed.

## Outcome

Implementation verified for the approved lean-local scope. The amendment now records the legacy cleanup invariant in authority docs, workflow mechanics, subagent guidance, validation docs, canonical skills, generated skill mirrors, and narrow guardrail checks. All required ledger proof passed; runtime Go proof was not applicable because no runtime Go/module files changed. No residual blocker or reopen target remains for this amendment.
