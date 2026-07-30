---
name: digitalocean-benchmark-runner
description: "DigitalOcean benchmark: Use for authorized Go/PostgreSQL/HTTP measurements on ephemeral Droplets. Own paid-resource security, comparable evidence, artifacts, and cleanup; Skip local or undecided workloads."
---

# DigitalOcean Benchmark Runner

Read [Benchmarking](../../../docs/benchmarking.md) and the [DigitalOcean runbook](references/digitalocean.md). Keep measurement in the repository's existing `make bench*` targets; this skill owns only remote execution.

## Execute

1. Confirm the workload, maximum spend, benchmark level, and whether external paid writes are authorized. `check`, `list`, and `image-list` are read-only; `create`, `run`, and `image-build` incur cost, while a created snapshot keeps incurring storage cost until deletion. One authorization covers agent-selected placements and bounded retries that stay inside its cost, security, and proof envelope; do not seek approval per host or attempt.
2. For an ordinary benchmark request, use DigitalOcean only when `doctl` is already installed and the selected context is authorized. Otherwise stop the remote path and use the matching local command. When the user explicitly asks to configure DigitalOcean, follow the runbook's account/workstation onboarding: the user owns browser, payment, token, and passphrase entry; the agent may perform the requested local setup and read-only verification but must never request or echo those secrets. A real setup smoke still needs explicit paid-write authorization.
3. Run `scripts/dev/benchmark-remote.sh list` and resolve placement from current provider data before provisioning. `DO_BENCH_REGION` and `DO_BENCH_SIZE` are agent-owned inputs unless geography or hardware is part of the accepted claim. After a capacity or `422 Size is not available in this region` response, confirm cleanup and never retry the same size/region pair: keep the dedicated size and select another advertised region first, then select a sufficient CPU-Optimized dedicated size within the authorized budget. Keep every comparison on one realized Droplet; Shared CPU proves wiring only. If no remote placement preserves the claim, use the matching local path when it can close the accepted proof and report a blocker only after valid routes are exhausted.
4. Account for one Droplet per Go or database session and two same-region Droplets per decision-grade external HTTP session, without exceeding the Team's current resource limit. When repeated startup is material, also run `image-list` and use the runbook's reusable snapshot. Prefer its temporary least-privilege builder token; reuse an existing broader context only when the user explicitly authorizes that trade-off.
5. Give every concurrent session its own `DO_BENCH_STATE_FILE`. One state file owns one Droplet; never attach to, reuse, operate, or delete another service's session. The default path is isolated across repository working directories but not across concurrent sessions in the same directory.
6. Use `run -- <command>` for one result. Use one retained session for baseline/candidate comparison: `create`, `sync`, `exec` baseline, change the local source, `sync`, `exec` candidate and comparison, `fetch`, then `destroy`.
7. Keep both sides on the same Droplet. Never compare raw results across ephemeral Droplets; use a stable dedicated testbed before persistent history or blocking thresholds.
8. For decision-grade external HTTP load, use separate target and generator state files in one region/VPC, allow only the generator private IP to the target port, and collect telemetry on both hosts. A single host proves wiring or low load only.
9. On every exit path, fetch useful evidence and destroy each owned Droplet, firewall, and tag. If cleanup fails, return the exact state file and retry command; powering off does not stop billing. A golden snapshot is the explicit exception: retain it only as the named reusable image, report its ongoing cost, and remove it when superseded or no longer used.

Complete only when raw benchmark/provider/host evidence is local, correctness proof remains independent, and no paid resource remains unless the user explicitly retained it.
