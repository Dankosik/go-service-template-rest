# Resource Closure And Iteration Probes

## Behavior Change Thesis
When loaded for resource cleanup or iteration-probe symptoms, this file makes the model require completion probes and correct release lifetime instead of likely mistake "Close exists, so it is fine" or "just add `defer` wherever the resource is opened."

## When To Load
Load when a Go review touches `Body.Close`, `rows.Close`, `rows.Err`, `scanner.Err`, files, timers, tickers, cancel funcs, partial `Read`, loops that open resources, or `defer` placement that changes resource lifetime.

## Decision Rubric
- Check both release and completion. Some APIs require a close/release call and a separate post-iteration error probe.
- For `database/sql.Rows`, closing is not enough; after iteration, check `rows.Err()` before treating the result as complete.
- For `bufio.Scanner`, check `scanner.Err()` after the scan loop unless the code intentionally accepts truncated/partial input and documents why.
- For `http.Response.Body`, ensure the body is closed on all paths after a successful response; when HTTP/1.x connection reuse matters, check whether the body is read to completion before close or intentionally closed early to avoid draining a large or untrusted body. Do not hide a prior read/status error behind a later close error unless the close error matters for writes or protocols.
- Avoid `defer` inside long or unbounded loops when it accumulates open bodies/files/rows until function return. Close within the iteration or move the work to a helper with a bounded scope.
- For timers and tickers, check the effective Go version and reason: Go 1.23+ no longer needs `Stop` only for GC recovery, but `Stop` can still be required to prevent future events, preserve older-version behavior, or make `Reset` or shutdown semantics correct.
- For `io.Reader.Read`, handle `n > 0` before or along with `err`; do not discard bytes just because `err == io.EOF` or another terminal error arrived.
- Keep DB query semantics, retry budgets, and goroutine shutdown in their specialist lanes; this reference owns Go/stdlib contract shape.

## Imitate
```text
[high] [go-idiomatic-review] internal/report/export.go:88
Issue: The rows loop returns the collected reports without checking rows.Err().
Impact: A mid-stream database scan error can be reported as a successful partial export, so callers may persist incomplete data.
Suggested fix: After the loop, check rows.Err() and return that error before returning the collected reports.
Reference: rows iteration completion contract
```

Copy the completion-probe proof: the risk is hidden partial success after iteration, not just "missing error check".

```text
[medium] [go-idiomatic-review] internal/importer/read.go:47
Issue: The scanner loop ignores scanner.Err() after Scan returns false.
Impact: Oversized tokens or read failures can terminate scanning early while the importer treats the truncated input as complete.
Suggested fix: Check scanner.Err() after the loop and return it with context before committing the parsed result.
Reference: scanner completion contract
```

Copy the false-at-EOF distinction: `Scan` returning false is not automatically clean EOF.

```text
[high] [go-idiomatic-review] internal/proxy/fetch.go:61
Issue: The loop defers resp.Body.Close() on every fetched page.
Impact: Large paginated responses keep all bodies open until FetchAll returns, which can exhaust connections or file descriptors.
Suggested fix: Close the body before the next iteration, or move per-page processing into a helper where defer has per-response scope.
Reference: defer lifetime shape
```

Copy the lifetime framing: `defer` is wrong only because the surrounding function has a longer lifetime than each resource.

```text
[medium] [go-idiomatic-review] internal/io/read.go:39
Issue: The Read loop returns immediately on err before appending the bytes from the same call.
Impact: Readers are allowed to return n > 0 with a non-nil error, so the final chunk can be dropped.
Suggested fix: Process n bytes first, then handle err, treating io.EOF as completion after the bytes are consumed.
Reference: io.Reader partial-read contract
```

Copy the exact stdlib contract: the finding proves how a legal `Read` result loses data.

## Reject
```text
rows.Close is present, so the database resource is handled.
```

Reject because closing does not prove iteration completed without an error.

```text
Just defer Body.Close inside the loop.
```

Reject when the defer lifetime is the whole function and the loop can open many resources.

```text
Ignore the close error everywhere.
```

Reject as a blanket rule. Close errors on read-only response bodies are often not actionable, but writer/file close errors may be the only signal that buffered data failed to flush.

## Agent Traps
- Do not turn every missing `Close` error check into a finding; decide whether the API uses close as a meaningful commit/flush signal.
- Do not use `defer` as an automatic fix without checking loop lifetime.
- Do not skip `rows.Err()` or `scanner.Err()` because tests already consumed normal input.
- Do not turn this into SQL transaction, retry, or shutdown review; hand off those specialist concerns.
- Do not demand broad `go test ./...` if a focused package test can prove the resource/probe contract.

## Validation Shape
- Add a fake iterator/scanner/reader path that fails after yielding partial data and assert the function returns an error instead of success.
- Add a loop test with a close-counting fake body when lifetime is the defect.
- For timer/ticker ownership, include the module Go version when Stop/GC behavior matters; use deterministic cancellation or a fake clock where the repo already has one.
- Run focused package tests for the changed resource path; add `go vet` only if the suspected issue maps to a vet analyzer.

## Handoffs
- Hand off SQL query and transaction correctness to DB/cache review.
- Hand off goroutine, timer, ticker, and shutdown depth to concurrency or reliability review.
- Hand off response status/API semantics to API or chi review.
- Hand off performance proof for resource retention to performance review.
