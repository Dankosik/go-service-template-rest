# Runtime Forensics For Go Incidents

## When To Load
Load this reference when a Go process or test is alive but not making progress, deadlocked, leaking goroutines, growing memory, stuck in shutdown, or producing a panic whose first stack is insufficient.

Use it before restarting or editing when the next action could destroy runtime evidence.

## Fast Artifact Map

| Symptom | Best first artifact | Why |
|---|---|---|
| hang, deadlock, stuck shutdown | goroutine dump | shows blocked send, receive, lock, wait, syscall, or shutdown drain |
| goroutine count grows | goroutine profile over time | distinguishes warmup from leaked owners |
| memory or RSS grows | heap profile over time | shows retained objects or allocation pressure |
| high CPU | CPU profile | shows active hot paths |
| low CPU but high latency | goroutine dump, block profile, trace | points at waiting, serialization, or scheduler behavior |
| lock or channel wait | block or mutex profile | shows wait sites and contention |
| scheduler or wakeup mystery | execution trace | shows timing and goroutine state transitions |

## Commands
If the service exposes `net/http/pprof`:

```bash
curl -o goroutine.txt 'http://127.0.0.1:6060/debug/pprof/goroutine?debug=2'
curl -o heap.pprof 'http://127.0.0.1:6060/debug/pprof/heap'
curl -o block.pprof 'http://127.0.0.1:6060/debug/pprof/block'
curl -o mutex.pprof 'http://127.0.0.1:6060/debug/pprof/mutex'
curl -o cpu.pprof 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
curl -o trace.out 'http://127.0.0.1:6060/debug/pprof/trace?seconds=5'
go tool pprof -top cpu.pprof
go tool pprof -top heap.pprof
go tool trace trace.out
```

For a local or test reproducer:

```bash
go test ./path/to/pkg -run '^TestName$' -count=1 -timeout=30s -v
go test ./path/to/pkg -run '^TestName$' -trace trace.out -count=1
go test ./path/to/pkg -run '^TestName$' -blockprofile block.out -blockprofilerate=1 -count=1
go test ./path/to/pkg -run '^TestName$' -mutexprofile mutex.out -mutexprofilefraction=1 -count=1
go test ./path/to/pkg -run '^TestName$' -cpuprofile cpu.out -memprofile mem.out -count=1
go tool pprof -top block.out
go tool pprof -top mutex.out
go tool pprof -top cpu.out
go tool pprof -top mem.out
go tool trace trace.out
```

For a stuck process, capture a dump before restart when operationally safe:

```bash
kill -QUIT <pid>
docker kill --signal=QUIT <container>
```

When panic output hides relevant goroutines:

```bash
GOTRACEBACK=all go test ./path/to/pkg -run '^TestName$' -count=1 -v
```

When scheduler or GC runtime events are relevant:

```bash
GODEBUG=schedtrace=1000,scheddetail=1 go test ./path/to/pkg -run '^TestName$' -count=1 -v
GODEBUG=gctrace=1 go test ./path/to/pkg -run '^TestName$' -count=1 -v
```

## Evidence To Capture
- timestamp, process identity, version or commit, and load condition
- exact capture command and artifact path
- two samples when growth or leak is suspected, with elapsed time between them
- top repeated goroutine stacks and blocked operation type
- profile top output plus the profile file when it must be revisited
- whether pprof endpoints were temporary or already protected operational endpoints

## Bad Debugging Moves
- restarting a hung process before capturing the goroutine state
- collecting CPU, heap, block, mutex, and trace at once and treating the combined picture as precise
- using CPU profiles to debug a process that is mostly waiting
- exposing pprof endpoints broadly or leaving temporary debug ports open
- assuming one heap snapshot proves a leak without a second point in time

## Good Debugging Moves
- pick the artifact that matches the wait or growth class
- keep tools isolated when one profiler can distort another
- compare goroutine or heap profiles across time for leak claims
- use execution trace when ordering and timing relationships matter more than aggregate samples
- remove or close any temporary runtime endpoint after the evidence is captured

## Source Links
- [Go diagnostics](https://go.dev/doc/diagnostics)
- [runtime/pprof package](https://pkg.go.dev/runtime/pprof)
- [net/http/pprof package](https://pkg.go.dev/net/http/pprof)
- [runtime/trace package](https://pkg.go.dev/runtime/trace)
- [cmd/trace tool](https://pkg.go.dev/cmd/trace)
- [Go blog: execution traces](https://go.dev/blog/execution-traces-2024)
