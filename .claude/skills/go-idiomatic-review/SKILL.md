---
name: go-idiomatic-review
description: "Review Go code changes for language-level correctness, toolchain-aware language-native or standard-library reinvention, error contracts, receiver and method-set safety, nil and zero-value behavior, ownership leaks, and standard-library contract misuse with real merge-risk impact. Use whenever a Go PR, diff, refactor, or incident fix may hide Go-semantic defects, even if the request is phrased as a generic code review or another review lane also applies."
---

# Go Idiomatic Review

## Purpose
Protect changed Go code from language-level, standard-library, and public-surface mistakes that create correctness, diagnosability, or long-term maintenance risk.

## Specialist Stance
- Review Go semantics and standard-library contracts as correctness surfaces, not style trivia.
- Prioritize error contracts, context lifetime, receiver/copy safety, nil behavior, exported APIs, and mutable ownership leaks.
- Prefer language-native and stdlib-first fixes when local wrappers add no real semantic value.
- Hand off business, DB/cache, concurrency, security, or architecture ownership when idiomatic review only reveals the boundary.

## When To Use
- review Go PRs, diffs, incident fixes, and refactors where correctness may be weakened by non-idiomatic Go
- use even on generic review requests when the change touches error handling, context, exported APIs, interfaces, sync primitives, slices, maps, `[]byte`, or stdlib wrappers such as `http.Header` and `url.Values`
- stay in the Go lane; hand off deeper domain ownership instead of drifting into redesign

## Review Posture
- Stay read-only and advisory.
- Review changed files and directly affected tests first.
- If approved task artifacts exist, treat them as governing intent.
- Do not invent policy when those artifacts are missing; still report clear Go-level defects visible in the code.
- Findings come first and must be ordered by merge risk, not by section order or style preference.
- Green tests are not proof that Go semantics, nil behavior, or exported contracts are safe.
- Always run a toolchain-aware language-native and stdlib-first pass; unnecessary reinvention of current Go capabilities is part of idiomatic review, not optional polish.

## Scope
- review error semantics, context propagation, control flow, and lifetime handling
- review receiver choice, method sets, must-not-copy state, and zero-value friendliness
- review slice, map, and `[]byte` ownership, aliasing, and exported-surface safety
- review package globals, `init` side effects, stdlib wrapper correctness, naming, and docs
- review whether changed code reimplements builtins or standard-library behavior that the repository's current Go toolchain already provides
- review whether validation matches the changed Go-risk surface

## Boundaries
Do not:
- turn idiomatic review into architecture redesign or deep specialist review
- block on taste-only comments with no correctness or maintenance impact
- take primary ownership of business rules, DB/cache policy, concurrency lifecycle, or security depth
- treat micro-optimization as idiomatic review unless it changes correctness, ownership, or API clarity
- confuse shorter code with clearer or safer code

## Core Defaults
- Correctness comes before style.
- Prefer explicit, readable, toolchain-compatible Go over clever abstraction.
- If a claim depends on Go version or stdlib contract details, say so instead of applying folklore.
- Treat builtins and the standard library as the default baseline. Repo-local reinvention is review-worthy when it adds drift surface without carrying extra contract semantics.
- Errors are contract values, not logs-only side effects.
- Request-scoped work must preserve caller-owned context and cancellation.
- Export the smallest surface you can defend.
- Prefer zero-value-usable types and obvious ownership of mutable data.
- Prioritize direct runtime failure, hidden success, data corruption, ownership leak, and contract-breakage findings before secondary cleanliness observations.
- Separate independent defects when the blast radius or fix differs.

## Expertise

### Risk Calibration And Finding Selection
- Prioritize findings that can produce caller-visible failure, panic, hidden success, data corruption, or broken public contracts.
- Separate independent Go failure modes even when they share a code region; do not bundle a must-not-copy defect, ownership leak, and doc gap into one finding unless the fix is genuinely the same.
- Keep documentation, naming, and package-tidiness comments below stronger correctness or contract findings unless the exported behavior is itself ambiguous.
- For exported helpers and shared-state containers, zero-value panic risk and mutable-ownership leaks usually outrank stylistic cleanup.

### Version-Sensitive Language Rules
- State when a finding depends on Go version, compiler behavior, or a standard-library semantic guarantee.
- Do not repeat outdated folklore. Example: the classic `for range` loop-variable capture warning changed materially in Go 1.22; only raise it when the actual version and escape pattern still make it a bug.
- Prefer citing the specific semantic rule over cargo-cult advice.

