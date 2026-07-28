#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

STATE_FILE="${DO_BENCH_STATE_FILE:-.artifacts/bench/remote/digitalocean.state}"
KNOWN_HOSTS_FILE="${STATE_FILE}.known_hosts"
PROVIDER_EVIDENCE_FILE="${STATE_FILE}.provider.txt"
SSH_PRIVATE_KEY="${DO_BENCH_SSH_PRIVATE_KEY:-${HOME}/.ssh/digitalocean-bench}"
SSH_PUBLIC_KEY="${DO_BENCH_SSH_PUBLIC_KEY:-${SSH_PRIVATE_KEY}.pub}"
DOCTL_CONTEXT="${DO_BENCH_CONTEXT:-}"
REGION="${DO_BENCH_REGION:-fra1}"
SIZE="${DO_BENCH_SIZE:-c-4}"
IMAGE="${DO_BENCH_IMAGE:-ubuntu-24-04-x64}"
GOLDEN_IMAGE="${DO_BENCH_GOLDEN_IMAGE:-0}"
SSH_CIDR="${DO_BENCH_SSH_CIDR:-auto}"
ENV_FILE="${DO_BENCH_ENV_FILE:-}"
TELEMETRY="${DO_BENCH_TELEMETRY:-0}"
SSH_RETRY_DELAY="${DO_BENCH_SSH_RETRY_DELAY:-5}"
PROVIDER_POLL_DELAY="${DO_BENCH_PROVIDER_POLL_DELAY:-2}"
IMAGE_BUILD_SIZE="${DO_BENCH_IMAGE_BUILD_SIZE:-s-1vcpu-1gb}"
IMAGE_BASE="${DO_BENCH_IMAGE_BASE:-ubuntu-24-04-x64}"
IMAGE_NAME="${DO_BENCH_IMAGE_NAME:-}"
IMAGE_REFERENCE_FILE="${DO_BENCH_IMAGE_REFERENCE_FILE:-.artifacts/bench/remote/golden-image.env}"
IMAGE_GO_TOOLCHAIN="${DO_BENCH_IMAGE_GO_TOOLCHAIN:-}"
IMAGE_DOCKER_IMAGES="${DO_BENCH_IMAGE_DOCKER_IMAGES:-}"
REMOTE_USER="benchmark"
REMOTE_DIR="/opt/benchmark/source"

DROPLET_ID=""
FIREWALL_ID=""
TAG_NAME=""
PUBLIC_IP=""
PRIVATE_IP=""
DROPLET_NAME=""
SOURCE_REVISION=""
SOURCE_DIRTY=""
SOURCE_FINGERPRINT=""
CREATED_AT=""
CLOUD_INIT_TEMP=""
CLEANUP_ON_EXIT=0

DOCTL=(doctl)
if [[ -n "${DOCTL_CONTEXT}" ]]; then
	DOCTL+=(--context "${DOCTL_CONTEXT}")
fi

usage() {
	cat <<'EOF'
usage: scripts/dev/benchmark-remote.sh <command> [args]

commands:
  check                              validate local tools, credentials, key, region, size, and image
  list                               list and count all benchmark Droplets in the Team
  image-list                         list reusable benchmark snapshots in the Team
  image-build                        build one reusable snapshot and delete its builder Droplet
  create                             create and provision one protected benchmark Droplet
  sync                               replace remote source with tracked and non-ignored local files
  exec <command> [args...]           execute in the remote source tree and record host telemetry
  fetch                              download .artifacts/bench from the Droplet
  ssh                                open an interactive shell on the Droplet
  status                             show the current Droplet, firewall, and tag
  private-ip                         print the current Droplet's VPC address
  allow-from-state <state> <port>    allow TCP from another runner's private IP
  destroy                            delete the Droplet, firewall, and tag; powering off does not stop billing
  run -- <command> [args...]         create, sync, execute, fetch, and always destroy

configuration:
  DO_BENCH_CONTEXT                   optional doctl auth context
  DO_BENCH_REGION                    default: fra1
  DO_BENCH_SIZE                      default: c-4 (CPU-Optimized, dedicated vCPUs)
  DO_BENCH_IMAGE                     default: ubuntu-24-04-x64
  DO_BENCH_GOLDEN_IMAGE              1 skips package installation for a prepared snapshot; default: 0
  DO_BENCH_IMAGE_BUILD_SIZE          image builder default: s-1vcpu-1gb (25 GiB disk)
  DO_BENCH_IMAGE_BASE                image builder base; default: ubuntu-24-04-x64
  DO_BENCH_IMAGE_NAME                optional snapshot name; default includes current UTC time
  DO_BENCH_IMAGE_REFERENCE_FILE      image-build output; default: .artifacts/bench/remote/golden-image.env
  DO_BENCH_IMAGE_GO_TOOLCHAIN        image-build Go toolchain; default: local go env GOVERSION when available
  DO_BENCH_IMAGE_DOCKER_IMAGES       optional space-separated digest-pinned images to pre-pull
  DO_BENCH_SSH_PRIVATE_KEY           default: ~/.ssh/digitalocean-bench
  DO_BENCH_SSH_PUBLIC_KEY            default: <private-key>.pub
  DO_BENCH_SSH_CIDR                  default: auto-detected public IPv4/32
  DO_BENCH_STATE_FILE                use a distinct file for every concurrent runner
  DO_BENCH_ENV_FILE                  optional ignored file copied remotely as .env.bench
  DO_BENCH_TELEMETRY                 1 records CPU, memory, disk, and network every 5s; default: 0
  DO_BENCH_SSH_RETRY_DELAY           seconds between SSH retries; default: 5
  DO_BENCH_PROVIDER_POLL_DELAY       seconds between deletion checks; default: 2
EOF
}

doctl_cmd() {
	"${DOCTL[@]}" "$@"
}

set_state_file() {
	STATE_FILE="$1"
	KNOWN_HOSTS_FILE="${STATE_FILE}.known_hosts"
	PROVIDER_EVIDENCE_FILE="${STATE_FILE}.provider.txt"
}

require_command() {
	command -v "$1" >/dev/null 2>&1 || {
		echo "required command not found: $1" >&2
		return 1
	}
}

