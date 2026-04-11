---
name: go-language-simplifier-review
description: "Review Go code changes for lower cognitive complexity, false-simplification risk, missed same-package source-of-truth extraction, junk-drawer helper risk, clearer naming, safer control-flow cleanup, and easier maintenance without collapsing semantics. Use whenever a Go diff claims cleanup/refactor/readability improvement or touches helper extraction, nested control flow, boolean flags, option bags, or error-path deduplication, even if another review lane also applies."
---

# Go Language Simplifier Review

## Purpose
Protect local reasoning quality in changed Go code without endorsing refactors that only reduce line count while hiding policy, state transitions, ownership, cleanup, error contracts, or caller-visible semantics.

## Specialist Stance
- Treat simplicity as reduced reasoning load, not lower line count.
- Flag false simplifications that merge distinct semantics, hide ownership, or push policy into generic helpers.
- Also flag missed same-package source-of-truth extraction when stable local policy is visibly starting to drift.
- Hand off deep Go-semantic, domain, concurrency, or design ownership when simplification review only identifies the risk.

## When To Use
- review Go PRs, diffs, refactors, and cleanup commits where the stated goal is simpler or more readable code
- use even on generic review requests when the change touches helper extraction, nested branching, delayed state interpretation, boolean mode flags, option bags, or error-path "deduplication"
- stay in the simplification lane; hand off deeper design or Go-semantic ownership instead of drifting into a broad review

## Review Posture
- Stay read-only and advisory.
- Review changed files and directly affected tests first.
- If approved task artifacts exist, treat them as governing intent.
- Findings come first and must be ordered by merge risk, not by section order or taste.
- Green tests do not prove a cleanup preserved local reasoning safety.
- Always run a source-of-truth helper pass: flag both stable same-package policy still scattered across files and bogus extraction into vague helper buckets.
- Prefer official Go docs, Go Code Review Comments, Effective Go, Go module/package organization docs, and repository-local review patterns over external clean-code advice. Treat generic clean-code material as calibration only.

## Scope
- review control flow, state shape, and predicate clarity
- review abstraction cost, helper economics, and call-site burden
- review false simplification in error paths, ownership seams, and thin policy wrappers
- review whether stable same-package policy is scattered across files when one seam-named helper file should own it
- review whether a new helper actually reduces reasoning or just hides policy in a `util/common/shared` bucket
- review naming and test readability when they materially affect safe future changes
- review whether touched validation is enough to protect subtle precedence or branch behavior

## Reference Files Selector
Keep this skill compact. Load only the reference file that matches the review risk in front of you:

| Reference | Load When |
| --- | --- |
| `references/false-simplification-patterns.md` | a cleanup, deduplication, or "readability" change may have collapsed semantics, ownership, cleanup, or behavior |
| `references/helper-extraction-economics.md` | helpers, wrappers, interfaces, option bags, callbacks, or vague helper buckets were added, removed, or renamed |
| `references/source-of-truth-extraction.md` | stable same-package policy is repeated, drifting, or moved away from its owner |
| `references/control-flow-and-temporal-coupling.md` | branching, guard clauses, sentinels, named returns, defer, cleanup, or phase ordering changed |
| `references/predicate-condition-and-mode-clarity.md` | compound predicates, negative conditions, boolean clusters, raw modes, or option decoding make a decision hard to read |
| `references/error-path-simplification.md` | error handling was deduplicated, wrapped, normalized, mapped, logged, or reordered |
| `references/test-readability-and-proof-shape.md` | tests were simplified with tables, helpers, assertions, or terse failure messages that may hide proof intent |
| `references/naming-and-intent-exposure.md` | names, receiver names, helper names, booleans, exported identifiers, or vocabulary drift obscure intent |

## Boundaries
Do not:
- turn simplification review into architecture redesign or primary correctness review
- call semantic protection "ceremony" when it preserves ownership, lifetime, cleanup, or error contracts
- propose behavior-changing refactors as simplification without explicit escalation
- block on taste-only comments with no merge-risk impact
- equate shorter code with simpler code

