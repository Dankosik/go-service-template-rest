#!/usr/bin/env bash
set -euo pipefail

tool=${1:-}
root=$(git rev-parse --show-toplevel)
common_dir=$(git rev-parse --git-common-dir)
[[ ${common_dir} == /* ]] || common_dir=${root}/${common_dir}

case "$(uname -s):$(uname -m):${tool}" in
	Darwin:arm64:actionlint) version=1.7.12; platform=darwin_arm64; checksum=aba9ced2dee8d27fecca3dc7feb1a7f9a52caefa1eb46f3271ea66b6e0e6953f ;;
	Darwin:x86_64:actionlint) version=1.7.12; platform=darwin_amd64; checksum=5b44c3bc2255115c9b69e30efc0fecdf498fdb63c5d58e17084fd5f16324c644 ;;
	Linux:aarch64:actionlint|Linux:arm64:actionlint) version=1.7.12; platform=linux_arm64; checksum=325e971b6ba9bfa504672e29be93c24981eeb1c07576d730e9f7c8805afff0c6 ;;
	Linux:x86_64:actionlint|Linux:amd64:actionlint) version=1.7.12; platform=linux_amd64; checksum=8aca8db96f1b94770f1b0d72b6dddcb1ebb8123cb3712530b08cc387b349a3d8 ;;
	Darwin:arm64:shellcheck) version=0.11.0; platform=darwin.aarch64; checksum=339b930feb1ea764467013cc1f72d09cd6b869ebf1013296ba9055ab2ffbd26f ;;
	Darwin:x86_64:shellcheck) version=0.11.0; platform=darwin.x86_64; checksum=c2c15e08df0e8fbc374c335b230a7ee958c313fa5714817a59aa59f1aa594f51 ;;
	Linux:aarch64:shellcheck|Linux:arm64:shellcheck) version=0.11.0; platform=linux.aarch64; checksum=68a8133197a50beb8803f8d42f9908d1af1c5540d4bb05fdfca8c1fa47decefc ;;
	Linux:x86_64:shellcheck|Linux:amd64:shellcheck) version=0.11.0; platform=linux.x86_64; checksum=b7af85e41cc99489dcc21d66c6d5f3685138f06d34651e6d34b42ec6d54fe6f6 ;;
	*) echo "unsupported validation tool platform: ${tool} $(uname -s)/$(uname -m)" >&2; exit 2 ;;
esac

cache_dir=${CODEX_TOOL_CACHE:-${common_dir}/codex/tools}/native/${tool}-${version}-${platform}
binary=${cache_dir}/${tool}
lock=${cache_dir}.lock

valid() {
	case "${tool}" in
		actionlint) [[ -x ${binary} ]] && [[ $("${binary}" -version 2>/dev/null | sed -n '1p') == "${version}" ]] ;;
		shellcheck) [[ -x ${binary} ]] && [[ $("${binary}" --version 2>/dev/null | awk '$1 == "version:" { print $2; exit }') == "${version}" ]] ;;
	esac
}

if ! valid; then
	command -v curl >/dev/null 2>&1 || { echo "curl is required to install ${tool}" >&2; exit 2; }
	mkdir -p "$(dirname "${cache_dir}")"
	acquired=false
	while ! valid; do
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
		tmp=$(mktemp -d)
		case "${tool}" in
			actionlint)
				archive=actionlint_${version}_${platform}.tar.gz
				url=https://github.com/rhysd/actionlint/releases/download/v${version}/${archive}
				;;
			shellcheck)
				archive=shellcheck-v${version}.${platform}.tar.gz
				url=https://github.com/koalaman/shellcheck/releases/download/v${version}/${archive}
				;;
		esac
		curl -fsSL "${url}" -o "${tmp}/${archive}"
		printf '%s  %s\n' "${checksum}" "${tmp}/${archive}" | shasum -a 256 -c - >/dev/null
		mkdir -p "${cache_dir}"
		if [[ ${tool} == actionlint ]]; then
			tar -xzf "${tmp}/${archive}" -C "${cache_dir}" actionlint
		else
			tar -xzf "${tmp}/${archive}" --strip-components=1 -C "${cache_dir}" "shellcheck-v${version}/shellcheck"
		fi
		rm -rf -- "${tmp}" "${lock}"
		trap - EXIT INT TERM
		valid || { echo "installed ${tool} does not report ${version}" >&2; exit 1; }
	fi
fi

printf '%s\n' "${binary}"
