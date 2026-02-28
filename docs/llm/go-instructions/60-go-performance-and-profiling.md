# Go performance and profiling instructions for LLMs

## Load policy
- Load: Optional
- Use when:
  - Optimizing latency, throughput, CPU, memory, allocation, lock contention, or scheduler behavior
  - Writing benchmarks, profiles, traces, or performance investigations
  - Deciding whether to use PGO
- Do not load when: The user is not asking about performance, profiling, tracing, or optimization

## Performance principles

- Do not optimize by guesswork.
- Measure first, then optimize the proven bottleneck.
- Keep readable code unless measurement demonstrates a worthwhile gain from a more complex version.
- Prefer algorithmic, data-flow, and allocation-structure improvements over tiny syntax tricks.

## Investigation workflow

Follow this order unless the task strongly suggests otherwise:
1. Confirm the symptom and the metric that matters.
2. Reproduce the issue with a benchmark, profile, trace, or realistic workload.
3. Identify the hot path or contention point.
4. Change the smallest thing that plausibly improves the bottleneck.
5. Measure again.
6. Keep the simpler version unless the improvement is real and meaningful.

## Benchmark guidance

- Use `go test -bench` for focused performance checks.
- Make benchmark inputs realistic enough to exercise meaningful behavior.
- Keep setup outside the timed loop when practical.
- Consider allocation reporting when allocation count matters.
- Interpret microbenchmarks carefully; they do not replace whole-system profiling.

## Profiling guidance

- Use `pprof` to investigate CPU, heap, allocs, goroutine, block, or mutex profiles.
- For long-running services, `net/http/pprof` is a practical way to expose profiling endpoints when appropriate.
- Use profiles to find where time or memory actually goes before rewriting code.
- Do not assume the slow-looking function is the real bottleneck without measurement.

## Trace guidance

- Use `go tool trace` when scheduler behavior, blocking, wakeups, goroutine interactions, or latency spikes matter.
- Prefer tracing over guesswork for complicated concurrency problems.
- Combine trace findings with code-level reasoning about cancellation, backpressure, and synchronization.

## Concurrency and allocation considerations

- If performance issues involve goroutines or locking, pair this file with the concurrency file.
- Reduce unnecessary allocations in hot loops only when a profile shows they matter.
- Be careful with pooling and reuse. Complexity and lifetime bugs can outweigh minor wins.
- Avoid premature `sync.Pool` usage unless profiling shows it helps.

## PGO guidance

- Consider profile-guided optimization for release builds only after collecting representative CPU profiles.
- Use PGO as a measured optimization step, not as a substitute for algorithmic fixes.
- Do not assume PGO will rescue a poor design.

## Common anti-patterns to avoid

- Rewriting code for performance with no benchmark or profile
- Interpreting one microbenchmark as proof of end-to-end improvement
- Trading away clarity for tiny, unverified wins
- Overusing pooling, buffering, or manual reuse in non-hot code
- Ignoring blocking, contention, or scheduler effects in concurrent systems
- Optimizing allocation count when the true bottleneck is I/O, locking, or algorithmic complexity

## What good output looks like

- The optimization target is explicit.
- Measurement drives the proposed changes.
- The code remains maintainable after the improvement.
- Benchmarks, profiles, or traces are part of the reasoning.
- More invasive optimizations are reserved for confirmed hot paths.

## Checklist

Before finalizing, verify that:
- A real metric or symptom is identified.
- A benchmark, profile, or trace supports the change.
- The proposed optimization targets the measured bottleneck.
- Complexity was not added for an unproven gain.
- PGO is suggested only when the workload and build process justify it.
