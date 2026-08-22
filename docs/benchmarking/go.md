# Go And In-Process HTTP Benchmarks

Put `BenchmarkXxx` beside the package it measures and use `B.Loop`. Validate the
smallest correctness invariant outside the timed interval.

Capture comparable samples directly:

```bash
go test -run '^$' -bench BenchmarkCalculateTotal -benchmem -count 10 \
  ./internal/orders > .artifacts/bench/baseline.txt

# Apply the candidate, then run the same command to current.txt.
go test -run '^$' -bench BenchmarkCalculateTotal -benchmem -count 10 \
  ./internal/orders > .artifacts/bench/current.txt

go tool -modfile=tools/go.mod benchstat \
  .artifacts/bench/baseline.txt .artifacts/bench/current.txt
```

Keep setup outside `B.Loop` only when production does not pay for it. Use
`RunParallel` only for an accepted concurrent workload. Do not combine samples
from different machines or materially different environments.

For attribution, use the matching native profile:

```bash
go test -run '^$' -bench BenchmarkCalculateTotal -cpuprofile cpu.pprof ./internal/orders
go tool pprof -top cpu.pprof
```

Use `-memprofile`, `-blockprofile`, `-mutexprofile`, or `-trace` only for the
observed symptom. Profiles explain a measured delta; they are not pass/fail
gates.

PGO accepts only a representative CPU profile with recorded source, workload,
Go version, interval, and digest. Build with:

```bash
make build-pgo PGO_PROFILE=<representative-cpu.pprof>
```

Compare the PGO binary with `PGO_PROFILE=off` under the same workload. Never
commit a generic `default.pgo`.
