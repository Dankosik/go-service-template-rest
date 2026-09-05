# Implementation workflow efficiency: verification

Date: 2026-09-05. Accepted outcome: apply the six instruction changes from the
Billing retirement investigation to the source template. Preserve money/data
safety, independent review, explicit phase limits, and external-effect authority.
Active Billing candidates, ledgers, tasks, and consumer checkouts are not changed.

## Observed pressure and change

The retirement investigation covered the resumed September 5 execution and its
T04R migration recovery. Primary task identities are
`01a06e86-9e30-7fa0-93a2-ba56ad60b561`,
`01a066f6-6079-7f00-9b4e-aa09b94a8111`, and
`01a070c7-1815-7580-8922-7264889095b5`.

| Observed pressure | Changed owner and expected decision delta | Boundary retained |
| --- | --- | --- |
| Diagnostic 5s/1s limits were adopted as migration policy without runtime attribution. | Evidence Contract requires source, scope, and units before adopting hard constraints; Review challenges that attribution. | Correct the agent assumption through its decision owner; retain actual writer and safety budgets. |
| A novel relay was compiled before its real barrier was exercised; local proof initially depended on unavailable qualification hardware. | Test Design requires an existing witness or minimal authorized normal/fault probe when control/platform feasibility is unknown; Test Plan V1 distinguishes that witness from product acceptance. | Known controls need no extra probe; local correctness cannot replace production qualification. |
| Writer-candidate tasks waited on applied production schema although rollout had a separate owner. | Planning places dependencies at the first consuming action and checks whether isolated preparation can use a stable contract. | Existing packets must be corrected before dispatch; overlapping owners, auto-deploy merges, and real effects remain gated. |
| Fixture defects and mechanical identity updates risked reopening unaffected phases and reviews. | Implementation retains same-unit control repairs; Transition and Review retain unchanged semantic scope after an exact delta check. | Changed oracle, mechanism, risk, or admission still requires its owner and applicable review. |
| A root and Orchestrator both relayed status while recovery accumulated attempts. | Transition selects one active continuation coordinator and a discriminating result/work window for uncertain recovery. | Result delivery and parent responsibility survive; a checkpoint changes strategy, never authority or acceptance. |
| Coordinating and implementation checkouts used different workflow/model policies. | Agent Harness carries the selected revision/locators and checks effective native model/effort at actor or checkout changes. | Current user/native authority prevails; no bulk synchronization of active worktrees. |

These are refinements of conditional methods and ownership, not new safety
policy. Existing evidence reuse and bounded delta review remain canonical.
No new role, skill, workflow phase, scheduler, test runner, or journal was added.
Changes to the Go skills, AGENTS.md, Codex adapter, and skill-authoring document
that predated this task were preserved byte-for-byte.

## Fixed comparison

The eight changed files matched baseline commit
`e3ff15d11638bac1011260052c272b7eb7c3d045` before editing:

- `docs/agent-harness.md`
- `docs/spec-first-workflow/interfaces/test-plan-v1.md`
- `docs/spec-first-workflow/phases/implementation.md`
- `docs/spec-first-workflow/phases/planning/ledger-contract.md`
- `docs/spec-first-workflow/phases/test-design.md`
- `docs/spec-first-workflow/shared/evidence-contract.md`
- `docs/spec-first-workflow/shared/review.md`
- `docs/spec-first-workflow/shared/transition.md`

The frozen comparison also included the same five context documents in both
conditions: AGENTS.md, workflow router, Codex adapter, Planning, and Resume.
Some of that context contained pre-existing local edits. The 13-file candidate
manifest SHA-256 is
`963586bc4edbd6cad82aeb934c15af147dd1bdf1d1642b053ebf1ec76e5859b0`.
Frozen inputs, exact patch, cases, and manifest remain in the task-local
snapshot `/private/tmp/implementation-workflow-in2lhbx8/`; this temporary
location is not a durable dependency of the workflow.

The eight owners grew from 3,745 to 4,393 words. Bootstrap content, the number
of semantic owners, and the ordinary-path skill catalog were unchanged by this
task. Added text is conditional; this count measures instruction size, not speed.

