# Tool routing for large evidence sets

Route a bounded collection stage through Programmatic Tool Calling when several independent telemetry, metric, or log sources need filtering, joining, deduplication, subtraction, ranking, or validation.

Before the program runs, name the eligible read-only tools and set the input bounds: one target cluster or service, one time window, a record limit per source, concurrency at most four, and one retry for transient failures. The program calls those named tools and returns exactly:

```json
{
  "window": {},
  "reset_boundaries": [],
  "resource_deltas": [],
  "top_workloads": [],
  "anomalies": [],
  "missing_sources": [],
  "evidence_refs": []
}
```

Stop when each bounded source has succeeded or returned a structured failure after its retry.

Keep hypothesis ranking, approval decisions, side-effecting actions, `EXPLAIN` safety decisions, and final validation in the direct model/tool flow. Direct calls fit stages where one result is small, each result changes the next decision, or the final answer must preserve citations or native artifacts.
