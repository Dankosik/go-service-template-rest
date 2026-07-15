# Lean skill instruction design
status: ready

## Target instruction topology

The target keeps the existing authority chain and removes duplicate prompt
surface:

```text
AGENTS.md and workflow phase owners
  -> docs/skill-authoring.md
  -> thin session and direct-method skills
  -> go-specialist-router
       -> specialist-contract.md
       -> one specialist SKILL.md per materially affected independent axis
            -> references/index.md only when detail can change the result
                 -> one symptom-matched reference by default
```

`docs/skill-authoring.md` owns the authoring rule: a skill is a behavioral
adapter, not a knowledge base. `AGENTS.md` adds only a pointer under Instruction
Ownership. Existing workflow and phase files remain authoritative; their method
is linked, not restated.

`.agents/skills/specialist-contract.md` remains the shared non-triggerable owner
for primary-axis selection, spec/review boundaries, evidence, return shape, and
overlap arbitration. Compress it to the decisions every specialist needs.
Move all detailed maintainability, lifecycle, and neighboring-axis examples to
the sole non-triggerable owner `.agents/skills/specialist-arbitration.md`,
loaded only when ownership is ambiguous.

## Skill-family rewrite

Inspect every canonical `.agents/skills/*/SKILL.md` against the authoring
standard. Keep a compliant thin skill unchanged except for routing clarity;
rewrite every oversized or duplicate body.

- Session/router skills: link the owning phase, state owned outcome, stop rule,
  and return. Do not restate phase method.
- Specialist spec/review skills: link the shared contract; retain only the
  unique domain decision or invariant, decisive evidence model, named
  escalation owners, and checkable return.
- Direct-method skills (`go-coder`, `go-test-implementation`,
  `go-systematic-debugging`, `go-verification-before-completion`, closeout):
  link existing execution/test/proof authorities; retain only their unique
  method and stop condition. Add no shared execution contract.
- The maintainability trio keeps mutually exclusive triggers: Go/stdlib
  contract, local behavior-preserving readability, and explicit harsh
  whole-diff structural overbuild.
- Test skills keep distinct outcomes: proof design, executable test code,
  test-quality review, and final claim verification.

For a specialist with a multi-row reference selector, move that table to
`references/index.md`. `SKILL.md` loads the index only when domain detail could
change a decision or finding. A skill with zero or one obvious reference does
not gain an index merely for uniformity.

Descriptions are front-loaded and concise. Hard Go skills keep the checker-
owned `Use when` / `Own` / `Skip when` clauses. Other skills use the same
positive trigger and decisive exclusion semantics without forced boilerplate.

The authoring budgets are heuristics checked by review and size reporting, not
hard line-count failures: 50-150 words for session/router adapters, 100-250 for
ordinary specialists, and 250-500 for genuinely non-obvious methods. An
exception stays only when its exact failure mode is named in the skill.

## Specialist routing

Add `.agents/skills/go-specialist-router/` with:

- `SKILL.md`: reconstruct affected surfaces, select one primary specialist per
  independent axis, arbitrate overlap through the shared contract, read each
  selected canonical specialist `SKILL.md` as a local instruction reference,
  and apply all selected methods locally. It never delegates or performs
  nested skill invocation.
- `references/index.md`: compact declarative axis-to-specialist map for all 29
  current shared-contract specialists. It links
  `../../specialist-arbitration.md` only for ambiguous overlap and does not repeat
  those distinctions.
- `evals/evals.json`: the four checker-owned categories. The cases include a
  coupled API/data case requiring both specialists, a clean/no-specialist case,
  a single-axis case that rejects neighbors, and an unresolved-policy case
  that routes to the decision owner instead of review invention.

Stage 1, in the current completion, adds the router but leaves all 29
specialists implicitly eligible. Stage 2 adds
`agents/openai.yaml` with `policy.allow_implicit_invocation: false` to those 29
skills only after the live activation gate passes. No inactive or misleading
metadata files are added in Stage 1.

## Checker and eval ownership

