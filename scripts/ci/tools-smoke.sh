#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "${ROOT_DIR}"

registered_tool_paths() {
	awk '
		$1 == "tool" && $2 == "(" { tools = 1; next }
		tools && $1 == ")" { exit }
		tools && NF { print $1 }
	' tools/go.mod
}

tool_name() {
	local path=$1 name=${1##*/}
	if [[ ${name} =~ ^v[0-9]+$ ]]; then
		path=${path%/*}
		name=${path##*/}
	fi
	printf '%s\n' "${name}"
}

registered=$(registered_tool_paths)
[[ -n ${registered} ]] || {
	echo "tools/go.mod registers no tools" >&2
	exit 1
}

smoke_registered_tools() {
	while IFS= read -r owner; do
		tool=$(tool_name "${owner}")
		go tool -modfile=tools/go.mod -n "${tool}" >/dev/null
		echo "tool smoke passed: ${tool}"
	done <<<"${registered}"
}

if [[ ${TOOLS_SMOKE_ALL:-} == 1 ]]; then
	smoke_registered_tools
	exit
fi

base_ref=${BASE_REF:-origin/main}
git rev-parse --verify "${base_ref}^{commit}" >/dev/null 2>&1 || {
	echo "tools smoke base is unavailable: ${base_ref}; set BASE_REF to a readable commit" >&2
	exit 2
}
tmp=$(mktemp -d)
trap 'rm -rf -- "${tmp}"' EXIT
diff_file=${tmp}/tools.diff

if ! git diff --quiet "${base_ref}" HEAD -- scripts/ci/tools-smoke.sh ||
	! git diff --quiet -- scripts/ci/tools-smoke.sh ||
	! git diff --cached --quiet -- scripts/ci/tools-smoke.sh; then
	smoke_registered_tools
	exit
fi

{
	git diff "${base_ref}" HEAD -- tools/go.mod
	git diff -- tools/go.mod
	git diff --cached -- tools/go.mod
} >"${diff_file}"

candidates=$(
	awk '
		/^\+\+\+/ { next }
		/^\+/ {
			line=substr($0, 2)
			gsub(/^[[:space:]]+/, "", line)
			split(line, fields, /[[:space:]]+/)
			if (fields[1] ~ /[.][A-Za-z0-9-]+\//) print fields[1]
		}
	' "${diff_file}" | LC_ALL=C sort -u
)

if [[ -z ${candidates} ]]; then
	echo "not applicable: no tools/go.mod dependency line changed"
	exit
fi

tools=''
while IFS= read -r candidate; do
	owner=''
	if grep -Fqx "${candidate}" <<<"${registered}"; then
		owner=${candidate}
	elif why=$(go -C tools mod why -m "${candidate}" 2>/dev/null); then
		owner=$(grep -Fxf <(printf '%s\n' "${registered}") <<<"${why}" | head -n 1 || true)
	fi
	if [[ -z ${owner} ]]; then
		echo "tools smoke: no registered tool owner found for ${candidate}; tidy remains the proof" >&2
		continue
	fi
	tool=$(tool_name "${owner}")
	grep -Fqx "${tool}" <<<"${tools}" || tools="${tools}${tool}"$'\n'
done <<<"${candidates}"

if [[ -z ${tools} ]]; then
	echo "not applicable: changed dependencies have no registered tool owner"
	exit
fi

while IFS= read -r tool; do
	[[ -n ${tool} ]] || continue
	go tool -modfile=tools/go.mod -n "${tool}" >/dev/null
	echo "tool smoke passed: ${tool}"
done <<<"${tools}"
