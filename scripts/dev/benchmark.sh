#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

GO_BIN="${GO:-go}"
K6_IMAGE_DEFAULT="grafana/k6:2.1.0@sha256:65c920dc067d5e2e00befbf982af6ad6ad0117034e8b1c65817c7975c52d4669"
K6_IMAGE="${K6_IMAGE:-${K6_IMAGE_DEFAULT}}"

usage() {
	echo "usage: $0 <capture|compare|http|inspect-http|pgo-manifest|verify-pgo|source-fingerprint|pgo-fingerprint|self-test>"
}

die() {
	echo "$*" >&2
	exit 1
}

require_value() {
	local name="$1" value="$2"
	[[ -n "${value}" && "${value}" != *$'\n'* && "${value}" != *$'\r'* ]] ||
		die "${name} must be a non-empty single-line value"
}

require_positive_integer() {
	local name="$1" value="$2"
	[[ "${value}" =~ ^[1-9][0-9]*$ ]] || die "${name} must be a positive integer"
}

sha256_file() {
	shasum -a 256 "$1" | awk '{print $1}'
}

repository_fingerprint() (
	local snapshot_dir object_dir tree
	snapshot_dir="$(mktemp -d "${TMPDIR:-/tmp}/benchmark-source.XXXXXX")"
	trap 'rm -rf -- "${snapshot_dir}"' EXIT
	mkdir "${snapshot_dir}/objects"
	object_dir="$(CDPATH='' cd -- "$(git rev-parse --git-path objects)" && pwd)"

	snapshot_git() {
		GIT_INDEX_FILE="${snapshot_dir}/index" \
			GIT_OBJECT_DIRECTORY="${snapshot_dir}/objects" \
			GIT_ALTERNATE_OBJECT_DIRECTORIES="${object_dir}" \
			git "$@"
	}

	snapshot_git read-tree HEAD
	snapshot_git add -A -- .
	tree="$(snapshot_git write-tree)"
	printf '%s' "${tree}" | shasum -a 256 | awk '{print $1}'
)

pgo_candidate_fingerprint() (
	local snapshot_dir object_dir tree path
	snapshot_dir="$(mktemp -d "${TMPDIR:-/tmp}/pgo-source.XXXXXX")"
	trap 'rm -rf -- "${snapshot_dir}"' EXIT
	mkdir "${snapshot_dir}/objects"
	object_dir="$(CDPATH='' cd -- "$(git rev-parse --git-path objects)" && pwd)"

	snapshot_git() {
		GIT_INDEX_FILE="${snapshot_dir}/index" \
			GIT_OBJECT_DIRECTORY="${snapshot_dir}/objects" \
			GIT_ALTERNATE_OBJECT_DIRECTORIES="${object_dir}" \
			git "$@"
	}

	snapshot_git read-tree --empty
	for path in Makefile build/docker/Dockerfile go.mod go.sum cmd/internal cmd/service internal migrations; do
		[[ ! -e "${path}" ]] || snapshot_git add -A -- "${path}"
	done
	tree="$(snapshot_git write-tree)"
	printf '%s' "${tree}" | shasum -a 256 | awk '{print $1}'
)

revision_fingerprint() {
	git rev-parse "$1^{tree}" | shasum -a 256 | awk '{print $1}'
}

schema_fingerprint() {
	local path="${BENCH_SCHEMA_PATH:-}" file relative
	local -a files=()
	if [[ -z "${path}" ]]; then
		printf 'none'
		return
	fi
	[[ -e "${path}" ]] || die "benchmark schema path not found: ${path}"
	if [[ -f "${path}" ]]; then
		git hash-object "${path}"
		return
	fi
	while IFS= read -r -d '' file; do files+=("${file}"); done < <(find "${path}" -type f -print0 | LC_ALL=C sort -z)
	[[ "${#files[@]}" -gt 0 ]] || die "benchmark schema path contains no files: ${path}"
	{
		for file in "${files[@]}"; do
			relative="${file#"${path}"/}"
			printf '%s\t%s\n' "${relative}" "$(git hash-object "${file}")"
		done
	} | git hash-object --stdin
}

module_fingerprint() {
	shasum -a 256 go.mod go.sum | shasum -a 256 | awk '{print $1}'
}

command_string() {
	printf '%q ' "$@"
}

