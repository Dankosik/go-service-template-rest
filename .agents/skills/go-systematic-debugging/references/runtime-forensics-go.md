# Runtime Forensics For Go Incidents

## Load When

Load when a Go process/test is alive but stalled, growing, stuck in drain, or
panicking without enough stack evidence. Performance-profile selection belongs
to the benchmarking/performance owner.

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

Use one artifact per hypothesis: goroutine dump for current blocking, heap for
retained allocation, mutex for holder contention, block for waiter delay, trace
for scheduling. Growth requires two time-separated samples under comparable
load. CPU/trace captures longer than the 65-second diagnostics write timeout are
truncated even when the artifact parses.

## Reject

Reject restart-before-capture and captures written beside packages. Store them
under `.artifacts/`, record timestamp/load/build/capture command, and restore the
pprof gate to its shipped state.

## Prove

Name the hypothesis each artifact can falsify, elapsed sample interval for
growth, target build, and whether diagnostics exposure changed and was closed.
