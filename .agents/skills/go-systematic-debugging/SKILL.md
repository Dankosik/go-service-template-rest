---
name: go-systematic-debugging
description: "Debug Go service bugs, flaky tests, build failures, hangs, deadlocks, timeout incidents, and production regressions with root-cause-first diagnosis, concrete Go runtime forensics, and evidence-backed verification. Use whenever the user reports a panic, race, leak, flaky test, stuck goroutine, context deadline, or unexplained CI/build failure, even if they only ask to 'fix the test' or 'investigate the incident'."
---

# Go Systematic Debugging

## Purpose
Find and verify the real root cause of Go defects before finalizing a fix, using the smallest diagnostic surface that can prove or disprove the suspected failure mode.

## Scope
- debug failing tests, flaky CI, build failures, runtime panics, hangs, deadlocks, goroutine leaks, timeout incidents, pool saturation, and integration regressions in Go services
- establish deterministic reproduction or characterize the exact intermittency pattern when deterministic repro is not available yet
- choose the right Go diagnostic lane, capture the right evidence, and shrink the failure to the first broken invariant
- implement and verify the smallest safe fix once root cause is confirmed
- make contract or design escalation explicit when the safe fix is no longer local

## Boundaries
Do not:
- treat feature design or broad refactoring as the main task
- silently change API, data, security, reliability, or architecture semantics under defect pressure
- bundle several speculative fixes together
- keep permanent debugging scaffolding when short-lived diagnostics were enough
- default to raising timeouts, adding retries, or widening sleeps before proving the actual bottleneck

## Specialist Stance
- Evidence over intuition.
- Fix the source of bad state, not only the crash site.
- One primary hypothesis at a time; only parallelize evidence capture when the tracks are independent.
- Match the diagnostic tool to the failure class; do not reach for profiles, traces, or dump capture out of habit.
- Keep the blast radius small and the proof fresh.
- Keep workflow and product decisions outside the debugging lane; debugging proves the failure mode and the smallest safe fix.

## Debugging Modes
Default to the smallest mode that can prove the bug:
- `fast path`: local deterministic test or build failure with a short causal chain
- `deep dive`: flaky, concurrency-sensitive, hang, timeout, pool-saturation, or production-only failure where runtime forensics are needed

Do not let deep-dive tooling delay a narrow deterministic fix that is already proven.

## Lazy Reference Selector
Use `SKILL.md` as the lane selector. References are compact rubrics and example banks, not exhaustive Go documentation or generic checklists. Load at most one reference by default: choose the file most likely to change the next diagnostic decision. Load multiple references only when the task clearly spans independent decision pressures, such as a CI-only flake that also contains sleep-based synchronization, or a live hang where the profile type is genuinely unclear.

