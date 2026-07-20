#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEMP_DIR="$(mktemp -d)"
FIXTURE_DIR="${TEMP_DIR}/fixture"
FAKE_BIN="${TEMP_DIR}/bin"
FAKE_DO_ROOT="${TEMP_DIR}/provider"
FAKE_DO_LOG="${TEMP_DIR}/doctl.log"
FAKE_SSH_LOG="${TEMP_DIR}/ssh.log"
STATE_FILE="${TEMP_DIR}/digitalocean.state"
SSH_KEY="${TEMP_DIR}/id_ed25519"

cleanup() {
	rm -rf "${TEMP_DIR}"
}
trap cleanup EXIT

fail() {
	echo "remote benchmark lifecycle check failed: $*" >&2
	exit 1
}

state_value() {
	awk -F= -v key="$2" '$1 == key { sub(/^[^=]*=/, ""); print; exit }' "$1"
}

command_line() {
	local pattern="$1"
	awk -v pattern="${pattern}" 'index($0, pattern) { print NR; exit }' "${FAKE_DO_LOG}"
}

mkdir -p "${FIXTURE_DIR}/scripts/dev" "${FIXTURE_DIR}/test" "${FAKE_BIN}" "${FAKE_DO_ROOT}"
cp "${ROOT_DIR}/scripts/dev/benchmark-remote.sh" "${FIXTURE_DIR}/scripts/dev/benchmark-remote.sh"
cp "${ROOT_DIR}/scripts/dev/benchmark.sh" "${FIXTURE_DIR}/scripts/dev/benchmark.sh"
cp "${ROOT_DIR}/test/postgres_integration_test.go" "${FIXTURE_DIR}/test/postgres_integration_test.go"
printf 'benchmark fixture v1\n' >"${FIXTURE_DIR}/README.md"
git -C "${FIXTURE_DIR}" init -q
git -C "${FIXTURE_DIR}" add README.md scripts/dev/benchmark-remote.sh scripts/dev/benchmark.sh test/postgres_integration_test.go
git -C "${FIXTURE_DIR}" -c user.name=benchmark-check -c user.email=benchmark-check@example.invalid commit -qm fixture
ssh-keygen -q -t ed25519 -N '' -f "${SSH_KEY}"

cat >"${FAKE_BIN}/doctl" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

mkdir -p "${FAKE_DO_ROOT}"
printf '%s\n' "$*" >>"${FAKE_DO_LOG}"

case "${1:-} ${2:-} ${3:-}" in
"version  ")
	echo 'doctl version 1.146.0-release'
	;;
"compute ssh-key get")
	echo '1 benchmark aa:bb:cc'
	;;
"compute region list")
	echo 'fra1 true'
	;;
"compute size list")
	echo 'c-4 4 8192 50 0.125 84.00'
	echo 's-1vcpu-1gb 1 1024 25 0.00893 6.00'
	;;
"compute image get")
	if [[ "${4:-}" == '654321' ]]; then
		echo '654321 snapshot Ubuntu false 25 2026-01-01T00:00:00Z'
	else
		echo '123 base Ubuntu true 7 2026-01-01T00:00:00Z'
	fi
	;;
"compute image list-user")
	if [[ -f "${FAKE_DO_ROOT}/snapshot" ]]; then
		printf '654321 %s snapshot Ubuntu 25 2026-01-01T00:00:00Z\n' "$(cat "${FAKE_DO_ROOT}/snapshot")"
	fi
	;;
"compute tag create")
	printf '%s\n' "$4" >"${FAKE_DO_ROOT}/tag"
	echo "$4"
	;;
"compute tag list")
	[[ ! -f "${FAKE_DO_ROOT}/tag" ]] || cat "${FAKE_DO_ROOT}/tag"
	;;
"compute tag delete")
	[[ -f "${FAKE_DO_ROOT}/tag" ]] || exit 1
	rm -f "${FAKE_DO_ROOT}/tag"
	;;
