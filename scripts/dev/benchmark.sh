#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

BENCH_PACKAGE="${BENCH_PACKAGE:-./...}"
BENCH_PATTERN="${BENCH_PATTERN:-.}"
BENCH_COUNT="${BENCH_COUNT:-10}"
BENCH_TIME="${BENCH_TIME:-1s}"
BENCH_TAGS="${BENCH_TAGS:-}"
BENCH_OUTPUT="${BENCH_OUTPUT:-.artifacts/bench/current.txt}"
BENCH_BASELINE="${BENCH_BASELINE:-.artifacts/bench/baseline.txt}"
BENCH_CURRENT="${BENCH_CURRENT:-.artifacts/bench/current.txt}"
BENCH_COMPARE_OUTPUT="${BENCH_COMPARE_OUTPUT:-.artifacts/bench/comparison.txt}"
BENCH_PROFILE="${BENCH_PROFILE:-cpu}"
BENCH_PROFILE_DIR="${BENCH_PROFILE_DIR:-.artifacts/bench/profiles}"
BENCH_WORKLOAD_ID="${BENCH_WORKLOAD_ID:-go-benchmark}"
BENCH_DEPENDENCY_IMAGE="${BENCH_DEPENDENCY_IMAGE:-none}"
BENCH_SCHEMA_PATH="${BENCH_SCHEMA_PATH:-}"
K6_IMAGE_DEFAULT="grafana/k6:2.1.0@sha256:65c920dc067d5e2e00befbf982af6ad6ad0117034e8b1c65817c7975c52d4669"
K6_IMAGE="${K6_IMAGE:-${K6_IMAGE_DEFAULT}}"
HTTP_BENCH_SCRIPT="${HTTP_BENCH_SCRIPT:-test/performance/http/single-flow.js}"
HTTP_BENCH_ARTIFACT_DIR="${HTTP_BENCH_ARTIFACT_DIR:-.artifacts/bench/http}"
HTTP_BENCH_ENV_FILE="${HTTP_BENCH_ENV_FILE-.env.bench}"
HTTP_BENCH_DOCKER_NETWORK="${HTTP_BENCH_DOCKER_NETWORK:-}"
HTTP_BENCH_RAW_SAMPLES="${HTTP_BENCH_RAW_SAMPLES:-0}"
# profile:grpc-reference-benchmark:start
GRPC_BENCH_SCRIPT="${GRPC_BENCH_SCRIPT:-test/performance/grpc/all-cardinalities.js}"
GRPC_BENCH_ARTIFACT_DIR="${GRPC_BENCH_ARTIFACT_DIR:-.artifacts/bench/grpc}"
GRPC_BENCH_MODE="${GRPC_BENCH_MODE:-synthetic}"
GRPC_BENCH_RAW_SAMPLES="${GRPC_BENCH_RAW_SAMPLES:-0}"
GRPC_BENCH_WORKLOAD_ID="${GRPC_BENCH_WORKLOAD_ID:-grpc-reference-all-cardinalities}"
GRPC_BENCH_PAYLOAD_BYTES="${GRPC_BENCH_PAYLOAD_BYTES:-64}"
GRPC_BENCH_STREAM_MESSAGES="${GRPC_BENCH_STREAM_MESSAGES:-4}"
GRPC_BENCH_VUS="${GRPC_BENCH_VUS:-1}"
GRPC_BENCH_WARMUP_DURATION="${GRPC_BENCH_WARMUP_DURATION:-3s}"
GRPC_BENCH_DURATION="${GRPC_BENCH_DURATION:-10s}"
GRPC_BENCH_RPC_TIMEOUT="${GRPC_BENCH_RPC_TIMEOUT:-10s}"
GRPC_BENCH_DOCKER_BIN="${GRPC_BENCH_DOCKER_BIN:-docker}"
GRPC_BENCH_GO_BIN="${GRPC_BENCH_GO_BIN:-go}"
GRPC_BENCH_SERVER_BINARY="${GRPC_BENCH_SERVER_BINARY:-}"
GRPC_BENCH_READY_TIMEOUT_TICKS="${GRPC_BENCH_READY_TIMEOUT_TICKS:-100}"
# profile:grpc-reference-benchmark:end

usage() {
	echo "usage: $0 <run|compare|profile|http|http-inspect|check|source-fingerprint>"
	# profile:grpc-reference-benchmark:start
	echo "       $0 <grpc|grpc-smoke|grpc-inspect>"
	# profile:grpc-reference-benchmark:end
}

require_positive_integer() {
	local name="$1"
	local value="$2"
	if [[ ! "${value}" =~ ^[1-9][0-9]*$ ]]; then
		echo "${name} must be a positive integer, got '${value}'"
		exit 1
	fi
}

require_metadata_value() {
	local name="$1"
	local value="$2"
	if [[ -z "${value}" || "${value}" == *$'\n'* || "${value}" == *$'\r'* ]]; then
		echo "${name} must be a non-empty single-line value"
		exit 1
	fi
}

schema_fingerprint() {
	local schema_path="$1"
	local file relative_path
	local -a files=()

	if [[ -z "${schema_path}" ]]; then
		printf 'none'
		return
	fi
	if [[ ! -e "${schema_path}" ]]; then
		echo "benchmark schema path not found: ${schema_path}" >&2
		exit 1
	fi
	if [[ -f "${schema_path}" ]]; then
		git hash-object "${schema_path}"
		return
	fi

	while IFS= read -r -d '' file; do
		files+=("${file}")
	done < <(find "${schema_path}" -type f -print0 | sort -z)
	if [[ "${#files[@]}" -eq 0 ]]; then
		echo "benchmark schema path contains no files: ${schema_path}" >&2
		exit 1
	fi

	{
		for file in "${files[@]}"; do
			relative_path="${file#"${schema_path}"/}"
			printf '%s\t%s\n' "${relative_path}" "$(git hash-object "${file}")"
		done
	} | git hash-object --stdin
}

repository_fingerprint() (
	local snapshot_dir object_dir tree
	# A temporary index captures Git path/content/mode semantics in one pass
	# without touching the repository index or spawning a process per file.
	snapshot_dir=$(mktemp -d "${TMPDIR:-/tmp}/benchmark-source-fingerprint.XXXXXX")
	trap 'rm -rf -- "${snapshot_dir}"' EXIT
	mkdir "${snapshot_dir}/objects"
	object_dir=$(CDPATH='' cd -- "$(git rev-parse --git-path objects)" && pwd)

	snapshot_git() {
		GIT_INDEX_FILE="${snapshot_dir}/index" \
			GIT_OBJECT_DIRECTORY="${snapshot_dir}/objects" \
			GIT_ALTERNATE_OBJECT_DIRECTORIES="${object_dir}" \
			git "$@"
	}

	snapshot_git read-tree HEAD
	snapshot_git add -A -- .
	tree=$(snapshot_git write-tree)
	printf '%s' "${tree}" | shasum -a 256 | awk '{ print $1 }'
)

