---
name: go-systematic-debugging
description: "Use when a Go bug, flaky test, build failure, hang, deadlock, timeout incident, or production regression needs a supported root cause; Own evidence-driven diagnosis and, only when authorized, the smallest causal fix and proof; Skip when the task is feature implementation, unresolved policy design, broad review, or claim-only verification."
---

# Go Systematic Debugging

## Accepted Input And Boundary

Use this skill for Go test or CI flakes, compilation and generation failures, panics, incorrect state, races, hangs, deadlocks, goroutine leaks, timeout incidents, pool or queue saturation, and production regressions. First classify the request as `diagnosis_only` or `fix_authorized`; both lanes trace the symptom to the first supported broken invariant.

Do not turn defect pressure into feature design, broad refactoring, bundled speculative fixes, permanent debug scaffolding, timeout inflation, retry widening, or client-visible API, data, security, reliability, ownership, or rollout changes without accepted policy.

## Method

- In `diagnosis_only`, inspect and run non-mutating experiments, then return the supported root cause, rejected hypotheses, residual uncertainty, and next falsifying experiment. Do not edit implementation or tests.
- In `fix_authorized`, diagnose first, then apply the smallest causal correction and verify it against the original failure signal.
- Start with the exact symptom, environment, failing scope, and evidence that exists now. Preserve volatile evidence—first stack, dump, trace, profile, logs, failing seed, or incident timing—before restart or edits can destroy it.
- Reproduce and classify the failure. Pin input, seed, order, CPU/race setting, build tags, environment, runtime state, and frequency as applicable, then shrink scope without removing the triggering condition.
- Use `fast_path` for a deterministic test, build failure, or short causal chain. Use `deep_dive` only when intermittency, concurrency, live hangs, saturation, or production-only conditions require runtime forensics.
- Trace backward to the first boundary where state, ownership, timing, or contract was already wrong. Distinguish that root cause from the loudest symptom and do not patch only the final failure site.
- State one primary hypothesis as `I think <cause> because <evidence>`, run the smallest discriminating experiment, and reject the hypothesis when the signal does not move. Keep alternatives visible without combining remedies.
- Match evidence to the failure class: selected sources for build drift, repetition/order/race controls for flakes, goroutine state for hangs, the right profile or trace for CPU/retention/wait/timeline questions, and budget/capacity attribution for timeouts.
- When triggered, inspect caller context and semantic errors, shared state and aliasing, locks and channels, goroutine exits, timers, resource cleanup, transactions, external waits, runtime state, and retry amplification.
- In `fix_authorized`, correct the earliest valid owner, add only recurrence guardrails justified by the proven defect, remove temporary diagnostics, and stop after root-cause proof if a safe repair needs an unresolved public contract, data, security, timeout/retry, ownership, rollout, or architecture decision.

## Symptom-Driven Reference Selector

Name the behavior-change thesis before loading a reference. Load at most one by default; use more only for independent pressures, such as an order-sensitive CI flake that also uses sleep-based coordination.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| CI-only, intermittent, order-, scheduler-, CPU-, race-, or environment-sensitive tests. | [flaky-repro-controls-go.md](references/flaky-repro-controls-go.md) | Isolate one control variable and replay a pinned failure instead of trusting one lucky pass. |
| `time.Sleep`, guessed polling, async readiness, or slower-CI timing. | [condition-based-waiting-go.md](references/condition-based-waiting-go.md) | Wait on an observable condition or event instead of widening sleeps and hiding lifecycle defects. |
| Deterministic panic, bad state, typed nil, payload regression, aliasing, or error-chain mismatch. | [root-cause-tracing-go.md](references/root-cause-tracing-go.md) | Backtrack to the first broken invariant instead of guarding the crash site. |
| Live hang, deadlock, goroutine leak, process stall, or stuck shutdown. | [runtime-forensics-go.md](references/runtime-forensics-go.md) | Capture perishable goroutine/runtime state before restart or edits. |
| CPU, heap, goroutine, block, mutex, or execution-trace choice is unclear. | [pprof-trace-and-profile-selection.md](references/pprof-trace-and-profile-selection.md) | Select the artifact that answers the active CPU, retention, wait, contention, or timeline question instead of collecting everything. |
| `context.Canceled`, deadline, HTTP latency, DB/queue/pool wait, saturation, or retry amplification. | [context-timeout-and-saturation-debugging.md](references/context-timeout-and-saturation-debugging.md) | Attribute the budget to work, capacity wait, coordination wait, or retry amplification before changing policy. |
| Build, tags, toolchain, module/workspace, embed, generated files, or `GOOS`/`GOARCH`/CGO drift. | [build-and-generated-artifact-debugging.md](references/build-and-generated-artifact-debugging.md) | Prove selected sources and generator authority instead of editing runtime logic or derived output. |
| Root cause is proven and recurrence guardrails are being considered. | [defense-in-depth-go.md](references/defense-in-depth-go.md) | Add only the owning-layer guardrail justified by the failure rather than broad hardening or redesign. |
| A fix exists, temporary diagnostics remain, or completion is about to be claimed. | [fix-verification-and-scaffolding-cleanup.md](references/fix-verification-and-scaffolding-cleanup.md) | Match RED/GREEN proof to the defect and remove temporary scaffolding before reporting success. |

## Proof, Return, And Stop

Every reproduction or verification record names:

- exact command or runtime capture action and working directory/package scope;
- critical environment, build tags, input/fixture, seed/order, iteration count, CPU, race, and timeout controls;
- first failing or passing signal and the saved artifact path when one exists;
- first broken boundary, accepted hypothesis, rejected alternatives, and why the experiment discriminates;
- minimal fix scope, cleanup of diagnostic scaffolding, and remaining uncertainty.

Prefer the narrow failing command, then broaden only when the defect class needs it. Record RED evidence and replay the same signal for fresh GREEN proof after an authorized fix. A flake needs repeated/order/race evidence matched to its trigger; a hang needs liveness evidence; a build defect needs the failing build/generation surface; a runtime incident needs the closest replay, smoke, metric, or captured signal available. A narrow pass never proves repository-wide correctness.

`diagnosis_only` succeeds when the symptom is reproducible or precisely characterized, the earliest broken invariant and causal path are supported, rejected hypotheses are recorded, and the next experiment is explicit if uncertainty remains. `fix_authorized` additionally requires the smallest causal fix, matching fresh proof, and cleanup of temporary diagnostics.

If root cause is not proven, stop with the next concrete falsifying experiment—not a patch list. Escalate when required evidence is unavailable, ownership cannot be localized, the only safe fix changes approved behavior or policy, the defect is primarily a design/domain/security/reliability/performance/data-cache decision, or fresh regression proof cannot be obtained. Never report `fixed` from intuition, one lucky flake pass, a wider timeout, or unrelated green checks.

Return a compact debug envelope: lane and symptom; reproducer and controls; key evidence and rejected hypotheses; first broken invariant and root cause; minimal fix or next experiment; commands/results; escalation decision; residual risk.