"compute firewall create")
	name=''
	previous=''
	for argument in "$@"; do
		if [[ "${previous}" == '--name' ]]; then
			name="${argument}"
			break
		fi
		previous="${argument}"
	done
	[[ -n "${name}" ]]
	printf '%s\n' "${name}" >"${FAKE_DO_ROOT}/firewall"
	echo '11111111-1111-1111-1111-111111111111'
	;;
"compute firewall get")
	echo '11111111-1111-1111-1111-111111111111 benchmark active'
	;;
"compute firewall list")
	if [[ -f "${FAKE_DO_ROOT}/firewall" ]]; then
		printf '11111111-1111-1111-1111-111111111111 %s\n' "$(cat "${FAKE_DO_ROOT}/firewall")"
	fi
	;;
"compute firewall delete")
	[[ -f "${FAKE_DO_ROOT}/firewall" ]] || exit 1
	rm -f "${FAKE_DO_ROOT}/firewall"
	;;
"compute droplet create")
	printf '%s\n' "$4" >"${FAKE_DO_ROOT}/droplet"
	previous=''
	for argument in "$@"; do
		if [[ "${previous}" == '--user-data-file' ]]; then
			cp "${argument}" "${FAKE_DO_ROOT}/cloud-init"
			break
		fi
		previous="${argument}"
	done
	echo '123456'
	;;
"compute droplet get")
	if [[ "$*" == *'PublicIPv4,PrivateIPv4'* ]]; then
		echo '203.0.113.2 10.0.0.2'
	elif [[ "$*" == *'--format Status'* ]]; then
		if [[ -f "${FAKE_DO_ROOT}/off" ]]; then
			echo 'off'
		else
			echo 'active'
		fi
	else
		echo '123456 benchmark 203.0.113.2 10.0.0.2'
	fi
	;;
"compute droplet-action snapshot")
	previous=''
	for argument in "$@"; do
		if [[ "${previous}" == '--snapshot-name' ]]; then
			printf '%s\n' "${argument}" >"${FAKE_DO_ROOT}/snapshot"
			break
		fi
		previous="${argument}"
	done
	[[ -f "${FAKE_DO_ROOT}/snapshot" ]]
	echo '777 completed snapshot'
	;;
"compute droplet snapshots")
	[[ -f "${FAKE_DO_ROOT}/snapshot" ]]
	printf '654321 %s snapshot Ubuntu 25 2026-01-01T00:00:00Z\n' "$(cat "${FAKE_DO_ROOT}/snapshot")"
	;;
"compute droplet list")
	echo '999999 production-api fra1 2 4096 active production'
	if [[ -f "${FAKE_DO_ROOT}/droplet" ]]; then
		printf '123456 %s\n' "$(cat "${FAKE_DO_ROOT}/droplet")"
		if [[ -f "${FAKE_DO_ROOT}/droplet-delete-pending" ]]; then
			rm -f "${FAKE_DO_ROOT}/droplet" "${FAKE_DO_ROOT}/droplet-delete-pending" "${FAKE_DO_ROOT}/off"
			touch "${FAKE_DO_ROOT}/droplet-delete-delay-exercised"
		fi
	fi
	;;
"compute droplet delete")
	[[ -f "${FAKE_DO_ROOT}/droplet" ]] || exit 1
	if [[ ! -f "${FAKE_DO_ROOT}/droplet-delete-delay-exercised" ]]; then
		touch "${FAKE_DO_ROOT}/droplet-delete-pending"
	else
		rm -f "${FAKE_DO_ROOT}/droplet" "${FAKE_DO_ROOT}/off"
	fi
	;;
*)
	echo "unexpected fake doctl command: $*" >&2
	exit 1
	;;
esac
EOF

cat >"${FAKE_BIN}/ssh-keyscan" <<'EOF'
#!/usr/bin/env bash
echo 'example.invalid ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIFakeBenchmarkHostKey'
EOF

