# Test Design

Turn risky behavior into explicit proof obligations before implementation. Keep obvious tests in `tasks.md`; create `test-plan.md` only when a scenario matrix adds value.

## Read When

- Proof spans several scenarios, failure modes, or levels.
- Public contracts, migrations/data, security, money, concurrency/lifecycle, retries, async work, compatibility, or rollout behavior changes.
- A regression needs non-obvious fail-before proof.
- Planning would otherwise choose scenario classes or proof levels.

## Inputs

- Ready spec and design, including carried risks/proof obligations.
- Existing nearby tests, fixtures, contract/drift checks, and repository validation commands.
- Existing test plan for repair work.

## Outputs

Either compact proof rows in `tasks.md` or `test-plan.md` with:

```text
ID | source decision/risk | level | setup/action | expected observable | fail-before signal or reason unavailable | proof command | residual gap/reopen owner
```

Choose the smallest convincing level: unit, integration, contract, or e2e smoke. Include happy path, material failure/edge/negative paths, and protected-domain branches that could regress. Broad scenario labels are not enough when they hide several state transitions or side effects.

## Review

When structured or orchestrated work triggers test design, run an independent QA review before planning. Direct work uses one only when the user or risk requires it. The reviewer checks traceability, observables, determinism, fail-before expectations, command feasibility, and residual gaps; it does not write tests or change behavior.

When test design owns that review, findings return to the owning root for same-session repair and fresh re-review. An explicitly user-requested standalone QA review returns findings only and stops read-only.

## Stop Rule

Continue to planning when every material risk has an owner, proof level, observable, and executable or honestly unavailable check. Reopen specification/design when a scenario cannot be written without deciding behavior, failure policy, ownership, or rollout.

Do not create a test plan whose only content is headings or generic “add coverage” tasks.
