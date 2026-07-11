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

Missing test code or fixtures are a valid fail-before signal only when their expected contents are mechanically derivable from approved sources. If setup requires choosing fields, values, ordering, encoding, trust material, or security policy, reopen design instead.

An honestly unavailable target, budget, fixture, command, environment, or other proof input may remain only for a task and claim already outside the accepted current implementation completion. If a mandatory completion path needs it, test design returns `FAIL` and reopens the accepted-outcome owner; only that owner may narrow or split the outcome before the excluded task and claim are routed to a later ledger.

## Review

When structured or orchestrated work triggers test design, run an independent QA review before planning. Direct work uses one only when the user or risk requires it. The reviewer checks traceability, setup derivability, observables, determinism, fail-before expectations, command feasibility, and residual gaps; it does not write tests or change behavior.

The reviewer returns one verdict under the shared convergence contract:

- `PASS`: every material risk has a credible, owned, executable proof path or an authorized recorded acceptance with evidence, owner, and reopen condition, and no affected lens is uncovered;
- `CONCERNS`: a bounded downstream proof obligation still needs explicit owner disposition and fresh review; it does not permit planning, including when the check is honestly unavailable outside current completion;
- `FAIL`: a missing scenario, observable, proof level, feasible command path, owner, or upstream decision prevents honest planning.

When test design owns that review, findings return to the owning root for same-session repair and fresh re-review until convergence. An explicitly user-requested standalone QA review returns findings only and stops read-only.

## Stop Rule

Continue to planning when every material risk has an owner plus an executable check, a dispositioned downstream obligation, or an authorized recorded residual-risk acceptance with evidence and reopen condition, and the required review has returned `PASS`. Reopen specification/design when a scenario cannot be written without deciding behavior, failure policy, ownership, or rollout.

Do not create a test plan whose only content is headings or generic “add coverage” tasks.
