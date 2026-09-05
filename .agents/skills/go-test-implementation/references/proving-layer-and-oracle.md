# Proving Layer And Oracle

## Load When
Symptom: an approved requirement, invariant, review finding, or bug report has to
become named Go tests, or the layer that should carry the proof is unsettled.

## Decide
- Name the observable before the package: returned value, persisted row, response
  status and body, count of effects performed, wrapped error category, or a
  goroutine that exited. The observable picks the layer, never the reverse.
- Two questions disqualify an oracle. Could this assertion have been written from
  the approved behavior alone, without reading the implementation? If not it is
  derived, and it restates the code instead of judging it. Could a plausible
  regression in this scenario's named obligation pass the oracle? If yes,
  strengthen the discriminator or report the narrower evidence without closing
  the unmet obligation. Broad assertions such as
  `err != nil`, `code >= 400`, or a non-empty slice are insufficient when the
  contract requires a more specific outcome.
- Prove duplicate-request semantics through the identity returned and the number
  of effects performed, not the reservation key, fingerprint, or row the current
  code happens to use.
- Put the proof where the behavior is owned. What an engine produces — lock
  exclusion, `ORDER BY`, constraint violation, mux routing — is invisible through
  a fake of that engine, so the test passes having proved nothing; see
  [postgres-integration-proof.md](postgres-integration-proof.md). What Go code
  owns — mapping, wrapping, validation, suppression — needs no container, and
  paying for one buys latency rather than evidence.
- Reach for a fuzz target when the oracle is an invariant over generated input —
  a parser, decoder, or validator — rather than over cases you can name, as
  `FuzzParseDuration` and `FuzzStrictToken` already do here.
- Missing exactness is an escalation, not a blank to fill. A status code, cache
  key, tenant rule, or retry policy the approved behavior never named stays
  unasserted; inventing one freezes a decision nobody made.
- Isolation for `t.Parallel()` is the judgment no gate makes. The mechanical
  cases fail loudly — `t.Setenv` panics in a parallel test — while a package-level
  fake, a shared fixture, or a fixed port fails quietly and only under load.

## Reject
- Whole-struct equality where a few fields are contractual: it breaks on
  timestamps and added fields, and names none of them as the reason.
- A name like `TestCreate` over one stating condition and outcome: the failure
  line then reports which call broke, not which behavior regressed.

## Prove
- Run the named test with `-count=1 -vet=off`, matching the repository's own test
  commands and leaving vet to `make lint-package` or `make lint-all`.
- Add the package command when helpers, fixtures, or shared setup changed.
- A plain `go test` run over a fuzz target executes the seed corpus only, so it
  proves the seeds and not exploration; a focused `go test -fuzz` is the bounded run
  that explores.
- Report the observable each test rejects. A count of added tests is not evidence.