### Language-Native And Standard-Library First Findings
- Check the repository's declared Go version before tolerating compatibility helpers, wrapper utilities, or older pre-generic patterns.
- Treat custom helpers or wrappers as idiomatic-review findings when current Go builtins or the standard library already express the same contract and the local code adds no real semantic value.
- This lane is broad on purpose: value selection, min/max logic, slice or map operations, sorting, comparison, cloning, error inspection, context helpers, path or URL handling, string or byte transforms, time helpers, and stdlib-wrapper utilities should all default to language-native or stdlib solutions first.
- Do not demand a builtin or stdlib replacement when the helper carries real contract meaning: ownership isolation, nil-versus-empty preservation, bounds policy, normalization policy, error identity, repeated business meaning, or actual compatibility with an older supported Go version.
- If the builtin or stdlib version is almost enough, check whether the remaining semantic gap is real and externally relevant. If not, prefer the native form and report the wrapper as unnecessary drift.
- When raising this kind of finding, explain both the simpler native replacement and why the local reinvention increases maintenance, review, or semantic-drift risk.

### Control Flow And Readability
- Prefer guard clauses and early returns for the happy path.
- Flag unnecessary nesting, mixed abstraction levels, and functions with multiple unrelated responsibilities when they obscure failure behavior.
- Treat shadowing or reused names as merge risk when they hide the live error, context, or mutable value being acted on.
- Flag control flow that makes it hard to tell whether side effects already happened before failure.

### Error Semantics And Contracts
- Require errors to be returned or handled explicitly, not swallowed behind logs or converted to `(nil, nil)`.
- Preserve inspectable contracts: use `%w` when callers must inspect causes, and `errors.Is` or `errors.As` rather than string matching.
- Distinguish sentinel, typed, joined, and opaque errors deliberately; flag code that destroys the chosen contract at package boundaries.
- Keep error messages lowercase and punctuation-free unless an external contract says otherwise.
- Reject panic for normal error handling and avoid double-logging the same failure path.
- If code compares errors directly, verify that the compared value is actually the exported contract at that seam.

### Context And Lifetime Semantics
- Require `ctx context.Context` first where cancellation, deadlines, or request identity matter.
- Flag storing contexts in structs, passing nil context, or replacing caller context with `context.Background()` inside request flows.
- Require derived contexts to cancel on all resource-owning paths.
- Preserve `context.Canceled` and `context.DeadlineExceeded` semantics rather than flattening them into generic errors.
- Flag API shapes that make callers lose control of cancellation ownership.

### Receivers, Method Sets, And Copy Safety
- Require receiver choice to match mutation, identity, and method-set intent.
- Flag mixed pointer and value receivers when they create surprising interface satisfaction or copy semantics.
- Flag value receivers on types containing `sync.Mutex`, `sync.RWMutex`, `sync.Once`, `sync.WaitGroup`, atomic state, `strings.Builder`, or other must-not-copy fields after first use.
- Treat pointer-to-mutex and similar indirection as a smell unless shared indirection is intentional and documented.
- Review method values and closures carefully when they capture a receiver copy that diverges from the caller's expectation.
- Verify nil-receiver behavior is deliberate on exported pointer methods.

### Interfaces, Concrete Types, And Abstraction Load
- Prefer concrete types unless real consumer-side substitution exists.
- Flag interface-per-struct, producer-owned mock interfaces, and pass-through abstraction layers with no policy value.
- Require exported interfaces to encode behavior boundaries, not internal testing convenience.
- Flag interfaces that are too wide for the call sites they serve.
- Treat hidden implementation coupling behind tiny wrapper interfaces as maintainability risk, not abstraction success.

### Zero Values, Nil, And Typed-Nil Pitfalls
- Prefer types with useful or at least harmless zero values when practical.
- Flag constructors that are mandatory only because the type cannot survive its own zero value without good reason.
- Flag typed-nil interface traps, nil map writes, nil channel semantics leaking into APIs, and ambiguous nil-vs-empty return contracts.
- When factories or constructors return interfaces, check disabled or empty branches for typed-nil concrete values and explain the observable caller contract, not just the underlying Go trick.
- If the safe fix changes optional or disabled behavior, make the contract alternatives explicit: real `nil`, a safe no-op implementation, or a signature that reports absence directly.
- Review whether exported APIs distinguish absent, empty, and zero deliberately.
- Treat nil-vs-empty normalization as medium or lower by default unless encoding, marshaling, or public-contract semantics make it externally observable.
- For collections and byte buffers, make ownership and mutability boundaries explicit.