cat >"${FAKE_BIN}/ssh" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >>"${FAKE_SSH_LOG}"
if [[ "${FAKE_SSH_FAIL_CLOUD_INIT_ONCE:-0}" == 1 && "$*" == *'cloud-init status --wait'* && ! -f "${FAKE_DO_ROOT}/cloud-init-ssh-failed" ]]; then
	touch "${FAKE_DO_ROOT}/cloud-init-ssh-failed"
	exit 255
fi
if [[ "$*" == *'shutdown -h now'* ]]; then
	touch "${FAKE_DO_ROOT}/off"
	exit 255
fi
cat >/dev/null
EOF

cat >"${FAKE_BIN}/curl" <<'EOF'
#!/usr/bin/env bash
echo '203.0.113.10'
EOF

chmod +x "${FAKE_BIN}/doctl" "${FAKE_BIN}/ssh-keyscan" "${FAKE_BIN}/ssh" "${FAKE_BIN}/curl"

run_runner() {
	(
		cd "${FIXTURE_DIR}"
		PATH="${FAKE_BIN}:${PATH}" \
			FAKE_DO_ROOT="${FAKE_DO_ROOT}" \
			FAKE_DO_LOG="${FAKE_DO_LOG}" \
			FAKE_SSH_LOG="${FAKE_SSH_LOG}" \
		FAKE_SSH_FAIL_CLOUD_INIT_ONCE=1 \
		DO_BENCH_STATE_FILE="${STATE_FILE}" \
		DO_BENCH_SSH_PRIVATE_KEY="${SSH_KEY}" \
		DO_BENCH_SSH_PUBLIC_KEY="${SSH_KEY}.pub" \
		bash scripts/dev/benchmark-remote.sh "$@"
	)
}

run_runner create
[[ -f "${FAKE_DO_ROOT}/cloud-init-ssh-failed" ]] || fail 'transient cloud-init SSH failure was not exercised'
if run_runner create >/dev/null 2>&1; then
	fail 'a second runner reused an owned state file'
fi
list_output="$(run_runner list)"
grep -Fq 'benchmark Droplets: 1' <<<"${list_output}" || fail 'benchmark Droplet count is wrong'
grep -Fq 'bench-' <<<"${list_output}" || fail 'active benchmark Droplet is not listed'
if grep -Fq 'production-api' <<<"${list_output}"; then
	fail 'non-benchmark Droplet was included in the runner list'
fi
tag_create_line="$(command_line 'compute tag create')"
firewall_create_line="$(command_line 'compute firewall create')"
droplet_create_line="$(command_line 'compute droplet create')"
[[ -n "${tag_create_line}" && -n "${firewall_create_line}" && -n "${droplet_create_line}" ]] || fail 'create calls were not recorded'
((tag_create_line < firewall_create_line && firewall_create_line < droplet_create_line)) || fail 'tag and firewall must exist before the Droplet'
grep -F 'compute firewall create' "${FAKE_DO_LOG}" | grep -Fq -- '--tag-names' || fail 'firewall is not tag-bound'
grep -F 'compute droplet create' "${FAKE_DO_LOG}" | grep -Fq -- '--tag-names' || fail 'Droplet is not tagged at creation'

run_runner sync
first_fingerprint="$(state_value "${STATE_FILE}" SOURCE_FINGERPRINT)"
[[ "${first_fingerprint}" =~ ^[0-9a-f]{64}$ ]] || fail 'source fingerprint was not recorded'
printf 'benchmark fixture v2\n' >"${FIXTURE_DIR}/README.md"
run_runner sync
second_fingerprint="$(state_value "${STATE_FILE}" SOURCE_FINGERPRINT)"
[[ "${second_fingerprint}" =~ ^[0-9a-f]{64}$ && "${second_fingerprint}" != "${first_fingerprint}" ]] || fail 'source fingerprint did not track dirty source changes'
rm "${FIXTURE_DIR}/README.md"
run_runner sync
deleted_fingerprint="$(state_value "${STATE_FILE}" SOURCE_FINGERPRINT)"
[[ "${deleted_fingerprint}" =~ ^[0-9a-f]{64}$ && "${deleted_fingerprint}" != "${second_fingerprint}" ]] || fail 'tracked-file deletion was not synchronized and fingerprinted'

