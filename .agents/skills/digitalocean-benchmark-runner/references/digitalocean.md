# DigitalOcean benchmark runner runbook

Valid as of 2026-07-20. Refresh the CLI flags with `doctl version` and the
linked command reference before changing the runner. Refresh availability and
price with `scripts/dev/benchmark-remote.sh check` before creating resources.

- [Account and workstation onboarding](#account-and-workstation-onboarding)
- [Preflight and paid setup smoke](#preflight)
- [Reusable golden snapshot](#reusable-golden-snapshot)
- [Parallel sessions](#parallel-sessions-and-account-visibility)
- [Fast one-shot run](#fast-one-shot-run)
- [Baseline and candidate](#baseline-and-candidate-on-one-droplet)
- [External HTTP load](#external-http-load)
- [Evidence](#evidence-and-honesty-checks)
- [Cleanup and recovery](#cleanup-and-recovery)
- [Proof boundary](#proof-boundary)

## What this repository owns

`scripts/dev/benchmark-remote.sh` is a thin executor around the existing
benchmark commands. It does not create a second benchmark framework.

- `testing.B`, `httptest`, Testcontainers, k6, `benchstat`, and the existing
  `make bench*` commands continue to own measurement.
- The remote runner owns Droplet/firewall/tag lifecycle, source transfer, SSH,
  host evidence, artifact return, and cleanup.
- Tracked and non-ignored untracked files are transferred. `.git`, ignored
  secrets, caches, and local artifacts are not transferred.
- If the build needs private Go modules, prefer `make vendor` before `sync` or
  an already-operated read-only module proxy reachable by the Droplet. Do not
  forward the laptop's SSH agent to an untrusted remote build.
- Set `DO_BENCH_ENV_FILE=.env.bench` only when an ignored benchmark environment
  file is intentionally required. It is copied as mode `0600` and its contents
  are never written to benchmark metadata.
- The API token remains on the local machine. It is never sent to the Droplet.

## Account and workstation onboarding

This section is the reproducible handoff for a user who explicitly opts into
DigitalOcean. A new DigitalOcean account already has a default Team; a separate
Team or organization is not required for this runner. In the control panel, the
user must:

1. sign up or sign in, verify the account, and select the Team that will own the
   benchmark resources;
2. add a payment method under **Billing > Payment Methods**;
3. open **Resource Limits** and note the Droplet tier, count limit, and available
   plans;
4. open **Account > API > Personal access tokens**, create a token named for
   benchmark automation, choose an expiration, select **Custom Scopes**, and
   copy the token while it is shown.

Tier 1 requires no prepayment and currently permits three Basic Shared CPU
Droplets up to the $48 plan. A $50 one-time prepayment can unlock Tier 2
immediately; successful billing history or an approved limit request can also
increase the tier. Tier 2 adds some Dedicated CPU plans up to $84, including
the default `c-4` at the valid date. Do not upgrade merely to prove the
connection: use an available Basic plan for the setup smoke, then require a
dedicated plan for performance comparisons. Prices and limits shown by the
control panel and preflight win over this dated summary.

The user enters the API token and SSH-key passphrase only in local terminal
prompts. They never paste either value into chat. During explicit onboarding an
agent may inspect prerequisites, install `doctl` when asked, create non-secret
directories, and run the commands below. It must pause for account, payment,
token, and passphrase entry; resume with read-only verification; and obtain
separate authorization before the paid smoke.

Official contracts:

- [Manage payment methods](https://docs.digitalocean.com/platform/billing/manage-payment-methods/)
- [Create a personal access token](https://docs.digitalocean.com/reference/api/create-personal-access-token/)
- [View Team resource limits](https://docs.digitalocean.com/platform/teams/how-to/view-resource-limits/)
- [Default resource-limit tiers](https://docs.digitalocean.com/platform/resource-limits/)

### Install and authorize `doctl`

Install the official CLI on macOS:

```bash
brew install doctl
```

Create a dedicated custom-scope token in DigitalOcean. The normal runner needs:

```text
actions:read
droplet:read
droplet:create
droplet:delete
firewall:read
firewall:create
firewall:update
firewall:delete
image:read
regions:read
sizes:read
ssh_key:read
tag:read
tag:create
tag:delete
vpc:read
```

`firewall:update` is used only to allow a separate load generator's private IP.
Do not create a new operational token with global `api:write`; an existing
broader context is the explicit user-authorized exception described below.

Building a reusable snapshot is a rarer operation. For least privilege, keep
the normal `benchmarks` token unchanged and create a short-lived
`benchmarks-image-builder` token with every normal-runner scope above plus
exactly:

```text
droplet:update
image:create
```

DigitalOcean requires both scopes for the Droplet `snapshot` action. Initialize
the temporary context with `doctl auth init --context benchmarks-image-builder`,
entering the token only at the local prompt. Remove the context and revoke the
token after the image is built. Snapshot deletion is intentionally not granted
to either operational context; use the control panel, or a separate temporary
token with the documented `snapshot:delete` dependencies.

If the user explicitly authorizes an existing context that already has these
permissions, including a Full Access context, it may be used for the build;
creating a second token is not technically required. Keep the temporary scoped
context as the default recommendation because it limits the impact of token
exposure. Substitute the authorized context name in the image-build commands
below and do not remove it afterward if it is the user's retained context.

Initialize a named context. Enter the token only at the prompt:

```bash
doctl auth init --context benchmarks
doctl auth switch --context benchmarks
doctl auth list
export DO_BENCH_CONTEXT=benchmarks
```

The export is not a secret. It makes the intended context explicit for the
runner and later agents; place it in the user's shell profile only when the user
wants that context to be the workstation default. Validate the selected Team
without creating anything:

```bash
doctl --context benchmarks compute droplet list \
  --format ID,Name,Region,VCPUs,Memory,Status,Tags
```

### Create and register the SSH key

Do not overwrite an existing private key. Inspect first, and create a dedicated
Ed25519 key only when the path is absent:

```bash
ls -l ~/.ssh/digitalocean-bench ~/.ssh/digitalocean-bench.pub
ssh-keygen -t ed25519 -f ~/.ssh/digitalocean-bench -C digitalocean-bench
ssh-add --apple-use-keychain ~/.ssh/digitalocean-bench
```

`ls` returning “No such file” for both paths means creation is safe. If either
path already exists, stop and inspect it instead of running `ssh-keygen`.

On macOS, `ssh-add --apple-use-keychain` is the step that made the encrypted key
available to this runner and later non-interactive agents. Enter the passphrase
in that local prompt. On Linux, load the key into the user's already-operated
SSH agent or keyring; the runner will fail closed rather than ask an agent for a
passphrase.

The long-lived operational token intentionally has only `ssh_key:read`. Register
the public key once under **Settings > Security > SSH Keys > Add SSH Key**, or
use a separate short-lived setup token with `ssh_key:read` and
`ssh_key:create`. Token scopes cannot be edited after creation. For the
temporary-token CLI path:

```bash
doctl auth init --context benchmarks-setup
doctl --context benchmarks-setup compute ssh-key import digitalocean-bench \
  --public-key-file ~/.ssh/digitalocean-bench.pub
doctl auth switch --context benchmarks
doctl auth remove --context benchmarks-setup
```

Delete the temporary token in the control panel after the import. Verify the
registered fingerprint with the operational context:

```bash
ssh-keygen -E md5 -lf ~/.ssh/digitalocean-bench.pub
doctl --context benchmarks compute ssh-key list \
  --format ID,Name,FingerPrint
```

The runner already defaults to this dedicated key path. Set
`DO_BENCH_SSH_PRIVATE_KEY` only when a different path is intentional.

Do not export the API token or put it in `.env`, `.env.bench`, shell commands,
repository files, Droplet user data, or benchmark artifacts. `doctl` stores the
named context outside the repository.

Official contracts:

- [Install and configure `doctl`](https://docs.digitalocean.com/reference/doctl/how-to/install/)
- [`doctl auth init`](https://docs.digitalocean.com/reference/doctl/reference/auth/init/)
- [`doctl auth remove`](https://docs.digitalocean.com/reference/doctl/reference/auth/remove/)
- [API token scopes](https://docs.digitalocean.com/reference/api/scopes/)
- [`droplet:update` scope](https://docs.digitalocean.com/reference/api/scopes/droplet/update/)
- [`image:create` scope](https://docs.digitalocean.com/reference/api/scopes/image/create/)
- [`tag:create` scope](https://docs.digitalocean.com/reference/api/scopes/tag/create/)
- [`tag:delete` scope](https://docs.digitalocean.com/reference/api/scopes/tag/delete/)
- [Import an SSH public key](https://docs.digitalocean.com/reference/doctl/reference/compute/ssh-key/import/)
- [Manage Team SSH public keys](https://docs.digitalocean.com/platform/teams/how-to/upload-ssh-keys/)

## Preflight

```bash
DO_BENCH_CONTEXT=benchmarks make benchmark-remote-check
```

The check performs no external writes. It verifies local commands, the private
and public key pair, DigitalOcean authentication, the registered fingerprint,
region availability, size, image, and current provider price.

DigitalOcean is preferred only when `doctl` is already installed and the
selected context is authorized. If `doctl` is absent or authentication is not
usable, do not install it, start `doctl auth init`, create an account, or call
any paid lifecycle command. Return to `docs/benchmarking.md` and use the
matching local `make bench*` target. This is the normal path for repository
users who have not opted into DigitalOcean. Keep local Docker and service
prerequisites fail-closed.

Defaults:

```text
DO_BENCH_REGION=fra1
DO_BENCH_SIZE=c-4
DO_BENCH_IMAGE=ubuntu-24-04-x64
DO_BENCH_SSH_CIDR=auto
```

`c-4` is the default CPU-Optimized shape: four dedicated vCPUs and 8 GiB RAM.
As of the valid date it is $0.125/hour with an $84 monthly cap. DigitalOcean
bills CPU Droplets per second with a 60-second or $0.01 minimum. Billing begins
at creation and ends only at destruction; a powered-off Droplet still bills.
The script prints the current API price, which wins over this dated note.

If Tier 1 reports `DigitalOcean size is unavailable or unknown: c-4`, list the
plans actually available to the Team:

```bash
doctl --context benchmarks compute size list \
  --format Slug,VCPUs,Memory,Disk,PriceHourly,PriceMonthly
```

For connection testing only, select an available Basic plan. The Tier 1 setup
we verified used `s-4vcpu-8gb`:

```bash
DO_BENCH_CONTEXT=benchmarks \
DO_BENCH_SIZE=s-4vcpu-8gb \
make benchmark-remote-check
```

Do not silently use Shared CPU for a decision-grade comparison. Request Tier 2
or a limit increase and return to the dedicated `c-4` default, or run locally
and narrow the claim.

### Paid setup smoke

After preflight succeeds and the user explicitly authorizes one paid test,
exercise creation, SSH/Keychain, provisioning, source transfer, benchmark
execution, artifact download, and cleanup with a deliberately tiny stdlib
workload:

```bash
state="$PWD/.artifacts/bench/remote/setup-smoke.state"
DO_BENCH_CONTEXT=benchmarks \
DO_BENCH_SIZE=s-4vcpu-8gb \
DO_BENCH_STATE_FILE="$state" \
scripts/dev/benchmark-remote.sh run -- \
  make bench \
  BENCH_PACKAGE=crypto/sha256 \
  BENCH_PATTERN=BenchmarkHash8Bytes \
  BENCH_WORKLOAD_ID=digitalocean-setup-smoke \
  BENCH_COUNT=3 \
  BENCH_TIME=100ms
```

Replace `s-4vcpu-8gb` when preflight shows a different available Basic slug.
This smoke proves wiring only; its short Shared-CPU measurements are not
performance evidence. Success ends with artifact download and explicit
Droplet, firewall, and tag deletion. Confirm no owned runner remains:

```bash
DO_BENCH_CONTEXT=benchmarks scripts/dev/benchmark-remote.sh list
```

If the command exits before cleanup is confirmed and the state file remains,
run the same state through `destroy` immediately:

```bash
DO_BENCH_CONTEXT=benchmarks \
DO_BENCH_STATE_FILE="$state" \
scripts/dev/benchmark-remote.sh destroy
```

Choose one region and size for the whole comparison. A new Droplet may have a
different physical CPU model even under the same size slug, so do not compare
raw measurements from different Droplets. The Ubuntu slug can also move to a
new image revision; provider and host evidence records the realized testbed.

Official contracts:

- [Droplet pricing and billing](https://docs.digitalocean.com/products/droplets/details/pricing/)
- [`doctl compute size list`](https://docs.digitalocean.com/reference/doctl/reference/compute/size/list/)
- [`doctl compute region list`](https://docs.digitalocean.com/reference/doctl/reference/compute/region/list/)
- [`doctl compute image list`](https://docs.digitalocean.com/reference/doctl/reference/compute/image/list/)

## Reusable golden snapshot

Use this optional path when repeated creation of fresh Droplets makes package
installation a material part of the workflow. It does not speed the benchmark
itself. It removes repeated `apt` work from the measured `create -> ready`
interval while leaving source sync, exact Go toolchain selection, the benchmark
harness, host evidence, and same-Droplet comparison rules unchanged.

The builder defaults to `s-1vcpu-1gb`: one Shared vCPU, 1 GiB RAM, and a 25 GiB
disk. The small disk matters because DigitalOcean only permits a snapshot to
create a Droplet with an equal or larger disk. It is compatible with the 50 GiB
default `c-4`; a snapshot built from the previously tested 160 GiB
`s-4vcpu-8gb` would not be. Shared CPU is acceptable for this one-time package
installation because its measurements are never benchmark evidence.

The human must enter any new token locally. An agent can perform every remaining
step, but must still receive explicit authorization for the paid build and
persistent snapshot. First run a read-only preflight against the builder
settings, using either the recommended temporary context or an explicitly
authorized existing context:

```bash
DO_BENCH_CONTEXT=benchmarks-image-builder \
DO_BENCH_SIZE=s-1vcpu-1gb \
DO_BENCH_IMAGE=ubuntu-24-04-x64 \
make benchmark-remote-check
```

After authorization, build once:

```bash
DO_BENCH_CONTEXT=benchmarks-image-builder make benchmark-remote-image
```

When local `go` is available, the builder preloads `go env GOVERSION`; set
`DO_BENCH_IMAGE_GO_TOOLCHAIN=goX.Y.Z` only when a different target toolchain is
intentional. By default it derives the PostgreSQL and k6 image references from
this repository. `DO_BENCH_IMAGE_DOCKER_IMAGES` may replace that space-separated
list, but every replacement must be digest-pinned.

`image-build` creates the protected builder, installs Docker, Go bootstrap
tooling, build tools, and sysstat, preloads the current local Go toolchain and
the repository's digest-pinned PostgreSQL and k6 images, then verifies them. It
removes source, project caches, and key artifacts, runs
`cloud-init clean --logs --machine-id`, powers the Droplet off, creates the
snapshot, and deletes the builder Droplet, firewall, and tag. The generated
`.artifacts/bench/remote/golden-image.env` contains only the snapshot ID, name,
and `DO_BENCH_GOLDEN_IMAGE=1`; it contains no token or private key.
If the build fails and its state file remains, retry cleanup before anything
else:

```bash
DO_BENCH_CONTEXT=benchmarks-image-builder \
DO_BENCH_STATE_FILE=.artifacts/bench/remote/image-builder.state \
scripts/dev/benchmark-remote.sh destroy
```

When a temporary builder context was used, remove it and return to the
long-lived least-privilege runtime context. Do not run these removal steps when
the user chose an existing retained context. Then validate the snapshot before
a paid benchmark:

```bash
# Temporary builder context only; skip both steps for a retained context.
doctl auth remove --context benchmarks-image-builder
# Also revoke the temporary token in Account > API.
source .artifacts/bench/remote/golden-image.env
export DO_BENCH_CONTEXT=benchmarks
make benchmark-remote-check
scripts/dev/benchmark-remote.sh image-list
```

The runner uses minimal per-instance cloud-init for that snapshot: a fresh
benchmark user/key, rotated host keys, SSH hardening, and Docker startup, but no
package update/install. It fails closed if the image is not a snapshot or the
expected tools are missing. Every create records `ready_after_seconds` in its
provider evidence; compare that value with a public-Ubuntu create to verify the
startup gain on the current account instead of assuming it.

Do not bake repository source, private modules, `.env.bench`, project
module/build caches, or credentials into the image. The only preserved Go cache
is the explicitly preloaded public toolchain; the only Docker layers are the
digest-pinned benchmark dependencies. Refresh the snapshot only when its base
OS or stable dependency set changes, and validate the replacement before
deleting the old one. DigitalOcean currently charges Droplet snapshots at
$0.06/GB-month based on snapshot size, with a $0.01 minimum; the control panel
price wins over this dated note. Snapshots remain billable until manually
deleted:

```bash
scripts/dev/benchmark-remote.sh image-list
# With a separately authorized destructive context:
doctl --context benchmarks-snapshot-delete compute snapshot delete <snapshot-id>
```

Official contracts:

- [Create a powered-off Droplet snapshot](https://docs.digitalocean.com/products/snapshots/how-to/snapshot-droplets/)
- [Create Droplets from a snapshot and disk-size constraint](https://docs.digitalocean.com/products/snapshots/how-to/create-and-restore-droplets/)
- [`doctl compute droplet-action snapshot`](https://docs.digitalocean.com/reference/doctl/reference/compute/droplet-action/snapshot/)
- [`cloud-init clean` for golden images](https://docs.cloud-init.io/en/latest/reference/cli.html#clean)
- [Snapshot pricing](https://docs.digitalocean.com/products/snapshots/details/pricing/)
- [Delete snapshots](https://docs.digitalocean.com/products/snapshots/how-to/delete/)

## Parallel sessions and account visibility

List and count every existing runner created by this repository before starting a
new paid session:

```bash
scripts/dev/benchmark-remote.sh list
```

The command reads the Team's Droplets and reports resources with the runner's
reserved `bench-...` name prefix. For an unfiltered provider view, use:

```bash
doctl --context benchmarks compute droplet list \
  --format ID,Name,Region,VCPUs,Memory,Status,Tags
```

The count includes every power state because powering off does not stop
billing. It is observability, not a reservation or distributed lock. Check that
the existing count plus the planned sessions fits the Team's current Droplet
limit before creating anything. DigitalOcean owns the hard limit; view it and
request increases on the Team's **Settings > Limits** page. One Go or database
session normally needs one Droplet. Decision-grade external HTTP load needs
two: the target and the generator.

Each controlling process must own a distinct state path. One state file owns
exactly one Droplet and its firewall and tag:

```bash
state="$PWD/.artifacts/bench/remote/orders-candidate.state"
DO_BENCH_STATE_FILE="$state" scripts/dev/benchmark-remote.sh run -- \
  make bench BENCH_WORKLOAD_ID=orders-100-lines
```

Relative default state paths are naturally separate in different repository
working directories. Concurrent sessions launched from the same working
directory must set different `DO_BENCH_STATE_FILE` values. A two-host HTTP run
must use separate target and generator state files. A retained
baseline/candidate sequence may reuse its own state file because both sides
belong to one comparison on one testbed.

`create` and `run` always create a new Droplet; they never select an existing
runner from `list`. `status`, `sync`, `exec`, `fetch`, and `destroy` operate only
on the resource recorded in the caller's state file. Never pass another
service's state file and never delete another owner's runner merely to free a
slot. The owning process is responsible for its cleanup.

Official contracts:

- [`doctl compute droplet list`](https://docs.digitalocean.com/reference/doctl/reference/compute/droplet/list/)
- [View and increase Team resource limits](https://docs.digitalocean.com/platform/teams/how-to/view-resource-limits/)
- [Default resource-limit tiers](https://docs.digitalocean.com/platform/resource-limits/)

## Fast one-shot run

Use this only when one remote result is sufficient:

```bash
scripts/dev/benchmark-remote.sh run -- \
  make bench BENCH_PACKAGE=./internal/app/orders \
  BENCH_PATTERN=BenchmarkCalculateTotal \
  BENCH_WORKLOAD_ID=orders-100-lines
```

`run` creates and provisions the Droplet, synchronizes the current source,
executes the command, downloads `.artifacts/bench`, and destroys the Droplet,
firewall, and tag even when the command fails. It returns the benchmark failure
after cleanup.

Use the same form for a real-PostgreSQL benchmark:

```bash
scripts/dev/benchmark-remote.sh run -- \
  make bench-db BENCH_DB_PATTERN=BenchmarkOrderRepository \
  BENCH_DB_WORKLOAD_ID=orders-100k-warm
```

Cloud-init creates an unprivileged `benchmark` user, disables password and root
SSH login, installs Docker, Go bootstrap tooling, build tools, and sysstat, then
waits fail-closed for provisioning to finish. Go's toolchain selection downloads
the exact module toolchain before benchmark execution.

The Cloud Firewall exposes SSH only to `DO_BENCH_SSH_CIDR`. `auto` resolves the
caller's public IPv4 and uses `/32`; set it explicitly when a VPN, proxy, or
changing egress IP makes discovery wrong. The script creates a unique tag,
binds the firewall to it, and applies it while creating the Droplet, so the
Droplet is never intentionally created without the firewall. It then gets the
Droplet IP from the authenticated API, scans its initial host key, and pins
that key in the session-specific `known_hosts` file.

Official contracts:

- [Create a Droplet with `doctl`](https://docs.digitalocean.com/reference/doctl/reference/compute/droplet/create/)
- [Provide cloud-init user data](https://docs.digitalocean.com/products/droplets/how-to/provide-user-data/)
- [Wait for cloud-init](https://docs.cloud-init.io/en/latest/howto/wait_for_cloud_init.html)
- [Create a Cloud Firewall](https://docs.digitalocean.com/reference/doctl/reference/compute/firewall/create/)
- [Apply a firewall immediately with a Droplet tag](https://docs.digitalocean.com/products/networking/firewalls/how-to/create/)
- [Create a tag](https://docs.digitalocean.com/reference/doctl/reference/compute/tag/create/)
- [Connect with SSH](https://docs.digitalocean.com/products/droplets/how-to/connect-with-ssh/)

## Baseline and candidate on one Droplet

Use a retained session when claiming that code became faster or slower:

```bash
scripts/dev/benchmark-remote.sh create
scripts/dev/benchmark-remote.sh sync
scripts/dev/benchmark-remote.sh exec \
  make bench-baseline BENCH_PACKAGE=./internal/app/orders \
  BENCH_PATTERN=BenchmarkCalculateTotal \
  BENCH_WORKLOAD_ID=orders-100-lines

# Move the local checkout to the candidate without losing user work.

scripts/dev/benchmark-remote.sh sync
scripts/dev/benchmark-remote.sh exec \
  make bench BENCH_PACKAGE=./internal/app/orders \
  BENCH_PATTERN=BenchmarkCalculateTotal \
  BENCH_WORKLOAD_ID=orders-100-lines
scripts/dev/benchmark-remote.sh exec make bench-compare
scripts/dev/benchmark-remote.sh fetch
scripts/dev/benchmark-remote.sh destroy
```

`sync` replaces the remote source so deleted files do not survive, while
preserving `.artifacts/bench` between baseline and candidate. It records the
local revision, dirty state, and a SHA-256 fingerprint of every transferred
path, mode, and content object without uploading `.git`. The fingerprint is
recomputed after transfer and execution remains blocked if local source changed
mid-sync. For a candidate that already exists in the current dirty checkout,
materialize the baseline in a separate clean worktree and invoke `sync` there
with the same absolute `DO_BENCH_STATE_FILE`; do not reset or overwrite user
work.

For a small or noisy decision-critical delta, follow `docs/benchmarking.md`:
alternate baseline and candidate batches on the same Droplet and retain each
raw batch. A single baseline-then-candidate sequence is only adequate for a
stable material delta.

## External HTTP load

A single Droplet is acceptable for scenario wiring and bounded low load. It
cannot establish decision-grade service capacity because the application and
k6 contend for the same CPU and network.

For decision-grade load, create two same-region sessions:

```bash
target_state="$PWD/.artifacts/bench/remote/target.state"
generator_state="$PWD/.artifacts/bench/remote/generator.state"

cleanup() {
  DO_BENCH_STATE_FILE="$generator_state" scripts/dev/benchmark-remote.sh fetch || true
  DO_BENCH_STATE_FILE="$target_state" scripts/dev/benchmark-remote.sh fetch || true
  DO_BENCH_STATE_FILE="$generator_state" scripts/dev/benchmark-remote.sh destroy || true
  DO_BENCH_STATE_FILE="$target_state" scripts/dev/benchmark-remote.sh destroy || true
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

DO_BENCH_STATE_FILE="$target_state" scripts/dev/benchmark-remote.sh create
DO_BENCH_STATE_FILE="$target_state" \
DO_BENCH_ENV_FILE="$HOME/.config/my-service/benchmark-target.env" \
scripts/dev/benchmark-remote.sh sync
DO_BENCH_STATE_FILE="$target_state" scripts/dev/benchmark-remote.sh exec \
  docker compose --env-file .env.bench up -d

DO_BENCH_STATE_FILE="$generator_state" scripts/dev/benchmark-remote.sh create
DO_BENCH_STATE_FILE="$generator_state" scripts/dev/benchmark-remote.sh sync
DO_BENCH_STATE_FILE="$target_state" scripts/dev/benchmark-remote.sh \
  allow-from-state "$generator_state" 8080
DO_BENCH_STATE_FILE="$target_state" scripts/dev/benchmark-remote.sh private-ip
```

Set `HTTP_BENCH_BASE_URL` in the generator's ignored `.env.bench` to the printed
private target address, for example `http://10.x.x.x:8080`. Resync it explicitly
and run k6 with continuous generator telemetry. The pinned scenario records
p95 and p99 in `summary.json`; define the acceptance thresholds before the run:

```bash
DO_BENCH_STATE_FILE="$generator_state" \
DO_BENCH_ENV_FILE=.env.bench \
scripts/dev/benchmark-remote.sh sync

DO_BENCH_STATE_FILE="$generator_state" \
DO_BENCH_TELEMETRY=1 \
scripts/dev/benchmark-remote.sh exec make bench-http

DO_BENCH_STATE_FILE="$generator_state" scripts/dev/benchmark-remote.sh fetch
DO_BENCH_STATE_FILE="$generator_state" scripts/dev/benchmark-remote.sh destroy
DO_BENCH_STATE_FILE="$target_state" scripts/dev/benchmark-remote.sh destroy
```

Run the sequence from one controlling shell so its `EXIT` trap owns cleanup;
use a second terminal only for the concurrent target telemetry command. If the
laptop dies, the two state files remain the recovery authority.

Run a telemetry-holding command concurrently on the target for the load window,
for example `DO_BENCH_TELEMETRY=1 ... exec sleep 10m`, or use service-native
runtime telemetry. Reject a run when k6 reports dropped iterations, the
generator has less than roughly 20% CPU idle, swap is used, or its network is
saturated. Do not solve a generator bottleneck by changing the service.

Private VPC traffic avoids public transfer and the target firewall admits only
the generator's `/32`. `allow-from-state` requires `firewall:update` and both
state files to name the same region.

Official contracts:

- [Add firewall rules](https://docs.digitalocean.com/reference/doctl/reference/compute/firewall/add-rules/)
- [DigitalOcean VPCs](https://docs.digitalocean.com/products/networking/vpc/)
- [k6 large-test guidance](https://grafana.com/docs/k6/latest/testing-guides/running-large-tests/)

## Evidence and honesty checks

Every `exec` writes a unique directory under
`.artifacts/bench/remote/runs/` containing:

- source revision, dirty state, and exact transferred-source fingerprint;
- start/end UTC times and exit status;
- kernel, CPU topology/model, memory, load, Docker, and Go information;
- optional 5-second CPU/memory (`vmstat`, per-CPU `mpstat`), disk (`iostat`),
  and network-interface (`sar`) samples.

The local provider evidence records `doctl` version, Droplet shape, realized
image, VPC, firewall rules, and creation time. Existing benchmark metadata
continues to record Go settings, CPU identity/count, schema, dependency image,
and workload identity. Preserve raw baseline/candidate output and correctness
test evidence independently.

Continuous telemetry is off by default because monitoring can perturb narrow
microbenchmarks. Enable it for database, service-capacity, and external-load
runs where host saturation is a decision variable.

## Cleanup and recovery

Normal cleanup:

```bash
scripts/dev/benchmark-remote.sh status
scripts/dev/benchmark-remote.sh fetch
scripts/dev/benchmark-remote.sh destroy
```

The state file is written before the first external create. `destroy` first
reconciles the unique resource name with successful provider list responses,
then deletes the billable Droplet, firewall, and tag in that order. It updates
state after every confirmed deletion; an already-absent resource counts only
after a successful provider lookup proves absence. The command is therefore
retryable after a crash between provider creation/deletion and local state
updates. Do not use `power-off` as cleanup.

DigitalOcean deletion is eventually consistent. After each successful delete,
`destroy` waits up to 60 seconds for provider list responses to confirm absence
before clearing that resource from the state file. A timeout fails cleanup and
retains the unresolved state; rerun the same `destroy` command instead of
assuming that an empty or stale local view stopped billing.

If the terminal or laptop dies, retain the state file and run the same destroy
command later. If the state file is lost, list resources and locate the unique
`bench-YYYYMMDD-HHMMSS-PID` name:

```bash
doctl --context benchmarks compute droplet list \
  --format ID,Name,PublicIPv4,Region,Status
doctl --context benchmarks compute firewall list \
  --format ID,Name,DropletIDs,Status
doctl --context benchmarks compute tag list --format Name
```

Then delete the matching IDs explicitly:

```bash
doctl --context benchmarks compute droplet delete <droplet-id> --force
doctl --context benchmarks compute firewall delete <firewall-id> --force
doctl --context benchmarks compute tag delete <tag-name> --force
```

Official contracts:

- [Delete a Droplet](https://docs.digitalocean.com/reference/doctl/reference/compute/droplet/delete/)
- [Delete a firewall](https://docs.digitalocean.com/reference/doctl/reference/compute/firewall/delete/)
- [Delete a tag](https://docs.digitalocean.com/reference/doctl/reference/compute/tag/delete/)

## Proof boundary

- An ephemeral CPU-Optimized Droplet supports relative baseline/candidate
  evidence collected in one session. It does not make absolute values portable
  across machines or days.
- PostgreSQL Testcontainers on local Droplet SSD storage proves the declared
  fixture and client path, not a production managed-database SLO. The default
  regular CPU-Optimized `c-4` shape does not guarantee NVMe; use a currently
  documented [Premium CPU-Optimized or Storage-Optimized
  shape](https://docs.digitalocean.com/products/droplets/concepts/choosing-a-plan/)
  only when NVMe itself is part of the claim.
- Two Droplets prove controlled HTTP throughput, latency, errors, and
  saturation for the declared topology. Production dependencies, traffic mix,
  regions, TLS, autoscaling, and noisy neighbors still require runtime evidence.
- Persistent history or a blocking regression gate requires a stable dedicated
  testbed, established variance, a material threshold, and a named owner.
