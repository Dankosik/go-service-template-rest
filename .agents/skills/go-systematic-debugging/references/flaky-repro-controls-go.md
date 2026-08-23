# Flaky Reproduction Controls For Go

## Load When

Load when a Go test fails only under repetition, on CI, under `-race`,
`-shuffle`, a specific `-cpu`, or in wider package scope.

A test that is flaky because it sleeps on wall-clock time is a test-design
problem, not a reproduction problem: `go-test-implementation` owns
`testing/synctest`, which this repository already uses for time-driven behavior
(`internal/health/cached_test.go`, `internal/background`, the bootstrap
lifecycle tests). Reach for a fake clock there before widening any timeout here.

## Decide

- **Read the recorded failure before reproducing it.** CI and `gotestsum`
  preserve per-test output and ordering, which usually names the failure class
  before a single command is rerun.

- **Move one variable per command**, and name the class before combining: `-count`
  for repetition, `-race` for shared state, `-cpu` for scheduler sensitivity,
  `-shuffle` for order dependence. A combined stress shape is a later step for a
  hypothesis that needs it, not the opening move.

- **`-shuffle` only means something when several tests remain in scope.** Under a
  narrow `-run`, it shuffles nothing worth shuffling. When it fails, the seed it
  prints is the reproducer — replay `-shuffle=<seed> -count=1` before editing
  anything, because the next `-shuffle=on` run will not be the same order.

- **Repeated test commands take `-vet=off`** here: mandatory lint owns `govet` for
  the current tree, so leaving default vet on re-lints the package on every one of
  100 iterations. The repository's own flake gate is
  a bounded `go test -vet=off -count=5 -shuffle=on <scope>` run; the
  race gate is `ALLOW_HEAVY=1 make test-race`.

- **Integration-tagged flakes have pinned commands.** `make test-messaging-race`
  and `make test-outbox-race` run `-p=1 -count=1 -race -tags=integration` over a
  fixed `-run` set. If the flaky test is in that set, reproduce with that target's
  shape — `-p=1` is load-bearing, because those packages share a database and a
  broker.

- **Record frequency as data** — `7/100`, with the command that produced it.
  "Sometimes" cannot be compared against the same number after the fix, which is
  the only evidence that a probabilistic failure got rarer rather than luckier.

## Reject

Reject a single green local run as evidence against a CI-only flake, and reject
the same run count after the fix as evidence it is gone: a 1-in-20 failure passes
19 consecutive runs routinely. Match the repetition to the observed rate, or state
the residual risk in those terms.

Reject skipping or deleting the test before deciding whether the flake is
test-only or a real production race that the test merely exposes intermittently.

## Prove

Capture the exact command, package and test selector, `-count`, `-race`, `-cpu`,
`-shuffle` seed, tags, relevant env, the first distinct failing stack or
assertion rather than the run summary, and the failure frequency. After the fix,
rerun that same command shape and report the new frequency against the old one.
