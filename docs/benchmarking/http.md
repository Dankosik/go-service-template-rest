# External HTTP Benchmarks

The repository provides `test/performance/http/single-flow.js` and an ignored
`.env.bench` input. Copy the example and replace every workload value before
measurement:

```bash
cp test/performance/http/single-flow.env.example .env.bench
export BENCH_BUDGET='<accepted client-visible budget>'
export BENCH_RESPONSE_OWNER='<response owner>'
make benchmark-http
```

The command owns the pinned k6 image and writes durable evidence instead of
losing `handleSummary` output with the removed container:

- `summary.json`: thresholds, workload configuration, and aggregate metrics;
- `run.meta`: candidate, scenario digest, generator environment, budget, and
  response owner without header values or request bodies;
- `k6.log`: command output.

Set `HTTP_BENCH_ARTIFACT_DIR=.artifacts/bench/http/<candidate>` to retain
multiple runs. The command refuses to overwrite an existing receipt.

Use a Docker network and container DNS name instead of `--network host` when
the service runs in Compose by setting `HTTP_BENCH_DOCKER_NETWORK`. Keep
credentials in ignored local or secret-managed inputs, never URLs, metadata, or
committed files.

The scenario supports steady and ramping arrival rates, checks the expected
status, applies a request timeout, and fails its configured latency, error,
correctness, or dropped-iteration budget. Use a feature-owned scenario when the
claim requires login, multiple requests, extracted identifiers, or weighted
flows.

Do not use health endpoints as a proxy for business capacity. Reject a run when
the generator itself is CPU, memory, swap, or network saturated.

`make performance-harness-check` performs a non-load k6 inspection. It proves
scenario wiring only and is the CI gate for changes under `test/performance/`.
