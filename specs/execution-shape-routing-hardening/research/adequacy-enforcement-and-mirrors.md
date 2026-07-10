# R3: Adequacy, Enforcement, And Mirrors

## Question And Scope

What semantic adequacy checks, guardrails, eval cases, CI hooks, and mirror-availability contract are required to regression-protect repaired routing behavior?

Coverage: `B01-F06`, `B01-F09`, and `B01-F11`. This lane was read-only, advisory, and used explicit `no-skill`. Candidate enforcement models below remain specification inputs.

## Confirmed Authority And Current Behavior

### Shape Authority

- `AGENTS.md` owns shape triggers and hard boundaries (`AGENTS.md:17-24`; `docs/spec-first-workflow.md:5-7`).
- The authoritative matrix requires the smallest correctness-preserving shape and sends every protected trigger, broad audit/review, user-requested subagents, or explicit strict boundary to full orchestrated (`AGENTS.md:61-83`).
- Formal workflow-plan adequacy is currently described as required for full-orchestrated or high-risk work (`AGENTS.md:189-195`).
- The artifact model owns recording after a newly discovered trigger, but only says to block/condition the current artifact and move to the fuller route (`docs/spec-first-workflow/shared/artifact-model.md:53-57`).

### Adequacy Gate

- The adequacy skill is advisory and read-only, not an approval authority (`.agents/skills/workflow-plan-adequacy-challenge/SKILL.md:8-12`, `.agents/skills/workflow-plan-adequacy-challenge/SKILL.md:34-42`, `.agents/skills/workflow-plan-adequacy-challenge/SKILL.md:125`).
- It receives the selected execution shape as an input (`.agents/skills/workflow-plan-adequacy-challenge/SKILL.md:44-55`).
- Its checklist tests master/phase consistency, lane/gate status, artifact proportionality, handoff, and skip rationale (`.agents/skills/workflow-plan-adequacy-challenge/SKILL.md:59-76`).
- It does not explicitly load and evaluate every authoritative trigger, attempt to falsify the recorded shape, compare prior/target shape, or consume a stale-state/reclassification record.
- The challenger agent repeats the same selected-shape input boundary without requiring trigger-matrix evaluation (`.codex/agents/challenger-agent.toml:29-40`).

Evidence-backed inference: a packet can be internally proportional to a wrong recorded shape and still satisfy the current checklist.

### Trigger-Predicate Drift

| Surface | Current predicate |
| --- | --- |
| Authority | `full orchestrated or high-risk` (`AGENTS.md:195`) |
| Adequacy skill description | `full-orchestrated, high-risk, complex workflow-control, or agent-backed` (`.agents/skills/workflow-plan-adequacy-challenge/SKILL.md:3`) |
| Workflow-planning core default | `full-orchestrated, high-risk, or agent-backed` (`.agents/skills/workflow-planning-session/SKILL.md:90`) |
| Workflow-planning execution step | Adds `complex workflow-control` (`.agents/skills/workflow-planning-session/SKILL.md:147-151`) |
| Planning reference | Direct skip; lightweight-local waiver/challenge; full or agent-backed requires challenge (`.agents/skills/workflow-planning-session/references/adequacy-challenge-and-stop-boundary.md:9-16`) |
| Challenger agent | Before `non-trivial or agent-backed` work (`.codex/agents/challenger-agent.toml:16-21`) |

Confirmed terminology gaps:

- `high-risk` is not a closed predicate in the trigger matrix, which instead uses `high-impact` plus named protected triggers (`AGENTS.md:68-83`).
- `complex workflow-control` is not defined in the inspected authority.
- Bare `agent-backed` is broader than authoritative `user-requested agent-backed` and conflicts with `B01-F07` (`AGENTS.md:70`).
- The reference emits `lightweight local`, while the wrapper limits that phrase to read-time compatibility (`.agents/skills/workflow-planning-session/SKILL.md:83-85`).

### Guardrail, Eval, Make, And CI Coverage

