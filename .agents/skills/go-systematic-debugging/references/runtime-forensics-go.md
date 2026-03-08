# Runtime Forensics For Go Incidents

## When To Use
Use this reference when the dominant symptom is one of these:
- process is alive but not making progress
- goroutine count, memory, or RSS keeps climbing
- CPU is unexpectedly high or unexpectedly low during a stall
- lock contention, queue buildup, or scheduler behavior is the real unknown

Do not collect every artifact by default. Match the artifact to the symptom.

## Fast Artifact Map

| Symptom | Best first artifact | Why |
|---|---|---|
| Hang or deadlock suspicion | goroutine dump | shows who is blocked on send, receive, lock, wait, or syscall |
| Lock or queue wait | block or mutex profile | shows where waiting time accumulates |
| Runnable goroutine bursts, wakeups, scheduler oddities | `go tool trace` | shows scheduling, network, timers, and goroutine state transitions |
| Memory or goroutine growth | heap profile and goroutine profile | distinguishes one-time growth from leak-like monotonic growth |
| High CPU | CPU profile | shows hot functions instead of guessing |

## Useful Commands

If the service exposes `pprof`:

```bash
go tool pprof http://127.0.0.1:6060/debug/pprof/goroutine
go tool pprof http://127.0.0.1:6060/debug/pprof/heap
go tool pprof http://127.0.0.1:6060/debug/pprof/block
go tool pprof http://127.0.0.1:6060/debug/pprof/mutex
go tool pprof http://127.0.0.1:6060/debug/pprof/profile?seconds=20
```

For a local or test reproducer:

```bash
go test ./path/to/pkg -run '^TestName$' -trace trace.out
go test ./path/to/pkg -run '^TestName$' -blockprofile block.out -mutexprofile mutex.out
go test ./path/to/pkg -run '^TestName$' -cpuprofile cpu.out -memprofile mem.out
go tool trace trace.out
go tool pprof -top block.out
go tool pprof -top mutex.out
go tool pprof -top cpu.out
go tool pprof -top mem.out
```

For a stuck process, capture a goroutine dump before restart when operationally safe:

```bash
kill -QUIT <pid>
```

Containerized equivalent:

```bash
docker kill --signal=QUIT <container>
```

If panic output is too shallow or only the current goroutine is visible, retry with:

```bash
GOTRACEBACK=all
```

## What To Look For

### Goroutine Dump
- many goroutines blocked on the same channel send or receive
- shutdown goroutines waiting on `WaitGroup` or drain paths that can never complete
- handlers blocked in DB, network, or lock acquisition rather than CPU work
- repeated stacks that point to one owner cycle

### Block Or Mutex Profile
- one lock or channel wait site dominating wait time
- lock-held-across-I/O patterns
- queue wait accumulating at a stage that should have been bounded

### Trace
- long runnable queues with little forward progress
- timer-driven wakeup storms
- unexpected serialization across what should be parallel work
- bursts of goroutine creation without matching completion

### Heap Or Goroutine Evidence
- monotonic growth after steady-state traffic
- large retained structures or unbounded buffers
- worker or ticker goroutines that never exit after request completion or shutdown

## Safety Notes
- Capture the smallest artifact that answers the current question.
- Avoid CPU profiling an already overloaded production path unless the cost is acceptable.
- Prefer dump-first before restart for hangs; otherwise the evidence is gone.
- Do not keep `pprof` endpoints or verbose diagnostics exposed longer than needed.