| Reference | Symptom trigger | Behavior change thesis |
|---|---|---|
| `references/flaky-repro-controls-go.md` | CI-only, order-sensitive, scheduler-sensitive, repeated, `-race`, `-shuffle`, CPU-count, or environment-shaped test failures | When loaded for intermittent Go test failures, this file makes the model isolate repetition, order, race, CPU, and environment variables instead of mixing knobs or claiming a flake is fixed from one lucky pass. |
| `references/condition-based-waiting-go.md` | `time.Sleep`, guessed polling, async readiness, or slower-CI timing failures in tests | When loaded for sleep-based async tests, this file makes the model wait on an observable condition or event instead of inflating sleeps or hiding goroutine lifecycle bugs. |
| `references/root-cause-tracing-go.md` | Deterministic panics, bad state, typed-nil surprises, payload/state regressions, or error-chain mismatches where the failing line may not be the source | When loaded for deterministic symptoms with upstream bad state, this file makes the model backtrack to the first broken invariant instead of patching the crash site or adding a defensive nil guard. |
| `references/runtime-forensics-go.md` | Live hangs, deadlocks, goroutine leaks, process stalls, stuck shutdown, SIGQUIT dumps, or volatile runtime evidence | When loaded for a live stalled or leaking Go process, this file makes the model capture the most perishable runtime artifact before restart or edits instead of destroying evidence. |
| `references/pprof-trace-and-profile-selection.md` | CPU, heap, goroutine, block, mutex, or execution-trace evidence is needed but the right artifact is unclear | When loaded for ambiguous profile or trace choices, this file makes the model select the artifact that matches active CPU, retention, waiting, or timeline questions instead of collecting everything or using CPU profiles for blocked work. |
| `references/context-timeout-and-saturation-debugging.md` | `context.Canceled`, `context.DeadlineExceeded`, test timeouts, HTTP latency, DB pool wait, queue wait, retry amplification, or saturation | When loaded for timeout or saturation symptoms, this file makes the model attribute time to the budget owner, capacity wait, work time, or retry amplification instead of raising timeouts or adding retries first. |
| `references/build-and-generated-artifact-debugging.md` | `go build`, test compilation, generated files, build tags, `GOOS`/`GOARCH`, `CGO_ENABLED`, module/workspace, embedding, or generator drift | When loaded for build or generated-artifact failures, this file makes the model prove selected files, tags, toolchain, and generator source of truth instead of changing runtime logic or hand-editing generated output. |
| `references/defense-in-depth-go.md` | Root cause is proven and recurrence guardrails are being considered | When loaded after a root-cause fix, this file makes the model add only the owning-layer guardrail justified by the failure mode instead of broad hardening, retries, metrics, or redesign. |
| `references/fix-verification-and-scaffolding-cleanup.md` | Likely fix is in place or success is about to be reported | When loaded before a completion claim, this file makes the model match RED/GREEN proof to the original defect and remove temporary diagnostics instead of overclaiming from a narrow pass. |

## Expertise

### Choose The Debugging Lane
Pick the dominant lane first, then broaden only if the evidence forces it.

- `compile or build failure`
  - confirm the exact failing package, build tags, generated files, and toolchain or env drift
  - start with `go build ./...` or the narrow failing package before touching runtime reasoning
  - read `references/build-and-generated-artifact-debugging.md` when generated files, build constraints, or toolchain drift are plausible

- `deterministic test or panic`
  - reduce to the narrowest failing test with `go test ./path/to/pkg -run '^TestName$' -count=1 -v`
  - preserve the first stack trace and first broken assertion; later noise is secondary
  - check nil or typed-nil, zero-value misuse, context replacement, aliasing, and state ownership before rewriting logic
  - read `references/root-cause-tracing-go.md` when the crash site might not be the first broken boundary

- `flake or order-sensitive test`
  - characterize frequency and trigger conditions with repetition, `-shuffle`, `-race`, and controlled CPU parallelism
  - isolate time, randomness, global state, environment, temp-path, port, and cleanup leakage
  - read `references/flaky-repro-controls-go.md` when the next decision is reproducer shape; read `references/condition-based-waiting-go.md` when sleep or readiness replacement is the next decision

- `hang, deadlock, or goroutine leak`
  - capture goroutine state before editing code
  - inspect who is blocked on send, receive, lock, wait, or shutdown drain
  - use `references/runtime-forensics-go.md` for dump and profile capture patterns
  - use `references/pprof-trace-and-profile-selection.md` when the first artifact choice is not obvious

- `timeout, cancellation, or saturation incident`
  - identify who owned the budget, where time was spent, and whether the failure is real work, queue wait, lock wait, connection-pool wait, or retry amplification
  - preserve `context.Canceled` and `context.DeadlineExceeded` semantics; do not normalize them into vague timeout strings
  - read `references/context-timeout-and-saturation-debugging.md` before widening timeouts or adding retries

- `data or integration failure`
  - trace payload shape, query shape, lock scope, retry behavior, DNS or TLS or connectivity, cache staleness, and transaction scope
  - do not widen retries or timeouts until boundary evidence explains why the dependency failed

### Reproduction And Control Knobs
Prefer concrete, replayable commands over narrative descriptions.

Useful Go defaults:
- `go test ./path/to/pkg -run '^TestName$' -count=1 -v`
- `go test ./path/to/pkg -run '^TestName$' -count=50`
- `go test ./path/to/pkg -run '^TestName$' -shuffle=on -count=50`
- `go test -race ./path/to/pkg/...`
- `go test ./path/to/pkg -run '^TestName$' -cpu=1,4`
- `go test ./path/to/pkg -json`
- `go build ./...`

