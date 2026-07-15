# Goal
status: ready
Completion: Stage 1 is integrated with every canonical skill rewritten as a lean behavior-preserving adapter, the implicit specialist router and deterministic enforcement/eval infrastructure present, all 29 specialists still implicitly eligible, the full accepted local proof green, the task bundle removed, and the remaining live-behavior limitation reported without claiming Stage 2 activation.
Blocked stop: Stop the current task on any missing Stage 1 behavior, ownership, oracle, executable local proof, or unrelated dirty-tree conflict and reopen its named spec/design/test/planning owner. Missing runner, judge, cost authority, accepted Stage 1 baseline, or native runtime-event evidence does not block Stage 1; it excludes Stage 2 and must remain in closeout as unproved future work.

## Implementation entry prerequisite

After this ledger receives independent `PASS` and its status is set to `ready`,
but before creating the root Codex Goal or launching T1, root commits only the
exact ready `specs/lean-skill-instructions/` bundle on the implementation
branch and records that commit as the accepted starting state. This temporary
artifact commit implements no canonical change. Every fresh T1-T3 Worker starts
from the latest integrated branch that retains the tracked bundle; Workers do
not edit workflow status. If the ready bundle cannot be committed without
unrelated changes, Planning is not ready and must be repaired before
implementation.

After T1 and T2 are root-accepted, root updates only that task's checkbox and
acceptance evidence in `tasks.md`, commits the root-owned ledger update on the
integrated branch, and uses that branch for the next fresh Worker. These
progress/evidence bytes are expected and do not invalidate the accepted
decision content. T3 is terminal: root records its Worker result and acceptance
in transient execution context before integrating the bundle deletion, then
uses the frozen integrated diff and fresh terminal evidence for closeout. Any
change to spec, design, test strategy, task outcome, dependency, owner, or
proof semantics reopens the owning phase and requires fresh review before
another Worker starts.

- [x] T1: The existing canonical skill catalog and shared instruction owners are rewritten as lean behavioral adapters while preserving every skill name, authority boundary, unique invariant, escalation, and explicit entry point.
  - Source: `spec.md#behavior-and-contract-delta` items 1-7 and `design/overview.md#skill-family-rewrite`, `#target-instruction-topology`, and `#ownership-and-file-changes`.
  - Owner/surface: `docs/skill-authoring.md`; the single pointer in `AGENTS.md`; `.agents/skills/specialist-contract.md`; new non-triggerable `.agents/skills/specialist-arbitration.md`; every existing `.agents/skills/*/SKILL.md`; only those `.agents/skills/*/references/index.md` files needed to extract a non-trivial selector. Worker preserves the tracked `specs/lean-skill-instructions/**` starting-state bundle; after root acceptance, root may update only T1 progress/evidence before T2. Do not add `go-specialist-router`, change eval bundles/checker/harness files, or create any `agents/openai.yaml` in this task.
  - Depends on: none.
  - Proof: Catalog semantics and authority are preserved while default instruction surface is reduced; root reviews every skill diff against its baseline and TD-01, confirms each trigger/outcome/method/stop/return/reference boundary and every budget exception, and runs `go run ./scripts/ci/hard-skills-check`, `go run ./scripts/ci/hard-skills-check size-report --baseline-ref 34d9776`, and `git diff --check`; expected observable is a green existing hard-skill contract, a complete reviewed catalog with no lost unique invariant, valid links from every changed `SKILL.md`, materially smaller reported hard-skill bodies without a correctness threshold, no renamed/deleted skill, and no policy metadata.
  - Acceptance evidence: Root accepted native App Worker `019f67fc-0e6b-7c33-95a3-98868d26b3bd` after six correction/review passes and catalog-wide baseline/TD-01 inspection; T2's broader checks exposed and the same T1 Worker repaired the sole unquoted description plus the exact `grilling`, `go-coder`, and `go-test-implementation` routing entry points. Fresh root proof: hard-skill check exit 0; size report exit 0 with effective aggregate `27,747 -> 5,609` words; workflow routing check exit 0 across 20 canonical files; workflow behavior eval check exit 0 across 45 cases and 41 invariants; `git diff --check` exit 0; catalog-local Markdown links valid; all 52 existing descriptions are quoted one-line values; 35 semantic selector indexes present; task bundle otherwise unchanged; no renamed/deleted skill, router, policy metadata, or T2 surface change.
  - Reopen if: A skill cannot be shortened without losing a decision-changing invariant or explicit entry point; reopen Technical Design.