metadata_value() {
	local file="$1" key="$2"
	awk -v key="${key}" '
		index($0, key "=") == 1 {
			count++
			print substr($0, length(key) + 2)
		}
		END { if (count != 1) exit 2 }
	' "${file}"
}

comparable_keys() {
	printf '%s\n' \
		format benchmark_command benchmark_package benchmark_pattern \
		benchmark_count benchmark_time benchmark_tags workload_id accepted_budget \
		response_owner dependency_id schema_fingerprint benchmark_cpu host kernel \
		logical_cpus gomaxprocs go_version goos goarch goamd64 cgo_enabled \
		module_fingerprint
}

compare_metadata() {
	local baseline="$1" current="$2" key baseline_value current_value
	while IFS= read -r key; do
		baseline_value="$(metadata_value "${baseline}" "${key}")" || {
			echo "baseline metadata is missing unique key ${key}" >&2
			return 1
		}
		current_value="$(metadata_value "${current}" "${key}")" || {
			echo "current metadata is missing unique key ${key}" >&2
			return 1
		}
		if [[ "${baseline_value}" != "${current_value}" ]]; then
			echo "benchmark metadata mismatch for ${key}: baseline='${baseline_value}' current='${current_value}'" >&2
			return 1
		fi
	done < <(comparable_keys)
}

capture() (
	local package="${BENCH_PACKAGE:-}" pattern="${BENCH_PATTERN:-}"
	local output="${BENCH_OUTPUT:-}" count="${BENCH_COUNT:-10}" time="${BENCH_TIME:-1s}"
	local tags="${BENCH_TAGS:-}" workload="${BENCH_WORKLOAD_ID:-}"
	local budget="${BENCH_BUDGET:-}" owner="${BENCH_RESPONSE_OWNER:-}"
	local dependency="${BENCH_DEPENDENCY_ID:-none}" output_dir raw meta cpu logical_cpus
	local -a command

	require_value BENCH_PACKAGE "${package}"
	require_value BENCH_PATTERN "${pattern}"
	require_value BENCH_OUTPUT "${output}"
	require_value BENCH_WORKLOAD_ID "${workload}"
	require_value BENCH_BUDGET "${budget}"
	require_value BENCH_RESPONSE_OWNER "${owner}"
	require_value BENCH_DEPENDENCY_ID "${dependency}"
	require_positive_integer BENCH_COUNT "${count}"
	[[ "${package}" != "./..." && "${package}" != *[[:space:]]* ]] ||
		die "BENCH_PACKAGE must name one concrete package"

	output_dir="$(dirname "${output}")"
	mkdir -p "${output_dir}"
	raw="$(mktemp "${output_dir}/.benchmark.XXXXXX")"
	meta="$(mktemp "${output_dir}/.benchmark-meta.XXXXXX")"
	trap 'rm -f -- "${raw}" "${meta}"' EXIT

	command=("${GO_BIN}" test -mod=readonly -run '^$' -bench "${pattern}" -benchmem -benchtime "${time}" -count "${count}")
	[[ -z "${tags}" ]] || command+=(-tags "${tags}")
	command+=("${package}")
	"${command[@]}" >"${raw}"
	grep -Eq '^Benchmark[^[:space:]]*[[:space:]]' "${raw}" ||
		die "no benchmarks matched ${package} ${pattern}"

	cpu="$(awk '/^cpu:/ { sub(/^cpu:[[:space:]]*/, ""); print; exit }' "${raw}")"
	logical_cpus="$(getconf _NPROCESSORS_ONLN 2>/dev/null || true)"
	if [[ -z "${logical_cpus}" ]] && command -v sysctl >/dev/null 2>&1; then
		logical_cpus="$(sysctl -n hw.logicalcpu 2>/dev/null || true)"
	fi

	{
		printf 'format=go-benchmark-evidence-v1\n'
		printf 'recorded_at_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
		printf 'git_revision=%s\n' "$(git rev-parse HEAD)"
		printf 'source_fingerprint=%s\n' "$(repository_fingerprint)"
		printf 'benchmark_command=%s\n' "$(command_string "${command[@]}")"
		printf 'benchmark_package=%s\n' "${package}"
		printf 'benchmark_pattern=%s\n' "${pattern}"
		printf 'benchmark_count=%s\n' "${count}"
		printf 'benchmark_time=%s\n' "${time}"
		printf 'benchmark_tags=%s\n' "${tags:-none}"
		printf 'workload_id=%s\n' "${workload}"
		printf 'accepted_budget=%s\n' "${budget}"
		printf 'response_owner=%s\n' "${owner}"
		printf 'dependency_id=%s\n' "${dependency}"
		printf 'schema_fingerprint=%s\n' "$(schema_fingerprint)"
		printf 'benchmark_cpu=%s\n' "${cpu:-unknown}"
		printf 'host=%s\n' "$(hostname)"
		printf 'kernel=%s\n' "$(uname -sr)"
		printf 'logical_cpus=%s\n' "${logical_cpus:-unknown}"
		printf 'gomaxprocs=%s\n' "${GOMAXPROCS:-default}"
		printf 'go_version=%s\n' "$("${GO_BIN}" env GOVERSION)"
		printf 'goos=%s\n' "$("${GO_BIN}" env GOOS)"
		printf 'goarch=%s\n' "$("${GO_BIN}" env GOARCH)"
		printf 'goamd64=%s\n' "$("${GO_BIN}" env GOAMD64)"
		printf 'cgo_enabled=%s\n' "$("${GO_BIN}" env CGO_ENABLED)"
		printf 'module_fingerprint=%s\n' "$(module_fingerprint)"
	} >"${meta}"

	mv "${raw}" "${output}"
	mv "${meta}" "${output}.meta"
	echo "benchmark evidence: ${output} ${output}.meta"
)