Capture the knobs that materially change behavior:
- package or command path
- env toggles like `GOOS`, `GOARCH`, `CGO_ENABLED`, `TZ`, `GOMAXPROCS`, feature flags, DSNs, or dependency endpoints
- repetition count, shuffle mode or seed, CPU setting, race on or off, and timeout
- exact input fixture or payload

Once a failing seed, order, or load shape is known, pin it and keep shrinking the reproducer.

Keep order and race experiments distinct when that matters:
- use a narrow single-test reproducer to prove local race, lifecycle, or timing behavior
- use wider package or subtest scope when you actually need order dependence or shuffled overlap
- `-shuffle` is only diagnostic when multiple tests or subtests remain in scope; pairing it with an over-narrow `-run` can fake coverage of the real CI order condition

### Evidence Collection And Boundary Tracing
- Trace the failing path across transport, application, domain, persistence, cache, queue, and external dependency boundaries.
- Capture the first boundary where the invariant was already broken or first allowed through.
- Prefer exact artifacts over paraphrase:
  - input fixture or request payload
  - failing stack or error chain
  - relevant log lines with correlation or request ID
  - profile, trace, dump, or query evidence when the symptom fits
- Keep temporary diagnostics safe, bounded, and easy to remove. No secret leakage. No unbounded-cardinality logging.

For boundary-first backtracking, read `references/root-cause-tracing-go.md`.

### Command Evidence Contract
Every claimed reproduction or verification step should make these explicit:
- exact command
- working directory or package scope
- critical env toggles
- iteration shape, for example `-count=50` or `-cpu=1,4`
- the key failing or passing signal
- any saved artifact path such as `cpu.out`, `trace.out`, or copied logs

If you cannot supply this, state the evidence gap instead of softening the claim.

### Stack, Dump, And Runtime Forensics
Use heavier forensics only when the symptom justifies them.

- `panic or crash`
  - preserve the first stack trace
  - when stack depth is truncated or multiple goroutines matter, capture a fuller traceback such as `GOTRACEBACK=all`

- `stuck process or deadlock suspicion`
  - capture a goroutine dump, often via `SIGQUIT`, before restarting the process when that is safe
  - identify repeated blocked locations, ownership cycles, or shutdown waits

- `contention or queueing suspicion`
  - use block and mutex profiles when waiting, lock hold time, or serialized throughput is the symptom
  - use `go tool trace` when scheduler behavior, wakeups, runnable goroutine bursts, or fan-out stalls are the real unknowns

- `leak or growth suspicion`
  - use goroutine and heap evidence when goroutine count, memory, or RSS only moves upward
  - distinguish one-time warmup from monotonic growth

Read `references/runtime-forensics-go.md` for concrete commands and artifact interpretation.

### Go-Specific Suspicion List
Check these early when the symptom fits:
- typed-nil interface values that are non-nil at the interface boundary
- nil channel semantics causing permanent block, or nil map writes causing panic
- unsynchronized map access, copied `sync.WaitGroup` or `Mutex`, or shared-state aliasing across goroutines
- goroutines without a bounded completion or cancellation path
- `context.Background()` or new root contexts replacing inbound request context
- locks, DB transactions, or critical sections held across network or disk I/O
- hidden global state in tests: env vars, clocks, randomness, temp paths, ports, singletons, or leftover goroutines
- retries, timers, or sleeps masking the real dependency or coordination defect

### Single-Hypothesis Experiments
- State one primary hypothesis clearly: `I think <cause> because <evidence>`.
- Choose the smallest experiment that changes one variable and can falsify it.
- Reject the hypothesis quickly if the experiment does not move the signal.
- Keep alternative hypotheses visible when the evidence is not yet discriminating.

### Flaky Test Stabilization
Treat flakes as a class of defect, not as “probably timing”.

- Characterize the flake:
  - order dependence
  - shared state or cleanup leakage
  - time or clock dependence
  - scheduler sensitivity
  - dependency or network leakage
  - race or goroutine lifecycle bug