write_run_metadata() {
	local output="$1"
	local revision dirty source_fingerprint benchmark_cpu logical_cpus benchmark_schema_fingerprint

	revision="${BENCH_SOURCE_REVISION:-$(git rev-parse --verify HEAD 2>/dev/null || echo unknown)}"
	benchmark_cpu="$(awk '/^cpu:/ { sub(/^cpu:[[:space:]]*/, ""); print; exit }' "${output}")"
	logical_cpus="$(getconf _NPROCESSORS_ONLN 2>/dev/null || true)"
	if [[ -z "${logical_cpus}" ]] && command -v sysctl >/dev/null 2>&1; then
		logical_cpus="$(sysctl -n hw.logicalcpu 2>/dev/null || true)"
	fi
	if [[ -n "${BENCH_SOURCE_DIRTY:-}" ]]; then
		dirty="${BENCH_SOURCE_DIRTY}"
	elif [[ -n "$(git status --porcelain 2>/dev/null || true)" ]]; then
		dirty=true
	else
		dirty=false
	fi
	require_metadata_value BENCH_SOURCE_REVISION "${revision}"
	if [[ "${dirty}" != "true" && "${dirty}" != "false" ]]; then
		echo "BENCH_SOURCE_DIRTY must be true or false, got '${dirty}'"
		exit 1
	fi
	source_fingerprint="${BENCH_SOURCE_FINGERPRINT:-$(repository_fingerprint)}"
	if [[ ! "${source_fingerprint}" =~ ^[0-9a-f]{64}$ ]]; then
		echo "BENCH_SOURCE_FINGERPRINT must be a lowercase SHA-256 digest"
		exit 1
	fi
	benchmark_schema_fingerprint="$(schema_fingerprint "${BENCH_SCHEMA_PATH}")"

	{
		printf 'recorded_at_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
		printf 'git_revision=%s\n' "${revision}"
		printf 'git_dirty=%s\n' "${dirty}"
		printf 'source_fingerprint=%s\n' "${source_fingerprint}"
		printf 'benchmark_package=%s\n' "${BENCH_PACKAGE}"
		printf 'benchmark_pattern=%s\n' "${BENCH_PATTERN}"
		printf 'benchmark_count=%s\n' "${BENCH_COUNT}"
		printf 'benchmark_time=%s\n' "${BENCH_TIME}"
		printf 'benchmark_tags=%s\n' "${BENCH_TAGS:-none}"
		printf 'benchmark_workload_id=%s\n' "${BENCH_WORKLOAD_ID}"
		printf 'benchmark_dependency_image=%s\n' "${BENCH_DEPENDENCY_IMAGE}"
		printf 'benchmark_schema_fingerprint=%s\n' "${benchmark_schema_fingerprint}"
		printf 'benchmark_cpu=%s\n' "${benchmark_cpu:-unknown}"
		printf 'logical_cpus=%s\n' "${logical_cpus:-unknown}"
		printf 'gomaxprocs=%s\n' "${GOMAXPROCS:-default}"
		printf 'go_version=%s\n' "$(go version)"
		printf 'goos=%s\n' "$(go env GOOS)"
		printf 'goarch=%s\n' "$(go env GOARCH)"
		printf 'cgo_enabled=%s\n' "$(go env CGO_ENABLED)"
	} >"${output}.meta"
}

run_benchmark() {
	local output_dir temporary_output status
	local -a go_test_args warm_args

	require_positive_integer BENCH_COUNT "${BENCH_COUNT}"
	require_metadata_value BENCH_WORKLOAD_ID "${BENCH_WORKLOAD_ID}"
	require_metadata_value BENCH_DEPENDENCY_IMAGE "${BENCH_DEPENDENCY_IMAGE}"
	output_dir="$(dirname "${BENCH_OUTPUT}")"
	mkdir -p "${output_dir}"
	temporary_output="$(mktemp "${output_dir}/.benchmark.XXXXXX")"

	go_test_args=(
		-run='^$'
		-bench="${BENCH_PATTERN}"
		-benchmem
		-benchtime="${BENCH_TIME}"
		-count="${BENCH_COUNT}"
	)
	if [[ -n "${BENCH_TAGS}" ]]; then
		go_test_args+=(-tags="${BENCH_TAGS}")
	fi
	warm_args=(-run='^$' -bench='^$')
	if [[ -n "${BENCH_TAGS}" ]]; then
		warm_args+=(-tags="${BENCH_TAGS}")
	fi
	go test "${warm_args[@]}" "${BENCH_PACKAGE}" >/dev/null

	if go test "${go_test_args[@]}" "${BENCH_PACKAGE}" >"${temporary_output}" 2>&1; then
		status=0
	else
		status=$?
	fi

	cat "${temporary_output}"
	if [[ "${status}" -ne 0 ]]; then
		rm -f "${temporary_output}"
		exit "${status}"
	fi
	if ! grep -Eq '^Benchmark[^[:space:]]*[[:space:]]' "${temporary_output}"; then
		rm -f "${temporary_output}"
		echo "no benchmarks matched package '${BENCH_PACKAGE}' and pattern '${BENCH_PATTERN}'"
		exit 1
	fi

	mv "${temporary_output}" "${BENCH_OUTPUT}"
	write_run_metadata "${BENCH_OUTPUT}"
	echo "benchmark result: ${BENCH_OUTPUT}"
	echo "environment metadata: ${BENCH_OUTPUT}.meta"
}

