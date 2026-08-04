# Test Design

Own the falsification strategy for risky accepted behavior before implementation. Produce proof obligations and a Planning handoff; test and fixture code belong to Implementation, while a missing behavior, failure-policy, ownership, or rollout decision reopens its accepted owner. Keep obvious proof as an inline handoff for Planning; create `test-plan.md` only when a scenario matrix adds value. Test Design does not edit `tasks.md`.

## Read When

- Proof spans several scenarios, failure modes, or levels.
- Public contracts, migrations/data, security, money, performance budgets or scale-sensitive hot paths, concurrency/lifecycle, retries, replay, async work, compatibility, or rollout behavior changes.
- A regression needs non-obvious fail-before proof.
- Planning would otherwise choose scenario classes or proof levels.

## Inputs

- Ready spec and design, including carried risks/proof obligations.
- Existing nearby tests, fixtures, contract/drift checks, and repository validation commands.
- Existing test plan for repair work.

## Method

Run this universal kernel before applying any conditional branch:

1. Reconstruct the material proof-obligation surface independently from the
   ready spec/design, carried risks, and affected current boundaries:
   - map every material acceptance claim, invariant, deliberately preserved
     behavior, and carried risk to its observable results, state transitions,
     failure/recovery behavior, regressions, and protected side effects;
   - scan each affected boundary for triggered negative/failure,
     retry/replay/async, compatibility, concurrency/lifecycle, security/money,
     data/migration, performance, and rollout obligations; omission from an
     upstream artifact is not evidence that a lens is untriggered;
   - close a triggered lens only with an obligation or an inspected
     current-boundary anchor showing that it contributes none, and record only
     decision-relevant exclusions.

   Test names, package boundaries, and implementation structure may locate
   proof but do not define obligations.
2. Give each proof obligation exactly one current disposition: sufficient existing proof; existing proof to strengthen; one or more `TD-*` scenarios; a named non-test falsifier; or explicitly authorized residual-risk acceptance with evidence, owner, and reopen condition. Omission is not disposition.
3. For every non-residual disposition, satisfy the **falsifier contract**:
   - name the plausible incorrect observable behavior or regression;
   - control the setup and action or fault trigger;
   - derive the expected result from accepted authority independent of the
     production path under test;
   - choose the narrowest complementary proof boundary plus a full runnable
     command, or an exact non-command procedure when automation cannot establish
     the oracle.

   The oracle must change verdict when the named wrong behavior occurs. Treat an
   internal interaction as the observable only when it is itself the accepted
   boundary.

### Conditional Branches

Structural and execution signals establish only the fact they directly observe.
Source or instruction text, file presence, compilation or lint success,
coverage, and a green command or suite are not behavioral oracles merely
because they pass. For behavioral proof, exercise a controlled path and observe
the accepted result, state, emission, or side effect. Use exact text or
file-shape checks only for an accepted artifact or structural contract, such as
generated configuration, protocol output, a diagnostic contract, or required
repository layout.

Route performance proof decisions through `go-performance` and
[Benchmarking](../../benchmarking.md); keep the accepted workload and scale
boundary, measured path or structural oracle, budget or structural constraint,
baseline/candidate equivalence when comparison applies, and independent
correctness proof explicit.

Route non-obvious Go proving decisions through `go-test-strategy`; this phase
still owns the complete obligation matrix and Planning handoff. Map only
triggered failure classes: language or package behavior to a focused package
test; parser or input space to table or fuzz proof; shared state or lifecycle to
deterministic coordination plus triggered race, liveness, or leak proof; HTTP,
OpenAPI, or generated behavior to boundary or drift proof; SQL, cache,
migration, or durable state to stateful integration or rehearsal; and
repository completion only to triggered repository-native gates.

For triggered security or money obligations, design proof with a controlled
actor, authority or tenant, credential or value boundary and an oracle for both
the accepted result and the absence of unauthorized state, emission, or effect.
For retry, replay, async, or compatibility obligations, design proof that
controls identity, order, duplication, or version at the accepted boundary and
observes authoritative final state and external effects. For rollout
obligations, design a procedure that exercises the accepted deployment
transition and observes the named client-visible behavior and runtime signal at
the pre-cutover, post-cutover, rollback, or recovery checkpoint that carries the
claim. Scanners, acknowledgements, migration or deployment status, and dashboard
availability establish only the fact they directly observe.

## Outputs

When a durable matrix is unnecessary, return a compact inline proof handoff for Planning. Otherwise create `test-plan.md` and give each proof obligation one compact row:

