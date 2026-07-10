---
name: go-systematic-debugging
description: "Debug Go service bugs, flaky tests, build failures, hangs, deadlocks, timeout incidents, and production regressions with root-cause-first diagnosis, concrete Go runtime forensics, and evidence-backed verification. Use whenever the user reports a panic, race, leak, flaky test, stuck goroutine, context deadline, or unexplained CI/build failure, even if they only ask to 'fix the test' or 'investigate the incident'."
---

# Go Systematic Debugging

## Trigger And Scope

Use this skill for Go test or CI flakes, compilation and generation failures, panics, incorrect state, races, hangs, deadlocks, goroutine leaks, timeout incidents, pool or queue saturation, and production regressions. Diagnose the first broken invariant, implement the smallest local fix only after root cause is supported, and verify against the original failure signal.

Do not turn defect pressure into feature design, broad refactoring, several speculative fixes, permanent debug scaffolding, timeout inflation, retry widening, or client-visible/API/data/security/reliability/rollout changes without approval.

## Diagnostic Boundary

- Start with the exact symptom, environment, failing scope, and evidence that exists now. Preserve volatile evidence—first stack, dump, trace, profile, logs, failing seed, or incident timing—before restart or edits can destroy it.
- Use `fast_path` for a deterministic test, build failure, or short causal chain. Use `deep_dive` only when intermittency, concurrency, live hangs, saturation, or production-only conditions require runtime forensics.
- Debugging may implement a local behavior-preserving fix when authorized. If the safe fix changes a public contract, data model, security rule, retry/timeout policy, ownership boundary, rollout, or architecture, stop after root-cause proof and route the decision to its owner.

## Diagnostic Invariants

1. **Root cause precedes remediation.** Trace backward from the crash or symptom to the first boundary where state, ownership, timing, or contract was already wrong; do not patch only the final failure site.
2. **One primary hypothesis at a time.** State `I think <cause> because <evidence>`, choose the smallest falsifying experiment, and reject it when the signal does not move. Keep alternatives visible without bundling fixes.
3. **The reproducer is the smallest honest one.** Pin the failing input, seed, order, CPU/race setting, environment, or runtime state and shrink scope without removing the condition that triggers the defect.
4. **Evidence matches the failure class.** Use build selection evidence for compile drift, repetition/order/race controls for flakes, goroutine state for hangs, the correct profile or trace for CPU/retention/waiting/timeline questions, and budget/capacity attribution for timeouts.
5. **Boundary and lifecycle ownership stay explicit.** Preserve caller context and semantic errors; inspect shared state, aliasing, locks, channels, goroutine exits, resource cleanup, transactions, external waits, and retry amplification when the symptom points there.
6. **The fix is causal and minimal.** Correct the earliest valid owner, add only recurrence guardrails justified by the proven defect, and remove temporary diagnostics that are not worth operating permanently.
7. **Verification replays the defect.** Record RED evidence, apply one minimal fix, rerun the matching GREEN proof fresh, and keep completion wording no broader than the command or incident signal proves.

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

## Required Evidence

Every reproduction or verification record names:

- exact command or runtime capture action and working directory/package scope;
- critical environment, build tags, input/fixture, seed/order, iteration count, CPU, race, and timeout controls;
- first failing or passing signal and the saved artifact path when one exists;
- first broken boundary, accepted hypothesis, rejected alternatives, and why the experiment discriminates;
- minimal fix scope, cleanup of diagnostic scaffolding, and remaining uncertainty.

Prefer the narrow failing command, then broaden only when the defect class needs it. A flake needs repeated/order/race evidence matched to its trigger; a hang needs liveness evidence; a build defect needs the failing build/generation surface; a runtime incident needs the closest replay, smoke, metric, or captured signal available. A passing narrow test is not repository-wide proof.

## Success, Escalation, And Stop Conditions

Success means the original symptom is reproducible or precisely characterized, the earliest broken invariant and causal path are supported by evidence, the smallest authorized fix is applied, the matching fresh signal turns green, and retained diagnostics are intentional and safe.

If root cause is not proven, stop with the next concrete falsifying experiment—not a patch list. Escalate when required evidence is unavailable, ownership cannot be localized, the only safe fix changes approved behavior or policy, the defect is primarily a design/domain/security/reliability/performance/data-cache decision, or fresh regression proof cannot be obtained. Never report `fixed` from intuition, one lucky flake pass, a wider timeout, or unrelated green checks.

Return a compact debug envelope: lane and symptom; reproducer and controls; key evidence and rejected hypotheses; first broken invariant and root cause; minimal fix or next experiment; commands/results; escalation decision; residual risk.