compare() (
	local baseline="${BENCH_BASELINE:-.artifacts/bench/baseline.txt}"
	local current="${BENCH_CURRENT:-.artifacts/bench/current.txt}"
	local output="${BENCH_COMPARE_OUTPUT:-.artifacts/bench/comparison.txt}"
	local temporary

	for file in "${baseline}" "${current}" "${baseline}.meta" "${current}.meta"; do
		[[ -f "${file}" ]] || die "benchmark evidence not found: ${file}"
	done
	compare_metadata "${baseline}.meta" "${current}.meta"
	mkdir -p "$(dirname "${output}")"
	temporary="$(mktemp "$(dirname "${output}")/.benchstat.XXXXXX")"
	trap 'rm -f -- "${temporary}"' EXIT
	"${GO_BIN}" tool -modfile=tools/go.mod benchstat "${baseline}" "${current}" >"${temporary}"
	mv "${temporary}" "${output}"
	echo "benchmark comparison: ${output}"
)

write_http_metadata() {
	local output="$1" scenario="$2"
	{
		printf 'format=http-benchmark-evidence-v1\n'
		printf 'recorded_at_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
		printf 'git_revision=%s\n' "$(git rev-parse HEAD)"
		printf 'source_fingerprint=%s\n' "$(repository_fingerprint)"
		printf 'scenario=%s\n' "${scenario}"
		printf 'scenario_sha256=%s\n' "$(sha256_file "${scenario}")"
		printf 'k6_image=%s\n' "${K6_IMAGE}"
		printf 'accepted_budget=%s\n' "${BENCH_BUDGET}"
		printf 'response_owner=%s\n' "${BENCH_RESPONSE_OWNER}"
		printf 'host=%s\n' "$(hostname)"
		printf 'kernel=%s\n' "$(uname -sr)"
		printf 'docker_server_version=%s\n' "$(docker version --format '{{.Server.Version}}')"
		printf 'docker_network=%s\n' "${HTTP_BENCH_DOCKER_NETWORK:-host}"
	} >"${output}"
}

require_docker() {
	command -v docker >/dev/null 2>&1 || die "docker is required"
	docker info >/dev/null 2>&1 || die "docker daemon is unavailable"
}

