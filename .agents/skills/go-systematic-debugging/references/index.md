# Reference Selector

Each row names a pressure where this repository's own wiring overrides the
obvious answer. State the expected behavior change before loading; load one.

| Symptom | Load | Behavior change |
| --- | --- | --- |
| A live process is stalled, deadlocked, leaking goroutines or memory, or stuck in shutdown, and a restart or a signal is being considered. | [runtime-forensics-go.md](runtime-forensics-go.md) | Capture through the diagnostics listener this service already ships, in the order that keeps perishable state — instead of standing up a temporary pprof endpoint or killing the process with `SIGQUIT`. |
| A Go test fails only under repetition, on CI, under `-race`, `-shuffle`, or a specific `-cpu`. | [flaky-repro-controls-go.md](flaky-repro-controls-go.md) | Move one control variable at a time and replay the recorded seed — instead of one combined stress command whose pass proves nothing about which knob mattered. |

Everything else in debugging has an owner, and none of it has a reference here:

- **Boundary-crossing symptoms, hop localization, and cause-versus-victim** —
  `SKILL.md` states the both-sides rule and names the neighbor inventory;
  [`production-diagnosis`](../../../../docs/universal-disciplines/production-diagnosis/SKILL.md)
  owns the method.
- **Backtracking to the first broken invariant** — `SKILL.md`'s own thesis. Go
  semantics along the way (typed nil in an interface, `errors.Is`/`As`, context
  errors) belong to `go-idiomatic`.
- **Waiting on a condition instead of sleeping** — `go-test-implementation`, via
  `testing/synctest`.
- **Which profile answers a measured delta** — `docs/benchmarking.md` and
  `go-performance`.
- **Build tags, toolchain, and generated-artifact authority** — `go-coder`'s
  generated source-of-truth table, which names the canonical input and its
  `*-generate` / `*-check` pair.
- **Proof that matches the claim, and reporting the gap** —
  `go-verification-before-completion`.
- **Timeout, retry, and degradation policy** — `go-reliability`.

Adding a reference back requires a decision it would change that its owner above
does not already make.
