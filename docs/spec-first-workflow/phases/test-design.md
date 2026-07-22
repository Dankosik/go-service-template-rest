# Test Design

Turn risky behavior into explicit proof obligations before implementation. Keep obvious proof as an inline handoff for Planning; create `test-plan.md` only when a scenario matrix adds value. Test Design does not edit `tasks.md`.

## Read When

- Proof spans several scenarios, failure modes, or levels.
- Public contracts, migrations/data, security, money, concurrency/lifecycle, retries, async work, compatibility, or rollout behavior changes.
- A regression needs non-obvious fail-before proof.
- Planning would otherwise choose scenario classes or proof levels.

## Inputs

- Ready spec and design, including carried risks/proof obligations.
- Existing nearby tests, fixtures, contract/drift checks, and repository validation commands.
- Existing test plan for repair work.

## Method

Before writing scenarios, disposition every material acceptance claim, invariant, state transition, failure mode, and protected side effect affected by the change as existing sufficient proof, existing proof to strengthen, one or more `TD-*` scenarios, a named non-test proof that can falsify the claim, or an explicitly authorized residual-risk acceptance with evidence, owner, and reopen condition. Omission is not disposition. Derive this proof surface from approved behavior and affected contract, runtime, state, trust, and lifecycle boundaries, not from existing test names or implementation branches.

For each resulting proof obligation, design the smallest falsifier: a controlled setup/action/failure trigger plus an oracle that rejects a plausible incorrect observable result, state, emission, or side effect. Then select the narrowest complementary proving level and runnable command that can establish that oracle.

Route non-obvious Go proving decisions through `go-test-strategy`; this phase
still owns the complete obligation matrix and Planning handoff. Map only
triggered failure classes: language or package behavior to a focused package
test; parser or input space to table or fuzz proof; shared state or lifecycle to
deterministic coordination plus triggered race, liveness, or leak proof; HTTP,
OpenAPI, or generated behavior to boundary or drift proof; SQL, cache,
migration, or durable state to stateful integration or rehearsal; and
repository completion only to triggered repository-native gates.

## Outputs

When a durable matrix is unnecessary, return a compact inline proof handoff for Planning. Otherwise create `test-plan.md` with the canonical row shape:

```text
TD-ID | source claim/risk | scenario class | current proof/gap | proof boundary/type | controlled setup/action/failure trigger | oracle: result plus relevant state/emissions/forbidden side effects | fail-before/regression discriminator or phase-allowed reason unavailable | proof command/family | residual gap/reopen owner
```

Choose the smallest set of complementary proof boundaries that jointly proves the claim: unit, integration, contract, component/process, e2e smoke, or a repository-specific proof type. Each level owns a distinct observable; broader proof does not merely duplicate narrower proof. Include happy path, material failure/edge/negative paths, and protected-domain branches only when triggered by the accepted change.

Each scenario must distinguish a material behavior or failure mechanism and name a plausible incorrect observable behavior or regression that its oracle would reject. Reuse or strengthen existing proof before proposing a new test, but call it sufficient only after inspecting its setup, exercised path, oracle/assertions, isolation, and runnable command; its name, prior green status, or coverage hit is not proof. Merge rows with the same risk, trigger, oracle, and reopen path. Broad scenario labels and coverage alone are not proof. A proof command is adequate only when it executes the relevant path and its result can establish the named oracle; successful execution alone is insufficient.

Missing test code or fixtures establish a proof gap, not behavioral fail-before evidence. Define their required contents only when they are mechanically derivable from approved sources. If setup requires choosing fields, values, ordering, encoding, trust material, or security policy, reopen design instead.

When fail-before evidence adds no useful discrimination or is honestly unavailable, record why and name the nearest existing falsifying signal. This never waives proof required by the accepted current implementation completion.

An honestly unavailable target, budget, fixture, command, environment, or other proof input may remain only for a task and claim already outside the accepted current implementation completion. If a mandatory completion path needs it, test design returns `FAIL` and reopens the accepted-outcome owner; only that owner may narrow or split the outcome before the excluded task and claim are routed to a later ledger.

## Review

Apply focused root self-review when test design is triggered. Run independent QA review only when the shared review trigger applies. The reviewer falsifies the Outputs above without writing tests or changing behavior.

The reviewer returns one verdict under the shared convergence contract:

- `PASS`: every material risk has a credible, owned, executable proof path or an authorized recorded acceptance with evidence, owner, and reopen condition, and no affected lens is uncovered;
- `CONCERNS`: a bounded downstream proof obligation may move after its owner, observable, and reopen condition are recorded and no test-strategy decision remains open;
- `FAIL`: a missing scenario, observable, proof level, feasible command path, owner, or upstream decision prevents honest planning.

When test design owns that review, findings return to the owning root for disposition; fresh review follows only `FAIL` repair or material candidate change. An explicitly user-requested standalone QA review returns the complete review result and stops read-only.

## Stop Rule

Continue to planning when every material acceptance claim, invariant, transition, failure mode, and protected side effect is dispositioned as sufficient existing proof, an owned executable proof path, a named non-test falsifier, or an explicitly authorized residual-risk acceptance with evidence, owner, and reopen condition, and any triggered review has returned `PASS` or dispositioned `CONCERNS`. Reopen specification/design when a scenario cannot be written without deciding behavior, failure policy, ownership, or rollout.

Do not create a test plan whose only content is headings or generic “add coverage” tasks.