compare_benchmarks() {
	local output_dir temporary_output key baseline_value current_value

	[[ -f "${BENCH_BASELINE}" ]] || {
		echo "benchmark baseline not found: ${BENCH_BASELINE}"
		exit 1
	}
	[[ -f "${BENCH_CURRENT}" ]] || {
		echo "current benchmark result not found: ${BENCH_CURRENT}"
		exit 1
	}
	[[ -f "${BENCH_BASELINE}.meta" ]] || {
		echo "benchmark baseline metadata not found: ${BENCH_BASELINE}.meta"
		exit 1
	}
	[[ -f "${BENCH_CURRENT}.meta" ]] || {
		echo "current benchmark metadata not found: ${BENCH_CURRENT}.meta"
		exit 1
	}
	for key in benchmark_package benchmark_pattern benchmark_count benchmark_time benchmark_tags benchmark_workload_id benchmark_dependency_image benchmark_schema_fingerprint benchmark_cpu logical_cpus gomaxprocs go_version goos goarch cgo_enabled; do
		baseline_value="$(sed -n "s/^${key}=//p" "${BENCH_BASELINE}.meta")"
		current_value="$(sed -n "s/^${key}=//p" "${BENCH_CURRENT}.meta")"
		if [[ -z "${baseline_value}" || -z "${current_value}" ]]; then
			echo "benchmark metadata is missing ${key}"
			exit 1
		fi
		if [[ "${baseline_value}" != "${current_value}" ]]; then
			echo "benchmark metadata mismatch for ${key}: baseline='${baseline_value}' current='${current_value}'"
			exit 1
		fi
	done

	output_dir="$(dirname "${BENCH_COMPARE_OUTPUT}")"
	mkdir -p "${output_dir}"
	bash "${ROOT_DIR}/scripts/run-go-tool.sh" benchstat -h >/dev/null 2>&1
	temporary_output="$(mktemp "${output_dir}/.benchstat.XXXXXX")"
	if ! bash "${ROOT_DIR}/scripts/run-go-tool.sh" benchstat "${BENCH_BASELINE}" "${BENCH_CURRENT}" >"${temporary_output}" 2>&1; then
		rm -f "${temporary_output}"
		exit 1
	fi
	cat "${temporary_output}"
	mv "${temporary_output}" "${BENCH_COMPARE_OUTPUT}"
	echo "comparison result: ${BENCH_COMPARE_OUTPUT}"
}

profile_benchmark() {
	local profile_file raw_output status
	local -a go_test_args profile_flags

	if [[ "${BENCH_PACKAGE}" == "./..." || "${BENCH_PACKAGE}" =~ [[:space:]] ]]; then
		echo "BENCH_PACKAGE must name one concrete package for profiling"
		exit 1
	fi
	require_metadata_value BENCH_WORKLOAD_ID "${BENCH_WORKLOAD_ID}"
	require_metadata_value BENCH_DEPENDENCY_IMAGE "${BENCH_DEPENDENCY_IMAGE}"

	mkdir -p "${BENCH_PROFILE_DIR}"
	case "${BENCH_PROFILE}" in
	cpu)
		profile_file="cpu.pprof"
		profile_flags=(-cpuprofile="${profile_file}")
		;;
	memory)
		profile_file="memory.pprof"
		profile_flags=(-memprofile="${profile_file}" -memprofilerate=1)
		;;
	block)
		profile_file="block.pprof"
		profile_flags=(-blockprofile="${profile_file}" -blockprofilerate=1)
		;;
	mutex)
		profile_file="mutex.pprof"
		profile_flags=(-mutexprofile="${profile_file}" -mutexprofilefraction=1)
		;;
	trace)
		profile_file="trace.out"
		profile_flags=(-trace="${profile_file}")
		;;
	*)
		echo "BENCH_PROFILE must be one of: cpu, memory, block, mutex, trace"
		exit 1
		;;
	esac

	raw_output="${BENCH_PROFILE_DIR}/${BENCH_PROFILE}.txt"
	go_test_args=(
		-run='^$'
		-bench="${BENCH_PATTERN}"
		-benchmem
		-benchtime="${BENCH_TIME}"
		-count=1
		-o="${BENCH_PROFILE_DIR}/benchmark.test"
		-outputdir="${BENCH_PROFILE_DIR}"
	)
	if [[ -n "${BENCH_TAGS}" ]]; then
		go_test_args+=(-tags="${BENCH_TAGS}")
	fi

	if go test \
		"${go_test_args[@]}" \
		"${profile_flags[@]}" \
		"${BENCH_PACKAGE}" >"${raw_output}" 2>&1; then
		status=0
	else
		status=$?
	fi

	cat "${raw_output}"
	if [[ "${status}" -ne 0 ]]; then
		exit "${status}"
	fi
	if ! grep -Eq '^Benchmark[^[:space:]]*[[:space:]]' "${raw_output}"; then
		echo "no benchmarks matched package '${BENCH_PACKAGE}' and pattern '${BENCH_PATTERN}'"
		exit 1
	fi

	BENCH_COUNT=1 write_run_metadata "${raw_output}"
	printf 'benchmark_profile=%s\n' "${BENCH_PROFILE}" >>"${raw_output}.meta"
	echo "diagnostic artifact: ${BENCH_PROFILE_DIR}/${profile_file}"
	if [[ "${BENCH_PROFILE}" == "trace" ]]; then
		echo "inspect with: go tool trace ${BENCH_PROFILE_DIR}/${profile_file}"
	elif [[ "${BENCH_PROFILE}" == "memory" ]]; then
		echo "allocation churn: go tool pprof -top -alloc_space ${BENCH_PROFILE_DIR}/${profile_file}"
		echo "retained heap: go tool pprof -top -inuse_space ${BENCH_PROFILE_DIR}/${profile_file}"
	else
		echo "inspect with: go tool pprof -top ${BENCH_PROFILE_DIR}/${profile_file}"
	fi
}

ensure_docker() {
	bash "${ROOT_DIR}/scripts/lib/require-docker.sh" "HTTP load benchmarks"
}

# profile:grpc-reference-benchmark:start
ensure_grpc_docker() {
	DOCKER_BIN="${GRPC_BENCH_DOCKER_BIN}" \
		bash "${ROOT_DIR}/scripts/lib/require-docker.sh" "gRPC load benchmarks"
}
# profile:grpc-reference-benchmark:end

