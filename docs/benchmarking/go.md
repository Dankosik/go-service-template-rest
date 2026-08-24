# Go And In-Process HTTP Benchmarks

Put `BenchmarkXxx` beside the package it measures and use `B.Loop`. Validate the
smallest correctness invariant outside the timed interval.

Name the accepted budget and the person or team that must respond before
capture; do not use placeholder values. The repository's runnable example is
the request access-log hot path:

```bash
export BENCH_BUDGET='<accepted unit, percentile or capacity budget>'
export BENCH_RESPONSE_OWNER='<response owner>'

BENCH_PACKAGE=./internal/infra/http \
BENCH_PATTERN='^BenchmarkAccessLog$' \
BENCH_WORKLOAD_ID=access-log-level-paths-v1 \
BENCH_OUTPUT=.artifacts/bench/baseline.txt \
make benchmark-capture

# Apply the candidate, then capture the same workload.
BENCH_PACKAGE=./internal/infra/http \
BENCH_PATTERN='^BenchmarkAccessLog$' \
BENCH_WORKLOAD_ID=access-log-level-paths-v1 \
BENCH_OUTPUT=.artifacts/bench/current.txt \
make benchmark-capture

make benchmark-compare
```

Keep setup outside `B.Loop` only when production does not pay for it. Use
`RunParallel` only for an accepted concurrent workload. Do not combine samples
from different machines or materially different environments; the comparison
target rejects those mismatches before reporting a delta.

For attribution, use the matching native profile:

```bash
go test -run '^$' -bench '^BenchmarkAccessLog$' -cpuprofile cpu.pprof ./internal/infra/http
go tool pprof -top cpu.pprof
```

Use `-memprofile`, `-blockprofile`, `-mutexprofile`, or `-trace` only for the
observed symptom. Profiles explain a measured delta; they are not pass/fail
gates.

PGO accepts only a representative CPU profile with recorded source, workload,
Go version, interval, digest, response owner, and candidate fingerprint. The
profile and manifest are ignored by Git. Create the manifest only after the
owner has accepted the profile for the current source tree:

```bash
PGO_PROFILE=<representative-cpu.pprof> \
PGO_PROFILE_SOURCE_REVISION=<40-hex-profile-build-commit> \
PGO_WORKLOAD_ID=<accepted-workload-id> \
PGO_RESPONSE_OWNER=<response-owner> \
PGO_PROFILE_GO_VERSION=<profile-go-version> \
PGO_PROFILE_GOOS=<profile-goos> \
PGO_PROFILE_GOARCH=<profile-goarch> \
PGO_CAPTURE_INTERVAL=<profile-interval> \
PGO_CAPTURED_AT_UTC=<profile-capture-RFC3339-time> \
make pgo-manifest

make build
make build-pgo \
  PGO_PROFILE=<representative-cpu.pprof> \
  PGO_MANIFEST=<representative-cpu.pprof.meta>
```

The off binary is `bin/service`; the PGO binary is `bin/service-pgo`. Compare
them under the same workload and independent correctness proof. Any change in
the service build inputs invalidates the manifest until the response owner
accepts profile fit again; unrelated docs and tests do not. A target, Go
toolchain, workload, schema, or material hot-path change requires a fresh
profile, not just a reissued manifest. Canonical CI/CD images remain
`PGO_PROFILE=off` until a production profile-supply and rollout owner is
accepted. Never commit a generic `default.pgo`.