http_capture() {
	local env_file="${HTTP_BENCH_ENV_FILE:-.env.bench}"
	local artifact_dir="${HTTP_BENCH_ARTIFACT_DIR:-.artifacts/bench/http}"
	local scenario="test/performance/http/single-flow.js" artifact_abs variable
	local -a docker_args

	require_value BENCH_BUDGET "${BENCH_BUDGET:-}"
	require_value BENCH_RESPONSE_OWNER "${BENCH_RESPONSE_OWNER:-}"
	require_docker
	[[ -f "${env_file}" ]] || die "HTTP benchmark environment file not found: ${env_file}"
	[[ -f "${scenario}" ]] || die "HTTP benchmark scenario not found: ${scenario}"
	mkdir -p "${artifact_dir}"
	for file in summary.json run.meta k6.log; do
		[[ ! -e "${artifact_dir}/${file}" ]] || die "refusing to overwrite ${artifact_dir}/${file}"
	done
	artifact_abs="$(CDPATH='' cd -- "${artifact_dir}" && pwd)"
	write_http_metadata "${artifact_dir}/run.meta" "${scenario}"

	docker_args=(--rm --user "$(id -u):$(id -g)" --network "${HTTP_BENCH_DOCKER_NETWORK:-host}")
	docker_args+=(--env-file "${env_file}")
	for variable in \
		HTTP_BENCH_WORKLOAD_ID HTTP_BENCH_BASE_URL HTTP_BENCH_PATH HTTP_BENCH_METHOD \
		HTTP_BENCH_RATE HTTP_BENCH_DURATION HTTP_BENCH_START_RATE HTTP_BENCH_STAGES_JSON \
		HTTP_BENCH_PRE_ALLOCATED_VUS HTTP_BENCH_TIMEOUT HTTP_BENCH_LATENCY_PERCENTILE \
		HTTP_BENCH_LATENCY_MS HTTP_BENCH_MAX_ERROR_RATE HTTP_BENCH_EXPECTED_STATUS \
		HTTP_BENCH_HEADERS_JSON HTTP_BENCH_BODY; do
		[[ -z "${!variable:-}" ]] || docker_args+=(-e "${variable}")
	done
	docker_args+=(-e K6_SUMMARY_PATH=/artifacts/summary.json)
	docker_args+=(-v "${ROOT_DIR}/test/performance/http:/work:ro" -v "${artifact_abs}:/artifacts")
	if ! docker run "${docker_args[@]}" "${K6_IMAGE}" run /work/single-flow.js 2>&1 |
		tee "${artifact_dir}/k6.log"; then
		die "HTTP benchmark failed; inspect ${artifact_dir}/k6.log"
	fi
	[[ -s "${artifact_dir}/summary.json" ]] || die "HTTP benchmark produced no durable summary"
	echo "HTTP benchmark evidence: ${artifact_dir}/"
}

inspect_http() {
	require_docker
	docker run --rm \
		--env-file test/performance/http/single-flow.env.example \
		-v "${ROOT_DIR}/test/performance/http:/work:ro" \
		"${K6_IMAGE}" inspect --include-system-env-vars /work/single-flow.js >/dev/null
	echo "HTTP benchmark inspection passed"
}

validate_pgo_manifest() {
	local manifest="$1"
	awk '
		BEGIN {
			required["format"] = 1
			required["profile_sha256"] = 1
			required["profile_source_revision"] = 1
			required["profile_source_fingerprint"] = 1
			required["accepted_build_fingerprint"] = 1
			required["workload_id"] = 1
			required["response_owner"] = 1
			required["go_version"] = 1
			required["goos"] = 1
			required["goarch"] = 1
			required["capture_interval"] = 1
			required["captured_at_utc"] = 1
		}
		{
			key = $0
			sub(/=.*/, "", key)
			value = substr($0, length(key) + 2)
			if (!(key in required) || value == "" || ++seen[key] != 1) exit 2
		}
		END {
			for (key in required) if (seen[key] != 1) exit 2
		}
	' "${manifest}"
}

