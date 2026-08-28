#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "${ROOT_DIR}"

render_config() {
	local module module_regex line
	module=$(go list -m -f '{{.Path}}')
	module_regex=${module//./\\.}
	while IFS= read -r line || [[ -n ${line} ]]; do
		line=${line//__MODULE_REGEX__/${module_regex}}
		printf '%s\n' "${line//__MODULE__/${module}}"
	done <.golangci.yml
}

if [[ ${1:-} == --render-config ]]; then
	render_config
	exit
fi

expected=$(awk '$1 == "github.com/golangci/golangci-lint/v2" { sub(/^v/, "", $2); print $2; exit }' tools/go.mod)
[[ -n ${expected} ]] || {
	echo "golangci-lint version is absent from tools/go.mod" >&2
	exit 1
}

common_dir=$(git rev-parse --git-common-dir)
[[ ${common_dir} == /* ]] || common_dir=${ROOT_DIR}/${common_dir}
fingerprint=$(
	{
		uname -srm
		go version
		shasum -a 256 tools/go.mod tools/go.sum
	} | shasum -a 256 | awk '{print $1}'
)
cache_dir=${CODEX_TOOL_CACHE:-${common_dir}/codex/tools}/${fingerprint}
binary=${cache_dir}/golangci-lint
lock=${cache_dir}.lock

valid_binary() {
	[[ -x ${binary} ]] && "${binary}" version 2>/dev/null | grep -Eq "(^|[[:space:]])v?${expected}([[:space:]]|$)"
}

if ! valid_binary; then
	mkdir -p "$(dirname "${cache_dir}")"
	acquired=false
	while ! valid_binary; do
		if mkdir "${lock}" 2>/dev/null; then acquired=true; break; fi
		owner_pid=$(awk -F= '$1 == "pid" { print $2; exit }' "${lock}/owner" 2>/dev/null || true)
		if [[ -n ${owner_pid} ]] && ! kill -0 "${owner_pid}" 2>/dev/null; then
			rm -f "${lock}/owner"
			rmdir "${lock}" 2>/dev/null || true
			continue
		fi
		sleep 1
	done
	if [[ ${acquired} == true ]]; then
		trap 'rm -rf -- "${lock}"' EXIT INT TERM
		printf 'pid=%s\n' "$$" >"${lock}/owner"
		mkdir -p "${cache_dir}"
		tmp=${binary}.tmp.$$
		go build -modfile=tools/go.mod -o "${tmp}" github.com/golangci/golangci-lint/v2/cmd/golangci-lint
		mv "${tmp}" "${binary}"
		valid_binary || {
			echo "cached golangci-lint does not report ${expected}" >&2
			exit 1
		}
		rm -rf -- "${lock}"
		trap - EXIT INT TERM
	fi
fi

case ${1:-} in
run | fmt)
	command=$1
	shift
	config_dir=$(mktemp -d -t golangci-config)
	config=${config_dir}/.golangci.yml
	trap 'rm -rf -- "${config_dir}"' EXIT INT TERM
	render_config >"${config}"
	"${binary}" "${command}" --config "${config}" "$@"
	;;
*)
	"${binary}" "$@"
	;;
esac