validate_ipv4() {
	local value="$1"
	local part number
	local -a parts

	IFS=. read -r -a parts <<<"${value}"
	[[ "${#parts[@]}" -eq 4 ]] || return 1
	for part in "${parts[@]}"; do
		[[ "${part}" =~ ^[0-9]+$ ]] || return 1
		number=$((10#${part}))
		((number >= 0 && number <= 255)) || return 1
	done
}

validate_cidr() {
	local value="$1"
	local address="${value%/*}"
	local prefix="${value#*/}"

	[[ "${value}" == */* ]] || return 1
	validate_ipv4 "${address}" || return 1
	[[ "${prefix}" =~ ^[0-9]+$ ]] || return 1
	((prefix >= 0 && prefix <= 32))
}

validate_boolean() {
	[[ "$1" == "0" || "$1" == "1" ]]
}

state_value() {
	local file="$1"
	local key="$2"

	awk -F= -v key="${key}" '$1 == key { sub(/^[^=]*=/, ""); print; exit }' "${file}"
}

write_state() {
	local temporary

	mkdir -p "$(dirname "${STATE_FILE}")"
	temporary="$(mktemp "${STATE_FILE}.XXXXXX")"
	chmod 600 "${temporary}"
	{
		printf 'DROPLET_ID=%s\n' "${DROPLET_ID}"
		printf 'FIREWALL_ID=%s\n' "${FIREWALL_ID}"
		printf 'TAG_NAME=%s\n' "${TAG_NAME}"
		printf 'PUBLIC_IP=%s\n' "${PUBLIC_IP}"
		printf 'PRIVATE_IP=%s\n' "${PRIVATE_IP}"
		printf 'DROPLET_NAME=%s\n' "${DROPLET_NAME}"
		printf 'REGION=%s\n' "${REGION}"
		printf 'SOURCE_REVISION=%s\n' "${SOURCE_REVISION}"
		printf 'SOURCE_DIRTY=%s\n' "${SOURCE_DIRTY}"
		printf 'SOURCE_FINGERPRINT=%s\n' "${SOURCE_FINGERPRINT}"
		printf 'CREATED_AT=%s\n' "${CREATED_AT}"
	} >"${temporary}"
	mv "${temporary}" "${STATE_FILE}"
}

load_state() {
	[[ -f "${STATE_FILE}" ]] || {
		echo "remote benchmark state not found: ${STATE_FILE}" >&2
		return 1
	}

	DROPLET_ID="$(state_value "${STATE_FILE}" DROPLET_ID)"
	FIREWALL_ID="$(state_value "${STATE_FILE}" FIREWALL_ID)"
	TAG_NAME="$(state_value "${STATE_FILE}" TAG_NAME)"
	PUBLIC_IP="$(state_value "${STATE_FILE}" PUBLIC_IP)"
	PRIVATE_IP="$(state_value "${STATE_FILE}" PRIVATE_IP)"
	DROPLET_NAME="$(state_value "${STATE_FILE}" DROPLET_NAME)"
	REGION="$(state_value "${STATE_FILE}" REGION)"
	SOURCE_REVISION="$(state_value "${STATE_FILE}" SOURCE_REVISION)"
	SOURCE_DIRTY="$(state_value "${STATE_FILE}" SOURCE_DIRTY)"
	SOURCE_FINGERPRINT="$(state_value "${STATE_FILE}" SOURCE_FINGERPRINT)"
	CREATED_AT="$(state_value "${STATE_FILE}" CREATED_AT)"

	[[ -z "${DROPLET_ID}" || "${DROPLET_ID}" =~ ^[0-9]+$ ]] || {
		echo "invalid Droplet ID in ${STATE_FILE}" >&2
		return 1
	}
	[[ -z "${FIREWALL_ID}" || "${FIREWALL_ID}" =~ ^[0-9a-f-]+$ ]] || {
		echo "invalid firewall ID in ${STATE_FILE}" >&2
		return 1
	}
	[[ -z "${DROPLET_NAME}" || "${DROPLET_NAME}" =~ ^bench-[0-9]{8}-[0-9]{6}-[0-9]+$ ]] || {
		echo "invalid Droplet name in ${STATE_FILE}" >&2
		return 1
	}
	[[ -z "${TAG_NAME}" || "${TAG_NAME}" == "${DROPLET_NAME}" ]] || {
		echo "invalid tag name in ${STATE_FILE}" >&2
		return 1
	}
	[[ -z "${PUBLIC_IP}" ]] || validate_ipv4 "${PUBLIC_IP}" || {
		echo "invalid public IP in ${STATE_FILE}" >&2
		return 1
	}
	[[ -z "${PRIVATE_IP}" ]] || validate_ipv4 "${PRIVATE_IP}" || {
		echo "invalid private IP in ${STATE_FILE}" >&2
		return 1
	}
	[[ "${REGION}" =~ ^[a-z0-9-]+$ ]] || {
		echo "invalid region in ${STATE_FILE}" >&2
		return 1
	}
	[[ -z "${SOURCE_REVISION}" || "${SOURCE_REVISION}" == "unknown" || "${SOURCE_REVISION}" =~ ^[0-9a-f]{40,64}$ ]] || {
		echo "invalid source revision in ${STATE_FILE}" >&2
		return 1
	}
	[[ -z "${SOURCE_DIRTY}" || "${SOURCE_DIRTY}" == "true" || "${SOURCE_DIRTY}" == "false" ]] || {
		echo "invalid source dirty flag in ${STATE_FILE}" >&2
		return 1
	}
	[[ -z "${SOURCE_FINGERPRINT}" || "${SOURCE_FINGERPRINT}" =~ ^[0-9a-f]{64}$ ]] || {
		echo "invalid source fingerprint in ${STATE_FILE}" >&2
		return 1
	}
}

ssh_key_fingerprint() {
	ssh-keygen -E md5 -lf "${SSH_PUBLIC_KEY}" | awk '{ sub(/^MD5:/, "", $2); print $2; exit }'
}

ssh_args() {
	printf '%s\0' \
		-i "${SSH_PRIVATE_KEY}" \
		-o IdentitiesOnly=yes \
		-o BatchMode=yes \
		-o StrictHostKeyChecking=yes \
		-o "UserKnownHostsFile=${KNOWN_HOSTS_FILE}" \
		-o ConnectTimeout=10
	if [[ "${OSTYPE:-}" == darwin* ]]; then
		printf '%s\0' -o UseKeychain=yes
	fi
}

remote_ssh() {
	local -a args=()
	while IFS= read -r -d '' value; do
		args+=("${value}")
	done < <(ssh_args)
	ssh "${args[@]}" "${REMOTE_USER}@${PUBLIC_IP}" "$@"
}

check_prerequisites() {
	local command fingerprint region_line size_line image_line

	for command in doctl ssh ssh-keygen ssh-keyscan tar git curl awk sed shasum stat; do
		require_command "${command}"
	done
	[[ -f "${SSH_PRIVATE_KEY}" ]] || {
		echo "SSH private key not found: ${SSH_PRIVATE_KEY}" >&2
		return 1
	}
	[[ -f "${SSH_PUBLIC_KEY}" ]] || {
		echo "SSH public key not found: ${SSH_PUBLIC_KEY}" >&2
		return 1
	}
	validate_boolean "${TELEMETRY}" || {
		echo "DO_BENCH_TELEMETRY must be 0 or 1" >&2
		return 1
	}
	validate_boolean "${GOLDEN_IMAGE}" || {
		echo "DO_BENCH_GOLDEN_IMAGE must be 0 or 1" >&2
		return 1
	}
	[[ "${SSH_RETRY_DELAY}" =~ ^[0-9]+([.][0-9]+)?$ ]] || {
		echo "DO_BENCH_SSH_RETRY_DELAY must be a non-negative number" >&2
		return 1
	}
	[[ "${PROVIDER_POLL_DELAY}" =~ ^[0-9]+([.][0-9]+)?$ ]] || {
		echo "DO_BENCH_PROVIDER_POLL_DELAY must be a non-negative number" >&2
		return 1
	}

	fingerprint="$(ssh_key_fingerprint)"
	[[ -n "${fingerprint}" ]] || {
		echo "failed to calculate the SSH public-key fingerprint" >&2
		return 1
	}
	doctl_cmd compute ssh-key get "${fingerprint}" --format ID,Name,FingerPrint --no-header >/dev/null

	region_line="$(doctl_cmd compute region list --format Slug,Available --no-header | awk -v region="${REGION}" '$1 == region { print; exit }')"
	[[ -n "${region_line}" && "${region_line##*[[:space:]]}" == "true" ]] || {
		echo "DigitalOcean region is unavailable or unknown: ${REGION}" >&2
		return 1
	}
	size_line="$(doctl_cmd compute size list --format Slug,VCPUs,Memory,Disk,PriceHourly,PriceMonthly --no-header | awk -v size="${SIZE}" '$1 == size { print; exit }')"
	[[ -n "${size_line}" ]] || {
		echo "DigitalOcean size is unavailable or unknown: ${SIZE}" >&2
		echo "check the Team's Resource Limits or set DO_BENCH_SIZE to an available slug" >&2
		return 1
	}
	image_line="$(doctl_cmd compute image get "${IMAGE}" --format ID,Type,Distribution,Public,MinDisk,Created --no-header)"
	[[ -n "${image_line}" ]] || {
		echo "DigitalOcean image is unavailable or unknown: ${IMAGE}" >&2
		return 1
	}
	if [[ "${GOLDEN_IMAGE}" == 1 && "$(awk '{ print $2; exit }' <<<"${image_line}")" != snapshot ]]; then
		echo "DO_BENCH_GOLDEN_IMAGE=1 requires a DigitalOcean snapshot image" >&2
		return 1
	fi

	echo "DigitalOcean benchmark runner preflight passed"
	echo "region: ${region_line}"
	echo "size (slug, vCPUs, MiB, GiB, hourly USD, monthly cap USD): ${size_line}"
	echo "image ${IMAGE} (ID, type, distribution, public, minimum disk GiB, created): ${image_line}"
	echo "SSH key fingerprint: ${fingerprint}"
	doctl version
}

list_runners() {
	local listing count

	require_command doctl
	listing="$(doctl_cmd compute droplet list --format ID,Name,Region,VCPUs,Memory,Status,Tags --no-header)"
	count="$(awk '$2 ~ /^bench-[0-9]/ { count++ } END { print count + 0 }' <<<"${listing}")"

	if [[ "${count}" -eq 0 ]]; then
		echo "no benchmark Droplets"
	else
		printf 'ID\tNAME\tREGION\tVCPUS\tMEMORY_MIB\tSTATUS\tTAGS\n'
		awk '$2 ~ /^bench-[0-9]/' <<<"${listing}"
	fi
	printf 'benchmark Droplets: %s\n' "${count}"
}

list_images() {
	local listing count

	require_command doctl
	listing="$(doctl_cmd compute image list-user --format ID,Name,Type,Distribution,MinDisk,Created --no-header)"
	count="$(awk '$2 ~ /^go-bench-/ { count++ } END { print count + 0 }' <<<"${listing}")"

	if [[ "${count}" -eq 0 ]]; then
		echo "no reusable benchmark snapshots"
	else
		printf 'ID\tNAME\tTYPE\tDISTRIBUTION\tMIN_DISK_GIB\tCREATED\n'
		awk '$2 ~ /^go-bench-/' <<<"${listing}"
	fi
	printf 'benchmark snapshots: %s\n' "${count}"
}

write_cloud_init() {
	local output="$1"
	local public_key

	public_key="$(awk 'NF >= 2 { print $1 " " $2; exit }' "${SSH_PUBLIC_KEY}")"
	[[ "${public_key}" == ssh-*\ * ]] || {
		echo "unsupported or malformed SSH public key: ${SSH_PUBLIC_KEY}" >&2
		return 1
	}

	{
		cat <<EOF
#cloud-config
EOF
		if [[ "${GOLDEN_IMAGE}" == 0 ]]; then
			cat <<'EOF'
groups:
  - docker
EOF
		fi
		cat <<EOF
users:
  - default
  - name: benchmark
    groups: [docker]
    sudo: ALL=(ALL) NOPASSWD:ALL
    shell: /bin/bash
    lock_passwd: true
    ssh_authorized_keys:
      - ${public_key}
ssh_pwauth: false
disable_root: true
ssh_deletekeys: true
EOF
		if [[ "${GOLDEN_IMAGE}" == 0 ]]; then
			cat <<'EOF'
package_update: true
package_upgrade: false
packages:
  - build-essential
  - ca-certificates
  - curl
  - docker.io
  - docker-compose-v2
  - git
  - golang-go
  - make
  - sysstat
write_files:
  - path: /etc/ssh/sshd_config.d/99-benchmark-runner.conf
    permissions: '0644'
    content: |
      PasswordAuthentication no
      KbdInteractiveAuthentication no
      PermitRootLogin no
runcmd:
  - install -d -m 0755 -o benchmark -g benchmark /opt/benchmark
  - systemctl enable --now docker
  - systemctl reload ssh
EOF
		fi
	} >"${output}"
}

resolve_ssh_cidr() {
	local address

	if [[ "${SSH_CIDR}" == "auto" ]]; then
		address="$(curl -4 -fsS --max-time 10 https://api.ipify.org)"
		SSH_CIDR="${address}/32"
	fi
	validate_cidr "${SSH_CIDR}" || {
		echo "DO_BENCH_SSH_CIDR must be an IPv4 CIDR, got '${SSH_CIDR}'" >&2
		return 1
	}
}

wait_for_ssh() {
	local attempt temporary
	local -a args=()

	mkdir -p "$(dirname "${KNOWN_HOSTS_FILE}")"
	temporary="$(mktemp "${KNOWN_HOSTS_FILE}.XXXXXX")"
	for attempt in $(seq 1 60); do
		if ssh-keyscan -T 5 -H "${PUBLIC_IP}" >"${temporary}" 2>/dev/null && [[ -s "${temporary}" ]]; then
			mv "${temporary}" "${KNOWN_HOSTS_FILE}"
			chmod 600 "${KNOWN_HOSTS_FILE}"
			break
		fi
		sleep "${SSH_RETRY_DELAY}"
	done
	[[ -s "${KNOWN_HOSTS_FILE}" ]] || {
		rm -f "${temporary}"
		echo "SSH host key was not available within the configured retry window" >&2
		return 1
	}

	while IFS= read -r -d '' value; do
		args+=("${value}")
	done < <(ssh_args)
	for attempt in $(seq 1 60); do
		if ssh "${args[@]}" "${REMOTE_USER}@${PUBLIC_IP}" true >/dev/null 2>&1; then
			return 0
		fi
		sleep "${SSH_RETRY_DELAY}"
	done
	echo "SSH authentication was not available within the configured retry window" >&2
	return 1
}

wait_for_cloud_init() {
	local attempt status

	for attempt in $(seq 1 60); do
		if remote_ssh sudo cloud-init status --wait; then
			return 0
		else
			status=$?
		fi
		[[ "${status}" -eq 255 ]] || return "${status}"
		echo "SSH connection was interrupted while waiting for cloud-init; retrying" >&2
		sleep "${SSH_RETRY_DELAY}"
	done
	echo "SSH connection did not remain available within the configured retry window" >&2
	return 1
}

create_runner() {
	local fingerprint droplet_output firewall_output addresses create_started_epoch ready_at

	[[ ! -e "${STATE_FILE}" ]] || {
		echo "remote benchmark state already exists: ${STATE_FILE}" >&2
		echo "destroy that runner or choose another DO_BENCH_STATE_FILE" >&2
		return 1
	}
	check_prerequisites
	resolve_ssh_cidr
	fingerprint="$(ssh_key_fingerprint)"
	CLOUD_INIT_TEMP="$(mktemp)"
	write_cloud_init "${CLOUD_INIT_TEMP}"

	DROPLET_NAME="bench-$(date -u +%Y%m%d-%H%M%S)-$$"
	TAG_NAME="${DROPLET_NAME}"
	CREATED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	create_started_epoch="$(date +%s)"
	CLEANUP_ON_EXIT=1
	write_state
	doctl_cmd compute tag create "${TAG_NAME}"

	firewall_output="$(doctl_cmd compute firewall create \
		--name "${DROPLET_NAME}" \
		--inbound-rules "protocol:tcp,ports:22,address:${SSH_CIDR}" \
		--outbound-rules "protocol:tcp,ports:all,address:0.0.0.0/0 protocol:udp,ports:all,address:0.0.0.0/0 protocol:icmp,address:0.0.0.0/0" \
		--tag-names "${TAG_NAME}" \
		--format ID \
		--no-header)"
	FIREWALL_ID="$(awk 'NR == 1 { print $1 }' <<<"${firewall_output}")"
	[[ "${FIREWALL_ID}" =~ ^[0-9a-f-]+$ ]] || {
		echo "failed to read the created firewall ID" >&2
		return 1
	}
	write_state

	droplet_output="$(doctl_cmd compute droplet create "${DROPLET_NAME}" \
		--region "${REGION}" \
		--size "${SIZE}" \
		--image "${IMAGE}" \
		--ssh-keys "${fingerprint}" \
		--tag-names "${TAG_NAME}" \
		--enable-private-networking \
		--droplet-agent=false \
		--user-data-file "${CLOUD_INIT_TEMP}" \
		--wait \
		--format ID \
		--no-header)"
	DROPLET_ID="$(awk 'NR == 1 { print $1 }' <<<"${droplet_output}")"
	[[ "${DROPLET_ID}" =~ ^[0-9]+$ ]] || {
		echo "failed to read the created Droplet ID" >&2
		return 1
	}
	write_state

	addresses="$(doctl_cmd compute droplet get "${DROPLET_ID}" --format PublicIPv4,PrivateIPv4 --no-header)"
	PUBLIC_IP="$(awk 'NR == 1 { print $1 }' <<<"${addresses}")"
	PRIVATE_IP="$(awk 'NR == 1 { print $2 }' <<<"${addresses}")"
	validate_ipv4 "${PUBLIC_IP}" || {
		echo "created Droplet has no valid public IPv4 address" >&2
		return 1
	}
	validate_ipv4 "${PRIVATE_IP}" || {
		echo "created Droplet has no valid VPC IPv4 address" >&2
		return 1
	}
	write_state

	wait_for_ssh
	if ! wait_for_cloud_init; then
		remote_ssh sudo cloud-init status --long || true
		remote_ssh sudo tail -n 200 /var/log/cloud-init-output.log || true
		return 1
	fi
	remote_ssh 'docker version >/dev/null && go version && getconf _NPROCESSORS_ONLN'
	ready_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
	mkdir -p "$(dirname "${PROVIDER_EVIDENCE_FILE}")"
	{
		printf 'created_at_utc=%s\n' "${CREATED_AT}"
		printf 'ready_at_utc=%s\n' "${ready_at}"
		printf 'ready_after_seconds=%s\n' "$(( $(date +%s) - create_started_epoch ))"
		printf 'ssh_cidr=%s\n' "${SSH_CIDR}"
		doctl version
		doctl_cmd compute droplet get "${DROPLET_ID}" --format ID,Name,PublicIPv4,PrivateIPv4,VCPUs,Memory,Disk,Region,Image,VPCUUID,Status,Tags
		doctl_cmd compute firewall get "${FIREWALL_ID}" --format ID,Name,Status,DropletIDs,Tags,InboundRules,OutboundRules
	} >"${PROVIDER_EVIDENCE_FILE}"
	rm -f "${CLOUD_INIT_TEMP}"
	CLOUD_INIT_TEMP=""
	CLEANUP_ON_EXIT=0
	echo "benchmark runner ready: ${DROPLET_NAME} (${PUBLIC_IP}, private ${PRIVATE_IP})"
	echo "state: ${STATE_FILE}"
}

wait_for_power_off() {
	local attempt status

	for attempt in $(seq 1 60); do
		if status="$(doctl_cmd compute droplet get "${DROPLET_ID}" --format Status --no-header)"; then
			[[ "${status}" != off ]] || return 0
		fi
		sleep "${PROVIDER_POLL_DELAY}"
	done
	echo "Droplet ${DROPLET_ID} did not power off within the configured polling window" >&2
	return 1
}

prepare_snapshot() {
	local shutdown_status=0

	if remote_ssh 'sudo rm -rf /opt/benchmark/source /home/benchmark/.cache && sudo rm -f /home/benchmark/.ssh/authorized_keys /root/.ssh/authorized_keys /etc/ssh/ssh_host_* && sudo cloud-init clean --logs --machine-id && sudo shutdown -h now'; then
		:
	else
		shutdown_status=$?
		[[ "${shutdown_status}" -eq 255 ]] || return "${shutdown_status}"
	fi
	wait_for_power_off
}

warm_image_dependencies() {
	local postgres_image k6_image image
	local -a images=()

	if [[ -z "${IMAGE_GO_TOOLCHAIN}" ]] && command -v go >/dev/null 2>&1; then
		IMAGE_GO_TOOLCHAIN="$(go env GOVERSION)"
	fi
	if [[ -n "${IMAGE_GO_TOOLCHAIN}" ]]; then
		[[ "${IMAGE_GO_TOOLCHAIN}" =~ ^go[0-9]+\.[0-9]+(\.[0-9]+)?$ ]] || {
			echo "DO_BENCH_IMAGE_GO_TOOLCHAIN must look like go1.26.5" >&2
			return 1
		}
		remote_ssh env "GOTOOLCHAIN=${IMAGE_GO_TOOLCHAIN}" go version
	fi

	if [[ -z "${IMAGE_DOCKER_IMAGES}" ]]; then
		if [[ -r internal/infra/postgres/pgtest/pgtest.go ]]; then
			postgres_image="$(sed -n 's/^const DefaultImage = "\(.*\)"$/\1/p' internal/infra/postgres/pgtest/pgtest.go)"
		fi
		if [[ -r scripts/dev/benchmark.sh ]]; then
			k6_image="$(sed -n 's/^K6_IMAGE_DEFAULT="\(.*\)"$/\1/p' scripts/dev/benchmark.sh)"
		fi
		IMAGE_DOCKER_IMAGES="${postgres_image:-} ${k6_image:-}"
	fi
	read -r -a images <<<"${IMAGE_DOCKER_IMAGES}"
	for image in "${images[@]}"; do
		[[ -n "${image}" ]] || continue
		[[ "${image}" =~ ^[^[:space:]]+@sha256:[[:xdigit:]]{64}$ ]] || {
			echo "snapshot Docker images must be digest-pinned, got '${image}'" >&2
			return 1
		}
		remote_ssh docker pull "${image}"
	done
}

write_image_reference() {
	local snapshot_id="$1"
	local snapshot_name="$2"
	local temporary

	mkdir -p "$(dirname "${IMAGE_REFERENCE_FILE}")"
	temporary="$(mktemp "${IMAGE_REFERENCE_FILE}.XXXXXX")"
	chmod 600 "${temporary}"
	{
		printf '# Generated by scripts/dev/benchmark-remote.sh image-build. No secrets.\n'
		printf 'export DO_BENCH_IMAGE=%s\n' "${snapshot_id}"
		printf 'export DO_BENCH_GOLDEN_IMAGE=1\n'
		printf 'export DO_BENCH_IMAGE_NAME=%s\n' "${snapshot_name}"
	} >"${temporary}"
	mv "${temporary}" "${IMAGE_REFERENCE_FILE}"
}

build_image() {
	local snapshot_name snapshot_line snapshot_id destroy_status=0

	SIZE="${IMAGE_BUILD_SIZE}"
	IMAGE="${IMAGE_BASE}"
	GOLDEN_IMAGE=0
	if [[ -z "${DO_BENCH_STATE_FILE+x}" ]]; then
		set_state_file ".artifacts/bench/remote/image-builder.state"
	fi
	snapshot_name="${IMAGE_NAME:-go-bench-$(date -u +%Y%m%d-%H%M%S)}"
	[[ "${snapshot_name}" =~ ^go-bench-[a-zA-Z0-9._-]+$ ]] || {
		echo "DO_BENCH_IMAGE_NAME must start with go-bench- and contain only letters, numbers, dot, underscore, or hyphen" >&2
		return 1
	}

	create_runner
	CLEANUP_ON_EXIT=1
	warm_image_dependencies
	prepare_snapshot
	doctl_cmd compute droplet-action snapshot "${DROPLET_ID}" --snapshot-name "${snapshot_name}" --wait >/dev/null
	snapshot_line="$(doctl_cmd compute droplet snapshots "${DROPLET_ID}" --format ID,Name,Type,Distribution,MinDisk,Created --no-header | awk -v name="${snapshot_name}" '$2 == name { print; exit }')"
	snapshot_id="$(awk '{ print $1; exit }' <<<"${snapshot_line}")"
	[[ "${snapshot_id}" =~ ^[0-9]+$ ]] || {
		echo "snapshot action completed but image '${snapshot_name}' could not be resolved" >&2
		return 1
	}
	{
		printf 'snapshot_created_at_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
		printf 'snapshot=%s\n' "${snapshot_line}"
		printf 'snapshot_go_toolchain=%s\n' "${IMAGE_GO_TOOLCHAIN:-not-preloaded}"
		printf 'snapshot_docker_images=%s\n' "${IMAGE_DOCKER_IMAGES:-none}"
	} >>"${PROVIDER_EVIDENCE_FILE}"
	echo "snapshot created; cleaning up builder next: ${snapshot_line}"
	write_image_reference "${snapshot_id}" "${snapshot_name}"
	if destroy_runner; then
		CLEANUP_ON_EXIT=0
	else
		destroy_status=$?
	fi
	[[ "${destroy_status}" -eq 0 ]] || return "${destroy_status}"

	echo "reusable benchmark snapshot ready: ${snapshot_line}"
	echo "runtime configuration: source ${IMAGE_REFERENCE_FILE}"
}

source_files() {
	local file

	while IFS= read -r -d '' file; do
		if [[ -e "${file}" || -L "${file}" ]]; then
			printf '%s\0' "${file}"
		fi
	done < <(git ls-files --cached --others --exclude-standard -z)
}

source_object_id() {
	local file="$1"
	local link_target

	if [[ ! -L "${file}" ]]; then
		git hash-object --no-filters -- "${file}"
		return
	fi
	if ! link_target="$({ readlink "./${file}" || exit; printf x; })"; then
		echo "failed to read benchmark source symlink: ${file}" >&2
		return 1
	fi
	link_target="${link_target%x}"
	printf '%s' "${link_target}" | git hash-object --stdin
}

source_fingerprint() {
	local file mode object_id

	{
		while IFS= read -r -d '' file; do
			if mode="$(stat -f '%Lp' -- "${file}" 2>/dev/null)"; then
				:
			else
				mode="$(stat -c '%a' -- "${file}")"
			fi
			object_id="$(source_object_id "${file}")"
			printf '%s\0%s\0%s\0' "${mode}" "${object_id}" "${file}"
		done < <(source_files)
	} | shasum -a 256 | awk '{ print $1 }'
}

sync_source() {
	local env_path fingerprint_after

	load_state
	[[ -n "${DROPLET_ID}" && -n "${PUBLIC_IP}" ]] || {
		echo "state does not contain an active Droplet" >&2
		return 1
	}
	SOURCE_REVISION=""
	SOURCE_DIRTY=""
	SOURCE_FINGERPRINT=""
	write_state

	SOURCE_REVISION="$(git rev-parse --verify HEAD 2>/dev/null || printf unknown)"
	if [[ -n "$(git status --porcelain --untracked-files=normal)" ]]; then
		SOURCE_DIRTY=true
	else
		SOURCE_DIRTY=false
	fi
	SOURCE_FINGERPRINT="$(source_fingerprint)"

	source_files |
		COPYFILE_DISABLE=1 tar --no-xattrs --null -czf - --files-from=- |
		remote_ssh "set -e; rm -rf '${REMOTE_DIR}.next'; mkdir -p '${REMOTE_DIR}.next'; tar -xzf - -C '${REMOTE_DIR}.next'; if [ -d '${REMOTE_DIR}/.artifacts/bench' ]; then mkdir -p '${REMOTE_DIR}.next/.artifacts'; mv '${REMOTE_DIR}/.artifacts/bench' '${REMOTE_DIR}.next/.artifacts/bench'; fi; rm -rf '${REMOTE_DIR}'; mv '${REMOTE_DIR}.next' '${REMOTE_DIR}'"

	if [[ -n "${ENV_FILE}" ]]; then
		env_path="${ENV_FILE}"
		[[ "${env_path}" = /* ]] || env_path="${ROOT_DIR}/${env_path}"
		[[ -f "${env_path}" ]] || {
			echo "DO_BENCH_ENV_FILE not found: ${env_path}" >&2
			return 1
		}
		remote_ssh "umask 077; cat > '${REMOTE_DIR}/.env.bench'" <"${env_path}"
	fi
	fingerprint_after="$(source_fingerprint)"
	[[ "${SOURCE_FINGERPRINT}" == "${fingerprint_after}" ]] || {
		SOURCE_REVISION=""
		SOURCE_DIRTY=""
		SOURCE_FINGERPRINT=""
		write_state
		echo "local source changed during synchronization; run sync again" >&2
		return 1
	}
	write_state
	echo "source synchronized: revision=${SOURCE_REVISION} dirty=${SOURCE_DIRTY} fingerprint=${SOURCE_FINGERPRINT}"
}

shell_join() {
	local argument quoted result=""
	for argument in "$@"; do
		printf -v quoted '%q' "${argument}"
		result+="${result:+ }${quoted}"
	done
	printf '%s' "${result}"
}

execute_remote() {
	local command_line status

	[[ "$#" -gt 0 ]] || {
		echo "exec requires a command" >&2
		return 1
	}
	load_state
	[[ -n "${SOURCE_REVISION}" && -n "${SOURCE_DIRTY}" && -n "${SOURCE_FINGERPRINT}" ]] || {
		echo "sync the source before executing a benchmark" >&2
		return 1
	}
	validate_boolean "${TELEMETRY}" || {
		echo "DO_BENCH_TELEMETRY must be 0 or 1" >&2
		return 1
	}
	command_line="$(shell_join "$@")"

	set +e
	remote_ssh bash -s <<EOF
set -uo pipefail
cd '${REMOTE_DIR}'
	export BENCH_SOURCE_REVISION='${SOURCE_REVISION}'
	export BENCH_SOURCE_DIRTY='${SOURCE_DIRTY}'
	export BENCH_SOURCE_FINGERPRINT='${SOURCE_FINGERPRINT}'
run_dir=".artifacts/bench/remote/runs/\$(date -u +%Y%m%dT%H%M%SZ)-\$\$"
mkdir -p "\${run_dir}"
{
  date -u +recorded_at_utc=%Y-%m-%dT%H:%M:%SZ
  uname -a
  lscpu
  free -b
  cat /proc/loadavg
  docker version 2>/dev/null || true
  go version 2>/dev/null || true
} >"\${run_dir}/host-before.txt"
telemetry_pids=""
cleanup_telemetry() {
  if [[ -n "\${telemetry_pids}" ]]; then
    kill \${telemetry_pids} 2>/dev/null || true
    wait \${telemetry_pids} 2>/dev/null || true
  fi
}
trap cleanup_telemetry EXIT
if [[ '${TELEMETRY}' == '1' ]]; then
  vmstat 5 >"\${run_dir}/vmstat.txt" & telemetry_pids="\$!"
  mpstat -P ALL 5 >"\${run_dir}/mpstat.txt" & telemetry_pids="\${telemetry_pids} \$!"
  iostat -xz 5 >"\${run_dir}/iostat.txt" & telemetry_pids="\${telemetry_pids} \$!"
  sar -n DEV 5 >"\${run_dir}/network.txt" & telemetry_pids="\${telemetry_pids} \$!"
fi
set +e
${command_line}
status=\$?
set -e
cleanup_telemetry
trap - EXIT
{
  printf 'exit_status=%s\n' "\${status}"
  printf 'source_revision=%s\n' '${SOURCE_REVISION}'
  printf 'source_dirty=%s\n' '${SOURCE_DIRTY}'
  printf 'source_fingerprint=%s\n' '${SOURCE_FINGERPRINT}'
  date -u +finished_at_utc=%Y-%m-%dT%H:%M:%SZ
  uptime
  free -b
  cat /proc/loadavg
} >"\${run_dir}/host-after.txt"
exit "\${status}"
EOF
	status=$?
	set -e
	return "${status}"
}

fetch_artifacts() {
	load_state
	mkdir -p "${ROOT_DIR}"
	remote_ssh "cd '${REMOTE_DIR}' && test -d .artifacts/bench && tar -czf - .artifacts/bench" |
		tar -xzf - -C "${ROOT_DIR}"
	echo "benchmark artifacts downloaded to .artifacts/bench"
}

show_status() {
	load_state
	if [[ -n "${DROPLET_ID}" ]]; then
		doctl_cmd compute droplet get "${DROPLET_ID}" --format ID,Name,PublicIPv4,PrivateIPv4,VCPUs,Memory,Region,Image,Status
	else
		echo "Droplet already deleted"
	fi
	if [[ -n "${FIREWALL_ID}" ]]; then
		doctl_cmd compute firewall get "${FIREWALL_ID}" --format ID,Name,Status,DropletIDs,InboundRules
	else
		echo "firewall already deleted"
	fi
	if [[ -n "${TAG_NAME}" ]]; then
		doctl_cmd compute tag list --format Name --no-header | awk -v name="${TAG_NAME}" '$1 == name { print "tag: " $1; found=1 } END { if (!found) print "tag already deleted" }'
	else
		echo "tag already deleted"
	fi
}

open_shell() {
	load_state
	remote_ssh
}

allow_from_state() {
	local source_state="$1"
	local port="$2"
	local source_private source_region

	load_state
	[[ -n "${FIREWALL_ID}" ]] || {
		echo "target state has no active firewall" >&2
		return 1
	}
	[[ -f "${source_state}" ]] || {
		echo "source runner state not found: ${source_state}" >&2
		return 1
	}
	source_private="$(state_value "${source_state}" PRIVATE_IP)"
	source_region="$(state_value "${source_state}" REGION)"
	validate_ipv4 "${source_private}" || {
		echo "source state has no valid private IP" >&2
		return 1
	}
	[[ "${source_region}" == "${REGION}" ]] || {
		echo "target and source runners must use the same region/VPC" >&2
		return 1
	}
	[[ "${port}" =~ ^[0-9]+$ ]] && ((port >= 1 && port <= 65535)) || {
		echo "port must be between 1 and 65535" >&2
		return 1
	}
	doctl_cmd compute firewall add-rules "${FIREWALL_ID}" \
		--inbound-rules "protocol:tcp,ports:${port},address:${source_private}/32"
	echo "allowed TCP ${port} from ${source_private}/32"
}

reconcile_resources() {
	local droplet_list firewall_list tag_list recovered
	local failed=0 changed=0

	if droplet_list="$(doctl_cmd compute droplet list --format ID,Name --no-header)"; then
		recovered="$(awk -v name="${DROPLET_NAME}" '$2 == name { print $1; exit }' <<<"${droplet_list}")"
		if [[ -n "${recovered}" && ! "${recovered}" =~ ^[0-9]+$ ]]; then
			echo "invalid recovered Droplet ID: ${recovered}" >&2
			failed=1
		elif [[ "${DROPLET_ID}" != "${recovered}" ]]; then
			DROPLET_ID="${recovered}"
			if [[ -z "${recovered}" ]]; then
				PUBLIC_IP=""
				PRIVATE_IP=""
			fi
			changed=1
		fi
	else
		echo "failed to reconcile Droplet ${DROPLET_NAME}" >&2
		failed=1
	fi

	if firewall_list="$(doctl_cmd compute firewall list --format ID,Name --no-header)"; then
		recovered="$(awk -v name="${DROPLET_NAME}" '$2 == name { print $1; exit }' <<<"${firewall_list}")"
		if [[ -n "${recovered}" && ! "${recovered}" =~ ^[0-9a-f-]+$ ]]; then
			echo "invalid recovered firewall ID: ${recovered}" >&2
			failed=1
		elif [[ "${FIREWALL_ID}" != "${recovered}" ]]; then
			FIREWALL_ID="${recovered}"
			changed=1
		fi
	else
		echo "failed to reconcile firewall ${DROPLET_NAME}" >&2
		failed=1
	fi

	if tag_list="$(doctl_cmd compute tag list --format Name --no-header)"; then
		if [[ -n "${TAG_NAME}" ]] && ! awk -v name="${TAG_NAME}" '$1 == name { found=1 } END { exit !found }' <<<"${tag_list}"; then
			TAG_NAME=""
			changed=1
		fi
	else
		echo "failed to reconcile tag ${TAG_NAME}" >&2
		failed=1
	fi

	if [[ "${changed}" -eq 1 ]]; then
		write_state
	fi
	return "${failed}"
}

droplet_absent() {
	local listing

	listing="$(doctl_cmd compute droplet list --format ID,Name --no-header)" || return 1
	! awk -v id="${DROPLET_ID}" -v name="${DROPLET_NAME}" '$1 == id || $2 == name { found=1 } END { exit !found }' <<<"${listing}"
}

firewall_absent() {
	local listing

	listing="$(doctl_cmd compute firewall list --format ID,Name --no-header)" || return 1
	! awk -v id="${FIREWALL_ID}" -v name="${DROPLET_NAME}" '$1 == id || $2 == name { found=1 } END { exit !found }' <<<"${listing}"
}

tag_absent() {
	local listing

	listing="$(doctl_cmd compute tag list --format Name --no-header)" || return 1
	! awk -v name="${TAG_NAME}" '$1 == name { found=1 } END { exit !found }' <<<"${listing}"
}

wait_for_absence() {
	local resource="$1"
	local check="$2"
	local attempt

	for attempt in $(seq 1 30); do
		if "${check}"; then
			return 0
		fi
		sleep "${PROVIDER_POLL_DELAY}"
	done
	echo "${resource} deletion was not confirmed within 60 seconds" >&2
	return 1
}

destroy_runner() {
	local failed=0

	if [[ ! -f "${STATE_FILE}" ]]; then
		echo "no remote benchmark state to destroy: ${STATE_FILE}"
		return 0
	fi
	load_state
	if ! reconcile_resources; then
		failed=1
	fi
	if [[ -n "${DROPLET_ID}" ]]; then
		if doctl_cmd compute droplet delete "${DROPLET_ID}" --force; then
			if wait_for_absence "Droplet ${DROPLET_ID}" droplet_absent; then
				echo "deleted Droplet ${DROPLET_ID}; billing stopped"
				printf 'destroyed_at_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"${PROVIDER_EVIDENCE_FILE}"
				DROPLET_ID=""
				PUBLIC_IP=""
				PRIVATE_IP=""
				write_state
			else
				failed=1
			fi
		elif droplet_absent; then
			echo "Droplet ${DROPLET_ID} is already absent; billing is stopped"
			DROPLET_ID=""
			PUBLIC_IP=""
			PRIVATE_IP=""
			write_state
		else
			echo "failed to delete Droplet ${DROPLET_ID}" >&2
			failed=1
		fi
	fi
	if [[ -n "${FIREWALL_ID}" ]]; then
		if doctl_cmd compute firewall delete "${FIREWALL_ID}" --force; then
			if wait_for_absence "firewall ${FIREWALL_ID}" firewall_absent; then
				echo "deleted firewall ${FIREWALL_ID}"
				FIREWALL_ID=""
				write_state
			else
				failed=1
			fi
		elif firewall_absent; then
			echo "firewall ${FIREWALL_ID} is already absent"
			FIREWALL_ID=""
			write_state
		else
			echo "failed to delete firewall ${FIREWALL_ID}" >&2
			failed=1
		fi
	fi
	if [[ -n "${TAG_NAME}" ]]; then
		if doctl_cmd compute tag delete "${TAG_NAME}" --force; then
			if wait_for_absence "tag ${TAG_NAME}" tag_absent; then
				echo "deleted tag ${TAG_NAME}"
				TAG_NAME=""
				write_state
			else
				failed=1
			fi
		elif tag_absent; then
			echo "tag ${TAG_NAME} is already absent"
			TAG_NAME=""
			write_state
		else
			echo "failed to delete tag ${TAG_NAME}" >&2
			failed=1
		fi
	fi
	if [[ "${failed}" -eq 0 && -z "${DROPLET_ID}" && -z "${FIREWALL_ID}" && -z "${TAG_NAME}" ]]; then
		rm -f "${STATE_FILE}" "${KNOWN_HOSTS_FILE}"
	fi
	return "${failed}"
}

cleanup_on_exit() {
	local status=$?
	if [[ -n "${CLOUD_INIT_TEMP}" ]]; then
		rm -f "${CLOUD_INIT_TEMP}"
	fi
	if [[ "${CLEANUP_ON_EXIT}" -eq 1 ]]; then
		echo "cleaning up DigitalOcean benchmark resources" >&2
		if [[ ! -f "${STATE_FILE}" && -n "${DROPLET_NAME}" ]]; then
			doctl_cmd compute droplet delete "${DROPLET_NAME}" --force >/dev/null 2>&1 || true
		fi
		destroy_runner || echo "cleanup incomplete; retry: DO_BENCH_STATE_FILE='${STATE_FILE}' $0 destroy" >&2
	fi
	return "${status}"
}

run_once() {
	local command_status=0 fetch_status=0 destroy_status=0

	[[ "${1:-}" == "--" ]] || {
		echo "run requires -- before the remote command" >&2
		return 1
	}
	shift
	[[ "$#" -gt 0 ]] || {
		echo "run requires a remote command" >&2
		return 1
	}
	create_runner
	CLEANUP_ON_EXIT=1
	sync_source
	if execute_remote "$@"; then
		:
	else
		command_status=$?
	fi
	if fetch_artifacts; then
		:
	else
		fetch_status=$?
	fi
	if destroy_runner; then
		CLEANUP_ON_EXIT=0
	else
		destroy_status=$?
	fi
	if [[ "${command_status}" -ne 0 ]]; then
		return "${command_status}"
	fi
	if [[ "${fetch_status}" -ne 0 ]]; then
		return "${fetch_status}"
	fi
	return "${destroy_status}"
}

trap cleanup_on_exit EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

command="${1:-}"
shift || true
case "${command}" in
check)
	check_prerequisites
	;;
list)
	list_runners
	;;
image-list)
	list_images
	;;
image-build)
	[[ "$#" -eq 0 ]] || {
		usage
		exit 1
	}
	build_image
	;;
create)
	create_runner
	;;
sync)
	sync_source
	;;
exec)
	execute_remote "$@"
	;;
fetch)
	fetch_artifacts
	;;
ssh)
	open_shell
	;;
status)
	show_status
	;;
private-ip)
	load_state
	printf '%s\n' "${PRIVATE_IP}"
	;;
allow-from-state)
	[[ "$#" -eq 2 ]] || {
		usage
		exit 1
	}
	allow_from_state "$1" "$2"
	;;
destroy)
	destroy_runner
	;;
run)
	run_once "$@"
	;;
*)
	usage
	exit 1
	;;
esac
