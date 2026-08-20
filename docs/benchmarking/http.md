# External HTTP Benchmarks

Load for client-visible HTTP latency, error rate, offered load, or k6.

The repository pins Grafana k6 by version and image digest and provides
`test/performance/http/single-flow.js`. The starter scenario:

- supports steady `constant-arrival-rate` and staged
  `ramping-arrival-rate` open-model executors;
- checks the expected response status;
- applies an explicit request timeout;
- fails on the configured latency percentile, error-rate, correctness, or
  dropped-iteration budget;
- retains p95 and p99 trend statistics in the aggregate result;
- discards response bodies after the HTTP layer has consumed them;
- writes the aggregate summary, console log, and run metadata under
  `.artifacts/bench/http/`.

The latency threshold uses the custom `flow_duration` trend around the complete
`http.request` call. Unlike k6 `http_req_duration`, this wall-clock boundary
also includes time spent blocked, connecting, and negotiating TLS. The summary
records the workload identifier, executor, stages, timeout, percentile, and
budget without copying header values or the request body.

Copy the ready input template, then replace every workload value. Keep secrets
and environment-specific values in the ignored `.env.bench` file:

```bash
cp test/performance/http/single-flow.env.example .env.bench
```

The example values prove runner wiring only; they are not template SLOs.

Start the service and its representative dependencies, seed the named fixture,
then run:

```bash
make bench-http
```

Use the steady executor for smoke, steady-load, and soak workloads by setting
`HTTP_BENCH_RATE` and `HTTP_BENCH_DURATION`. Use the ramping executor for
stress and spike shapes by supplying a starting rate and compact JSON stages;
the rate and duration fields from the example file are then ignored:

```bash
HTTP_BENCH_START_RATE=0 \
HTTP_BENCH_STAGES_JSON='[{"target":20,"duration":"1m"},{"target":50,"duration":"2m"},{"target":0,"duration":"30s"}]' \
make bench-http
```

Every stage target is iterations per second because `timeUnit` is fixed at
`1s`. A stage duration of `0` is allowed for an instantaneous spike transition.
Pre-allocate enough VUs for the highest target before measurement.

The Docker-host address in the example reaches a service running on the host.
For a service on a Compose or Docker network, set
`HTTP_BENCH_DOCKER_NETWORK=<network>` and use its container DNS name in the base
URL. Do not put credentials in the URL. Header values and request bodies are
not copied into run metadata, but they still belong only in ignored local or
secret-managed configuration.

For secret injection without a file, export the same variables and run
`make bench-http HTTP_BENCH_ENV_FILE=`. Direct environment values override
matching values from the environment file.

The aggregate summary is the default evidence because per-sample streaming can
consume disk and distort the load generator. Enable it only for diagnosis:

```bash
make bench-http HTTP_BENCH_RAW_SAMPLES=1
```

That opt-in writes compressed `samples.json.gz`; it never silently enables
unbounded plain JSON output.

The generic scenario supports one request with an optional
`HTTP_BENCH_BODY` and JSON headers. When correctness depends on login, setup,
multiple requests, extracted identifiers, or weighted traffic classes, add a
feature-owned k6 scenario under `test/performance/http/` and select it with
`HTTP_BENCH_SCRIPT=<path>`. Keep scenario setup outside the measured default
function, tag each flow, and give each decision-relevant flow its own threshold.

Do not use `/health` or `/metrics` as a proxy for business-path capacity. Do not
add `sleep` to an arrival-rate scenario; the executor owns request arrival.
Pre-allocate enough VUs before the measured run; the starter deliberately does
not allocate more VUs during measurement because that work can skew load
generator results. Treat dropped iterations as a failed offered-load claim,
then distinguish insufficient VU allocation from service saturation before
changing the service.

For decision-grade high load, run k6 on a separate stable host and monitor its
CPU, memory, swap, and network along with the service. Keep roughly 20% load
generator CPU idle and reject a run with swap use or saturated generator
network; otherwise the generator, not the service, may be the bottleneck.

## References

- Grafana k6 [constant arrival rate](https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/constant-arrival-rate/)
- Grafana k6 [ramping arrival rate](https://grafana.com/docs/k6/latest/using-k6/scenarios/executors/ramping-arrival-rate/)
- Grafana k6 [arrival-rate VU allocation](https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/arrival-rate-vu-allocation/)
- Grafana k6 [built-in HTTP metrics](https://grafana.com/docs/k6/latest/using-k6/metrics/reference/)
- Grafana k6 [gRPC overview](https://grafana.com/docs/k6/latest/using-k6/protocols/grpc/)
- Grafana k6 [gRPC streams](https://grafana.com/docs/k6/latest/javascript-api/k6-net-grpc/stream/)
- Grafana k6 [result outputs](https://grafana.com/docs/k6/latest/results-output/)
- Grafana k6 [large-test generator sizing](https://grafana.com/docs/k6/latest/testing-guides/running-large-tests/)
- Grafana k6 [thresholds](https://grafana.com/docs/k6/latest/using-k6/thresholds/)
- Grafana k6 [dropped iterations](https://grafana.com/docs/k6/latest/using-k6/scenarios/concepts/dropped-iterations/)
