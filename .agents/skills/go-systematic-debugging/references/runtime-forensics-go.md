# Runtime Forensics For Go Incidents

## Behavior Change Thesis
When loaded for a live stalled or leaking Go process, this file makes the model capture the most perishable runtime artifact before restart or edits instead of destroying evidence.

## When To Load
Load when a Go process or test is alive but not making progress, deadlocked, leaking goroutines, growing memory, stuck in shutdown, or producing a panic whose first stack is insufficient.

## Decision Rubric
- Capture volatile evidence before restart when operationally safe.
- Pick the first artifact that matches the wait or growth class; do not collect every profile by habit.
- Use two time-separated samples when claiming goroutine, heap, or RSS growth.
- Prefer goroutine dumps for "who is blocked now"; use block/mutex profiles for accumulated wait and contention.
- Prefer local or protected pprof when the process must stay alive; use `SIGQUIT` only when its default exit-with-stack-dump behavior is acceptable or custom signal handling is confirmed.
- Use execution trace when ordering and timing relationships matter more than aggregate samples.
- Remove or close temporary runtime endpoints after capture.

## Fast Artifact Map

| Symptom | Best first artifact | Why |
|---|---|---|
| hang, deadlock, stuck shutdown | goroutine dump | shows blocked send, receive, lock, wait, syscall, or shutdown drain |
| goroutine count grows | goroutine profile over time | distinguishes warmup from leaked owners |
| memory or RSS grows | heap profile over time | shows retained objects or allocation pressure |
| high CPU | CPU profile | shows active hot paths |
| low CPU but high latency | goroutine dump, block profile, or trace | points at waiting, serialization, or scheduler behavior |
| lock or channel wait | block or mutex profile | shows wait sites and contention |
| scheduler or wakeup mystery | execution trace | shows timing and goroutine state transitions |

## Imitate

```bash
kill -QUIT <pid>
docker kill --signal=QUIT <container>
```

Use this only when terminating the process is acceptable, or when you have confirmed custom signal handling. By default, Go exits with a stack dump on `SIGQUIT`.

```bash
curl -o goroutine-1.txt 'http://127.0.0.1:6060/debug/pprof/goroutine?debug=2'
sleep 30
curl -o goroutine-2.txt 'http://127.0.0.1:6060/debug/pprof/goroutine?debug=2'
```

Use two samples when the claim is "goroutines are leaking" rather than "goroutines are currently blocked."

```bash
GOTRACEBACK=all go test ./path/to/pkg -run '^TestName$' -count=1 -v
```

Use this when panic output hides relevant goroutines.

## Reject

```bash
curl -o cpu.pprof .../profile
curl -o heap.pprof .../heap
curl -o block.pprof .../block
curl -o mutex.pprof .../mutex
curl -o trace.out .../trace
```

This over-collects, adds overhead, and can blur which artifact proved which hypothesis.

```text
Restarted the stuck service, then started investigating the deadlock.
```

This destroys the blocked goroutine state that would have shown the owner cycle.

## Agent Traps
- Using CPU profiles to debug mostly waiting processes.
- Assuming one heap snapshot proves a leak.
- Ignoring process identity, commit/version, timestamp, and load condition.
- Leaving temporary pprof endpoints reachable after the investigation.
- Checking in `*.pprof`, `trace.out`, or dumps by accident.

## Validation Shape
Record timestamp, process identity, version or commit, load condition, exact capture command, artifact path, repeated blocked stack or profile top summary, elapsed time between samples for growth claims, whether signal capture was expected to terminate the process, and whether runtime endpoints were already protected or temporary.
