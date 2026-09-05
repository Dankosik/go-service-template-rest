# Runtime Forensics For Go Incidents

## Load When

Load when a Go process/test is alive but stalled, growing, stuck in drain, or
panicking without enough stack evidence. Performance-profile selection belongs
to the benchmarking/performance owner. Load [goroutine leak
profiles](../../go-concurrency/references/goroutine-leak-profiles.md) for the
supported blocking model and lifecycle proof contract.

## Decide

Capture perishable evidence before process loss. `SIGQUIT` dumps goroutines and
terminates under default `GOTRACEBACK`; use it only when losing the process is
acceptable, otherwise use the gated diagnostics listener. Pprof ships disabled,
binds a broad diagnostics address when enabled, and exposes heap/cmdline data,
so enabling it is a deployment audience decision rather than an invisible
debugging step.

Read `/debug/buildinfo` with every capture because build VCS data is absent from
the runtime binary and rollout identity can differ by instance. Diagnostics stay
up through drain, making the shutdown window the useful capture period.

Use `goroutineleak` for high-confidence permanent blocking on supported
primitives and the ordinary goroutine profile for broad inventory, including
I/O waits. A non-empty leak profile is strong evidence, but its stack names the
blocking site rather than necessarily the lifecycle root cause. An empty leak
profile does not exclude I/O, syscall, busy-loop, custom-synchronization, or
formally reachable leaks. Requesting it runs a special GC analysis, so capture
it on demand after a liveness signal; never scrape it as a routine metric.

For a service capture:

```bash
base=http://127.0.0.1:9090
stamp=$(date -u +%Y%m%dT%H%M%SZ)
artifact_dir=".artifacts/runtime/$stamp"
mkdir -p "$artifact_dir"
curl -fsS "$base/debug/buildinfo" > "$artifact_dir/buildinfo.json"
curl -fsS "$base/debug/pprof/goroutineleak" > "$artifact_dir/goroutineleak.pb.gz"
curl -fsS "$base/debug/pprof/goroutineleak?debug=1" > "$artifact_dir/goroutineleak.txt"
curl -fsS "$base/debug/pprof/goroutine?debug=2" > "$artifact_dir/goroutines.txt"
go tool pprof -top "$artifact_dir/goroutineleak.pb.gz" > "$artifact_dir/goroutineleak.top.txt"
```

`/debug/buildinfo` is service-only. For worker binaries, store the equivalent
immutable deployment or image revision from the control plane beside the
capture instead.

Use one artifact per hypothesis: heap for retained allocation, mutex for holder
contention, block for waiter delay, and trace for scheduling. Growth requires
two time-separated samples under comparable load. CPU/trace captures longer
than the 65-second diagnostics write timeout are truncated even when the
artifact parses.

For a non-empty leak profile, trace from blocking site to the expected unblock
owner and the lost stop/join edge. If only goroutine count grows, compare the
ordinary profile under matched load and inspect unsupported waits and stale
registries. During shutdown, distinguish process-exit-only abandonment from a
reusable component that lost its join or is closing dependencies beneath live
work.

## Reject

Reject restart-before-capture and captures written beside packages. Store them
under `.artifacts/`, record timestamp/load/build/capture command, and restore the
pprof gate to its shipped state.

## Prove

Name the hypothesis each artifact can falsify, elapsed sample interval for
growth, target build, and whether diagnostics exposure changed and was closed.