### Slices, Maps, Buffers, And Ownership
- Flag returning or storing mutable slices or maps when callers can mutate internal state or observe aliasing.
- Remember that map and slice headers are small value types but still alias underlying state; copying the header does not create isolation.
- Flag appends that retain large backing arrays or leak scratch-buffer ownership across boundaries when that affects correctness or memory safety.
- Prefer clone or copy on boundary crossings when the contract requires isolation.
- Treat map iteration order assumptions as correctness defects.
- For `[]byte`, `bytes.Buffer`, and similar mutable payloads, verify whether the callee may retain or expose backing storage after return.
- For caches and shared-state containers, ownership leaks across the package boundary often outrank receiver-style cleanup because they can corrupt state after locks are released.

### Resources, Defer, And Standard-Library Discipline
- Verify cleanup of `rows.Close`, `Body.Close`, files, timers, tickers, cancel funcs, unlocks, and other release paths.
- Flag `defer` inside hot or long-lived loops when it can accumulate resources or obscure lifetime.
- Check `rows.Err`, `scanner.Err`, HTTP response body handling, partial `Read` contracts, and timer or ticker stop semantics where relevant.
- Prefer stdlib helper methods that preserve contract behavior over raw map or field mutation when the type exposes them, for example `http.Header`, `url.Values`, and similar wrappers.
- Pointer-to-map, pointer-to-slice, or pointer-to-interface exported APIs are usually a smell unless nilability or mutation semantics truly require that shape.
- Treat package-level mutable state and `init` side effects as contract risk when they alter runtime behavior implicitly.

### Package Surface, Globals, Naming, And Docs
- Keep package responsibility focused and import direction clear.
- Minimize exported surface and use `internal/` where privacy matters.
- Flag mutable exported globals, hidden wiring through `init`, and package APIs that require callers to know internal sequencing.
- Keep package-hygiene findings proportional: raise them to high severity when they create mutable process-wide contract, race surface, or exported API stickiness, not merely because they look untidy.
- Enforce Go naming norms, stable initialisms, and non-stuttering package APIs.
- Require boolean names that read as facts or questions, and exported docs that explain behavior or constraints rather than restating the name.

### Tests And Validation Signals
- Review whether touched tests prove the Go-level risk surface that changed: error inspection, nil handling, context cancellation, copy safety, alias isolation, or public-surface behavior.
- Suggest the smallest command set that honestly validates the changed risk surface.
- Expect `-race` only when the touched defect overlaps concurrency-owned behavior; otherwise hand off.
- Do not claim readiness without a clear verification path.

### Cross-Domain Handoffs
- Hand off deep race, goroutine-lifecycle, channel, or shutdown analysis to `go-concurrency-review`.
- Hand off DB/cache ownership and query semantics to `go-db-cache-review`.
- Hand off public API semantic depth or architecture drift to `go-design-review`.
- Hand off threat-depth analysis to `go-security-review`.
- Hand off profiling and hot-path proof questions to `go-performance-review`.
- Hand off coverage strategy completeness to `go-qa-review`.

## Finding Quality Bar
Each finding should include:
- exact `file:line`
- the concrete Go rule, semantic pitfall, or standard-library contract misuse
- why it creates correctness, diagnosability, or maintenance merge risk
- the smallest safe correction
- a validation command when useful
- whether the issue is local code drift, a specialist handoff, or needs design escalation
- when the finding depends on a builtin or stdlib alternative, cite the relevant Go-version or standard-library capability if that makes the recommendation materially clearer

Severity is merge-risk based:
- `critical`: confirmed Go-level defect with direct correctness, panic, data corruption, or operational risk
- `high`: strong evidence of meaningful correctness, API-contract, ownership, or must-not-copy risk
- `medium`: bounded but important idiomatic weakness with realistic maintenance or observability cost
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

## Escalate When
Escalate when:
- a safe correction changes the public API, exported zero-value contract, or approved package ownership model (`go-design-spec` or `go-architect-spec`)
- transport or API-visible error or status behavior must change (`api-contract-designer-spec` or `go-chi-spec`)
- the issue reveals missing reliability, security, data, or concurrency policy owned elsewhere (`go-reliability-spec`, `go-security-spec`, `go-db-cache-spec`, or `go-distributed-architect-spec`)
- local idiomatic cleanup is blocked by a broader design mistake or missing approved decision
