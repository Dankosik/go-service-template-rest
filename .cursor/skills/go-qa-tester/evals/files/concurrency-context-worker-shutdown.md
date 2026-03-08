We changed a background shard reconciler:

- It fans out up to 16 shard workers in parallel.
- The workers share one parent `context.Context`.
- If one worker returns a fatal error, sibling workers should stop promptly.
- Repository calls wrap underlying database errors with `%w`.
- The service must preserve `context.Canceled` and `context.DeadlineExceeded` instead of turning them into generic business errors.
- A previous bug used `context.Background()` inside one repository helper and caused shutdown hangs.
- Another previous bug relied on `time.Sleep` in tests and hid a blocked send after cancellation.

Do not edit repository files. I want the exact tests you would add, how you would keep them deterministic, and which validation commands you would run.