pgo_manifest() (
	local profile="${PGO_PROFILE:-}" manifest temporary
	local source_revision="${PGO_PROFILE_SOURCE_REVISION:-}"
	local workload="${PGO_WORKLOAD_ID:-}" owner="${PGO_RESPONSE_OWNER:-}"
	local go_version="${PGO_PROFILE_GO_VERSION:-}" interval="${PGO_CAPTURE_INTERVAL:-}"
	local goos="${PGO_PROFILE_GOOS:-}" goarch="${PGO_PROFILE_GOARCH:-}"
	local captured_at="${PGO_CAPTURED_AT_UTC:-}"

	require_value PGO_PROFILE "${profile}"
	require_value PGO_PROFILE_SOURCE_REVISION "${source_revision}"
	require_value PGO_WORKLOAD_ID "${workload}"
	require_value PGO_RESPONSE_OWNER "${owner}"
	require_value PGO_PROFILE_GO_VERSION "${go_version}"
	require_value PGO_PROFILE_GOOS "${goos}"
	require_value PGO_PROFILE_GOARCH "${goarch}"
	require_value PGO_CAPTURE_INTERVAL "${interval}"
	require_value PGO_CAPTURED_AT_UTC "${captured_at}"
	[[ -f "${profile}" ]] || die "PGO profile not found: ${profile}"
	git cat-file -e "${source_revision}^{commit}" 2>/dev/null || die "PGO source revision is unavailable"
	"${GO_BIN}" tool pprof -raw "${profile}" >/dev/null || die "invalid CPU profile: ${profile}"
	manifest="${PGO_MANIFEST:-${profile}.meta}"
	mkdir -p "$(dirname "${manifest}")"
	temporary="$(mktemp "$(dirname "${manifest}")/.pgo-manifest.XXXXXX")"
	trap 'rm -f -- "${temporary}"' EXIT
	{
		printf 'format=go-pgo-profile-v1\n'
		printf 'profile_sha256=%s\n' "$(sha256_file "${profile}")"
		printf 'profile_source_revision=%s\n' "${source_revision}"
		printf 'profile_source_fingerprint=%s\n' "$(revision_fingerprint "${source_revision}")"
		printf 'accepted_build_fingerprint=%s\n' "$(pgo_candidate_fingerprint)"
		printf 'workload_id=%s\n' "${workload}"
		printf 'response_owner=%s\n' "${owner}"
		printf 'go_version=%s\n' "${go_version}"
		printf 'goos=%s\n' "${goos}"
		printf 'goarch=%s\n' "${goarch}"
		printf 'capture_interval=%s\n' "${interval}"
		printf 'captured_at_utc=%s\n' "${captured_at}"
	} >"${temporary}"
	validate_pgo_manifest "${temporary}" || die "invalid PGO manifest: ${temporary}"
	mv "${temporary}" "${manifest}"
	echo "PGO manifest: ${manifest}"
)

verify_pgo() {
	local profile="${1:-}" manifest="${2:-}" candidate="${3:-}"
	local format digest source_revision source_fingerprint accepted_fingerprint
	local workload owner go_version goos goarch interval captured_at

	[[ -f "${profile}" ]] || die "PGO profile not found: ${profile}"
	[[ -f "${manifest}" ]] || die "PGO manifest not found: ${manifest}"
	validate_pgo_manifest "${manifest}" || die "invalid PGO manifest: ${manifest}"
	format="$(metadata_value "${manifest}" format)"
	digest="$(metadata_value "${manifest}" profile_sha256)"
	source_revision="$(metadata_value "${manifest}" profile_source_revision)"
	source_fingerprint="$(metadata_value "${manifest}" profile_source_fingerprint)"
	accepted_fingerprint="$(metadata_value "${manifest}" accepted_build_fingerprint)"
	workload="$(metadata_value "${manifest}" workload_id)"
	owner="$(metadata_value "${manifest}" response_owner)"
	go_version="$(metadata_value "${manifest}" go_version)"
	goos="$(metadata_value "${manifest}" goos)"
	goarch="$(metadata_value "${manifest}" goarch)"
	interval="$(metadata_value "${manifest}" capture_interval)"
	captured_at="$(metadata_value "${manifest}" captured_at_utc)"

	[[ "${format}" == go-pgo-profile-v1 ]] || die "unsupported PGO manifest format: ${format}"
	[[ "${digest}" == "$(sha256_file "${profile}")" ]] || die "PGO profile digest does not match manifest"
	[[ "${go_version}" == "$("${GO_BIN}" env GOVERSION)" ]] || die "PGO profile Go version does not match build toolchain"
	[[ "${goos}" == "${PGO_TARGET_GOOS:-$("${GO_BIN}" env GOOS)}" ]] || die "PGO profile GOOS does not match build target"
	[[ "${goarch}" == "${PGO_TARGET_GOARCH:-$("${GO_BIN}" env GOARCH)}" ]] || die "PGO profile GOARCH does not match build target"
	[[ "${source_revision}" =~ ^[0-9a-f]{40}$ ]] || die "invalid PGO source revision"
	[[ "${source_fingerprint}" =~ ^[0-9a-f]{64}$ ]] || die "invalid PGO source fingerprint"
	[[ "${accepted_fingerprint}" =~ ^[0-9a-f]{64}$ ]] || die "invalid accepted build fingerprint"
	[[ "${workload}" =~ ^[A-Za-z0-9._-]+$ ]] || die "invalid PGO workload identity"
	require_value response_owner "${owner}"
	[[ "${interval}" =~ ^[1-9][0-9]*(ms|s|m|h)$ ]] || die "invalid PGO capture interval"
	[[ "${captured_at}" =~ ^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z$ ]] || die "invalid PGO capture timestamp"
	"${GO_BIN}" tool pprof -raw "${profile}" >/dev/null || die "invalid CPU profile: ${profile}"

	if [[ -z "${candidate}" ]]; then
		git rev-parse --git-dir >/dev/null 2>&1 || die "candidate fingerprint is required outside a Git checkout"
		candidate="$(pgo_candidate_fingerprint)"
	fi
	[[ "${candidate}" == "${accepted_fingerprint}" ]] || die "PGO manifest was not accepted for this source fingerprint"
	if git rev-parse --git-dir >/dev/null 2>&1; then
		git cat-file -e "${source_revision}^{commit}" 2>/dev/null || die "PGO source revision is unavailable"
		[[ "${source_fingerprint}" == "$(revision_fingerprint "${source_revision}")" ]] ||
			die "PGO source fingerprint does not match its revision"
		git merge-base --is-ancestor "${source_revision}" HEAD || die "PGO source revision is not an ancestor of the candidate"
	fi
	echo "PGO profile verified: workload=${workload} source=${source_revision} owner=${owner}"
}

