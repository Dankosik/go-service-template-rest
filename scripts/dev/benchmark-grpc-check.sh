#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

mkdir -p "${ROOT_DIR}/.artifacts/bench/grpc"
CHECK_ROOT="$(mktemp -d "${ROOT_DIR}/.artifacts/bench/grpc/check.XXXXXX")"
UNSAFE_ROOT="$(mktemp -d "${ROOT_DIR}/.artifacts/bench/grpc-unsafe.XXXXXX")"
SENTINEL_PID=""

cleanup() {
	local status=$?
	trap - EXIT INT TERM
	if [[ -n "${SENTINEL_PID}" ]] && kill -0 "${SENTINEL_PID}" >/dev/null 2>&1; then
		kill -TERM "${SENTINEL_PID}" >/dev/null 2>&1 || true
		wait "${SENTINEL_PID}" 2>/dev/null || true
	fi
	rm -rf -- "${CHECK_ROOT}" "${UNSAFE_ROOT}"
	exit "${status}"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

FAKE_SERVER="${CHECK_ROOT}/fake-server.sh"
FAKE_DOCKER="${CHECK_ROOT}/fake-docker.sh"
FAKE_GO="${CHECK_ROOT}/fake-go.sh"

cat >"${FAKE_SERVER}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

printf '%s\n' "$$" >"${FAKE_SERVER_PID_FILE}"
case "${FAKE_SERVER_MODE:-success}" in
start-fail)
	exit 17
	;;
malformed-ready)
	printf 'GRPC_BENCH_READY=0.0.0.0:54321\n'
	;;
success)
	printf 'GRPC_BENCH_READY=127.0.0.1:54321\n'
	;;
*)
	exit 18
	;;
esac

trap 'exit 0' TERM INT
while :; do
	sleep 1
done
EOF

cat >"${FAKE_DOCKER}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
info)
	if [[ "${2:-}" == --format ]]; then
		printf 'fake-linux\n'
	fi
	exit 0
	;;
version)
	printf 'fake-1.0\n'
	exit 0
	;;
run)
	shift
	if [[ -n "${FAKE_DOCKER_PID_FILE:-}" ]]; then
		printf '%s\n' "$$" >"${FAKE_DOCKER_PID_FILE}"
	fi
	artifact_dir=""
	while (($# > 0)); do
		case "$1" in
		-v)
			mapping="$2"
			if [[ "${mapping}" == *:/artifacts ]]; then
				artifact_dir="${mapping%:/artifacts}"
			fi
			shift 2
			;;
		-u | -w | -e | --network)
			shift 2
			;;
		--rm)
			shift
			;;
		*)
			shift
			;;
		esac
	done
	case "${FAKE_DOCKER_MODE:-success}" in
	client-fail)
		printf 'fake k6 failure\n' >&2
		exit 42
		;;
	slow)
		sleep 2
		;;
	success) ;;
	*)
		exit 43
		;;
	esac
	if [[ -n "${artifact_dir}" ]]; then
		printf '{"fake":"summary"}\n' >"${artifact_dir}/summary.json"
	fi
	printf 'fake k6 success\n'
	;;
*)
	exit 44
	;;
esac
EOF

cat >"${FAKE_GO}" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

case "${1:-}" in
version)
	printf 'go version go1.25.0 fake/arm64\n'
	;;
env)
	case "${2:-}" in
	GOOS) printf 'fakeos\n' ;;
	GOARCH) printf 'arm64\n' ;;
	*) exit 45 ;;
	esac
	;;
*)
	exit 46
	;;
esac
EOF

chmod +x "${FAKE_SERVER}" "${FAKE_DOCKER}" "${FAKE_GO}"

sleep 60 &
SENTINEL_PID=$!

assert_sentinel_alive() {
	if ! kill -0 "${SENTINEL_PID}" >/dev/null 2>&1; then
		echo "gRPC benchmark runner terminated an unowned sentinel process"
		exit 1
	fi
}

