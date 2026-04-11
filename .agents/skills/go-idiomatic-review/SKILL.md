---
name: go-idiomatic-review
description: "Review Go code changes for language-level correctness, toolchain-aware language-native or standard-library reinvention, error contracts, receiver and method-set safety, nil and zero-value behavior, ownership leaks, and standard-library contract misuse with real merge-risk impact. Use whenever a Go PR, diff, refactor, or incident fix may hide Go-semantic defects, even if the request is phrased as a generic code review or another review lane also applies."
---

# Go Idiomatic Review

## Purpose
Protect changed Go code from language-level, standard-library, and exported-surface mistakes that create correctness, diagnosability, compatibility, or long-term maintenance risk.

## Specialist Stance
- Review Go semantics and standard-library contracts as correctness surfaces, not style trivia.
- Prioritize error contracts, context lifetime, receiver and copy safety, nil behavior, exported API shape, and mutable ownership leaks.
- Prefer language-native and standard-library fixes when local wrappers add no real semantic value.
- Stay in the Go-language review lane; hand off domain, concurrency, DB/cache, security, performance, reliability, or architecture depth instead of drifting into redesign.
- Treat Effective Go as useful core-language guidance with its official caveat: it was written for Go's 2009 release and is not actively updated. Prefer current release notes, pkg.go.dev docs, the Go spec, Go Code Review Comments, and official Go blog posts for version-sensitive claims.

## When To Use
- Review Go PRs, diffs, incident fixes, and refactors where correctness may be weakened by non-idiomatic Go.
- Use even on generic review requests when the change touches error handling, contexts, exported APIs, interfaces, sync primitives, slices, maps, `[]byte`, nil handling, receiver choice, or wrappers around standard-library types.
- Run a toolchain-aware pass when the repository's `go.mod` version may make newer builtins or packages available.

## Review Loop
1. Read the changed Go files, directly affected tests, and any approved task artifacts that define intent.
2. Identify the repository's Go version from `go.mod`, build tags, or stated toolchain constraints before making version-sensitive claims.
3. Choose the relevant review axes and lazily load only the needed reference files from `references/`.
4. Select findings by merge risk: direct failure, hidden success, panic, data corruption, ownership leak, broken public contract, or durable maintenance drift.
5. For each finding, name the concrete Go rule or stdlib contract, the observable impact, the smallest safe correction, and the validation signal.
6. Escalate or hand off when the fix needs another lane's ownership.

## Lazy Reference Selection
Load reference files only when the diff touches that axis or you need concrete examples/source anchors. Do not bulk-load the directory by default.

| Reference | Load when |
| --- | --- |
| `references/errors-and-contracts-review.md` | Errors are swallowed, logged instead of returned, string-matched, wrapped with `%w` or `%v`, joined, typed, sentinel-based, or exported as package contracts. |
| `references/context-and-lifetime-review.md` | `context.Context` is stored, replaced with `context.Background`, passed nil, omitted from request-scoped work, or derived without clear cancellation ownership. |
| `references/receivers-methodsets-and-copy-safety.md` | Receivers, method sets, interface satisfaction, value copies, `sync` fields, `strings.Builder`, `bytes.Buffer`, or pointer-to-map/slice/interface shapes changed. |
| `references/nil-zero-value-and-typed-nil.md` | Nil interfaces, typed-nil errors, nil maps/channels/slices, constructors, zero-value usability, absent vs empty semantics, or JSON-visible nil behavior changed. |
| `references/slices-maps-buffers-and-ownership.md` | Slices, maps, `[]byte`, buffers, `http.Header`, `url.Values`, cloning, aliasing, map iteration order, or mutable data crossing package boundaries changed. |
| `references/stdlib-first-modern-go-review.md` | Custom helpers duplicate current Go builtins or stdlib packages such as `errors`, `slices`, `maps`, `cmp`, `strings`, `bytes`, `net/url`, or `net/http`. |
| `references/exported-api-and-interface-shape.md` | Exported names, doc comments, package names, interfaces, constructors, compatibility, option structs, or public method/function signatures changed. |

If multiple axes apply, load the smallest set that explains the concrete finding. If a reference points to deeper concurrency, data, security, domain, or architecture policy, use it to frame the handoff rather than doing that review here.

## Core Axes
- Error semantics: preserve inspectable contracts with deliberate sentinel, typed, joined, wrapped, or opaque errors. Use `errors.Is` and `errors.As` when callers need cause inspection; do not string-match error text.
- Context lifetime: pass caller-owned `ctx context.Context` through request-scoped work, keep it first, avoid storing it in structs, and cancel derived contexts on all resource-owning paths.
- Receivers and method sets: match receiver choice to mutation, identity, interface satisfaction, and must-not-copy state. Avoid value receivers or value copies on types containing locks, atomics, builders, buffers, or other copy-sensitive fields.
- Nil and zero values: prefer useful or harmless zero values when practical. Make typed-nil, nil map writes, nil channel blocking, and nil-vs-empty public contracts explicit.
- Ownership: treat slices, maps, `[]byte`, buffers, headers, and URL values as aliasing surfaces. Clone or copy at boundaries when callers must not mutate internal state.
- Standard library first: prefer current builtins and stdlib helpers over local reinvention when the helper adds no compatibility, ownership, normalization, or domain contract.
- Exported surface: keep exported API small, documented, compatible, and consumer-oriented. Prefer concrete return types unless an interface represents a real behavior boundary.
- Resources and control flow: check cleanup and error probes such as `Body.Close`, `rows.Close`, `rows.Err`, `scanner.Err`, timer/ticker release, and `defer` lifetime where they are part of the changed Go contract.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the concrete Go rule, semantic pitfall, or standard-library contract misuse
- why it creates correctness, diagnosability, compatibility, ownership, or maintenance merge risk
- the smallest safe correction
- a validation command or test idea when useful
- whether the issue is local Go drift, a specialist handoff, or needs design escalation
- for version-sensitive stdlib or builtin recommendations, the relevant Go version or source anchor

Severity is merge-risk based:
- `critical`: confirmed Go-level defect with direct correctness, panic, data corruption, or operational risk
- `high`: strong evidence of meaningful correctness, API-contract, ownership, or must-not-copy risk
- `medium`: bounded but important idiomatic weakness with realistic maintenance, diagnosability, or compatibility cost
- `low`: local cleanup that materially improves clarity or contract safety

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
[severity] [go-idiomatic-review] [file:line]
Issue:
Impact:
Suggested fix:
Reference:
```

Start `Issue` with the plain-language defect. Add an `Axis:` label only when it materially disambiguates why the issue belongs in idiomatic Go review.

## Boundaries And Handoffs
- Hand off deep goroutine lifecycle, channel, lock-order, `sync/atomic`, or shutdown analysis to `go-concurrency-review`.
- Hand off DB/cache ownership, transaction, query, and invalidation semantics to `go-db-cache-review`.
- Hand off public API product semantics, package ownership, or architecture drift to `go-design-review` or architecture/spec lanes.
- Hand off auth, tenant isolation, injection, SSRF, secret handling, and abuse depth to `go-security-review`.
- Hand off profiling, benchmark sufficiency, allocation budgets, and hot-path tradeoffs to `go-performance-review`.
- Hand off coverage strategy completeness to `go-qa-review`.

## Escalate When
Escalate when:
- a safe correction changes a public API, exported zero-value contract, compatibility promise, or approved package ownership model
- transport or API-visible error/status behavior must change
- the issue reveals missing reliability, security, data, domain, concurrency, or distributed policy owned elsewhere
- local idiomatic cleanup is blocked by a broader design mistake or missing approved decision