The official [Astra guidance](https://developers.openai.com/api/docs/guides/latest-model)
was fetched during the preceding investigation on 2026-09-05. Its advice to
audit loaded instructions and calibrate verification informed these changes.
No model defaults or reasoning-effort policy changed in this task.

## Comparative interpretation replay

Two fresh native `reviewer-agent` subagents received identical replay briefs
and cases, differing only in the supplied frozen policy root:

- `workflow_probe_a`: baseline, `gpt-6-astra`, `high`, no inherited turns.
- `workflow_probe_b`: candidate, `gpt-6-astra`, `high`, no inherited turns.

Neither received expected answers or the other's results. They interpreted
hypothetical authoritative states and returned next actions, owners, held
proof/authority, and ambiguities. They did not execute the scenarios. Shared
native instructions and the supplied cases are confounders; one run per
condition is not a benchmark or a statistical comparison.

| Case input and question | Baseline result | Candidate result |
| --- | --- | --- |
| An accepted 5s DDL gate comes only from diagnostic PGOPTIONS; expensive proof is next. Who corrects it? | Hold acquisition and reopen the source decision; attribution procedure is underspecified. | Hold acquisition and correct source/scope through the decision owner. |
| A new TCP relay only compiles; 48 fault cases are proposed, local PostgreSQL is available, qualification hardware is not. What runs first? | Prefer a narrow barrier experiment; readiness without a witness remains ambiguous. | Require a normal/fault feasibility witness before affected readiness; keep qualification separate. |
| A parser correction uses an existing deterministic fixture and oracle. Is another probe needed? | No extra feasibility experiment. | No extra feasibility experiment. |
| A writer candidate depends on applied schema, but has stable local inputs; main auto-deploys. Can preparation proceed? | Planning must revise the dependency first; production effects remain gated. | Same disposition, with explicit first-consumer placement and auto-deploy boundary. |
| A fixture needs a SQL cast; separately someone proposes replacing independent commit observation with a success log. Who repairs each? | Lead repairs the cast; changed oracle returns upstream and cannot borrow delta-review approval. | Same safety/ownership result; ordinary control defects explicitly stay in the unit. |
| A root polls an Orchestrator that can deliver terminal/intervention messages. Which loop remains? | Prefer message delivery, but duplicate polling is not expressly excluded. | One active coordinator; former root retains terminal/intervention delivery without routine relaying. |
| Two identical EOF attempts exhaust a selected 20-minute window; a smaller authorized probe exists. What next? | Change probe without weakening safety or inventing authority. | Same result, with explicit checkpoint and unchanged unit/acceptance boundaries. |
| A selected Astra policy conflicts with an old worktree and native Terra inheritance. What is reconciled? | Verify effective model and resolve policy conflict; revision adoption is underspecified. | Reconcile selected revision/local owners and native state before the dependent decision. |
| Test Design only; heavy local work and spending denied; production writes unapproved. The witness needs an unavailable effect. Where does work stop? | Document the gap and stop; readiness without a witness is ambiguous. | Document/review the gap, do not claim affected readiness, and stop within Test Design. |

The baseline selected mostly the same safe next actions. The candidate made
several conditional requirements explicit and retained all nine cases' authority
boundaries. These observations support clearer interpretation, not a measured
reduction in real implementation time.

Both actors noted that the frozen subset omitted linked output-template files;
exact output-schema conformance was outside this replay. No changed workflow
rule depends on that reporting limitation.

## Structural proof and independent review

- `make template-owned-purity-check`: PASS, including role/config/skill parity
  and template-sync behavior. No Go, database, or image suite was needed.
- `git diff --check`: PASS.
- Relative file links in all eight changed owners resolve; baseline bytes match
  the named commit; all five pre-existing modified files retain their initial
  hashes.
- Fresh `workflow_change_review`, Astra/high: PASS, no findings. It verified all
  13 manifest entries and the eight-file delta, challenged all six outcomes,
  and rejected attempted oracle weakening, identity-based review bypass,
  unauthorized phase-only probes, and ordinary-work freezes.

No live delivery-speed improvement, production safety certification, consumer
adoption, or publication is claimed. Reopen measurement when comparing repeated
representative implementations with the same model, effort, harness, and inputs;
track time to the first working path, upstream reopens, harness repair time, and
retained required proof separately.
