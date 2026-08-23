#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "${ROOT_DIR}"

mode=run
provided_files=()
case "${1:-}" in
	--plan)
		mode=plan
		shift
		;;
	--self-test)
		mode=self-test
		shift
		;;
esac
if [[ ${1:-} == --files ]]; then
	shift
	provided_files=("$@")
fi

self_test() {
	local output
	output=$(bash "$0" --plan --files README.md)
	grep -q 'documentation=true' <<<"${output}"
	grep -q 'documentation: no repository-wide documentation validator' <<<"${output}"
	if grep -q '^  make ' <<<"${output}"; then return 1; fi

	output=$(bash "$0" --plan --files tools/go.mod)
	grep -q 'make tools-dependencies-check' <<<"${output}"
	if grep -q 'test-integration' <<<"${output}"; then return 1; fi

	output=$(bash "$0" --plan --files build/docker/Dockerfile)
	grep -q 'make dockerfile-check' <<<"${output}"
	grep -q 'make runtime-image-build' <<<"${output}"
	grep -q 'make runtime-image-check' <<<"${output}"
	grep -q 'make container-security' <<<"${output}"
	if grep -q 'test-integration-db' <<<"${output}"; then return 1; fi

	output=$(bash "$0" --plan --files scripts/integration-init.sh)
	grep -q 'INTEGRATION_INIT_ROWS=row_e1_http' <<<"${output}"
	if grep -q 'template-init-check' <<<"${output}"; then return 1; fi

	output=$(bash "$0" --plan --files internal/failure/failure.go internal/failure/removed.go)
	grep -q 'make test-package PKG=./internal/failure' <<<"${output}"
}

if [[ ${mode} == self-test ]]; then
	self_test
	exit
fi

tmp=$(mktemp -d)
trap 'rm -rf -- "${tmp}"' EXIT
files_path=${tmp}/files