```text
TD-ID | source claim/risk/preserved behavior/boundary invariant | disposition | plausible wrong observable behavior/regression | controlled setup; action or fault trigger; discriminating oracle, or authorized residual-risk acceptance | narrowest complementary proof boundary/type; full repository-native command that will execute the proof after implementation, or exact non-command procedure and why automation cannot establish the oracle | required fixture/input, canonical source, and status | proof owner; Planning placement constraint; reopen owner
```

For every executable disposition, record its completed falsifier contract,
including the full command, exercised path, and oracle that make the result
dispositive. Record every required fixture or proof input as existing from a
named source, behavior-significant and derived from approved authority,
decision-neutral and specified under the rule below, or unavailable from a
named owner; for an unavailable input, record the affected obligation and
reopen condition. Add current-proof gaps, scenario classes, fail-before
discriminators, non-test proof details, and residual-risk details only when
triggered.

Choose the smallest set of complementary proof boundaries that jointly proves the claim: unit, integration, contract, component/process, e2e smoke, or a repository-specific proof type. Each level owns a distinct observable; broader proof does not merely duplicate narrower proof. Include happy path, material failure/edge/negative paths, and protected-domain branches only when triggered by the accepted change.

Each scenario must distinguish a material behavior or failure mechanism and name a plausible incorrect observable behavior or regression that its oracle would reject. Reuse or strengthen existing proof before proposing a new test, and call it sufficient only after inspecting its setup, exercised path, oracle/assertions, isolation, and runnable command. Merge rows with the same risk, trigger, oracle, and reopen path.

Missing test code or fixture files establish proof gaps, not behavioral fail-before evidence. Test Design may specify the smallest synthetic fixture shape and decision-neutral values needed to make the setup and oracle discriminating; Implementation owns creating the test and fixture code. Derive behavior-significant fields, ordering, encoding, trust material, security policy, migration state, and rollout state from approved authority. If any such choice would decide accepted behavior or policy, reopen its owner instead of choosing it here.

When fail-before evidence adds no useful discrimination or is honestly unavailable, record why and name the nearest existing falsifying signal. This never waives proof required by the accepted current implementation completion.

An honestly unavailable target, budget, fixture, command, environment, or other proof input may remain only for a task and claim already outside the accepted current implementation completion. If a mandatory completion path needs it, test design returns `FAIL` and reopens the accepted-outcome owner; only that owner may narrow or split the outcome before the excluded task and claim are routed to a later ledger.

## Review

Apply focused root self-review when test design is triggered. Run independent QA review only when the shared review trigger applies. The reviewer falsifies the Outputs above without writing tests or changing behavior.

The reviewer returns one verdict under the shared convergence contract:

- `PASS`: every proof obligation has one valid disposition from the universal kernel; every non-residual disposition satisfies the Method and Outputs and is feasible from its named inputs; every residual-risk acceptance has authorization evidence, an owner, and a reopen condition; and no affected lens is uncovered;
- `CONCERNS`: a bounded downstream proof obligation may move after its owner, observable, and reopen condition are recorded and no test-strategy decision remains open;
- `FAIL`: a missing scenario, observable, proof level, feasible proof path, owner, or upstream decision prevents honest planning.

When test design owns that review, findings return to the owning root for disposition; fresh review follows only `FAIL` repair or material candidate change. An explicitly user-requested standalone QA review returns the complete review result and stops read-only.

## Stop Rule

Continue to Planning only after a bidirectional closure check: every material acceptance claim, deliberately preserved behavior, carried risk, affected-boundary invariant, and triggered proof lens maps to one final disposition, and every disposition maps back to one of those sources. A lens exclusion requires an inspected current-boundary anchor; upstream omission does not close it.

Every non-residual disposition must satisfy the falsifier contract and contain
every applicable proof-carrier, command or bounded procedure, fixture/input,
owner, Planning-placement, and reopen field required by Outputs. Existing proof
counts only after the required path-and-oracle inspection. Residual-risk
acceptance records authorization evidence, owner, and reopen condition.
Planning may choose order and placement only; it must not invent a plausible
wrong behavior, scenario, setup or trigger, oracle, proof boundary or level,
command or procedure, fixture or input, or proof or reopen owner. A mandatory
proof with an unavailable input is `FAIL`; any triggered review has returned
`PASS` or its `CONCERNS` are dispositioned.

Reopen Specification or Design when closure would require deciding behavior, failure policy, ownership, migration or rollout policy, or another accepted upstream rule.