cp "${STATE_FILE}" "${TEMP_DIR}/stale.state"
sed -e 's/^DROPLET_ID=.*/DROPLET_ID=/' -e 's/^FIREWALL_ID=.*/FIREWALL_ID=/' "${STATE_FILE}" >"${STATE_FILE}.next"
mv "${STATE_FILE}.next" "${STATE_FILE}"
run_runner destroy
[[ ! -e "${STATE_FILE}" ]] || fail 'state remained after reconciled cleanup'
[[ ! -e "${FAKE_DO_ROOT}/droplet" && ! -e "${FAKE_DO_ROOT}/firewall" && ! -e "${FAKE_DO_ROOT}/tag" ]] || fail 'provider resources remained after cleanup'
[[ -f "${FAKE_DO_ROOT}/droplet-delete-delay-exercised" ]] || fail 'eventually consistent Droplet deletion was not confirmed'

droplet_delete_line="$(command_line 'compute droplet delete')"
firewall_delete_line="$(command_line 'compute firewall delete')"
tag_delete_line="$(command_line 'compute tag delete')"
((droplet_delete_line < firewall_delete_line && firewall_delete_line < tag_delete_line)) || fail 'cleanup did not stop Droplet billing first'

cp "${TEMP_DIR}/stale.state" "${STATE_FILE}"
run_runner destroy
[[ ! -e "${STATE_FILE}" ]] || fail 'stale state was not cleared after provider-confirmed absence'
list_output="$(run_runner list)"
grep -Fq 'benchmark Droplets: 0' <<<"${list_output}" || fail 'cleaned benchmark Droplet remained in the account list'

run_runner image-build
[[ -f "${FAKE_DO_ROOT}/snapshot" ]] || fail 'golden snapshot was not created'
[[ ! -e "${FAKE_DO_ROOT}/droplet" && ! -e "${STATE_FILE}" ]] || fail 'image builder resources were not cleaned up'
grep -Fq 'GOTOOLCHAIN=go' "${FAKE_SSH_LOG}" || fail 'current Go toolchain was not preloaded'
grep -Fq 'postgres:17@sha256:' "${FAKE_SSH_LOG}" || fail 'pinned PostgreSQL image was not preloaded'
grep -Fq 'grafana/k6:2.1.0@sha256:' "${FAKE_SSH_LOG}" || fail 'pinned k6 image was not preloaded'
IMAGE_REFERENCE_FILE="${FIXTURE_DIR}/.artifacts/bench/remote/golden-image.env"
[[ -f "${IMAGE_REFERENCE_FILE}" ]] || fail 'golden image reference was not written'
grep -Fq 'export DO_BENCH_IMAGE=654321' "${IMAGE_REFERENCE_FILE}" || fail 'golden image ID is missing from the reference'
grep -Fq 'export DO_BENCH_GOLDEN_IMAGE=1' "${IMAGE_REFERENCE_FILE}" || fail 'golden image mode is missing from the reference'
list_output="$(run_runner image-list)"
grep -Fq 'benchmark snapshots: 1' <<<"${list_output}" || fail 'golden snapshot was not listed'

DO_BENCH_IMAGE=654321 DO_BENCH_GOLDEN_IMAGE=1 run_runner create
grep -F 'compute droplet create' "${FAKE_DO_LOG}" | tail -n 1 | grep -Fq -- '--image 654321' || fail 'golden image ID was not used to create the runner'
if grep -Fq 'package_update:' "${FAKE_DO_ROOT}/cloud-init"; then
	fail 'golden-image boot repeated package installation'
fi
if grep -Fq 'runcmd:' "${FAKE_DO_ROOT}/cloud-init"; then
	fail 'golden-image boot repeated base-image initialization'
fi
run_runner destroy

echo 'remote benchmark lifecycle check passed'
