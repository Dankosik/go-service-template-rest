---
name: digitalocean-benchmark-runner
description: "DigitalOcean benchmark: Use for authorized Go/PostgreSQL/HTTP measurement on ephemeral Droplets. Own evidence/cleanup; Skip local or undecided work."
metadata:
  invocation: model
  kind: method
---

# DigitalOcean Benchmark Runner

Read [Benchmarking](../../../docs/benchmarking.md) and the complete [DigitalOcean
runbook](references/digitalocean.md) before a paid action. Keep measurement in
the existing `make bench*` targets; this skill owns remote execution only.

Confirm the workload, benchmark level, maximum spend, and paid-write authority.
Read-only discovery may proceed without purchase authority; provisioning,
remote runs, snapshots, and retained storage may not. The user owns browser,
payment, token, and passphrase entry, while the agent selects an advertised
dedicated placement inside the authorized envelope.

Use `scripts/dev/benchmark-remote.sh` exactly as the runbook defines. Give each
concurrent session its own `DO_BENCH_STATE_FILE`; that state owns one Droplet
and never another session's resources. Compare baseline and candidate on the
same realized host. Decision-grade external HTTP load uses separate target and
generator hosts in one private network; shared CPU or one host proves wiring
only. After rejected capacity, confirm cleanup and choose a different advertised
placement rather than repeating the failed pair.

Fetch evidence and destroy every owned Droplet, firewall, tag, and superseded
snapshot on every exit path. A named reusable snapshot is the only retained
resource and carries an explicit ongoing-cost receipt. Complete when raw remote
evidence is local, correctness proof remains independent, and no unapproved paid
resource remains.
