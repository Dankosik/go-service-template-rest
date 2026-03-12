---
name: go-language-simplifier-review
description: "Review Go code changes for lower cognitive complexity, false-simplification risk, missed same-package source-of-truth extraction, junk-drawer helper risk, clearer naming, safer control-flow cleanup, and easier maintenance without collapsing semantics. Use whenever a Go diff claims cleanup/refactor/readability improvement or touches helper extraction, nested control flow, boolean flags, option bags, or error-path deduplication, even if another review lane also applies."
---

# Go Language Simplifier Review

## Purpose
Protect local reasoning quality in changed Go code without endorsing refactors that only reduce line count while hiding policy, state transitions, ownership, or caller-visible semantics.

## When To Use
- review Go PRs, diffs, refactors, and cleanup commits where the stated goal is simpler or more readable code
- use even on generic review requests when the change touches helper extraction, nested branching, delayed state interpretation, boolean mode flags, option bags, or error-path "deduplication"
- stay in the simplification lane; hand off deeper design or Go-semantic ownership instead of drifting into a broad review

## Review Posture
- Stay read-only and advisory.
- Review changed files and directly affected tests first.
- If `spec.md`, `plan.md`, or approved design notes exist, treat them as governing intent.
- Findings come first and must be ordered by merge risk, not by section order or taste.
- Green tests do not prove a cleanup preserved local reasoning safety.
- Always run a source-of-truth helper pass: flag both stable same-package policy still scattered across files and bogus extraction into vague helper buckets.

## Scope
- review control flow, state shape, and predicate clarity
- review abstraction cost, helper economics, and call-site burden
- review false simplification in error paths, ownership seams, and thin policy wrappers
- review whether stable same-package policy is scattered across files when one seam-named helper file should own it
- review whether a new helper actually reduces reasoning or just hides policy in a `util/common/shared` bucket
- review naming and test readability when they materially affect safe future changes
- review whether touched validation is enough to protect subtle precedence or branch behavior

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

### Risk Calibration And False Simplification
- Prioritize cases where a cleanup merges distinct behaviors, changes precedence, or makes side effects harder to trace.
- Treat line-count reduction, helper extraction, and deduplication as non-wins unless the reader now tracks fewer branches, locals, or hidden modes.
- Separate structural duplication from semantic duplication. Repeated shape is acceptable when each branch still owns different policy or externally visible behavior.
- Flag "refactors" that mostly move complexity out of sight rather than removing it.

### Abstraction Judgment And Helper Economics
- Flag single-use helpers, pass-through wrappers, and extract-method fan-out that force the reader to bounce across functions just to reconstruct one local decision.
- Keep helpers when they isolate stable policy, ownership, defaulting, lock scope, cleanup scope, stdlib quirks, or error normalization.
- Distinguish dead wrappers from thin policy seams. A helper can be only a few lines long and still matter because it clones data, preserves `errors.Is/As`, applies operation labels, or owns cleanup.
- Prefer inlining when the helper name adds no semantic compression and the body is the only place the logic is used.

### Source-Of-Truth Extraction Judgment
- Flag under-extraction when the same package carries repeated stable policy across files, especially normalization, mapping, validation, classification, section-reading, or label-shaping rules that are likely to drift if left file-local.
- Prefer one seam-named same-package helper file over multiple file-local copies when all three are true:
  - the logic names one stable domain noun or seam
  - the failure or ownership contract stays the same across call sites
  - the extraction reduces future drift without requiring mode flags, callbacks, or caller branching
- Flag over-extraction when helpers are named `util`, `utils`, `common`, `shared`, `helpers`, or similarly vague terms and the reader still has to rediscover the real policy elsewhere.
- Be skeptical of extracted helpers that need booleans, strings, callbacks, or option bags just to serve several call sites; they often merge distinct policy while pretending to simplify.
- Keep orchestration local. Response lifecycle, request flow, startup flow, and other phase-ordering logic should not be pulled into generic helpers just to reduce line count.

### Control Flow, State Shape, And Temporal Coupling
- Prefer a straight-line happy path with guard clauses when side-effect ordering remains explicit.
- Flag delayed interpretation via cross-branch sentinels such as `status`, `action`, `mode`, booleans, or shared `err` values that are only decoded at the tail.
- Narrow lifetimes of mutable locals. The more facts a reader must carry until the end of the function, the higher the merge risk.
- Treat manual phase machines encoded in locals or strings as simplification debt when explicit branch outcomes would be clearer.
- When flattening control flow, verify that side-effect order, deferred cleanup, and which error wins stay unchanged.

### Predicate And Condition Clarity
- Flag compound and negative predicates that force mental De Morgan expansion or hidden mode decoding.
- Treat clusters of booleans as hidden modes. Several flags often encode state that should be named explicitly.
- Prefer predicates that expose the decision being made, not the mechanics of how the inputs were derived.
- Be careful with extracted predicate helpers that hide policy terms the reader must still unpack elsewhere.

### API Surface And Call-Site Burden
- Flag flag-heavy signatures, same-typed positional parameters, raw strings with hidden meaning, `map[string]any` option blobs, and config decoding spread across the callee.
- Reject "simplifications" that replace useful small types, enums, or named options with shorter but more opaque call sites.
- For exported symbols, distinguish local cleanup from contract-shape change. If the public surface must move, escalate instead of treating it as casual simplification.
- Prefer one obvious way to call the function over highly configurable but cognitively heavy surfaces.

### Error-Path Simplification And Semantic Preservation
- Keep distinct failure classes distinct when callers or operators must reason about them differently.
- Flag helpers that collapse validation, conflict, retryable, not-found, or timeout errors into one generic bucket just to reduce duplication.
- Preserve inspectable contracts. If a refactor destroys `errors.Is/As`, caller-visible classification, or step identity, it is not behavior-preserving simplification.
- Prefer a little repetition over a generic error helper when each branch carries different semantics or status mapping.
- Keep which error wins explicit when cleanup, audit, rollback, or notification can also fail.

### Naming And Intent Exposure
- Require names that reveal role, phase, or policy, not generic mechanism words like `data`, `process`, `result`, or `do` when a sharper term exists.
- Flag vocabulary drift inside one feature area and booleans that do not read clearly at the call site.
- Prefer comments that explain why, constraints, or invariants; remove comments that merely narrate the code.

### Test Readability And Proof Shape
- Prefer tests whose setup, trigger, and expected failure or success are obvious on first read.
- Flag helper layers or giant tables that hide which branch, precedence rule, or invariant is actually being proven.
- Suggest simplification only when diagnosis improves. Terse tests that obscure the failure signal are not wins.
- If a refactor relies on subtle precedence or branch preservation, ask for focused tests or explicit validation commands.

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
