# Test Design

Use when material proof obligations, oracles, scenario classes, or proof levels
are non-obvious. Own falsification strategy; Implementation owns test and fixture
code.

## Inputs

Consume ready behavior/design, carried risks and proof obligations, existing
nearby tests/fixtures/contracts/drift checks, repository validation commands,
and current proof inputs.

## Method

1. Reconstruct every material acceptance claim, invariant, preserved behavior,
   boundary, and risk independently from current authority. A triggered
   negative/failure, retry/replay/async, compatibility, concurrency/lifecycle,
   security/money, data/migration, performance, or rollout lens needs an
   obligation or an inspected exclusion anchor.
2. Give each obligation one disposition: sufficient existing proof, existing
   proof to strengthen, planned scenario, non-test falsifier, or explicitly
   authorized residual risk with owner and reopen condition.
3. Apply [Falsifier](../rubrics/falsifier.md) to every non-residual disposition.
   Inspect existing proof's setup, exercised path, oracle, isolation, and
   runnable command before calling it sufficient.
4. Choose the smallest complementary proof boundaries that jointly prove the
   claims. Wider proof must add a distinct observable.
5. Record unavailable mandatory inputs as blockers; a behavior-significant
   fixture value or oracle that would choose policy reopens its upstream owner.

## Conditional Methods

- non-obvious Go proof selection -> `go-test-strategy`;
- workload/budget proof -> `go-performance` and [Benchmarking](../../benchmarking.md);
- a production seam is required solely for proof -> Go Code / Ownership Design;
- rollout proof -> accepted [Release Closure](../rubrics/release-closure.md).

## Output

Return [Test Plan V1](../interfaces/test-plan-v1.md) inline, or persist
`test-plan.md` through [Artifacts](../shared/artifacts.md) when a matrix must
survive. Each executable row includes the full repository-native command or an
exact non-command procedure and why automation cannot establish the oracle.

## Review

When shared [Review](../shared/review.md) triggers, the reviewer checks that
every material obligation has one feasible disposition and discriminating
oracle. A missing obligation, observable, proof level, owner, or mandatory input
is `FAIL`; only bounded downstream risk over accepted behavior may be
`CONCERNS`.

## Exit And Reopen

Exit when claims and proof rows reconcile in both directions and every
non-residual row satisfies Falsifier. Planning may choose only order and
placement. Reopen Specification or Design when proof would decide behavior,
failure/rollout policy, mechanism, or ownership.
