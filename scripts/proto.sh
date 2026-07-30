#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

usage() {
	echo "usage: $0 format|format-check|lint|generate|drift|breaking|check" >&2
}

has_proto() {
	find "$1" -type f -name '*.proto' -print -quit 2>/dev/null | grep -q .
}

module_roots() {
	if has_proto "${ROOT_DIR}/api/proto"; then
		printf '%s\n' "${ROOT_DIR}"
	fi
	if has_proto "${ROOT_DIR}/examples/grpc-reference-service/api/proto"; then
		printf '%s\n' "${ROOT_DIR}/examples/grpc-reference-service"
	fi
}

run_buf() {
	local module_root="$1"
	shift
	(
		cd "${module_root}"
		if [[ "${module_root}" == "${ROOT_DIR}" ]]; then
			bash ./scripts/run-buf.sh "$@"
		else
			bash ../../scripts/run-buf.sh "$@"
		fi
	)
}

check_schema_policy() {
	local module_root
	local proto
	local relative_proto
	local base_ref="${BASE_REF:-}"
	local base_commit=""
	local base_proto
	while IFS= read -r module_root; do
		while IFS= read -r -d '' proto; do
			relative_proto="${proto#"${ROOT_DIR}"/}"
			if grep -Eq '^[[:space:]]*edition[[:space:]]*=[[:space:]]*"2023";[[:space:]]*$' "${proto}"; then
				if ! grep -Eq '^[[:space:]]*option[[:space:]]+features\.\(pb\.go\)\.api_level[[:space:]]*=[[:space:]]*API_OPAQUE;[[:space:]]*$' "${proto}"; then
					echo "protobuf schema policy: ${relative_proto} must select schema-owned API_OPAQUE" >&2
					exit 1
				fi
				continue
			fi
			if grep -Eq '^[[:space:]]*edition[[:space:]]*=[[:space:]]*"2024";[[:space:]]*$' "${proto}"; then
				echo "protobuf schema policy: ${relative_proto} uses Edition 2024, which requires reopening and accepting the cross-language contract policy" >&2
				exit 1
			fi
			if grep -Eq '^[[:space:]]*syntax[[:space:]]*=[[:space:]]*"proto(2|3)";[[:space:]]*$' "${proto}"; then
				if [[ -z "${base_ref}" ]]; then
					echo "protobuf schema policy: ${relative_proto} uses legacy proto2/proto3 syntax; BASE_REF must identify a readable comparison base that retains the same path" >&2
					exit 1
				fi
				if [[ -z "${base_commit}" ]]; then
					if ! base_commit="$(git -C "${ROOT_DIR}" rev-parse --verify "${base_ref}^{commit}" 2>/dev/null)"; then
						echo "protobuf schema policy: invalid or unreadable legacy comparison base: ${base_ref}" >&2
						exit 1
					fi
				fi
				if ! base_proto="$(git -C "${ROOT_DIR}" show "${base_commit}:${relative_proto}" 2>/dev/null)"; then
					echo "protobuf schema policy: ${relative_proto} is not retained at ${base_ref}; new Go contracts must use Edition 2023 with schema-owned API_OPAQUE" >&2
					exit 1
				fi
				if ! grep -Eq '^[[:space:]]*syntax[[:space:]]*=[[:space:]]*"proto(2|3)";[[:space:]]*$' <<<"${base_proto}"; then
					echo "protobuf schema policy: ${relative_proto} was not a proto2/proto3 contract at ${base_ref}" >&2
					exit 1
				fi
				continue
			fi
			echo "protobuf schema policy: ${relative_proto} must declare Edition 2023 with schema-owned API_OPAQUE or be a baseline-retained proto2/proto3 contract" >&2
			exit 1
		done < <(find "${module_root}/api/proto" -type f -name '*.proto' -print0)
	done < <(module_roots)
}

format_proto() {
	local found=false
	local module_root
	while IFS= read -r module_root; do
		found=true
		run_buf "${module_root}" format -w
	done < <(module_roots)
	if [[ "${found}" == false ]]; then
		echo "no protobuf sources; skipping format"
	fi
}

format_check_proto() {
	local found=false
	local module_root
	while IFS= read -r module_root; do
		found=true
		run_buf "${module_root}" format --diff --exit-code
	done < <(module_roots)
	if [[ "${found}" == false ]]; then
		echo "no protobuf sources; format is current"
	fi
}

