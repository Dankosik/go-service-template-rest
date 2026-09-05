---
name: go-test-implementation
description: "Executable Go falsifiers. Use for a test-only change after the proof obligation, oracle, and proving layer are accepted, or when a non-routine fixture or harness must be built."
metadata:
  invocation: model
  kind: method
---

# Go Test Implementation

A test is an **executable falsifier**: it exists to reject the wrong behavior at the smallest deterministic layer, through an oracle independent of the code it judges.

`accepted obligation -> proving layer -> deterministic controls -> independent oracle -> executable proof -> disposition`

Routine focused tests required by a production unit remain with `go-coder`.

From every accepted proof obligation through its runnable command, build
`ExecutableProof{wrong_behavior, proving_layer, controls, fixture, oracle,
test, command, cleanup, result}`. Use the oracle as the anchor and choose the
smallest deterministic layer that rejects the wrong behavior. Source-string
presence substitutes for execution only when the exact text is itself the
accepted output contract. Load [the reference selector](references/index.md)
only when a concrete pressure changes the layer, controls, oracle, or command.

Complete when every accepted proof obligation has executable proof or a named
blocker, every proof carries an independent oracle, and the command actually
executes the named test. Return obligation dispositions, changed tests and
fixtures, commands, results, and gaps.
