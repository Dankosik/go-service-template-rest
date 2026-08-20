# Benchmarking

Use the narrowest benchmark that measures the accepted performance claim. This
repository provides one path for each normal proof level:

| Claim | Harness | Command |
| --- | --- | --- |
| Local CPU, allocations, or contention | Go `testing.B` | `make bench`, `make bench-compare`, `make bench-profile` |
| In-process handler, middleware, or composed router cost | `testing.B` plus `net/http/httptest` | the same Go benchmark commands |
| Real PostgreSQL query, repository, or pool cost | `testing.B` plus the existing Testcontainers integration seam | `make bench-db`, `make bench-db-compare` |
| Client-visible HTTP latency, error rate, and offered load | digest-pinned Grafana k6 in Docker | `make bench-http` |

Runtime telemetry is still required when production traffic, dependency state,
GC behavior, or capacity can materially change the conclusion. A benchmark is
controlled evidence, not a replacement for correctness tests or production
SLIs.

## Define the workload before measuring

Record the measured operation, representative fixture buckets, concurrency or
arrival model, warm/cold dependency state, machine or Docker testbed, and the
latency, throughput, allocation, or capacity budget. Keep materially different
input sizes, cache states, row counts, tenant shapes, and concurrency levels in
separate named cases; a combined average can hide a regression.

Use the same source revision, toolchain, power mode, dependency versions,
fixture shape, and benchmark parameters for baseline and candidate. Run on an
idle machine and do not keep rerunning until a desired statistical result
appears. Give each evidence set a stable workload identifier: a short name for
the fixture, state, and operation that can be compared without recording
secrets.

## Choose the execution environment

Prefer DigitalOcean when `doctl` is already installed and its selected context
is authorized. If either condition is false, stop the remote path and run the
matching local command: `make bench*` for Go and in-process HTTP,
`make bench-db*` for real PostgreSQL, or `make bench-http` for external HTTP.
An unavailable default size/region pair does not make the remote path
unavailable: after paid execution is authorized, the runner skill owns
equivalent placement and bounded fallback inside the accepted cost, security,
and proof envelope without asking the user to choose a host or region.
Do not install `doctl`, start authentication, create an account, or provision a
Droplet unless the user explicitly authorizes that setup or paid execution
envelope. An unavailable DigitalOcean account
is an expected local-fallback condition, not a successful remote proof. Local
Docker or service prerequisites still fail closed when the chosen benchmark
requires them.

When authorized remote execution is selected, apply
`digitalocean-benchmark-runner`. It owns provider discovery, paid authority,
resource identity, placement, remote execution, evidence retrieval, and cleanup.
This owner retains only workload and performance-claim semantics.

## Select one leaf

| Changed pressure | Load |
| --- | --- |
| Go `testing.B`, in-process HTTP, profiling, or PGO | [Go And In-Process HTTP](benchmarking/go.md) |
| PostgreSQL query, repository, or pool | [PostgreSQL](benchmarking/postgres.md) |
| External HTTP and k6 | [External HTTP](benchmarking/http.md) |

Load another leaf only for an independent changed pressure.

## Completion and CI policy

A performance claim closes only when:

- independent correctness proof is green;
- the benchmark boundary and workload match the claim;
- baseline and candidate evidence came from equivalent environments;
- the result satisfies the accepted material-change or absolute budget;
- any service-level claim also has the required saturation and runtime evidence.

These targets are explicit evidence commands, not default CI gates. Shared
GitHub-hosted runners are not a stable blocking regression oracle. Add
persistent history or blocking automation only after the benchmark is stable,
a dedicated comparable testbed exists, and a named owner has approved the
threshold and response to failure.

When benchmark infrastructure itself changes, run:

```bash
make benchmark-infra-check
```

This fail-closed smoke executes a hermetic create/sync/reconcile/destroy test of
the DigitalOcean lifecycle, the Go capture and pinned `benchstat` comparison
path, verifies PostgreSQL image provenance, then has pinned k6 inspect both the
steady and ramping scenario configurations. It proves wiring and validation
only; its deliberately tiny stdlib samples are not performance evidence and it
does not create cloud resources or need a running service.

Before relying on a benchmark level in a derived service, run one bounded live
smoke for that level. In particular, verify that Testcontainers can start the
real PostgreSQL dependency, that each profile type the service will use can be
opened by the matching Go tool, and that k6 can reach a real disposable target.
This adoption smoke is not a CI gate and its results are not service performance
evidence.
