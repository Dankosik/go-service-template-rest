# Go And In-Process HTTP Benchmarks

Load for `testing.B`, in-process HTTP, profiling, or PGO measurement.

Put `BenchmarkXxx` beside the package and code it measures. Use `B.Loop` for
new benchmarks:

```go
func BenchmarkParse(b *testing.B) {
	input := []byte(`{"id":"example"}`)
	b.ReportAllocs()

	for b.Loop() {
		if _, err := Parse(input); err != nil {
			b.Fatal(err)
		}
	}
}
```

Setup before `B.Loop` is excluded from timing. Keep per-operation construction
inside the measured interval when it belongs to the production operation.
Reset mutable state deliberately instead of allowing later iterations to
measure different data.

For HTTP, select one of two boundaries:

- benchmark a handler, middleware chain, or `internal/infra/http.NewRouter`
  with `httptest.NewRequest` and `httptest.NewRecorder` when process scheduling,
  sockets, TLS, and client behavior are outside the claim;
- use `httptest.NewServer` and its real `http.Client` when the claim includes
  loopback transport and response-body handling but does not require an
  independently deployed service.

Build a fresh request and recorder whenever the handler consumes or mutates
them. If request/recorder creation is test-harness setup rather than the
operation, bracket only `ServeHTTP` with `b.StartTimer` and `b.StopTimer`.
Otherwise leave creation timed. Always validate the response status and the
smallest correctness invariant outside the timed interval.

Use `RunParallel` only when concurrent callers are the accepted workload. It
does not prove service capacity, production worker bounds, or safe lifecycle
behavior.

### Capture and compare

The default run matches every Go benchmark, records allocations, repeats each
case 10 times, and saves raw output and environment metadata:

```bash
make bench

make bench \
  BENCH_PACKAGE=./internal/orders \
  BENCH_PATTERN=BenchmarkCalculateTotal \
  BENCH_WORKLOAD_ID=orders-100-lines \
  BENCH_COUNT=20 \
  BENCH_TIME=2s
```

The command performs an untimed compile-only warm-up before capture and resolves
the pinned `benchstat` tool before writing a comparison. A first run may take
longer while Go downloads modules or the tool, but those messages do not enter
the raw benchmark or comparison artifacts. This does not warm application or
dependency state; declare and prepare that state as part of the workload.

Capture both sides on the same testbed:

```bash
make bench-baseline \
  BENCH_PACKAGE=./internal/orders \
  BENCH_PATTERN=BenchmarkCalculateTotal \
  BENCH_WORKLOAD_ID=orders-100-lines
# Apply the candidate change.
make bench \
  BENCH_PACKAGE=./internal/orders \
  BENCH_PATTERN=BenchmarkCalculateTotal \
  BENCH_WORKLOAD_ID=orders-100-lines
make bench-compare
```

`make bench-compare` uses the pinned Go `benchstat` tool and fails when the
recorded package, pattern, count, benchmark time, build tags, Go environment,
workload identity, dependency image, schema fingerprint, CPU identity/count, or
`GOMAXPROCS` differ. Artifacts live under
`.artifacts/bench/`:

- `baseline.txt` and `current.txt`: raw Go benchmark output;
- adjacent `.meta` files: automatic source identity, settings, and Go
  environment;
- `comparison.txt`: the `benchstat` report.

`BENCH_COUNT=10` is a comparison-ready default, not an acceptance policy.
Statistical significance also does not establish that a delta is materially
important; judge it against the accepted budget.

The simple baseline-then-candidate sequence is sufficient for stable, material
deltas. For small or noisy decision-critical deltas, avoid run-order bias:
prepare two clean worktrees or prebuilt binaries, alternate one baseline and
one candidate batch on the same idle testbed, retain every raw batch, and give
the aggregated baseline and candidate inputs to `benchstat`. Use at least 10
samples per side and prefer 20 when the decision is sensitive; do not combine
samples from different machines or environments.

## Diagnose a measured delta

Collect one symptom-matched Go diagnostic from one concrete package:

```bash
make bench-profile \
  BENCH_PACKAGE=./internal/orders \
  BENCH_PATTERN=BenchmarkCalculateTotal \
  BENCH_PROFILE=cpu
```

`BENCH_PROFILE` accepts `cpu`, `memory`, `block`, `mutex`, or `trace`. Profiles
and traces explain a measured delta; they are not pass/fail gates:

- CPU attributes active compute time;
- memory `alloc_space` exposes allocation churn, while `inuse_space` exposes
  retained live heap;
- block exposes time waiting on channels, synchronization, and blocking
  operations;
- mutex exposes lock contention;
- trace exposes scheduler, goroutine, GC, network, and syscall timing.

The profile command prints the matching inspection command, including both
memory views. Use `-benchmem` results for bytes and allocations per operation;
use production telemetry when the claim depends on real GC or heap behavior.
For database work, pair client profiles with pool statistics and a
representative query plan when the bottleneck may be server-side.

Do not expose `/debug/pprof` merely for this workflow. A runtime profiling
endpoint needs its own access-control and operational decision.

## Use a representative CPU profile for PGO

PGO is a candidate build input only after the CPU profile matches the service
and workload whose performance matters. A synthetic benchmark profile can
validate the build path and guide investigation, but it is not a production
profile.

Record the profile's service/source revision, workload identity, Go version,
collection interval, and SHA-256 beside the private delivery artifact. Build
locally with:

```bash
make build-pgo PGO_PROFILE=<representative-cpu.pprof>
```

For the production image, stage the private profile inside the build context
and pass the same repository-relative `PGO_PROFILE` build argument. Every
activated local or image build validates that explicit profile with
`go tool pprof` before compilation. Compare the PGO binary against
`PGO_PROFILE=off` under the same workload and environment, with independent
correctness proof. Refresh the profile after a material compiler, workload,
schema, or hot-path change; rebuild with `off` for rollback. Never commit a
generic synthetic `default.pgo`, because Go's automatic discovery can silently
make it an input to later builds.

## References

- Go [`testing` benchmarks and `B.Loop`](https://pkg.go.dev/testing#hdr-Benchmarks)
- Go [`benchstat`](https://pkg.go.dev/golang.org/x/perf/cmd/benchstat)