- `required-guardrails-check.sh` is primarily required-file and regex/negative-regex enforcement (`scripts/ci/required-guardrails-check.sh:4-49`, `scripts/ci/required-guardrails-check.sh:66-88`).
- Its required-file set does not include the adequacy skill, workflow-planning skill, workflow-status skill, or one required challenger config (`scripts/ci/required-guardrails-check.sh:4-49`).
- It checks authority links, selected phrases, subagent status, and read-only agent configuration, but not semantic shape/reclassification behavior (`scripts/ci/required-guardrails-check.sh:173-218`).
- Workflow-status guardrails currently assert only specification-review and test-design report fields (`scripts/ci/required-guardrails-check.sh:253-254`).
- Workflow-planning evals cover a full feature, tiny direct skip, and competing control files, but not authoritative trigger audit, false shape, downgrade, or reclassification (`.agents/skills/workflow-planning-session/evals/evals.json:5-37`).
- No eval manifest exists under `workflow-plan-adequacy-challenge`.
- Workflow-status evals cover readiness and a supplied direct waiver but not shape/adequacy reporting (`.agents/skills/workflow-status/evals/evals.json:5-81`).
- CI runs guardrails plus agent/skill mirror checks, but no skill-eval runner (`.github/workflows/ci.yml:36-46`; `.github/workflows/cd.yml:138-148`).
- `ci-local` does the same for these workflow surfaces (`Makefile:374-380`); Make maps guardrails and mirrors directly to scripts (`Makefile:450-463`).

No repository-owned skill-eval runner or CI-wired model behavior harness was found.

### Canonical And Mirror Availability

Agent surfaces:

- Canonical: `.codex/agents/*.toml`; the registry points at canonical configs (`.codex/config.toml:5-13`).
- Runtime mirror: `.claude/agents/*.md` (`scripts/dev/sync-agents.sh:7-22`).
- A clean checkout may omit the mirror; `--check` succeeds and reports absence (`scripts/dev/sync-agents.sh:119-124`).

Skill surfaces:

- Canonical: `.agents/skills`.
- Runtime mirrors: `.claude/skills`, `.gemini/skills`, `.github/skills`, `.cursor/skills`, and `.opencode/skills` (`scripts/dev/sync-skills.sh:7-15`, `scripts/dev/sync-skills.sh:20-35`).
- Missing targets return success; summary text distinguishes present and absent counts (`scripts/dev/sync-skills.sh:86-92`, `scripts/dev/sync-skills.sh:132-155`).

All runtime mirrors are ignored (`.gitignore:9-14`) and were absent in the current checkout. Read-only lane checks returned:

- `scripts/dev/sync-agents.sh --check` -> `agents check complete (mirror absent)`;
- `scripts/dev/sync-skills.sh --check --strict` -> `skills check complete (strict; 0 present, 5 absent)`.

Evidence-backed inference: current CI validates exact mirror drift only when a local mirror exists; on a normal clean checkout it validates successful absence, not generated equivalence.

## `B01-F06`: Adequacy Does Not Falsify Shape

The current gate tests adequacy for the recorded shape rather than independently checking whether that shape survives the authoritative trigger matrix. This is circular validation.

Primary evidence: `AGENTS.md:61-83`; `.agents/skills/workflow-plan-adequacy-challenge/SKILL.md:44-76`; task-local closure at `specs/execution-shape-routing-hardening/workflow-plan.md:67`.

Classification: `blocks_spec`.

### Candidate Models

1. **Normalized trigger audit plus independent falsification**
   - Workflow planning records every canonical trigger as true/false/unknown, evidence, selected/prior shape, and reclassification/stale-state pointer.
   - Adequacy independently applies the canonical matrix and blocks false direct/lean, unresolved downgrade triggers, unsubstantiated full, or undisposed stale state.
2. **Formal adequacy for every durable workflow-planning packet**
   - Any session creating or substantially repairing master plus phase-local control receives the challenge; artifactless direct remains local.
   - This is broader than current authority and needs an explicit policy decision.
3. **Two-stage check**
   - A deterministic local matrix check always precedes shape selection; formal adequacy remains trigger-scoped and validates the recorded evidence.

