# Deterministic Time And Concurrency

## Load When
Symptom: the behavior under test is a schedule, a timeout, a backoff, a
cancellation, a shutdown ordering, or a handoff between goroutines.

## Decide
- `synctest.Test(t, func(t *testing.T) { ... })` is the entry point in
  `testing/synctest` on this toolchain (Go 1.27.0; GA since 1.25). The Go 1.24
  spelling — `synctest.Run` behind `GOEXPERIMENT=synctest` — no longer applies,
  and the current form is already used in
  `internal/infra/postgresoutbox/relay_test.go` and eleven other files.
- Inside a bubble `time.Sleep` is how the fake clock advances, not something to
  avoid. Time moves only when every goroutine in the bubble is durably blocked,
  so a sleep asks the clock to jump to the next scheduled wakeup. The identical
  call outside a bubble is the guess this file exists to replace.
- Put `synctest.Wait` between advancing time and asserting. It returns once every
  other bubble goroutine is durably blocked, which turns "the timer has fired by
  now" into an observation instead of a race.
- A goroutine parked on a channel created before the bubble, a real socket, or a
  syscall is not durably blocked, and one such goroutine stops the clock for the
  whole bubble; the current `testing/synctest` doc owns the full durably-blocked
  taxonomy.
- Read the two stalls differently. All goroutines durably blocked with no wakeup
  left is a deadlock, which `Test` panics on and names. One goroutine blocked
  outside the bubble is a hang: nothing panics until `go test -timeout` fires and
  reports `test timed out`, naming neither synctest nor the culprit. In that dump
  the offender is the bubble frame missing the `(durable)` marker —
  `[chan receive, synctest bubble 1]` beside healthy neighbours reading
  `[chan receive (durable), synctest bubble 1]`.
- Goroutine exit needs no separate assertion inside a bubble: `Test` waits for
  every bubble goroutine. Package-wide that job belongs to
  `goleak.VerifyTestMain(m)` in `TestMain`, already run in seven packages here.
- When the code genuinely leaves the bubble — a container, a subprocess, a real
  socket — drop synctest for an explicit ready signal and a buffered result
  channel, keeping a real timeout only as a failure diagnostic.

## Reject
- Cancel, sleep, assert nothing: it passes whether the goroutine returned, leaked,
  or panicked.
- `-count=N` repetition offered as determinism. Repetition shifts the odds of
  catching a race; it does not make an ordering observable.
- A clean `-race` run read as proof of liveness, ordering, or cancellation. It
  reports conflicting access on the paths that executed, and nothing else.

## Prove
- Focused test with `-count=1 -vet=off`.
- `ALLOW_HEAVY=1 make test-race` when the changed path shares state across goroutines.