self_test() (
	local tmp baseline current manifest key
	tmp="$(mktemp -d "${TMPDIR:-/tmp}/benchmark-self-test.XXXXXX")"
	trap 'rm -rf -- "${tmp}"' EXIT
	baseline="${tmp}/baseline.meta"
	current="${tmp}/current.meta"
	while IFS= read -r key; do printf '%s=value\n' "${key}"; done < <(comparable_keys) >"${baseline}"
	cp "${baseline}" "${current}"
	compare_metadata "${baseline}" "${current}"
	awk 'BEGIN { changed = 0 } /^workload_id=/ { print "workload_id=other"; changed = 1; next } { print } END { if (!changed) exit 2 }' \
		"${baseline}" >"${current}"
	if compare_metadata "${baseline}" "${current}" >/dev/null 2>&1; then
		die "benchmark metadata mismatch was accepted"
	fi
	manifest="${tmp}/profile.meta"
	printf '%s\n' \
		'format=go-pgo-profile-v1' \
		'profile_sha256=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa' \
		'profile_source_revision=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa' \
		'profile_source_fingerprint=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa' \
		'accepted_build_fingerprint=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb' \
		'workload_id=self-test' 'response_owner=self-test' 'go_version=go1.27.0' \
		'goos=linux' 'goarch=amd64' \
		'capture_interval=1s' 'captured_at_utc=2026-01-01T00:00:00Z' >"${manifest}"
	validate_pgo_manifest "${manifest}" || die "valid PGO manifest was rejected"
	printf 'workload_id=duplicate\n' >>"${manifest}"
	if validate_pgo_manifest "${manifest}" >/dev/null 2>&1; then
		die "duplicate PGO manifest key was accepted"
	fi
	[[ "${K6_IMAGE_DEFAULT}" =~ ^grafana/k6:[0-9]+\.[0-9]+\.[0-9]+@sha256:[0-9a-f]{64}$ ]] ||
		die "default k6 image is not versioned and digest-pinned"
	echo "benchmark evidence self-test passed"
)

case "${1:-}" in
capture) capture ;;
compare) compare ;;
http) http_capture ;;
inspect-http) inspect_http ;;
pgo-manifest) pgo_manifest ;;
verify-pgo) verify_pgo "${2:-}" "${3:-}" "${4:-}" ;;
source-fingerprint) repository_fingerprint ;;
pgo-fingerprint) pgo_candidate_fingerprint ;;
self-test) self_test ;;
*) usage; exit 1 ;;
esac