lint_proto() {
	local found=false
	local module_root
	check_schema_policy
	while IFS= read -r module_root; do
		found=true
		run_buf "${module_root}" lint
	done < <(module_roots)
	if [[ "${found}" == false ]]; then
		echo "no protobuf sources; lint is current"
	fi
}

generate_proto() {
	local found=false
	local module_root
	while IFS= read -r module_root; do
		found=true
		run_buf "${module_root}" generate
	done < <(module_roots)
	if [[ "${found}" == false ]]; then
		echo "no protobuf sources; skipping generation"
	fi
}

generated_output_for() {
	local module_root="$1"
	printf '%s\n' "${module_root}/internal/gen/proto"
}

check_stale_output_without_source() {
	local source_dir="$1"
	local output_dir="$2"
	if has_proto "${source_dir}"; then
		return
	fi
	if find "${output_dir}" -type f -name '*.go' -print -quit 2>/dev/null | grep -q .; then
		echo "stale protobuf generated output exists without source: ${output_dir#"${ROOT_DIR}"/}" >&2
		exit 1
	fi
}

drift_proto() (
	local snapshot
	local module_root
	local output_dir
	local snapshot_dir
	local relative_output
	local drift=0
	local found=false

	check_stale_output_without_source "${ROOT_DIR}/api/proto" "${ROOT_DIR}/internal/gen/proto"

	snapshot="$(mktemp -d)"
	trap 'rm -rf "${snapshot}"' EXIT

	while IFS= read -r module_root; do
		found=true
		output_dir="$(generated_output_for "${module_root}")"
		relative_output="${output_dir#"${ROOT_DIR}"/}"
		snapshot_dir="${snapshot}/${relative_output}"
		mkdir -p "${snapshot_dir}"
		if [[ -d "${output_dir}" ]]; then
			cp -R "${output_dir}/." "${snapshot_dir}/"
		else
			touch "${snapshot_dir}/.absent"
		fi
	done < <(module_roots)

	if [[ "${found}" == false ]]; then
		echo "protobuf generated output is current (no sources)"
		exit 0
	fi

	generate_proto

	while IFS= read -r module_root; do
		output_dir="$(generated_output_for "${module_root}")"
		relative_output="${output_dir#"${ROOT_DIR}"/}"
		snapshot_dir="${snapshot}/${relative_output}"
		if [[ -f "${snapshot_dir}/.absent" ]]; then
			if find "${output_dir}" -type f -name '*.go' -print -quit 2>/dev/null | grep -q .; then
				drift=1
				echo "protobuf codegen drift detected: generation created ${relative_output}"
			fi
			continue
		fi
		if diff -qr "${snapshot_dir}" "${output_dir}" >/dev/null; then
			continue
		fi
		drift=1
		echo "protobuf codegen drift detected: generation changed ${relative_output}"
		diff -ruN "${snapshot_dir}" "${output_dir}" || true
	done < <(module_roots)

	if [[ "${drift}" -ne 0 ]]; then
		echo "generated protobuf output was updated; review and commit it" >&2
		exit 1
	fi
	echo "protobuf generated output is current"
)

breaking_proto() {
	local base_ref="${BASE_REF:-}"
	local against

	if [[ -z "${base_ref}" ]]; then
		echo "BASE_REF is required" >&2
		exit 2
	fi
	if ! git -C "${ROOT_DIR}" rev-parse --verify "${base_ref}^{commit}" >/dev/null 2>&1; then
		echo "invalid or unreadable protobuf base ref: ${base_ref}" >&2
		exit 1
	fi
	if ! has_proto "${ROOT_DIR}/api/proto"; then
		echo "no owned protobuf contract; breaking check not applicable"
		return
	fi
	if ! git -C "${ROOT_DIR}" ls-tree -r --name-only "${base_ref}" -- api/proto |
		grep -E '\.proto$' >/dev/null; then
		echo "no base protobuf contract at ${base_ref}:api/proto; breaking check not applicable"
		return
	fi

	against=".git#ref=${base_ref},subdir=api/proto"
	(
		cd "${ROOT_DIR}"
		bash ./scripts/run-buf.sh breaking api/proto --against "${against}"
	)
}

case "${1:-}" in
format)
	format_proto
	;;
format-check)
	format_check_proto
	;;
lint)
	lint_proto
	;;
generate)
	generate_proto
	;;
drift)
	drift_proto
	;;
breaking)
	breaking_proto
	;;
check)
	format_check_proto
	lint_proto
	drift_proto
	;;
*)
	usage
	exit 2
	;;
esac