if ((${#provided_files[@]})); then
	printf '%s\n' "${provided_files[@]}" >"${files_path}"
else
	base_ref=${BASE_REF:-origin/main}
	git rev-parse --verify "${base_ref}^{commit}" >/dev/null 2>&1 || {
		echo "verification base is unavailable: ${base_ref}; set BASE_REF to a readable commit" >&2
		exit 2
	}
	{
		git diff --name-only "${base_ref}...HEAD"
		git diff --name-only
		git diff --cached --name-only
		git ls-files --others --exclude-standard
	} >"${files_path}"
fi
LC_ALL=C sort -u -o "${files_path}" "${files_path}"

if [[ ! -s ${files_path} ]]; then
	echo "no changed files; verification is not applicable"
	exit
fi
if grep -n '[[:space:]]' "${files_path}" >/dev/null; then
	echo "verification does not support changed paths containing whitespace" >&2
	exit 2
fi

surfaces_path=${tmp}/surfaces
bash ./scripts/ci/changed-surfaces.sh <"${files_path}" >"${surfaces_path}"
while IFS='=' read -r name value; do
	printf -v "surface_${name}" '%s' "${value}"
done <"${surfaces_path}"

is_true() {
	local variable=surface_$1
	[[ ${!variable:-false} == true ]]
}

kinds=()
arguments=()
reasons=()
displays=()
keys=''
not_applicable=()

add_command() {
	local kind=$1 argument=$2 reason=$3 display=$4 key
	key="${kind}|${argument}"
	grep -Fqx "${key}" <<<"${keys}" && return
	keys="${keys}${key}"$'\n'
	kinds[${#kinds[@]}]=${kind}
	arguments[${#arguments[@]}]=${argument}
	reasons[${#reasons[@]}]=${reason}
	displays[${#displays[@]}]=${display}
}

add_na() {
	not_applicable[${#not_applicable[@]}]="$1: $2"
}

root_dependencies=false
if is_true go_root_dependencies; then
	root_dependencies=true
	add_command make root-mod-check "root Go dependencies changed" "make root-mod-check"
	add_command make test-all "root dependency changes can affect every package" "make test-all"
fi
if is_true go_tool_dependencies; then
	add_command make tools-dependencies-check "tool dependencies changed" "make tools-dependencies-check"
fi

if is_true go_source; then
	go_dirs=${tmp}/go-dirs
	awk '/\.go$/ { sub("/[^/]+$", ""); if ($0 == "") print "."; else print }' "${files_path}" | LC_ALL=C sort -u >"${go_dirs}"
	while IFS= read -r dir; do
		[[ -n ${dir} ]] || continue
		package_files=$(awk -v dir="${dir}" '
			/\.go$/ {
				path=$0
				d=path
				sub("/[^/]+$", "", d)
				if (d == path) d="."
				if (d == dir) print path
			}
		' "${files_path}" | while IFS= read -r file; do
			if [[ -f ${file} ]]; then printf '%s ' "${file}"; fi
		done)
		package_files=${package_files% }
		pkg=./${dir}
		[[ ${dir} == . ]] && pkg=.
		if [[ -z ${package_files} ]]; then
			add_na go_source "${pkg} has no remaining changed Go file to format or lint"
			continue
		fi
		add_command fmt "${package_files}" "Go files changed in ${pkg}" "make fmt-files-check FILES='${package_files}'"
		if ! is_true go_lint_config; then
			add_command lint "${pkg}" "Go files changed in ${pkg}" "make lint-changed PKG=${pkg}"
		fi
		if [[ ${root_dependencies} != true ]]; then
			add_command test "${pkg}" "Go behavior changed in ${pkg}" "make test-package PKG=${pkg}"
		fi
	done <"${go_dirs}"
fi

if is_true go_lint_config; then
	add_command make lint-all "the complete lint configuration changed" "make lint-all"
fi
if is_true openapi; then
	add_command make check-openapi "OpenAPI source or generated output changed" "make check-openapi"
fi
if is_true protobuf; then
	if find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q . || [[ -f examples/grpc-reference-service/buf.yaml ]]; then
		add_command make check-proto "Protobuf source or generated output changed" "make check-proto"
	else
		add_na protobuf "no protobuf sources remain"
	fi
fi
if is_true sqlc; then
	if find internal/infra/postgres/queries internal/infra/postgres/sqlcgen -type f -print -quit 2>/dev/null | grep -q .; then
		add_command make check-sqlc "SQLC source or generated output changed" "make check-sqlc"
	else
		add_na sqlc "no query sources or generated output remain"
	fi
fi
if is_true module_initializer; then
	add_command make template-init-check "module initializer contract changed" "make template-init-check"
fi
if is_true integration_initializer; then
	add_command integration-init row_e1_http "integration initializer changed; use one representative row locally" "make integration-init-check INTEGRATION_INIT_ROWS=row_e1_http"
fi
if is_true agent_instructions; then
	add_command make check-instructions "agent instructions or their carriers changed" "make check-instructions"
fi
if is_true shell; then
	shell_files=$(awk '/\.sh$/ { print }' "${files_path}" | while IFS= read -r file; do
		if [[ -f ${file} ]]; then printf '%s ' "${file}"; fi
	done)
	shell_files=${shell_files% }
	if [[ -n ${shell_files} ]]; then
		add_command shell "${shell_files}" "shell sources changed" "make shellcheck SHELL_FILES='${shell_files}'"
	else
		add_na shell "no changed shell source remains"
	fi
fi
if is_true github_workflows; then
	add_command make actionlint "GitHub workflow or action source changed" "make actionlint"
fi
if is_true dependency_automation; then
	add_na dependency_automation "GitHub owns Dependabot schema validation; CI dependency review remains applicable"
fi
if is_true documentation; then
	add_na documentation "no repository-wide documentation validator is configured"
fi

if is_true db_integration; then
	add_command make test-integration-db "database integration owners changed" "make test-integration-db"
fi
if is_true messaging_integration; then
	add_command make test-integration-messaging "messaging integration owners changed" "make test-integration-messaging"
fi
if is_true process_integration; then
	add_command make test-integration-process "process integration owners changed" "make test-integration-process"
fi
if is_true integration_race; then
	add_command make test-integration-race "concurrency-sensitive integration owners changed" "make test-integration-race"
fi

needs_image=false
if is_true runtime_image || is_true image_security || is_true migrations; then
	needs_image=true
fi
if is_true runtime_image; then
	add_command make dockerfile-check "runtime image inputs changed" "make dockerfile-check"
fi
if is_true migrations; then
	add_command make migration-check "migration history or runner changed" "make migration-check"
fi
if [[ ${needs_image} == true ]]; then
	add_command image-build "${VERIFY_RUNTIME_IMAGE:-service:verify}" "one image is shared by selected runtime gates" "make runtime-image-build RUNTIME_IMAGE=${VERIFY_RUNTIME_IMAGE:-service:verify}"
fi
if is_true runtime_image && ! is_true migrations; then
	add_command image-check "${VERIFY_RUNTIME_IMAGE:-service:verify}" "runtime image lifecycle changed" "make runtime-image-check RUNTIME_IMAGE=${VERIFY_RUNTIME_IMAGE:-service:verify}"
fi
if is_true migrations; then
	add_command migration-validate "${VERIFY_RUNTIME_IMAGE:-service:verify}" "migrations require exact-image rehearsal" "make migration-validate RUNTIME_IMAGE=${VERIFY_RUNTIME_IMAGE:-service:verify}"
fi
if is_true image_security; then
	add_command image-security "${VERIFY_RUNTIME_IMAGE:-service:verify}" "runtime image inputs changed" "make container-security CONTAINER_IMAGE=${VERIFY_RUNTIME_IMAGE:-service:verify}"
fi

echo "files:"
sed 's/^/  /' "${files_path}"
echo "surfaces:"
sed 's/^/  /' "${surfaces_path}"
echo "commands:"
if ((${#kinds[@]} == 0)); then
	echo "  none"
else
	for i in "${!kinds[@]}"; do
		printf '  %s\n    because %s\n' "${displays[$i]}" "${reasons[$i]}"
	done
fi
if ((${#not_applicable[@]})); then
	echo "not applicable:"
	printf '  %s\n' "${not_applicable[@]}"
fi

[[ ${mode} == plan ]] && exit

candidate_input=${tmp}/candidate
git rev-parse HEAD >"${candidate_input}"
while IFS= read -r file; do
	if [[ -f ${file} || -L ${file} ]]; then
		shasum -a 256 "${file}"
	else
		printf 'deleted  %s\n' "${file}"
	fi
done <"${files_path}" >>"${candidate_input}"
candidate=$(shasum -a 256 "${candidate_input}" | awk '{print $1}')
command_summary=none
if ((${#displays[@]})); then
	command_summary=$(IFS='; '; echo "${displays[*]}")
fi
plan=$(printf '%s\n' "${command_summary}" | shasum -a 256 | awk '{print $1}')
go_environment=not-used
if is_true go_source || is_true go_root_dependencies || is_true go_tool_dependencies || is_true go_lint_config || \
	is_true openapi || is_true protobuf || is_true sqlc || is_true module_initializer || \
	is_true integration_initializer || is_true db_integration || is_true messaging_integration || \
	is_true process_integration || is_true integration_race || is_true migrations || is_true runtime_image; then
	go_environment=$(go version 2>/dev/null || echo unavailable)
fi
docker_environment=not-used
if is_true shell || is_true github_workflows || is_true db_integration || is_true messaging_integration || \
	is_true process_integration || is_true integration_race || is_true migrations || is_true runtime_image || is_true image_security; then
	docker_environment=$(docker version --format '{{.Client.Version}}/{{.Server.Version}}' 2>/dev/null || echo unavailable)
fi
environment_detail="$(uname -srm); ${go_environment}; docker=${docker_environment}"
environment=$(printf '%s\n' "${environment_detail}" | shasum -a 256 | awk '{print $1}')
receipt_dir=$(git rev-parse --git-path codex/verify)
receipt=${receipt_dir}/${candidate}-${plan}-${environment}.receipt

if [[ -f ${receipt} && ${VERIFY_FORCE:-} != 1 ]]; then
	echo "reusing exact passing receipt: ${receipt}"
	cat "${receipt}"
	exit
fi

execute() {
	local kind=$1 argument=$2
	case "${kind}" in
		make) make "${argument}" ;;
		fmt) make fmt-files-check FILES="${argument}" ;;
		lint) make lint-changed PKG="${argument}" ;;
		test) make test-package PKG="${argument}" ;;
		shell) make shellcheck SHELL_FILES="${argument}" ;;
		integration-init) make integration-init-check INTEGRATION_INIT_ROWS="${argument}" ;;
		image-build) APP_VERSION="verify-${candidate:0:12}" VCS_REF="$(git rev-parse HEAD)" make runtime-image-build RUNTIME_IMAGE="${argument}" ;;
		image-check) make runtime-image-check RUNTIME_IMAGE="${argument}" RUNTIME_EXPECTED_VERSION="verify-${candidate:0:12}" ;;
		migration-validate) make migration-validate RUNTIME_IMAGE="${argument}" RUNTIME_EXPECTED_VERSION="verify-${candidate:0:12}" ;;
		image-security) make container-security CONTAINER_IMAGE="${argument}" ;;
		*) echo "unknown verification command kind: ${kind}" >&2; return 2 ;;
	esac
}

started=$(date +%s)
if ((${#kinds[@]})); then
	for i in "${!kinds[@]}"; do
		command_started=$(date +%s)
		echo "==> ${displays[$i]}"
		if ! execute "${kinds[$i]}" "${arguments[$i]}"; then
			duration=$(($(date +%s) - started))
			printf 'claim: surface-aware verification\nresult: fail\ncandidate: %s\nscope: %s\ncommand: %s\ninputs: %s\nenvironment: %s\nduration: %ss\nstatus: not_verified\ngap_or_next_owner: failed command\n' \
				"${candidate}" "$(tr '\n' ',' <"${files_path}")" "${displays[$i]}" "$(tr '\n' ',' <"${files_path}")" "${environment_detail}" "${duration}" >&2
			exit 1
		fi
		printf '<== %s (%ss)\n' "${displays[$i]}" "$(($(date +%s) - command_started))"
	done
fi
duration=$(($(date +%s) - started))
mkdir -p "${receipt_dir}"
{
	printf 'claim: surface-aware verification\n'
	printf 'result: pass\n'
	printf 'candidate: %s\n' "${candidate}"
	printf 'scope: %s\n' "$(tr '\n' ',' <"${files_path}")"
	printf 'command: make verify [%s]\n' "${command_summary}"
	printf 'inputs: %s\n' "$(tr '\n' ',' <"${files_path}")"
	printf 'environment: %s\n' "${environment_detail}"
	printf 'duration: %ss\n' "${duration}"
	printf 'status: verified\n'
	if ((${#not_applicable[@]})); then
		printf 'not_applicable: %s\n' "$(IFS='; '; echo "${not_applicable[*]}")"
	fi
	printf 'gap_or_next_owner: none\n'
} >"${receipt}"
cat "${receipt}"
