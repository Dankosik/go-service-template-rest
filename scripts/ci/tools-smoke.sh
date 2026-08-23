#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "${ROOT_DIR}"

if [[ ${TOOLS_SMOKE_ALL:-} == 1 ]]; then
	for tool in buf validate golangci-lint oapi-codegen oasdiff goose gosec sqlc gitleaks nilaway benchstat deadcode goimports govulncheck protoc-gen-go-grpc protoc-gen-go gotestsum gofumpt; do
		go tool -modfile=tools/go.mod -n "${tool}" >/dev/null
		echo "tool smoke passed: ${tool}"
	done
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

{
	git diff "${base_ref}...HEAD" -- tools/go.mod
	git diff -- tools/go.mod
	git diff --cached -- tools/go.mod
} >"${diff_file}"

modules=$(
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

if [[ -z ${modules} ]]; then
	echo "not applicable: no tools/go.mod dependency line changed"
	exit
fi

tools=''
while IFS= read -r module; do
	owner=$(go -C tools mod why -m "${module}" | awk '!/^#/ && NF { print; exit }')
	case "${owner}" in
		github.com/bufbuild/buf/*) tool=buf ;;
		github.com/getkin/kin-openapi/cmd/validate) tool=validate ;;
		github.com/golangci/golangci-lint/*) tool=golangci-lint ;;
		github.com/oapi-codegen/oapi-codegen/*) tool=oapi-codegen ;;
		github.com/oasdiff/oasdiff*) tool=oasdiff ;;
		github.com/pressly/goose/*) tool=goose ;;
		github.com/securego/gosec/*) tool=gosec ;;
		github.com/sqlc-dev/sqlc/*) tool=sqlc ;;
		github.com/zricethezav/gitleaks/*) tool=gitleaks ;;
		go.uber.org/nilaway/*) tool=nilaway ;;
		golang.org/x/perf/cmd/benchstat) tool=benchstat ;;
		golang.org/x/tools/cmd/deadcode) tool=deadcode ;;
		golang.org/x/tools/cmd/goimports) tool=goimports ;;
		golang.org/x/vuln/cmd/govulncheck) tool=govulncheck ;;
		google.golang.org/grpc/cmd/protoc-gen-go-grpc) tool=protoc-gen-go-grpc ;;
		google.golang.org/protobuf/cmd/protoc-gen-go) tool=protoc-gen-go ;;
		gotest.tools/gotestsum*) tool=gotestsum ;;
		mvdan.cc/gofumpt*) tool=gofumpt ;;
		*)
			echo "tools smoke: no registered tool owner found for ${module}; tidy remains the proof" >&2
			continue
			;;
	esac
	grep -Fqx "${tool}" <<<"${tools}" || tools="${tools}${tool}"$'\n'
done <<<"${modules}"

if [[ -z ${tools} ]]; then
	echo "not applicable: changed dependencies have no registered tool owner"
	exit
fi

while IFS= read -r tool; do
	[[ -n ${tool} ]] || continue
	go tool -modfile=tools/go.mod -n "${tool}" >/dev/null
	echo "tool smoke passed: ${tool}"
done <<<"${tools}"
