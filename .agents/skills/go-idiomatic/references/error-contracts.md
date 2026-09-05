# Error Contracts

## Load When

Load when code creates, wraps, joins, logs, classifies, or recovers from an
error; defines a custom type; or attaches a cancellation cause.

## Decide

- Give an error one handling owner. Handle it where the layer completes a
  recovery, fallback, retry, translation, or terminal outcome; otherwise add
  safe operation context and return it. Logging and returning the same error is
  not handling: the terminal boundary owns the actionable record.
- Keep operation names static and free of runtime data. Use `failure.Op` when an
  unclassified boundary must retain the name in its safe class chain.
- `%w` publishes the cause's identity as package API. Use it only when callers
  may inspect that cause. Translate an implementation-detail dependency failure
  into a package-owned sentinel or type. `%v` is opaque only when the dependency
  text is safe there; do not use `errors.New(err.Error())` merely to hide identity.
- Use `errors.Is` for identity or declared equivalence and
  `errors.AsType[E]` for a type and its fields. Both traverse wrapped and joined
  error trees. Never classify by `Error()` text or compare a possibly wrapped
  sentinel with `==`.
- Use a custom error type only for caller-visible fields or behavior.
  Add `Unwrap` only when the cause is intentionally public.
- Use `errors.Join` for independently meaningful failures, not an ordinary
  causal chain. A match on one branch says nothing about its siblings.
- Preserve `context.Canceled` and `context.DeadlineExceeded` for owners that
  branch on them, but joining `cleanupErr` is not cancellation-only. Use
  `context.WithCancelCause` only when an observer reads the cause.
- Panic is not error propagation. Recover only at an isolation boundary that
  owns continuation, safe output, stack capture, and terminal reporting.
- Feature code returns feature errors. `failure.Mapper` performs
  transport-neutral classification; mapper order is first-match policy. HTTP
  and gRPC boundaries own their projections, and an unpublished code fails
  closed.

## Inspect

`fmt.Errorf("load user: %w", pgx.ErrNoRows)` from an exported adapter publishes
the driver's sentinel as package API; wrapping is not merely a style choice.

## Reject

- "Use `%w` because it is more idiomatic": wrapping is an API decision.
- "Log it here and return it": that duplicates the boundary record and may
  expose a different representation of the same cause.
- "Make this a custom error type": an opaque contextual error is smaller when
  no caller needs fields or behavior.
- "Retry because `net.Error.Temporary()` is true": temporary is deprecated and
  says nothing about repeatability, effect ambiguity, or the retry budget.

## Reopen

- Name the caller policy: distinguish, translate, redact, release, retry, or
  terminate.
- Route HTTP output to `go-api-contract`, gRPC status to `go-grpc`, disclosure to
  `go-observability`, retry to `go-reliability`, and cleanup to
  [lifetime-and-release.md](lifetime-and-release.md).

## Prove

- Assert the exported contract with `errors.Is` or `errors.AsType`, never a
  message string; add a negative assertion when a cause must stay opaque.
- For a join, prove every independently meaningful branch remains observable
  and that a single match cannot hide a sibling failure.
- Drive recovery through the isolation boundary and assert safe output and the
  terminal record.