Research recommendation for specification to evaluate: combine models 1 and 3, and require formal adequacy when full/protected evidence, durable workflow planning, or a challenged-route reclassification makes independent review material. This is not approval.

## `B01-F09`: Trigger Predicate And Terms Diverge

At least five different trigger predicates exist, and `high-risk`, `complex workflow-control`, and bare `agent-backed` lack closed authoritative meanings.

Classification: `blocks_spec`.

### Candidate Models

1. **One closed authoritative predicate** using only canonical terms: selected/proposed full shape; true/unknown full-trigger evidence; durable workflow-planning packet; or downgrade/reclassification from a formally challenged route.
2. **Durable-control predicate**: formal challenge whenever dedicated workflow planning writes durable control; direct/lean without control use local check.
3. **Glossary mapping**: retain convenience terms only when each maps exactly to canonical trigger evidence.

Research recommendation for specification to evaluate: one closed predicate with lower surfaces linking to it; eliminate undefined synonyms rather than multiply mappings. This is not approval.

## `B01-F11`: Semantic Enforcement And Mirror State Are Not Proven

Presence/string checks and optional-absent mirror checks do not prove routing semantics, behavior eval coverage, or generated mirror equivalence.

Classification: semantic enforcement and declared mirror-state contract are `blocks_spec`; actual external runtime discovery behavior is `proof_only` unless the accepted scope explicitly requires it.

### Candidate Enforcement Models

#### R3-E1: Deterministic Routing Contract Fixtures Plus Behavioral Evals

- Add a deterministic checker for direct/lean/full, protected triggers, authorization intent, reclassification, stale state, phase-file triggers, and adequacy predicates.
- Run it from Make and merge-blocking CI.
- Add adequacy/workflow-planning/workflow-status behavioral evals.
- Do not call evals CI-covered until a real bounded, credential-safe runner is present and invoked.
- Retain regex guardrails for owner links, required files, and forbidden stale terminology.

#### R3-E2: Normalized Control-Block Validator

- Define a stable machine-readable block for trigger audit, shape, prior shape, adequacy requirement/result, routing revision, and reclassification pointer.
- Validate fixtures and current/changed task packets, not completed historical bundles.
- Pair with behavioral challenger evals.

This is stronger but depends on `B01-F03`/`B01-F04` choosing a schema.

#### R3-E3: Hermetic Eval-Gated CI

Run model behavior evals in CI only if a deterministic, credential-safe, bounded-cost runner exists. No such runner is currently evidenced, so this remains conditional.

Research recommendation: R3-E1 as the baseline, optionally adding R3-E2 after the typed state schema exists. Treat R3-E3 as conditional proof, not assumed coverage.

### Candidate Mirror-Availability Model

Distinguish:

- `canonical_available`;
- `mirror_optional_absent`;
- `mirror_present_in_sync`;
- `mirror_present_stale`;
- `mirror_required_missing` only for a consumer explicitly declared to require it.

Candidate CI behavior:

- validate canonical files directly;
- render mirrors into temporary directories and compare deterministic generation;
- fail stale present mirrors;
- report optional absence as unavailable, not `in sync`;
- prove absent, exact, stale, required-missing, and target-only-file cases.

This preserves ignored/generated mirrors; it does not require committing them.

## Rejected Alternatives

- Let adequacy select/reclassify shape or approve handoff.
- Copy the full trigger matrix independently into every skill and agent.
- Treat proportional artifacts as proof the shape is correct.
- Check only the shape string without evidence or prior-state comparison.
- Treat the authorization boilerplate as an agent-backed trigger.
- Use all non-trivial work as the challenge trigger; that defeats lean compactness.
- Keep bare `agent-backed`, undefined `complex workflow-control`, or newly emitted `lightweight local`.
- Rely on regex phrase presence as semantic proof.
- Call eval JSON CI-covered without a runner.
- Require ignored mirrors to be committed.
- Make optional absence fail without changing the declared availability contract.
- Scan completed historical bundles as if they were newly authored state.