run_http_benchmark() {
	local mode="$1"
	local script_path="${HTTP_BENCH_SCRIPT}"
	local artifact_dir="${HTTP_BENCH_ARTIFACT_DIR}"
	local env_file="${HTTP_BENCH_ENV_FILE}"
	local raw_samples="${HTTP_BENCH_RAW_SAMPLES}"
	local resolved_env_file=""
	local host_uid host_gid variable_name
	local -a docker_args k6_args

	ensure_docker
	[[ "${raw_samples}" == 0 || "${raw_samples}" == 1 ]] || {
		echo "HTTP_BENCH_RAW_SAMPLES must be 0 or 1"
		exit 1
	}
	if [[ "${script_path}" == /* || "${script_path}" =~ (^|/)\.\.(/|$) ]]; then
		echo "HTTP_BENCH_SCRIPT must be a repository-relative path"
		exit 1
	fi
	[[ -f "${ROOT_DIR}/${script_path}" ]] || {
		echo "HTTP benchmark script not found: ${script_path}"
		exit 1
	}
	if [[ "${artifact_dir}" != .artifacts/bench/* || "${artifact_dir}" =~ (^|/)\.\.(/|$) ]]; then
		echo "HTTP_BENCH_ARTIFACT_DIR must stay under .artifacts/bench/"
		exit 1
	fi
	if [[ -n "${env_file}" ]]; then
		if [[ "${env_file}" == /* ]]; then
			resolved_env_file="${env_file}"
		else
			resolved_env_file="${ROOT_DIR}/${env_file}"
		fi
		[[ -f "${resolved_env_file}" ]] || {
			echo "HTTP benchmark environment file not found: ${env_file}"
			exit 1
		}
	fi

	mkdir -p "${ROOT_DIR}/${artifact_dir}"
	rm -f \
		"${ROOT_DIR}/${artifact_dir}/inspect.json" \
		"${ROOT_DIR}/${artifact_dir}/inspect.log" \
		"${ROOT_DIR}/${artifact_dir}/k6.log" \
		"${ROOT_DIR}/${artifact_dir}/samples.json" \
		"${ROOT_DIR}/${artifact_dir}/samples.json.gz" \
		"${ROOT_DIR}/${artifact_dir}/summary.json"

	host_uid="$(id -u 2>/dev/null || echo 0)"
	host_gid="$(id -g 2>/dev/null || echo 0)"
	docker_args=(
		--rm
		-u "${host_uid}:${host_gid}"
		-v "${ROOT_DIR}:/workspace:ro"
		-v "${ROOT_DIR}/${artifact_dir}:/artifacts"
		-w /workspace
		-e HOME=/tmp
		-e K6_NO_COLOR=true
		-e K6_SUMMARY_PATH=/artifacts/summary.json
	)
	if [[ -n "${HTTP_BENCH_DOCKER_NETWORK}" ]]; then
		docker_args+=(--network "${HTTP_BENCH_DOCKER_NETWORK}")
	else
		docker_args+=(--add-host=host.docker.internal:host-gateway)
	fi
	if [[ -n "${resolved_env_file}" ]]; then
		docker_args+=(--env-file "${resolved_env_file}")
	fi

	for variable_name in \
		HTTP_BENCH_WORKLOAD_ID \
		HTTP_BENCH_BASE_URL \
		HTTP_BENCH_PATH \
		HTTP_BENCH_METHOD \
		HTTP_BENCH_RATE \
		HTTP_BENCH_DURATION \
		HTTP_BENCH_START_RATE \
		HTTP_BENCH_STAGES_JSON \
		HTTP_BENCH_PRE_ALLOCATED_VUS \
		HTTP_BENCH_TIMEOUT \
		HTTP_BENCH_LATENCY_PERCENTILE \
		HTTP_BENCH_LATENCY_MS \
		HTTP_BENCH_MAX_ERROR_RATE \
		HTTP_BENCH_EXPECTED_STATUS \
		HTTP_BENCH_HEADERS_JSON \
		HTTP_BENCH_BODY; do
		if [[ -n "${!variable_name:-}" ]]; then
			docker_args+=(-e "${variable_name}")
		fi
	done

	if [[ "${mode}" == inspect ]]; then
		if ! docker run \
			"${docker_args[@]}" \
			"${K6_IMAGE}" \
			inspect --include-system-env-vars \
			"/workspace/${script_path}" \
			> >(tee "${ROOT_DIR}/${artifact_dir}/inspect.json") \
			2> >(tee "${ROOT_DIR}/${artifact_dir}/inspect.log" >&2); then
			echo "HTTP benchmark inspection failed; inspect ${artifact_dir}/inspect.log"
			exit 1
		fi
		echo "HTTP benchmark inspection: ${artifact_dir}/inspect.json"
		return
	fi

	local revision dirty source_fingerprint script_digest
	revision="${BENCH_SOURCE_REVISION:-$(git -C "${ROOT_DIR}" rev-parse --verify HEAD 2>/dev/null || echo unknown)}"
	if [[ -n "${BENCH_SOURCE_DIRTY:-}" ]]; then
		dirty="${BENCH_SOURCE_DIRTY}"
	elif [[ -n "$(git -C "${ROOT_DIR}" status --porcelain 2>/dev/null || true)" ]]; then
		dirty=true
	else
		dirty=false
	fi
	require_metadata_value BENCH_SOURCE_REVISION "${revision}"
	if [[ "${dirty}" != "true" && "${dirty}" != "false" ]]; then
		echo "BENCH_SOURCE_DIRTY must be true or false, got '${dirty}'"
		exit 1
	fi
	source_fingerprint="${BENCH_SOURCE_FINGERPRINT:-$(repository_fingerprint)}"
	if [[ ! "${source_fingerprint}" =~ ^[0-9a-f]{64}$ ]]; then
		echo "BENCH_SOURCE_FINGERPRINT must be a lowercase SHA-256 digest"
		exit 1
	fi
	script_digest="$(shasum -a 256 "${ROOT_DIR}/${script_path}" | awk '{print $1}')"
	{
		printf 'recorded_at_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
		printf 'git_revision=%s\n' "${revision}"
		printf 'git_dirty=%s\n' "${dirty}"
		printf 'source_fingerprint=%s\n' "${source_fingerprint}"
		printf 'k6_image=%s\n' "${K6_IMAGE}"
		printf 'scenario=%s\n' "${script_path}"
		printf 'scenario_sha256=%s\n' "${script_digest}"
		printf 'raw_samples=%s\n' "${raw_samples}"
		printf 'docker_network=%s\n' "${HTTP_BENCH_DOCKER_NETWORK:-default}"
		printf 'docker_server_version=%s\n' "$(docker version --format '{{.Server.Version}}')"
		printf 'docker_operating_system=%s\n' "$(docker info --format '{{.OperatingSystem}}')"
		printf 'docker_architecture=%s\n' "$(docker info --format '{{.Architecture}}')"
		printf 'environment_file_used=%s\n' "$([[ -n "${resolved_env_file}" ]] && echo true || echo false)"
	} >"${ROOT_DIR}/${artifact_dir}/run.meta"

	k6_args=(run)
	if [[ "${raw_samples}" == 1 ]]; then
		k6_args+=(--out json=/artifacts/samples.json.gz)
	fi
	if ! docker run \
		"${docker_args[@]}" \
		"${K6_IMAGE}" \
		"${k6_args[@]}" \
		"/workspace/${script_path}" 2>&1 |
		tee "${ROOT_DIR}/${artifact_dir}/k6.log"; then
		echo "HTTP benchmark failed; inspect ${artifact_dir}/k6.log"
		exit 1
	fi
	echo "HTTP benchmark artifacts: ${artifact_dir}/"
}

# profile:grpc-reference-benchmark:start
validate_grpc_benchmark_inputs() {
	require_metadata_value GRPC_BENCH_ARTIFACT_DIR "${GRPC_BENCH_ARTIFACT_DIR}"
	case "${GRPC_BENCH_ARTIFACT_DIR}" in
	.artifacts/bench/grpc | .artifacts/bench/grpc/*) ;;
	*)
		echo "GRPC_BENCH_ARTIFACT_DIR must stay under .artifacts/bench/grpc/"
		exit 1
		;;
	esac
	if [[ "${GRPC_BENCH_ARTIFACT_DIR}" =~ (^|/)\.\.(/|$) ]]; then
		echo "GRPC_BENCH_ARTIFACT_DIR must stay under .artifacts/bench/grpc/"
		exit 1
	fi
	if [[ "${GRPC_BENCH_SCRIPT}" == /* || "${GRPC_BENCH_SCRIPT}" =~ (^|/)\.\.(/|$) ]]; then
		echo "GRPC_BENCH_SCRIPT must be a repository-relative path"
		exit 1
	fi
	[[ -f "${ROOT_DIR}/${GRPC_BENCH_SCRIPT}" ]] || {
		echo "gRPC benchmark script not found: ${GRPC_BENCH_SCRIPT}"
		exit 1
	}
	[[ "${GRPC_BENCH_RAW_SAMPLES}" == 0 || "${GRPC_BENCH_RAW_SAMPLES}" == 1 ]] || {
		echo "GRPC_BENCH_RAW_SAMPLES must be 0 or 1"
		exit 1
	}
	[[ "${GRPC_BENCH_MODE}" == smoke || "${GRPC_BENCH_MODE}" == synthetic ]] || {
		echo "GRPC_BENCH_MODE must be smoke or synthetic"
		exit 1
	}
	require_positive_integer GRPC_BENCH_PAYLOAD_BYTES "${GRPC_BENCH_PAYLOAD_BYTES}"
	require_positive_integer GRPC_BENCH_STREAM_MESSAGES "${GRPC_BENCH_STREAM_MESSAGES}"
	require_positive_integer GRPC_BENCH_VUS "${GRPC_BENCH_VUS}"
	require_positive_integer GRPC_BENCH_READY_TIMEOUT_TICKS "${GRPC_BENCH_READY_TIMEOUT_TICKS}"
	require_metadata_value GRPC_BENCH_WORKLOAD_ID "${GRPC_BENCH_WORKLOAD_ID}"
	if ((GRPC_BENCH_PAYLOAD_BYTES > 1048576)); then
		echo "GRPC_BENCH_PAYLOAD_BYTES must be at most 1048576"
		exit 1
	fi
	if ((GRPC_BENCH_STREAM_MESSAGES > 1024)); then
		echo "GRPC_BENCH_STREAM_MESSAGES must be at most 1024"
		exit 1
	fi
	for duration_name in GRPC_BENCH_WARMUP_DURATION GRPC_BENCH_DURATION GRPC_BENCH_RPC_TIMEOUT; do
		if [[ ! "${!duration_name}" =~ ^[1-9][0-9]*(ms|s|m)$ ]]; then
			echo "${duration_name} must be a positive k6 duration using ms, s, or m"
			exit 1
		fi
	done
}

ensure_owned_grpc_artifact_dir() {
	local relative_path="$1"
	local current_path="${ROOT_DIR}"
	local component resolved_path owned_root
	local -a components

	case "${relative_path}" in
	.artifacts/bench/grpc | .artifacts/bench/grpc/*) ;;
	*)
		echo "gRPC benchmark artifact path must stay under .artifacts/bench/grpc/"
		exit 1
		;;
	esac
	IFS='/' read -r -a components <<<"${relative_path}"
	for component in "${components[@]}"; do
		if [[ -z "${component}" || "${component}" == . || "${component}" == .. ]]; then
			echo "gRPC benchmark artifact path contains an unsafe component"
			exit 1
		fi
		current_path="${current_path}/${component}"
		if [[ -L "${current_path}" ]]; then
			echo "gRPC benchmark artifact path contains a symlink: ${relative_path}"
			exit 1
		fi
	done

	mkdir -p "${current_path}"
	current_path="${ROOT_DIR}"
	for component in "${components[@]}"; do
		current_path="${current_path}/${component}"
		if [[ -L "${current_path}" ]]; then
			echo "gRPC benchmark artifact path contains a symlink: ${relative_path}"
			exit 1
		fi
	done

	owned_root="$(CDPATH='' cd "${ROOT_DIR}/.artifacts/bench/grpc" && pwd -P)"
	resolved_path="$(CDPATH='' cd "${current_path}" && pwd -P)"
	case "${resolved_path}" in
	"${owned_root}" | "${owned_root}"/*) ;;
	*)
		echo "gRPC benchmark artifact path escapes .artifacts/bench/grpc/: ${relative_path}"
		exit 1
		;;
	esac
}

grpc_docker_route() {
	case "$(uname -s)" in
	Darwin)
		printf 'host.docker.internal'
		;;
	Linux)
		printf '127.0.0.1'
		;;
	*)
		echo "gRPC benchmark Docker routing supports Darwin and Linux" >&2
		exit 1
		;;
	esac
}

write_grpc_run_metadata() {
	local output="$1"
	local listener_address="$2"
	local docker_target="$3"
	local route="$4"
	local revision dirty source_fingerprint scenario_digest schema_digest

	revision="${BENCH_SOURCE_REVISION:-$(git -C "${ROOT_DIR}" rev-parse --verify HEAD 2>/dev/null || echo unknown)}"
	if [[ -n "${BENCH_SOURCE_DIRTY:-}" ]]; then
		dirty="${BENCH_SOURCE_DIRTY}"
	elif [[ -n "$(git -C "${ROOT_DIR}" status --porcelain 2>/dev/null || true)" ]]; then
		dirty=true
	else
		dirty=false
	fi
	require_metadata_value BENCH_SOURCE_REVISION "${revision}"
	if [[ "${dirty}" != true && "${dirty}" != false ]]; then
		echo "BENCH_SOURCE_DIRTY must be true or false, got '${dirty}'"
		exit 1
	fi
	source_fingerprint="${BENCH_SOURCE_FINGERPRINT:-$(repository_fingerprint)}"
	if [[ ! "${source_fingerprint}" =~ ^[0-9a-f]{64}$ ]]; then
		echo "BENCH_SOURCE_FINGERPRINT must be a lowercase SHA-256 digest"
		exit 1
	fi
	scenario_digest="$(shasum -a 256 "${ROOT_DIR}/${GRPC_BENCH_SCRIPT}" | awk '{print $1}')"
	schema_digest="$(shasum -a 256 "${ROOT_DIR}/examples/grpc-reference-service/api/proto/reference/v1/echo.proto" | awk '{print $1}')"

	{
		printf 'recorded_at_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
		printf 'git_revision=%s\n' "${revision}"
		printf 'git_dirty=%s\n' "${dirty}"
		printf 'source_fingerprint=%s\n' "${source_fingerprint}"
		printf 'k6_image=%s\n' "${K6_IMAGE}"
		printf 'scenario=%s\n' "${GRPC_BENCH_SCRIPT}"
		printf 'scenario_sha256=%s\n' "${scenario_digest}"
		printf 'schema=examples/grpc-reference-service/api/proto/reference/v1/echo.proto\n'
		printf 'schema_sha256=%s\n' "${schema_digest}"
		printf 'evidence_level=%s\n' "${GRPC_BENCH_MODE}"
		printf 'workload_id=%s\n' "${GRPC_BENCH_WORKLOAD_ID}"
		printf 'server_listener=%s\n' "${listener_address}"
		printf 'docker_target=%s\n' "${docker_target}"
		printf 'docker_route=%s\n' "${route}"
		printf 'transport_security=plaintext-loopback\n'
		printf 'telemetry_mode=production-otel-no-exporter-never-sampled-json-discard\n'
		printf 'payload_bytes=%s\n' "${GRPC_BENCH_PAYLOAD_BYTES}"
		printf 'stream_messages=%s\n' "${GRPC_BENCH_STREAM_MESSAGES}"
		printf 'virtual_users=%s\n' "$([[ "${GRPC_BENCH_MODE}" == smoke ]] && echo 1 || echo "${GRPC_BENCH_VUS}")"
		printf 'connections_per_vu=1\n'
		printf 'warmup_duration=%s\n' "$([[ "${GRPC_BENCH_MODE}" == smoke ]] && echo none || echo "${GRPC_BENCH_WARMUP_DURATION}")"
		printf 'warmup_completion_budget=%s\n' "$([[ "${GRPC_BENCH_MODE}" == smoke ]] && echo none || echo "${GRPC_BENCH_RPC_TIMEOUT}")"
		printf 'measured_duration=%s\n' "$([[ "${GRPC_BENCH_MODE}" == smoke ]] && echo one-iteration || echo "${GRPC_BENCH_DURATION}")"
		printf 'rpc_timeout=%s\n' "${GRPC_BENCH_RPC_TIMEOUT}"
		printf 'raw_samples=%s\n' "${GRPC_BENCH_RAW_SAMPLES}"
		printf 'docker_server_version=%s\n' "$("${GRPC_BENCH_DOCKER_BIN}" version --format '{{.Server.Version}}')"
		printf 'docker_operating_system=%s\n' "$("${GRPC_BENCH_DOCKER_BIN}" info --format '{{.OperatingSystem}}')"
		printf 'docker_architecture=%s\n' "$("${GRPC_BENCH_DOCKER_BIN}" info --format '{{.Architecture}}')"
		printf 'server_go_version=%s\n' "$("${GRPC_BENCH_GO_BIN}" version)"
		printf 'server_goos=%s\n' "$("${GRPC_BENCH_GO_BIN}" env GOOS)"
		printf 'server_goarch=%s\n' "$("${GRPC_BENCH_GO_BIN}" env GOARCH)"
		printf 'server_resource_limits=unavailable\n'
		printf 'load_generator_resource_limits=unavailable\n'
		printf 'decision_grade_resource_observables=unavailable\n'
	} >"${output}"
}

run_grpc_inspect() (
	validate_grpc_benchmark_inputs
	ensure_grpc_docker

	local run_dir="${ROOT_DIR}/${GRPC_BENCH_ARTIFACT_DIR}/inspect"
	local host_uid host_gid
	ensure_owned_grpc_artifact_dir "${GRPC_BENCH_ARTIFACT_DIR}/inspect"
	rm -f "${run_dir}/inspect.json" "${run_dir}/inspect.log"
	host_uid="$(id -u 2>/dev/null || echo 0)"
	host_gid="$(id -g 2>/dev/null || echo 0)"

	if ! "${GRPC_BENCH_DOCKER_BIN}" run --rm \
		-u "${host_uid}:${host_gid}" \
		-v "${ROOT_DIR}:/workspace:ro" \
		-w /workspace \
		-e HOME=/tmp \
		-e K6_NO_COLOR=true \
		-e GRPC_BENCH_MODE="${GRPC_BENCH_MODE}" \
		"${K6_IMAGE}" \
		inspect --include-system-env-vars \
		"/workspace/${GRPC_BENCH_SCRIPT}" \
		>"${run_dir}/inspect.json" 2>"${run_dir}/inspect.log"; then
		cat "${run_dir}/inspect.log" >&2
		echo "gRPC benchmark inspection failed; inspect ${GRPC_BENCH_ARTIFACT_DIR}/inspect/inspect.log"
		exit 1
	fi
	echo "gRPC benchmark inspection: ${GRPC_BENCH_ARTIFACT_DIR}/inspect/inspect.json"
)

run_grpc_benchmark() {
	validate_grpc_benchmark_inputs
	ensure_grpc_docker

	local run_dir="${ROOT_DIR}/${GRPC_BENCH_ARTIFACT_DIR}/${GRPC_BENCH_MODE}"
	local server_binary="${GRPC_BENCH_SERVER_BINARY}"
	local ready_address=""
	local docker_host docker_target port route
	local host_uid host_gid tick status
	local -a docker_args k6_args
	owned_binary=""
	server_pid=""
	client_pid=""

	stop_grpc_owned_process() {
		local pid="$1"
		if [[ -z "${pid}" ]]; then
			return
		fi
		if kill -0 "${pid}" >/dev/null 2>&1; then
			kill -TERM "${pid}" >/dev/null 2>&1 || true
			for _ in $(seq 1 50); do
				kill -0 "${pid}" >/dev/null 2>&1 || break
				sleep 0.1
			done
			if kill -0 "${pid}" >/dev/null 2>&1; then
				kill -KILL "${pid}" >/dev/null 2>&1 || true
			fi
		fi
		wait "${pid}" 2>/dev/null || true
	}

	cleanup_grpc_benchmark() {
		local cleanup_status="$1"
		trap - EXIT INT TERM
		stop_grpc_owned_process "${client_pid}"
		stop_grpc_owned_process "${server_pid}"
		if [[ -n "${owned_binary}" ]]; then
			rm -f "${owned_binary}"
		fi
		exit "${cleanup_status}"
	}
	trap 'cleanup_grpc_benchmark $?' EXIT
	trap 'exit 130' INT
	trap 'exit 143' TERM

	ensure_owned_grpc_artifact_dir "${GRPC_BENCH_ARTIFACT_DIR}/${GRPC_BENCH_MODE}"
	rm -f \
		"${run_dir}/k6.log" \
		"${run_dir}/run.meta" \
		"${run_dir}/samples.json.gz" \
		"${run_dir}/server.log" \
		"${run_dir}/summary.json"

	if [[ -z "${server_binary}" ]]; then
		command -v "${GRPC_BENCH_GO_BIN}" >/dev/null 2>&1 || {
			echo "Go is required to build the gRPC benchmark server"
			exit 1
		}
		owned_binary="$(mktemp "${ROOT_DIR}/.artifacts/bench/.grpc-benchmark-server.XXXXXX")"
		server_binary="${owned_binary}"
		if ! "${GRPC_BENCH_GO_BIN}" build -o "${server_binary}" ./examples/grpc-reference-service/cmd/benchmark-server; then
			echo "failed to build the gRPC benchmark server"
			exit 1
		fi
	elif [[ ! -x "${server_binary}" ]]; then
		echo "GRPC_BENCH_SERVER_BINARY must name an executable file"
		exit 1
	fi

	"${server_binary}" >"${run_dir}/server.log" 2>&1 &
	server_pid=$!
	for ((tick = 0; tick < GRPC_BENCH_READY_TIMEOUT_TICKS; tick++)); do
		ready_address="$(sed -n 's/^GRPC_BENCH_READY=//p' "${run_dir}/server.log" | head -n 1)"
		if [[ -n "${ready_address}" ]]; then
			break
		fi
		if ! kill -0 "${server_pid}" >/dev/null 2>&1; then
			wait "${server_pid}" 2>/dev/null || status=$?
			server_pid=""
			cat "${run_dir}/server.log" >&2
			echo "gRPC benchmark server exited before readiness with status ${status:-0}"
			exit 1
		fi
		sleep 0.05
	done
	if [[ ! "${ready_address}" =~ ^127\.0\.0\.1:([1-9][0-9]{0,4})$ ]]; then
		echo "gRPC benchmark server emitted an invalid ready address: '${ready_address}'"
		exit 1
	fi
	port="${BASH_REMATCH[1]}"
	if ((port > 65535)); then
		echo "gRPC benchmark server emitted an invalid port: '${port}'"
		exit 1
	fi

	docker_host="$(grpc_docker_route)"
	docker_target="${docker_host}:${port}"
	case "$(uname -s)" in
	Darwin)
		route="docker-desktop-host"
		;;
	Linux)
		route="host-network"
		;;
	esac
	write_grpc_run_metadata "${run_dir}/run.meta" "${ready_address}" "${docker_target}" "${route}"

	host_uid="$(id -u 2>/dev/null || echo 0)"
	host_gid="$(id -g 2>/dev/null || echo 0)"
	docker_args=(
		run
		--rm
		-u "${host_uid}:${host_gid}"
		-v "${ROOT_DIR}:/workspace:ro"
		-v "${run_dir}:/artifacts"
		-w /workspace
		-e HOME=/tmp
		-e K6_NO_COLOR=true
		-e K6_SUMMARY_PATH=/artifacts/summary.json
		-e GRPC_BENCH_MODE="${GRPC_BENCH_MODE}"
		-e GRPC_BENCH_TARGET="${docker_target}"
		-e GRPC_BENCH_WORKLOAD_ID="${GRPC_BENCH_WORKLOAD_ID}"
		-e GRPC_BENCH_PAYLOAD_BYTES="${GRPC_BENCH_PAYLOAD_BYTES}"
		-e GRPC_BENCH_STREAM_MESSAGES="${GRPC_BENCH_STREAM_MESSAGES}"
		-e GRPC_BENCH_VUS="${GRPC_BENCH_VUS}"
		-e GRPC_BENCH_WARMUP_DURATION="${GRPC_BENCH_WARMUP_DURATION}"
		-e GRPC_BENCH_DURATION="${GRPC_BENCH_DURATION}"
		-e GRPC_BENCH_RPC_TIMEOUT="${GRPC_BENCH_RPC_TIMEOUT}"
	)
	if [[ "${route}" == host-network ]]; then
		docker_args+=(--network host)
	fi
	k6_args=(run)
	if [[ "${GRPC_BENCH_RAW_SAMPLES}" == 1 ]]; then
		k6_args+=(--out json=/artifacts/samples.json.gz)
	fi
	"${GRPC_BENCH_DOCKER_BIN}" \
		"${docker_args[@]}" \
		"${K6_IMAGE}" \
		"${k6_args[@]}" \
		"/workspace/${GRPC_BENCH_SCRIPT}" >"${run_dir}/k6.log" 2>&1 &
	client_pid=$!
	status=0
	wait "${client_pid}" || status=$?
	client_pid=""
	if [[ "${status}" -ne 0 ]]; then
		cat "${run_dir}/k6.log" >&2
		echo "gRPC benchmark failed; inspect ${GRPC_BENCH_ARTIFACT_DIR}/${GRPC_BENCH_MODE}/k6.log"
		exit 1
	fi
	cat "${run_dir}/k6.log"
	[[ -s "${run_dir}/summary.json" ]] || {
		echo "gRPC benchmark did not produce a non-empty summary"
		exit 1
	}
	echo "gRPC benchmark artifacts: ${GRPC_BENCH_ARTIFACT_DIR}/${GRPC_BENCH_MODE}/"
}
# profile:grpc-reference-benchmark:end

run_infrastructure_check() {
	local artifact_root="${BENCHMARK_INFRA_ARTIFACT_DIR:-.artifacts/bench/infra-check}"
	local postgres_test_image compose_postgres_image

	if [[ ! "${K6_IMAGE_DEFAULT}" =~ ^grafana/k6:[0-9]+\.[0-9]+\.[0-9]+@sha256:[[:xdigit:]]{64}$ ]]; then
		echo "K6_IMAGE_DEFAULT must be versioned and digest-pinned"
		exit 1
	fi
	postgres_test_image="$(sed -n 's/^const DefaultImage = "\(.*\)"$/\1/p' internal/infra/postgres/pgtest/pgtest.go)"
	compose_postgres_image="$(awk '/^[[:space:]]+image:[[:space:]]+postgres:/ { print $2; exit }' env/docker-compose.yml)"
	if [[ -z "${postgres_test_image}" || "${postgres_test_image}" != "${compose_postgres_image}" ]]; then
		echo "Testcontainers and Compose must use the same digest-pinned PostgreSQL image"
		exit 1
	fi
	if [[ ! "${postgres_test_image}" =~ ^postgres:[0-9]+@sha256:[[:xdigit:]]{64}$ ]]; then
		echo "PostgreSQL benchmark image must be versioned and digest-pinned"
		exit 1
	fi
	[[ -r "scripts/dev/benchmark-remote.sh" ]] || {
		echo "DigitalOcean benchmark runner is missing"
		exit 1
	}
	[[ -r "scripts/dev/benchmark-remote-check.sh" ]] || {
		echo "DigitalOcean benchmark lifecycle check is missing"
		exit 1
	}
	[[ -r ".agents/skills/digitalocean-benchmark-runner/SKILL.md" ]] || {
		echo "DigitalOcean benchmark runner skill is missing"
		exit 1
	}
	[[ -r ".agents/skills/digitalocean-benchmark-runner/references/digitalocean.md" ]] || {
		echo "DigitalOcean benchmark runner runbook is missing"
		exit 1
	}

	bash -n "${BASH_SOURCE[0]}"
	bash -n scripts/dev/benchmark-remote.sh
	bash -n scripts/dev/benchmark-remote-check.sh
	bash scripts/dev/benchmark-remote-check.sh
	BENCH_WORKLOAD_ID=infra-check-stdlib \
		BENCH_PACKAGE=crypto/sha256 \
		BENCH_PATTERN='BenchmarkHash8Bytes$' \
		BENCH_COUNT=3 \
	BENCH_TIME=1x \
		BENCH_OUTPUT="${artifact_root}/baseline.txt" \
		bash "${BASH_SOURCE[0]}" run
	cp "${artifact_root}/baseline.txt" "${artifact_root}/current.txt"
	cp "${artifact_root}/baseline.txt.meta" "${artifact_root}/current.txt.meta"
	BENCH_BASELINE="${artifact_root}/baseline.txt" \
		BENCH_CURRENT="${artifact_root}/current.txt" \
		BENCH_COMPARE_OUTPUT="${artifact_root}/comparison.txt" \
		bash "${BASH_SOURCE[0]}" compare

	HTTP_BENCH_ENV_FILE='' \
		HTTP_BENCH_ARTIFACT_DIR="${artifact_root}/http-steady" \
		HTTP_BENCH_WORKLOAD_ID=infra-check-steady \
		HTTP_BENCH_BASE_URL=http://example.invalid \
		HTTP_BENCH_PATH=/benchmark \
		HTTP_BENCH_METHOD=GET \
		HTTP_BENCH_RATE=1 \
		HTTP_BENCH_DURATION=1s \
		HTTP_BENCH_PRE_ALLOCATED_VUS=1 \
		HTTP_BENCH_TIMEOUT=1s \
		HTTP_BENCH_LATENCY_PERCENTILE=95 \
		HTTP_BENCH_LATENCY_MS=1000 \
		HTTP_BENCH_MAX_ERROR_RATE=0.01 \
		HTTP_BENCH_EXPECTED_STATUS=200 \
		bash "${BASH_SOURCE[0]}" http-inspect
	grep -Fq '"executor": "constant-arrival-rate"' "${artifact_root}/http-steady/inspect.json"
	grep -Fq '"flow_duration{flow:single_flow}"' "${artifact_root}/http-steady/inspect.json"
	grep -Fq '"p(99)"' "${artifact_root}/http-steady/inspect.json"

	HTTP_BENCH_ENV_FILE='' \
		HTTP_BENCH_ARTIFACT_DIR="${artifact_root}/http-ramping" \
		HTTP_BENCH_WORKLOAD_ID=infra-check-ramping \
		HTTP_BENCH_BASE_URL=http://example.invalid \
		HTTP_BENCH_PATH=/benchmark \
		HTTP_BENCH_METHOD=GET \
		HTTP_BENCH_START_RATE=0 \
		HTTP_BENCH_STAGES_JSON='[{"target":2,"duration":"1s"},{"target":0,"duration":"0"}]' \
		HTTP_BENCH_PRE_ALLOCATED_VUS=2 \
		HTTP_BENCH_TIMEOUT=1s \
		HTTP_BENCH_LATENCY_PERCENTILE=99 \
		HTTP_BENCH_LATENCY_MS=1000 \
		HTTP_BENCH_MAX_ERROR_RATE=0.01 \
		HTTP_BENCH_EXPECTED_STATUS=200 \
		bash "${BASH_SOURCE[0]}" http-inspect
	grep -Fq '"executor": "ramping-arrival-rate"' "${artifact_root}/http-ramping/inspect.json"
	grep -Fq '"startRate": 0' "${artifact_root}/http-ramping/inspect.json"

	echo "benchmark infrastructure check passed"
}

case "${1:-}" in
run)
	run_benchmark
	;;
compare)
	compare_benchmarks
	;;
profile)
	profile_benchmark
	;;
http)
	run_http_benchmark run
	;;
http-inspect)
	run_http_benchmark inspect
	;;
# profile:grpc-reference-benchmark:start
grpc)
	GRPC_BENCH_MODE=synthetic run_grpc_benchmark
	;;
grpc-smoke)
	GRPC_BENCH_MODE=smoke run_grpc_benchmark
	;;
grpc-inspect)
	run_grpc_inspect
	;;
# profile:grpc-reference-benchmark:end
check)
	run_infrastructure_check
	;;
source-fingerprint)
	repository_fingerprint
	;;
*)
	usage
	exit 1
	;;
esac
