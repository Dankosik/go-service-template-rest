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

## Feasibility

Separate local state/correctness proof from scale, platform-performance, and
production qualification. Attach each environmental requirement to the claim
that needs it; later qualification inputs do not hold an independently provable
local result. Local proof never substitutes for that qualification.

When a novel control or platform determines whether the plan can execute,
resolve that uncertainty before declaring the affected plan ready. Use existing
evidence or the smallest authorized disposable Research probe: demonstrate one
normal control path and one controlled fault at the hardest boundary before
expanding the matrix. This probe establishes control feasibility, not product
acceptance; Implementation still owns deliverable test and fixture code.
Check available resources and basic budget fit first. Compilation or a written
command alone cannot prove a new barrier works. Reuse a sufficient witness;
ordinary tests with known controls need no extra probe. If the witness needs
unavailable authority or a production change, carry that exact gap instead.

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