assert_owned_server_stopped() {
	local pid_file="$1"
	local pid
	[[ -s "${pid_file}" ]] || {
		echo "fake benchmark server did not record its PID"
		exit 1
	}
	pid="$(cat "${pid_file}")"
	for _ in $(seq 1 20); do
		kill -0 "${pid}" >/dev/null 2>&1 || break
		sleep 0.05
	done
	if kill -0 "${pid}" >/dev/null 2>&1; then
		ps -p "${pid}" -o pid=,ppid=,stat=,command= >&2 || true
		echo "owned fake benchmark server ${pid} survived runner cleanup"
		exit 1
	fi
	assert_sentinel_alive
}

assert_owned_client_stopped() {
	local pid_file="$1"
	local pid
	[[ -s "${pid_file}" ]] || {
		echo "fake Docker client did not record its PID"
		exit 1
	}
	pid="$(cat "${pid_file}")"
	for _ in $(seq 1 20); do
		kill -0 "${pid}" >/dev/null 2>&1 || break
		sleep 0.05
	done
	if kill -0 "${pid}" >/dev/null 2>&1; then
		ps -p "${pid}" -o pid=,ppid=,stat=,command= >&2 || true
		echo "owned fake Docker client ${pid} survived runner cleanup"
		exit 1
	fi
	assert_sentinel_alive
}

run_case() {
	local name="$1"
	local server_mode="$2"
	local docker_mode="$3"
	local expected="$4"
	local case_root="${CHECK_ROOT}/${name}"
	local artifact_relative="${case_root#"${ROOT_DIR}"/}"
	local pid_file="${case_root}/server.pid"
	local docker_pid_file="${case_root}/docker.pid"
	local status=0

	mkdir -p "${case_root}"
	FAKE_SERVER_MODE="${server_mode}" \
	FAKE_SERVER_PID_FILE="${pid_file}" \
	FAKE_DOCKER_MODE="${docker_mode}" \
	FAKE_DOCKER_PID_FILE="${docker_pid_file}" \
	GRPC_BENCH_SERVER_BINARY="${FAKE_SERVER}" \
	GRPC_BENCH_DOCKER_BIN="${FAKE_DOCKER}" \
	GRPC_BENCH_GO_BIN="${FAKE_GO}" \
	GRPC_BENCH_ARTIFACT_DIR="${artifact_relative}" \
	GRPC_BENCH_READY_TIMEOUT_TICKS=10 \
	BENCH_SOURCE_REVISION=fake-revision \
	BENCH_SOURCE_DIRTY=true \
	BENCH_SOURCE_FINGERPRINT=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
	bash scripts/dev/benchmark.sh grpc >"${case_root}/runner.log" 2>&1 || status=$?

	case "${expected}" in
	success)
		if [[ "${status}" -ne 0 ]]; then
			cat "${case_root}/runner.log" >&2
			echo "${name}: expected success, got status ${status}"
			exit 1
		fi
		;;
	failure)
		if [[ "${status}" -eq 0 ]]; then
			cat "${case_root}/runner.log" >&2
			echo "${name}: expected failure"
			exit 1
		fi
		;;
	esac
	assert_owned_server_stopped "${pid_file}"
	if [[ "${server_mode}" == success ]]; then
		assert_owned_client_stopped "${docker_pid_file}"
	fi
}

run_case success success success success
for key in \
	git_revision \
	git_dirty \
	source_fingerprint \
	k6_image \
	scenario \
	scenario_sha256 \
	schema \
	schema_sha256 \
	evidence_level \
	workload_id \
	server_listener \
	docker_target \
	docker_route \
	transport_security \
	telemetry_mode \
	payload_bytes \
	stream_messages \
	virtual_users \
	connections_per_vu \
	warmup_duration \
	warmup_completion_budget \
	measured_duration \
	rpc_timeout \
	raw_samples \
	docker_server_version \
	docker_operating_system \
	docker_architecture \
	server_go_version \
	server_goos \
	server_goarch \
	server_resource_limits \
	load_generator_resource_limits \
	decision_grade_resource_observables; do
	grep -Eq "^${key}=.+" "${CHECK_ROOT}/success/synthetic/run.meta" || {
		echo "success metadata is missing ${key}"
		exit 1
	}
done
grep -Fx 'server_go_version=go version go1.25.0 fake/arm64' "${CHECK_ROOT}/success/synthetic/run.meta"
grep -Fx 'server_goos=fakeos' "${CHECK_ROOT}/success/synthetic/run.meta"
grep -Fx 'server_goarch=arm64' "${CHECK_ROOT}/success/synthetic/run.meta"
grep -Fx 'warmup_completion_budget=10s' "${CHECK_ROOT}/success/synthetic/run.meta"

