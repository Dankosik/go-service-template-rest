# Test Design

Use when material proof obligations, oracles, scenario classes, or proof levels
are non-obvious. Own falsification strategy; Implementation owns test and fixture
code.

Apply `go-test-strategy` to ready behavior/design, carried risks, current proof
surfaces, and repository validation commands. Select another method only for an
independent pressure:

- workload/budget proof -> `go-performance` and [Benchmarking](../../benchmarking.md);
- a production seam is required solely for proof -> Go Code / Ownership Design;
- rollout proof -> accepted [Release Closure](../rubrics/release-closure.md).

## Output

Return [Test Plan V1](../interfaces/test-plan-v1.md) inline, or persist
`test-plan.md` through [Artifacts](../shared/artifacts.md) when a matrix must
survive. Each executable row includes the full repository-native command or an
exact non-command procedure and why automation cannot establish the oracle.

## Review

Apply shared [Review](../shared/review.md) before returning `ready`. The reviewer
checks every material obligation's disposition through Test Plan V1 and requires
a discriminating oracle for obligations needing proof. A missing obligation,
observable, proof level, owner, or mandatory design input is `FAIL`; only bounded
downstream risk over accepted behavior may be `CONCERNS`.

Done when claims and proof rows reconcile in both directions and every
non-residual row has a discriminating scenario, deterministic controls,
independent oracle, proving layer, and a feasible proof plan. Planning may choose
only order and placement. Reopen Specification or Design when proof would decide
behavior, failure/rollout policy, mechanism, or ownership.
