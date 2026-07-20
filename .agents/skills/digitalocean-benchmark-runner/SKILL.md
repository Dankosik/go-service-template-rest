---
name: digitalocean-benchmark-runner
description: "DigitalOcean benchmarks: Use when an agent must provision, operate, or clean up ephemeral DigitalOcean Droplets for Go, PostgreSQL, or external HTTP benchmark execution, including private dirty source or a separate k6 generator. Own secure paid-resource lifecycle, same-testbed comparison, artifact return, and cleanup; Skip when measurement stays local, the workload or budget is undecided, or only code-level performance analysis is requested."
---

# DigitalOcean Benchmark Runner

Read [Benchmarking](../../../docs/benchmarking.md) and the [DigitalOcean runbook](references/digitalocean.md). Keep measurement in the repository's existing `make bench*` targets; this skill owns only remote execution.

## Execute

1. Confirm the workload, budget, benchmark level, and whether external paid writes are authorized. `check`, `list`, and `image-list` are read-only; `create`, `run`, and `image-build` incur cost, while a created snapshot keeps incurring storage cost until deletion.
2. For an ordinary benchmark request, use DigitalOcean only when `doctl` is already installed and the selected context is authorized. Otherwise stop the remote path and use the matching local command. When the user explicitly asks to configure DigitalOcean, follow the runbook's account/workstation onboarding: the user owns browser, payment, token, and passphrase entry; the agent may perform the requested local setup and read-only verification but must never request or echo those secrets. A real setup smoke still needs explicit paid-write authorization.
3. Run `scripts/dev/benchmark-remote.sh list` before provisioning. Account for one Droplet per Go or database session and two per decision-grade external HTTP session, without exceeding the Team's current resource limit. When repeated startup is material, also run `image-list` and use the runbook's reusable snapshot. Prefer its temporary least-privilege builder token; reuse an existing broader context only when the user explicitly authorizes that trade-off.
4. Give every concurrent session its own `DO_BENCH_STATE_FILE`. One state file owns one Droplet; never attach to, reuse, operate, or delete another service's session. The default path is isolated across repository working directories but not across concurrent sessions in the same directory.
5. Use `run -- <command>` for one result. Use one retained session for baseline/candidate comparison: `create`, `sync`, `exec` baseline, change the local source, `sync`, `exec` candidate and comparison, `fetch`, then `destroy`.
6. Keep both sides on the same Droplet. Never compare raw results across ephemeral Droplets; use a stable dedicated testbed before persistent history or blocking thresholds.
7. For decision-grade external HTTP load, use separate target and generator state files in one region/VPC, allow only the generator private IP to the target port, and collect telemetry on both hosts. A single host proves wiring or low load only.
8. On every exit path, fetch useful evidence and destroy each owned Droplet, firewall, and tag. If cleanup fails, return the exact state file and retry command; powering off does not stop billing. A golden snapshot is the explicit exception: retain it only as the named reusable image, report its ongoing cost, and remove it when superseded or no longer used.

Complete only when raw benchmark/provider/host evidence is local, correctness proof remains independent, and no paid resource remains unless the user explicitly retained it.
