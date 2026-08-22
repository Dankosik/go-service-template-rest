# External HTTP Benchmarks

The repository provides `test/performance/http/single-flow.js` and an ignored
`.env.bench` input. Replace every example workload value before measurement.

Run the pinned image directly:

```bash
docker run --rm --network host --env-file .env.bench \
  -v "$PWD/test/performance/http:/work:ro" \
  grafana/k6:2.1.0@sha256:65c920dc067d5e2e00befbf982af6ad6ad0117034e8b1c65817c7975c52d4669 \
  run /work/single-flow.js
```

Use a Docker network and container DNS name instead of `--network host` when
the service runs in Compose. Keep credentials in ignored local or
secret-managed inputs, never URLs or committed files.

The scenario supports steady and ramping arrival rates, checks the expected
status, applies a request timeout, and fails its configured latency, error,
correctness, or dropped-iteration budget. Use a feature-owned scenario when the
claim requires login, multiple requests, extracted identifiers, or weighted
flows.

Do not use health endpoints as a proxy for business capacity. Reject a run when
the generator itself is CPU, memory, swap, or network saturated.