`scripts/ci/hard-skills-check` gains one generic discovery pass over every
triggerable `.agents/skills/*/SKILL.md`. That pass validates universal
frontmatter/name, quoted one-line non-empty description, and local-link
integrity. Description concision and trigger/exclusion quality remain review-
and size-report-owned for non-hard skills. The
explicit hard-skill inventory retains its narrower `Use when` / `Own` / `Skip
when` clauses, shared-contract linkage, eval-category coverage, and size
reporting. The `size-report` command expands to every discovered triggerable
skill and emits per-skill, derived family, and whole-catalog aggregates without
a pass/fail reduction threshold. Families are derived as shared-contract
specialist, direct execution/method, exact specialist router, or remaining
workflow/session skill. Preserve current hard-skill rename mappings for
baseline comparison; unchanged names map directly and the new router has a
zero baseline. Add `go-specialist-router` as a canonical hard skill and selected
live-eval skill. It also owns an all-or-none Stage 2 policy check: zero
explicit-only policies is valid Stage 1; once any target policy is false,
exactly every specialist that directly loads `specialist-contract.md` must set
`agents/openai.yaml` `policy.allow_implicit_invocation: false`, no other skill
may carry that false policy, and `go-specialist-router` must remain implicitly
eligible. Derive the expected set from current canonical skill links, excluding
the exact router name before separately asserting its eligibility, rather than
duplicate a 29-name list. Do not add another checker or a hard word-count gate.

The hard-skill checker validates eval bundles for execution/method skills as
well as specialists; only the shared-contract link remains specialist-only.
The live-selected set retains the current three skills and adds the analysis's
five pilots—`go-coder`, `go-verification-before-completion`,
`go-security-review`, `go-implementation-ownership-review`, and
`go-systematic-debugging`—plus `go-specialist-router`. This produces nine
selected skills and 36 core cases.

Normalize the five newly selected pilot bundles to the existing selected-core
contract: exactly one self-contained, file-free core case in each category,
one-line expected output, non-empty `assertions`, and no core `expectations`.
For `go-security-review`, keep one representative `domain_defect` core and give
the other two domain cases non-empty `trial_class` values. For
`go-implementation-ownership-review`, add one compact file-free core
`domain_defect` case and retain the three fixture-backed domain trials as
auxiliary cases. Convert the coder, verification, security-review, and
debugging core expectation lists into assertions without deleting their
oracles.

The representative security core is existing ID `1` (identity/JWT plus
tenant/object authorization); IDs `0` and `2` become `safety_authority`
auxiliary trials because they protect browser-session and credential-reset
trust boundaries. The representative systematic-debugging core is existing
ID `0` (flake characterization and causal isolation); IDs `1` and `2` become
`safety_authority` auxiliary trials because they protect volatile incident
evidence and prevent an unowned timeout/retry-policy change. Preserve every
existing auxiliary oracle. The checker accepts only `standard` or
`safety_authority` when `trial_class` is non-empty and rejects every other
value; this keeps the targeted live harness's trial count and acceptance bar
materializable from the eval bundle.

`scripts/dev/hard-skills-evals.sh` emits that 36-case manifest, reports the
dynamic count, and requires
`WORKFLOW_EVAL_COST_AUTHORIZED=true` before external runner/judge execution.
Snapshot sealing derives the selected skill set from the emitted canonical
manifest and removes every selected skill's `evals/` directory from both
runner-visible snapshots; it has no hard-coded three-skill list. A harness test
asserts all nine selected eval bundles are absent before either runner starts.
The check-only path remains free of model calls. Update the checker tests and
command documentation for the new selected skill and cost gate.

Stage 2 uses a separate implicit-discovery runner ABI. Before runner completion
the harness passes only variant, sealed repository snapshot, an opaque neutral
path to the user prompt, and an opaque metadata output path—never case ID,
category, expected output, skill name, or an explicit skill target. The prompt
contains no `$skill` mention. The runner writes one closed JSON object:

```json
{"schema_version":1,"prompt_sha256":"<64 hex>","repository_commit":"<40 hex>","routing_mode":"implicit","explicit_skill_mentions":[],"forced_skills":[],"selected_skills":[{"name":"go-specialist-router","source":"implicit"}],"provenance_source":"runtime_events"}
```