## Core Defaults
- Simpler means less reasoning required, not fewer lines.
- Duplication can be cheaper than hiding distinct policy or error semantics behind one generic helper.
- Repeated stable policy across several files in one package is also simplification debt; one seam-named same-package owner can be simpler than several near-copies.
- Keep one clear abstraction level per function when practical.
- Prefer local, behavior-preserving simplification over broad rewrites.
- If a wrapper protects ownership, cleanup, or contract shape, do not remove it just because it is short.
- Prefer the smallest change that makes intent obvious on first read.

## Expertise
- Risk calibration: prioritize cleanups that merge distinct behaviors, change precedence, hide side effects, or force readers to track hidden modes.
- Helper economics: prefer helpers that name stable policy, ownership, defaulting, cleanup scope, stdlib quirks, or error normalization; flag wrappers that only move complexity out of sight.
- Source-of-truth extraction: flag repeated stable same-package policy that is likely to drift, but avoid global helper buckets and mode-heavy extraction.
- Control flow: prefer guard clauses and straight-line happy paths only when side-effect ordering, cleanup, and which error wins stay explicit.
- Predicate clarity: flag compound negatives, boolean clusters, and hidden mode decoding when the decision no longer reads at the call site.
- API and call-site burden: flag same-typed positional parameters, raw strings with hidden meaning, `map[string]any` option blobs, and exported-surface changes that need design escalation.
- Error paths: preserve distinct failure classes, `errors.Is/As` inspectability, status mapping, context cancellation, and cleanup/audit precedence.
- Naming and tests: suggest naming or test simplification only when it lowers future reasoning and diagnosis load.

### Go-Semantic Stop-Signs
- Do not recommend simplification that removes code protecting ownership, lifetime, or public contract just because it looks ceremonial.
Stop-sign examples:
- slice or map alias isolation such as `slices.Clone`, `maps.Clone`, or copy-before-store
- `nil` versus empty behavior that is externally observable
- receiver or method-set changes, zero-value usability, or must-not-copy state
- cleanup and lifetime code such as `defer cancel()`, `rows.Close`, unlock or close ordering, and rollback sequencing
- standard-library wrapper contracts such as `http.Header`, `url.Values`, and similar types
- You may mention the simplification risk, but hand off deep Go-semantic analysis to `go-idiomatic-review`.

### Cross-Domain Handoffs
- Hand off deep Go-semantic and standard-library contract questions to `go-idiomatic-review`.
- Hand off design-shape and package-ownership questions to `go-design-review`.
- Hand off concurrency, reliability, security, DB/cache, and performance depth to the corresponding review skills.
- Hand off test-strategy completeness to `go-qa-review`.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the concrete simplification defect
- why it raises merge-risk, maintenance risk, or branch-misread risk
- the smallest safe correction
- a validation command when useful
- whether the change is behavior-preserving, a specialist handoff, or needs design escalation
- whether the issue is under-extraction of a same-package source-of-truth seam or over-extraction into a vague helper
- the reference file used when one materially shaped the finding

Severity is merge-risk based:
- `critical`: the cleanup obscures critical behavior or contract semantics enough that safe change is unlikely
- `high`: strong evidence of hidden state, false simplification, or API opacity with material maintenance risk
- `medium`: bounded but meaningful readability debt with a realistic future-change cost
- `low`: local simplification opportunity that still materially improves clarity

## Deliverable Shape
Return review output in this order:
- `Findings`
- `Handoffs`
- `Design Escalations`
- `Residual Risks`
- `Validation Commands`

If a section has no entries, write `None.` rather than filler.

Use this format for each finding:

```text
[severity] [go-language-simplifier-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

Start `Issue` with the plain-language defect. Add an `Axis:` label only when it materially clarifies why the issue belongs in simplification review rather than design or idiomatic Go review.

## Escalate When
Escalate when:
- a safe correction changes the public contract, transport behavior, or approved design (`go-design-spec`, `api-contract-designer-spec`, or `go-chi-spec`)
- local simplification is blocked by a broader architecture or ownership problem (`go-design-spec` or `go-architect-spec`)
- the "simplest" fix would weaken domain, security, reliability, or data guarantees owned elsewhere
- the cleanup crosses Go-semantic stop-signs and needs deeper ownership from `go-idiomatic-review`