## Required Fail-Before Proof

### Routing And Adequacy

- Direct with concurrency/retry trigger -> blocking false-shape finding.
- Lean with public/generated-contract trigger -> blocking escalation.
- Lean with unknown validation ownership -> block or full until evidence closes the trigger.
- Full selected only because skills/agents exist -> challenge smallest-shape compliance.
- Full-to-lean with unresolved trigger -> block.
- Lean-to-full without stale-state disposition -> block/reopen `F04`.
- Capability authorization only -> no full trigger.
- Substantive user-required agent evidence -> full/formal route.
- Challenger remains advisory/read-only and cannot approve or mutate state.
- Research not expected does not imply adequacy skip or same-session collapse.
- Phase-file trigger is checked independently of mandatory review.

### Propagation And CI

- Missing canonical adequacy skill/challenger config -> guardrail fail.
- Forbidden undefined trigger terms on lower surfaces -> guardrail fail.
- Missing false-lean/reclassification planning evals -> guardrail fail.
- Missing adequacy eval manifest -> guardrail fail.
- Missing workflow-status shape/adequacy eval -> guardrail fail.
- Missing Make/CI routing-contract checker invocation -> guardrail fail.
- `complete` adequacy with an unresolved blocking finding -> semantic check fail.

### Mirrors

- All mirrors absent -> explicit optional-absence result plus successful temporary render proof.
- Present exact -> pass.
- Present stale -> fail.
- Required consumer mirror absent -> fail.
- Optional target-only files follow declared strict/non-strict behavior.
- Read-only check leaves `git diff --exit-code` clean outside task-local research writes.

## Compatibility Consequences

- A required trigger-audit block can make active workflow packets incomplete; completed historical bundles remain untouched.
- Reclassification fields need deterministic read compatibility without pretending old packets satisfy new-write requirements.
- Expanding formal challenge to every durable lean packet is a policy change and must be explicit.
- Older `lightweight local` can remain a read alias but should not be emitted.
- Requiring committed mirrors would contradict the current generated-local model.
- A temporary-render check can strengthen CI without changing tracked mirror policy.

## Cross-Finding Constraints

- `F01`: direct writer eligibility belongs in direct sanity/adequacy evidence, but R3 cannot decide the actor.
- `F02`: the classifier writes trigger evidence; adequacy only falsifies it.
- `F03`: adequacy result must not overload artifact/phase/lifecycle status.
- `F04`: adequacy consumes one reclassification model and must not create another.
- `F05`: research skip is independent of adequacy and collapse.
- `F07`: capability authorization is not substantive agent-backed intent.
- `F08`: phase-file trigger is independently testable.
- `F10`: status needs shape, adequacy, reclassification, and freshness visibility.
- Broadening bare `agent-backed` would reopen `F07`/`F09`; requiring committed mirrors would create a new compatibility conflict.

## Missing Evidence And Specification Handoff

- No repository-owned skill-eval runner is present or CI-wired.
- External runtime discovery behavior for Claude, Gemini, GitHub, Cursor, and OpenCode is not provable from repository sources.
- No machine-readable canonical trigger model exists.
- No adequacy-specific eval corpus exists.
- No current CI result proves LLM behavioral compliance.

| Finding | Recommended specification destination | Reopen implication |
| --- | --- | --- |
| `B01-F06` | `Execution-Shape Evidence Contract` and `Adequacy Falsification Algorithm` | Unresolved evidence/reclassification owner reopens `F02`/`F04`, not local skill invention. |
| `B01-F09` | `Canonical Adequacy Predicate And Terminology` | Undefined predicate blocks spec completion; runtime term mismatch reopens specification/routing. |
| `B01-F11` | `Semantic Enforcement Matrix`, `Fail-Before Catalog`, `Canonical And Mirror Availability`, `CI/Make Proof` | Missing semantic model blocks spec; external runtime discovery remains proof-only unless scope expands. |

Design implication: these findings do not independently require `design/overview.md` if the specification can hold one predicate, evidence block, state/transition table, and proof matrix compactly.