- [x] T2: Stage 1 gains the implicit `go-specialist-router`, full-catalog structural enforcement, normalized nine-skill/36-core eval definitions, derived answer-key sealing, cost-gated harness behavior, dormant fail-closed Stage 2 metadata/ABI checks, and accurate command documentation without activating explicit-only policies.
  - Source: `design/overview.md#specialist-routing`, `#checker-and-eval-ownership`, `#proof-and-cleanup`; `test-plan.md#scenario-matrix` TD-02 through TD-13, excluding unavailable live behavior and activation in TD-09/TD-11/TD-12.
  - Owner/surface: `.agents/skills/go-specialist-router/{SKILL.md,references/index.md,evals/evals.json}`; the five newly selected pilot eval bundles plus the existing three selected bundles only where the accepted normalization requires; `scripts/ci/hard-skills-check/{checker.go,inventory.go,checker_test.go,emission.go,size.go,main.go}` only as needed; `scripts/dev/hard-skills-evals.sh`; `docs/build-test-and-development-commands.md`. Reuse the existing checker and harness; add no package, dependency, generator, Make target, second scanner, hard word-count threshold, or `agents/openai.yaml`.
  - Depends on: T1.
  - Proof: The integrated Stage 1 catalog is structurally closed and its deterministic proof surfaces reject the accepted mutations; run `go test ./scripts/ci/hard-skills-check`, `make fmt-check`, `go run ./scripts/ci/hard-skills-check`, `bash scripts/dev/hard-skills-evals.sh check`, `make workflow-routing-check`, `make workflow-behavior-evals-check`, `make guardrails-check`, `go run ./scripts/ci/hard-skills-check size-report --baseline-ref 34d9776`, and `git diff --check`. Expected observable: all commands pass; checker discovers every triggerable skill and validates bundle links; invalid `trial_class` and partial/extra false-policy states fail in mutation tests while zero-policy Stage 1 passes; the deterministic manifest has nine selected skills and exactly 36 core cases; root semantic review confirms the four router oracles, the new ownership core, all converted core obligations, and unchanged auxiliary oracles; both runner snapshots omit all selected eval directories; check-only makes no adapter call; unauthorized run fails before adapters; size output includes per-skill, derived-family, and whole-catalog totals with no reduction gate. Fake-adapter success is recorded only as ABI/isolation proof, never native routing provenance or behavioral equivalence.
  - Acceptance evidence: Root accepted native App Worker `019f680f-1d5c-7001-8bb6-dc6295572e9c` after repeated proof-matrix, sealed-runner-ABI, and effective-size-accounting corrections. Fresh root proof: uncached checker suite 81 passed; formatting, checker, selected-eval check, routing, 45-case/41-invariant behavior eval, guardrails, vet, size report, and `git diff --check` all exited 0. The router maps all 29 derived specialists; the manifest has nine skills and exactly 36 core rows; all accepted link, `trial_class`, policy, metadata, cost/no-call, snapshot, argv/path, payload-seal, family, and catalog mutations are exercised; converted and auxiliary oracles are preserved; the task bundle was otherwise unchanged; zero policy files and no forbidden dependency/package/generator/Make surfaces were added. Fake adapters prove only ABI/isolation; live Stage 1 behavior and native implicit-router provenance remain unproved.
  - Reopen if: Triggerable/specialist/selected-set derivation cannot be made deterministic, a selected oracle must change meaning, or the closed ABI would need to trust self-declared provenance; reopen Technical Design/Test Design and keep zero policy files.

