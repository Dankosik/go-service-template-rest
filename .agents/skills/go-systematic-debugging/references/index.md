# Reference Selector

Name the behavior-change thesis before loading a reference. Load at most one by default; use more only for independent pressures, such as an order-sensitive CI flake that also uses sleep-based coordination, or a boundary-crossing symptom whose owning hop then needs in-process tracing.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| The symptom crosses a service, client, job, or managed-dependency boundary: a caller-supplied value, a rejected or malformed request, a 4xx/5xx from or to a neighbor, a missing or duplicated effect, or a frontend-visible failure. | [cross-service-fault-localization.md](cross-service-fault-localization.md) | Localize the defect across the whole failing path from evidence on both sides of every hop, instead of assuming the repository holding the stack trace holds the defect. |
| CI-only, intermittent, order-, scheduler-, CPU-, race-, or environment-sensitive tests. | [flaky-repro-controls-go.md](flaky-repro-controls-go.md) | Isolate one control variable and replay a pinned failure instead of trusting one lucky pass. |
| `time.Sleep`, guessed polling, async readiness, or slower-CI timing. | [condition-based-waiting-go.md](condition-based-waiting-go.md) | Wait on an observable condition or event instead of widening sleeps and hiding lifecycle defects. |
| Deterministic panic, bad state, typed nil, payload regression, aliasing, or error-chain mismatch. | [root-cause-tracing-go.md](root-cause-tracing-go.md) | Backtrack to the first broken invariant instead of guarding the crash site. |
| Live hang, deadlock, goroutine leak, process stall, or stuck shutdown. | [runtime-forensics-go.md](runtime-forensics-go.md) | Capture perishable goroutine/runtime state before restart or edits. |
| CPU, heap, goroutine, block, mutex, or execution-trace choice is unclear. | [pprof-trace-and-profile-selection.md](pprof-trace-and-profile-selection.md) | Select the artifact that answers the active CPU, retention, wait, contention, or timeline question instead of collecting everything. |
| `context.Canceled`, deadline, HTTP latency, DB/queue/pool wait, saturation, or retry amplification. | [context-timeout-and-saturation-debugging.md](context-timeout-and-saturation-debugging.md) | Attribute the budget to work, capacity wait, coordination wait, or retry amplification before changing policy. |
| Build, tags, toolchain, module/workspace, embed, generated files, or `GOOS`/`GOARCH`/CGO drift. | [build-and-generated-artifact-debugging.md](build-and-generated-artifact-debugging.md) | Prove selected sources and generator authority instead of editing runtime logic or derived output. |
| Root cause is proven and recurrence guardrails are being considered. | [defense-in-depth-go.md](defense-in-depth-go.md) | Add only the owning-layer guardrail justified by the failure rather than broad hardening or redesign. |
| A fix exists, temporary diagnostics remain, or completion is about to be claimed. | [fix-verification-and-scaffolding-cleanup.md](fix-verification-and-scaffolding-cleanup.md) | Match RED/GREEN proof to the defect and remove temporary scaffolding before reporting success. |
