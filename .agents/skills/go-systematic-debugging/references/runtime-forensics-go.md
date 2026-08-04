# Runtime Forensics For Go Incidents

## Behavior Change Thesis

When loaded for a live Go process that is stalled, leaking, or stuck in
shutdown, this file makes the model capture the perishable artifact through the
listener this service already ships — instead of proposing a temporary
`net/http/pprof` endpoint, sending a signal that kills the process it was about
to inspect, or restarting first and reading stacks that no longer exist.

## When To Load

Load when a Go process or test is alive but not progressing: deadlock, goroutine
or memory growth, a stuck drain, or a panic whose first stack is insufficient.

Profile *fit* for a measured performance delta is not this file.
`docs/benchmarking.md` and `go-performance` own which artifact answers which
question — including the mutex-holder versus block-waiter distinction, and the
`make bench-profile` route that needs no listener at all.

## Decision Rubric

Capture order is the whole discipline: what dies with the process goes first.
[`production-diagnosis`](../../../../docs/universal-disciplines/production-diagnosis/SKILL.md)
owns when to stop capturing and mitigate.

- **`SIGQUIT` terminates the process.** Go's default `GOTRACEBACK` dumps every
  goroutine stack and then exits, so it is a capture *and* an outage. Use it only
  when losing the process is acceptable or custom signal handling is confirmed;
  otherwise read the dump over HTTP. On a test binary, `GOTRACEBACK=all` widens a
  panic's stack set without killing anything extra.

- **The profile handlers exist here, but ship off.**
  `observability.pprof.enabled` defaults to `false` (`internal/config/types.go`),
  and validation rejects enabling it with no `observability.metrics.addr` —
  the diagnostics listener binds every interface so Prometheus can reach it.
  `/debug/pprof/heap` discloses heap contents and `/debug/pprof/cmdline`
  discloses process arguments, which makes the gate
  (`APP__OBSERVABILITY__PPROF__ENABLED`) a deployment decision about audience
  rather than a debugging step to take unilaterally. Standing up a second,
  temporary endpoint instead is the same disclosure with none of the review.

- **`GET /debug/buildinfo` is ungated and answers "which build is this?"**
  Nothing else in the process can: the image builds `-buildvcs=false` with `.git`
  outside the context, so `runtime/debug.ReadBuildInfo` carries no revision, and
  the OCI label is unreachable from inside the container. Read it alongside every
  artifact — mid-rollout, "which of these two pods" is the question.

- **A stuck shutdown is still reachable.** The diagnostics listener is
  deliberately outside the drain
  (`cmd/service/internal/bootstrap/startup_server.go`) and closes only after
  `drainAndShutdown` returns, so `/metrics` and — when gated on — the profile
  handlers stay up for the entire drain window, including the in-flight requests
  hanging it. That window is the evidence; do not assume it is already gone.

- **One artifact per hypothesis; two time-separated samples for any growth
  claim.** `/debug/pprof/goroutine?debug=2` answers "who is blocked right now"; a
  single heap or goroutine snapshot cannot separate a leak from warm-up.

- **A capture longer than 65s is truncated.** `pprofWriteTimeout` raises the
  diagnostics write timeout to 65s precisely so a default 30s `?seconds=`
  completes. A larger `?seconds=` on `/debug/pprof/profile` or `/debug/pprof/trace`
  outlives the write timeout and returns a truncated artifact that still parses
  like a whole one.

## Reject

Reject restarting and then investigating. The blocked stacks that name the owner
of a deadlock cycle exist only in the live process, and a restart converts a
solvable incident into an unreproducible one.

Reject leaving captures where git will find them. `.gitignore` covers
`.artifacts/test/`, `.artifacts/bench/`, and `coverage.out` — a `.pprof`, a
`trace.out`, or a goroutine dump written beside the package under investigation
is not ignored. Write captures under `.artifacts/`, and return the pprof gate to
its shipped `false` when the investigation ends.

## Validation Shape

Record build identity from `/debug/buildinfo`, timestamp and load condition, the
exact capture command, the elapsed time between samples for any growth claim, and
whether the pprof gate was already enabled or was turned on for this
investigation and turned back off.