The selected list may contain other ordinary implicit skills, but candidate
evidence is valid only when it includes implicit `go-specialist-router`, has no
explicit mention or forced injection, and contains none of the 29 explicit-only
specialists. The harness validates the digest and snapshot fields and accepts
`provenance_source: runtime_events` only when the authorized adapter maps native
runtime events rather than its own routing assertion. Runner output and
metadata are sealed before the judge receives case ID and expected output.

If an authorized runtime/adapter cannot expose those discriminating signals,
the activation harness fails closed, proof design reopens, and Stage 1 remains
the shipped state.

The Stage 2 harness partitions the 36 cases by owner. The 24 cases owned by the
router or by selected specialists in the derived explicit-only set use the
implicit-discovery ABI and router provenance rules. The 12 `go-coder`,
`go-verification-before-completion`, and `go-systematic-debugging` cases remain
ordinary direct-method non-regression cases and do not require router
selection. Stage 1 runs all 36 as lean-rewrite comparisons.

The future Stage 2 activation run must use the immutable accepted Stage 1
commit as `WORKFLOW_EVAL_BASE_REF`. That baseline must already contain the
router, lean skills, and eval definitions and contain no false implicit policy;
the candidate delta under test is limited to the complete set of specialist
`agents/openai.yaml` policy files. The harness rejects a pre-Stage-1 or already
activated baseline and unrelated candidate changes in activation mode. Stage 2
must pass all router cases; rollback is removal of the policy files. Without
runner, judge, and cost authorization, Stage 2 is blocked, and Stage 1 checks
do not substitute for it.

For structural integrity, the checker validates local Markdown links in every
discovered skill bundle Markdown file, plus `specialist-contract.md` and
`specialist-arbitration.md`. Checker mutation tests break one selector-index
link and the router-index arbitration link and require both failures. This
reuses the existing link resolver and adds no second scanner.

## Ownership and file changes

- `docs/skill-authoring.md`: durable authoring source of truth.
- `AGENTS.md`: one ownership pointer only.
- `.agents/skills/specialist-contract.md` and exact
  `.agents/skills/specialist-arbitration.md`: common specialist mechanics and
  sole detailed overlap arbitration.
- `.agents/skills/go-specialist-router/**`: routing method, map, and evals.
- `.agents/skills/*/SKILL.md`: unique skill adapters.
- `.agents/skills/*/references/index.md`: extracted selectors only where a
  non-trivial selector currently exists.
- `scripts/ci/hard-skills-check/{checker.go,inventory.go,checker_test.go}`:
  all-skill discovery, bundle-wide Markdown-link validation, router inventory,
  metadata-set integrity, and structural regression proof.
- `scripts/dev/hard-skills-evals.sh`: selected router cases, count, and cost
  authorization gate.
- `docs/build-test-and-development-commands.md`: changed eval invocation and
  activation-proof boundary.

No new package, dependency, generator, generic contract, or workflow phase is
introduced. Existing references remain unless inspection proves they only
duplicate general professional knowledge or a higher canonical owner.

## Proof and cleanup

Fresh local proof for Stage 1:

1. `go test ./scripts/ci/hard-skills-check`
2. `go run ./scripts/ci/hard-skills-check`
3. `bash scripts/dev/hard-skills-evals.sh check`
4. `make workflow-routing-check`
5. `make workflow-behavior-evals-check`
6. `make guardrails-check`
7. `git diff --check`
8. `go run ./scripts/ci/hard-skills-check size-report --baseline-ref 34d9776`

Broaden only when a changed checker or workflow surface triggers another
repository-owned gate. Do not invent Make aliases or claim uncommitted changes
were covered by a ref-only drift command.

Cleanup removes selector tables and common mechanics from rewritten
`SKILL.md` bodies, stale links to moved content, temporary size reports, and the
task bundle only after the accepted completion—including any separately
authorized Stage 2—is actually closed. Git keeps history.

## Reopen conditions

- Reopen Specification if routing changes observable skill scope or removes an
  accepted explicit entry point.
- Reopen this design if checker inventory cannot represent the router, local
  reference loading is invalid in a supported runtime, or a family cannot be
  shortened without losing a decision-changing invariant.
- Reopen proof design if the selected eval harness cannot test the actual
  candidate metadata or distinguish multi-axis selection from generic advice.