- Prefer these controls:
  - repetition with `-count`
  - order perturbation or pinning with `-shuffle`
  - `-race` for shared-state suspicion
  - `-cpu` variation when scheduler sensitivity is plausible
  - deterministic seeds, fake clocks, isolated temp dirs, isolated ports, explicit env reset, and `t.Cleanup`

- Do not collapse distinct flake experiments into one vague command.
  - order-dependent suspicion: keep enough test or subtest scope for `-shuffle` to matter, then capture and replay the failing seed
  - local race or lifecycle suspicion: keep the narrow test target and use `-race`, `-count`, and `-cpu`

- Replace sleep-based guesses with condition-based waiting.
- Do not “fix” a flake only by inflating timeouts unless timing itself is the behavior under test.

Read `references/flaky-repro-controls-go.md` when the bug only appears in CI or under repetition. Add `references/condition-based-waiting-go.md` only when the fix also turns on sleep/readiness replacement.

### External Boundary And Resource Diagnostics
When the symptom sits around I/O, queues, DB, or caches, identify the concrete wait or failure source before changing policy.

- DB or transaction path
  - look for pool wait, lock wait, slow query shape, and transactions that stay open across network calls
- HTTP or RPC path
  - separate connect, DNS, TLS, request, response-body, and retry time from one generic timeout
- cache or queue path
  - distinguish stale read, cache stampede, missing invalidation, duplicate delivery, and ack-before-durable-write defects
- incident path
  - verify whether the service is doing slow real work, waiting on capacity, or waiting forever on coordination

Escalate if the only safe fix changes retry, timeout, durability, or contract semantics.

### Defense-In-Depth Remediation
After root cause is proven:
- fix the earliest valid boundary first
- add only the guardrails justified by the actual failure mode
- keep diagnostics that materially improve future triage; remove the rest

Use `references/defense-in-depth-go.md` when deciding which follow-up guardrails are worth keeping.

### Verification And Regression Proof
Require explicit RED/GREEN proof:
- failing reproduction or incident signal recorded
- minimal fix applied
- reproduction now passes or the incident signal is measurably gone

Validation guidance:
- build failure: rerun the failing build or narrow package build
- deterministic test: rerun the exact failing command
- flake or race: rerun repeated or `-race` evidence that matches the defect class
- runtime incident: rerun the relevant repro, smoke, or captured command set; keep the scope honest
- do not claim repository-wide safety from a narrow passing command

Fresh command evidence is required before any positive completion language.
Read `references/fix-verification-and-scaffolding-cleanup.md` when choosing the final proof set or removing temporary diagnostics.

## Boundaries And Handoffs
Keep workflow touchpoints minimal:
- if the confirmed safe fix changes API, data, security, reliability, rollout, or architecture semantics, stop at root-cause proof and hand the decision back to the orchestrator or the relevant specialist
- preserve short-lived diagnostic evidence in the debug note or command output; only ask for preserved research/artifact updates when the investigation is complex enough that the evidence must survive the current task
- for a local fix that stays within approved behavior, keep the artifact footprint small and focus on the debug envelope below

## Debugging Quality Bar
Each debugging conclusion should make the following explicit:
- debug lane
- failing symptom and reproducer
- boundary where the first invariant failed
- accepted and rejected hypotheses
- root-cause statement
- minimal fix scope
- verification commands and outcomes
- escalation decision
- residual risk or next evidence step if still uncertain

## Handoff Notes
When reporting debugging work, keep the debug envelope explicit:
- debug lane, symptom, and reproducer
- key evidence and rejected hypotheses
- root cause and minimal fix scope
- verification commands and observed results
- escalation decision and residual risk

If root cause is not proven yet, end with the next concrete experiment, not a speculative patch list.

## Escalate When
Escalate if:
- a fix is being proposed before reproducible evidence exists
- the necessary fix would materially change approved API, data, security, reliability, or architecture behavior
- the invariant cannot be localized without changing ownership or contract boundaries
- several speculative changes are being bundled together
- a flake is being “fixed” only by timeout inflation or broader sleeps
- no fresh regression proof exists for a claimed fix
- the defect is primarily a design, reliability, performance, or DB/cache policy problem rather than a local debugging gap