run_case start-failure start-fail success failure
run_case client-failure success client-fail failure
run_case malformed-ready malformed-ready success failure

signal_root="${CHECK_ROOT}/signal"
signal_relative="${signal_root#"${ROOT_DIR}"/}"
signal_pid_file="${signal_root}/server.pid"
signal_docker_pid_file="${signal_root}/docker.pid"
mkdir -p "${signal_root}"
FAKE_SERVER_MODE=success \
FAKE_SERVER_PID_FILE="${signal_pid_file}" \
FAKE_DOCKER_MODE=slow \
FAKE_DOCKER_PID_FILE="${signal_docker_pid_file}" \
GRPC_BENCH_SERVER_BINARY="${FAKE_SERVER}" \
GRPC_BENCH_DOCKER_BIN="${FAKE_DOCKER}" \
GRPC_BENCH_GO_BIN="${FAKE_GO}" \
GRPC_BENCH_ARTIFACT_DIR="${signal_relative}" \
GRPC_BENCH_READY_TIMEOUT_TICKS=10 \
BENCH_SOURCE_REVISION=fake-revision \
BENCH_SOURCE_DIRTY=true \
BENCH_SOURCE_FINGERPRINT=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa \
bash scripts/dev/benchmark.sh grpc >"${signal_root}/runner.log" 2>&1 &
runner_pid=$!
for _ in $(seq 1 50); do
	[[ -s "${signal_pid_file}" && -s "${signal_docker_pid_file}" ]] && break
	sleep 0.05
done
[[ -s "${signal_pid_file}" && -s "${signal_docker_pid_file}" ]] || {
	echo "signal case did not reach the owned server and Docker client"
	exit 1
}
kill -TERM "${runner_pid}"
signal_status=0
wait "${runner_pid}" || signal_status=$?
if [[ "${signal_status}" -eq 0 ]]; then
	cat "${signal_root}/runner.log" >&2
	echo "signal case expected a non-zero interrupted status"
	exit 1
fi
assert_owned_server_stopped "${signal_pid_file}"
assert_owned_client_stopped "${signal_docker_pid_file}"

printf 'preserve\n' >"${UNSAFE_ROOT}/sentinel"
unsafe_relative="${UNSAFE_ROOT#"${ROOT_DIR}"/}"
unsafe_status=0
GRPC_BENCH_SERVER_BINARY="${FAKE_SERVER}" \
GRPC_BENCH_DOCKER_BIN="${FAKE_DOCKER}" \
GRPC_BENCH_ARTIFACT_DIR="${unsafe_relative}" \
bash scripts/dev/benchmark.sh grpc >"${CHECK_ROOT}/unsafe.log" 2>&1 || unsafe_status=$?
if [[ "${unsafe_status}" -eq 0 || ! -f "${UNSAFE_ROOT}/sentinel" ]]; then
	cat "${CHECK_ROOT}/unsafe.log" >&2
	echo "unsafe artifact path was not rejected before deletion"
	exit 1
fi
assert_sentinel_alive

symlink_parent="${CHECK_ROOT}/symlink-escape"
symlink_relative="${symlink_parent#"${ROOT_DIR}"/}"
mkdir -p "${symlink_parent}"
ln -s "${UNSAFE_ROOT}" "${symlink_parent}/synthetic"
symlink_status=0
GRPC_BENCH_SERVER_BINARY="${FAKE_SERVER}" \
GRPC_BENCH_DOCKER_BIN="${FAKE_DOCKER}" \
GRPC_BENCH_ARTIFACT_DIR="${symlink_relative}" \
bash scripts/dev/benchmark.sh grpc >"${CHECK_ROOT}/symlink.log" 2>&1 || symlink_status=$?
if [[ "${symlink_status}" -eq 0 || ! -f "${UNSAFE_ROOT}/sentinel" ]]; then
	cat "${CHECK_ROOT}/symlink.log" >&2
	echo "symlink-escaping artifact path was not rejected before deletion"
	exit 1
fi
assert_sentinel_alive

echo "gRPC benchmark lifecycle check passed"