- [ ] T3: The accepted Stage 1 candidate is finalized, the temporary workflow bundle is removed at the final safe point, and terminal integrated proof is captured without performing or claiming live model evaluation or Stage 2 activation.
  - Source: `test-plan.md#scenario-matrix` TD-01, TD-04-TD-05, TD-11, TD-13-TD-14 and `design/overview.md#proof-and-cleanup`.
  - Owner/surface: Remove only `specs/lean-skill-instructions/` after T1 and T2 are root-accepted and their evidence is captured; root owns final integrated diff review, evidence reconciliation, and the post-integration ref-based docs-drift command. Canonical Stage 1 files from T1/T2 remain unchanged unless an implementation-owned failure is returned to its owning Worker before cleanup.
  - Depends on: T2.
  - Proof: Cleanup and full Stage 1 readiness are observable on the frozen bundle-free T3 managed-worktree candidate; before cleanup preserve accepted task evidence, then require `test ! -e specs/lean-skill-instructions`, inspect `git status --short`, and run `go test ./scripts/ci/hard-skills-check`, `make fmt-check`, `go run ./scripts/ci/hard-skills-check`, `bash scripts/dev/hard-skills-evals.sh check`, `make workflow-routing-check`, `make workflow-behavior-evals-check`, `make guardrails-check`, `go run ./scripts/ci/hard-skills-check size-report --baseline-ref 34d9776`, and `git diff --check`. Before root accepts T3 or integrates its deletion, root rechecks in that managed worktree whether `WORKFLOW_EVAL_RUNNER` and `WORKFLOW_EVAL_JUDGE` name executable adapters and `WORKFLOW_EVAL_COST_AUTHORIZED=true` is explicitly present. If all three are available, route and run TD-11's 36-case Stage 1 comparison from its fixed baseline there; otherwise record each exact missing input there. A failed authorized comparison blocks T3 acceptance and returns the evidence-backed defect to the owning T1/T2 Worker while the integrated branch still retains the task bundle. Only after T3 passes this conditional gate does root accept/integrate the deletion; then root runs `make docs-drift-check BASE_REF=9a1dc4fc01a628b593e9008ab2cdf895a8f8a02b HEAD_REF=<exact-accepted-stage-1-commit>`. Expected observable: task bundle absent from the accepted candidate, no unrelated deletion, all canonical files present, every local command green, docs drift sees the complete committed scripts/docs delta and passes, final diff matches only accepted Stage 1 scope, no `agents/openai.yaml` exists, and closeout either reports the actual authorized live result or names the unavailable inputs and states that live Stage 1 behavior and native router provenance were not proved.
  - Reopen if: The tracked bundle is absent, contains an unreviewed semantic authority change rather than root-only progress/evidence updates, or the ref-based docs gate cannot observe the complete candidate; reopen Planning before launching or accepting cleanup.

## No-implementation dispositions

- Stage 2's 29 `allow_implicit_invocation: false` policy files are excluded by
  the ready spec/design/test plan. They require a separate reopened design,
  proof plan, authority, and ledger after an accepted Stage 1 commit and an
  independently inspectable native runtime-event verifier exist.
- Live Stage 1 baseline/candidate comparison follows `test-plan.md` TD-11 and
  is currently unscheduled because runner, judge, and explicit model-cost
  authority are unavailable. Root rechecks those three inputs on the frozen,
  bundle-free T3 managed-worktree candidate before accepting or integrating
  T3: if all are available, run the accepted comparison there; otherwise record
  the exact missing inputs there. Structural, semantic-definition, and
  fake-adapter checks must never be reported as that proof.
- Existing skill names, explicit entry points, workflow phase authority,
  runtime service behavior, dependencies, packages, generators, and Make
  targets have no accepted change and therefore no implementation task.
