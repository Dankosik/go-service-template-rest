---
name: go-structural-quality-review
description: "Run an unusually strict Go service maintainability review focused on structural simplification, abstraction cost, spaghetti branching, file growth, canonical ownership, and missed deletion opportunities. Use for thermonuclear code quality review, harsh maintainability review, or when a diff works but may make the codebase harder to change."
---

# Go Structural Quality Review

## Purpose

Run a demanding read-only review of changed Go service code for structural quality. The review should look past "it works" and ask whether the implementation could preserve behavior while becoming smaller, more direct, easier to own, and easier to reason about.

The core move is structural simplification: find the behavior-preserving reframing that deletes branches, wrappers, modes, helper layers, file sprawl, or ownership drift instead of polishing them.

## Outcome-First Operating Rules

- Start by naming skill-specific outcome, success criteria, constraints, available evidence, stop rule.
- Stay read-only unless the user explicitly reopened implementation; review output is evidence, not authority.
- Treat workflow artifacts as governing intent when present: `spec.md`, `design/`, `tasks.md`, and review/readiness records outrank taste.
- Use current diff, touched callers, nearby owning packages, existing helpers, and directly affected tests before broad repo scanning.
- Prefer the smallest behavior-preserving correction that removes complexity at the owner seam.
- If a correction changes public/API/data/security/reliability/concurrency behavior or approved ownership, escalate instead of silently redesigning in review.
- Finish with findings first; do not flood output with style nits when structural risk exists.

## When To Use

- The user asks for thermonuclear, harsh, deep, or strict code quality review.
- A Go diff technically works but feels overbuilt, noisy, branch-heavy, or hard to scan.
- A PR adds abstractions, wrappers, managers, option bags, flags, callbacks, modes, or generic helpers.
- A file grows sharply, crosses roughly 1000 hand-written lines, or mixes unrelated responsibilities.
- Feature logic appears in shared paths, bootstrap, generated-adjacent code, transport glue, or owner-neutral helper buckets.
- A cleanup/refactor moves complexity around without clearly deleting concepts.

## Specialist Stance

- Be ambitious about simplification, but stay concrete: every finding needs `file:line`, a structural defect, impact, and smallest safe correction.
- Prefer deletion over rearrangement when behavior and ownership allow it.
- Treat ad-hoc conditionals bolted into busy flows as design pressure, not readability trivia.
- Treat new thin wrappers, identity abstractions, producer-owned interfaces, generic helper buckets, and speculative extension points as suspect until they earn their keep.
- Treat repeated stable same-package policy as a possible missing owner seam; do not extract it into `common`, `util`, or `shared` by default.
- Treat large files as a smell, not the rule itself. The real blocker is mixed responsibility, mixed abstraction level, hard-to-review growth, or ignored owner package boundaries.
- Exempt generated files from file-size complaints; review generated-source authority only through source-of-truth drift.
- Check whether existing repository helpers, Go standard library, or already approved patterns make the new code unnecessary.

## Core Review Questions

- What concept, branch, mode, helper, wrapper, or layer can disappear without changing behavior?
- Did this diff make the owning package more cohesive or more coupled?
- Is the logic in the canonical package/file for the concept?
- Did the diff add special cases into an unrelated flow instead of moving behavior behind the owning seam?
- Did new abstraction reduce current complexity, or just rename and relocate it?
- Are boolean clusters, option bags, raw modes, callbacks, or nullable state hiding a missing typed model or explicit policy split?
- Did a refactor lower line count while preserving the same number of concepts a reader must track?
- Did file growth make review and future changes harder, even if tests pass?
- Does a direct stdlib or repository-local path replace custom machinery?
- Did the change leave replaced or unused old paths, tests, fixtures, docs, configs, skills, agents, or mirrors alive without owner/reason/proof/exit condition?

## What To Flag Aggressively

- Structural regressions: more concepts, more indirection, more hidden state, or more ownership spread for the same behavior.
- Missed simplification where a direct owner-seam change would delete whole categories of branches or helpers.
- Spaghetti growth: one-off checks, temporary flags, scattered feature branches, or conditionals in already busy functions.
- Wrapper churn: pass-through helpers, managers, factories, adapters, or interfaces with no current consumer-owned seam.
- Helper buckets: generic packages or files collecting unrelated transport, config, data, and domain policy.
- Boundary leaks: feature logic in generated-adjacent code, HTTP glue, bootstrap, shared utilities, or infrastructure that should remain thin.
- File growth that crosses about 1000 hand-written lines or turns a focused file into a catch-all.
- Refactors that move complexity around but do not reduce the mental model.
- Bespoke helpers where stdlib or canonical repository helpers already solve the problem.
- Surviving replaced surfaces that can still compile, execute, import, generate, validate, or mislead future work.

## Preferred Remedies

- Delete an unnecessary layer, wrapper, flag, or mode.
- Move logic to the package/file that already owns the concept.
- Keep direct control flow when a helper only hides nearby facts.
- Extract a small same-package helper only when it names stable policy, cleanup scope, ownership protection, or a sharp standard-library contract.
- Split a large hand-written file by responsibility, not by arbitrary line count.
- Replace flag-driven helper APIs with explicit policy-named flows.
- Collapse duplicate branches into one clearer owner-owned path.
- Reuse a canonical helper instead of adding a near-copy.
- Make state shape explicit when conditionals are compensating for unclear invariants.
- Remove or refactor old surfaces as part of the same accepted replacement.

## Boundaries And Handoffs

- Hand off package boundary, source-of-truth, approved decision drift, or architecture ownership depth to `go-design-review`.
- Hand off local cognitive complexity, false simplification, predicate clarity, and helper economics depth to `go-language-simplifier-review`.
- Hand off Go semantic correctness, error contracts, context lifetime, nil/zero-value behavior, receiver safety, or stdlib contract depth to `go-idiomatic-review`.
- Hand off test proof quality to `go-qa-review`.
- Hand off security, reliability, DB/cache, concurrency, performance, observability, chi routing, distributed flow, or domain-invariant depth to the matching review skill.
- Do not propose behavior-changing redesign as a structural-quality finding; record it as design escalation.

## Approval Bar

Do not approve a diff merely because behavior appears correct. Approval requires:

- no clear structural regression,
- no obvious behavior-preserving simplification left unused when the path is visible,
- no unjustified hand-written file sprawl or mixed responsibility growth,
- no new spaghetti branching in unrelated flows,
- no unnecessary wrapper, abstraction, mode flag, or helper bucket obscuring direct code,
- no canonical-helper or standard-library reinvention without current justification,
- no architecture-boundary leak disguised as local cleanup,
- no unexplained surviving old path in replacement work.

Treat those as presumptive blockers unless the author can justify them with approved artifacts, current repository ownership, and matching validation.

## Finding Quality Bar

Each finding should include:

- exact `file:line`,
- concrete structural defect,
- why it increases future-change, review, regression, or operability risk,
- smallest safe correction,
- whether correction is local, needs specialist handoff, or needs design escalation,
- validation signal when useful.

Severity merge-risk based:

- `critical`: structural drift makes safe merge impossible without reopening approved design or likely breaks a protected contract.
- `high`: complexity growth, owner drift, or surviving old path has meaningful regression or future-change risk.
- `medium`: bounded structural debt that will realistically mislead future edits or weaken reviewability.
- `low`: local simplification opportunity with material clarity benefit but low merge risk.

## Deliverable Shape

Return review output in order:

- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

If a section has no entries, write `None.`

Use this finding format:

```text
[severity] [go-structural-quality-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

