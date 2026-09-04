#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "${ROOT_DIR}"

mode=run
locked=false
provided_files=()
case "${1:-}" in
	--plan) mode=plan; shift ;;
	--self-test) mode=self-test; shift ;;
	--locked) locked=true; shift ;;
esac
if [[ ${1:-} == --files ]]; then
	shift
	provided_files=("$@")
fi

fingerprint_candidate() {
	local file mode record hash head
	local -a hash_files=() records=()
	head=$(git rev-parse HEAD)
	while IFS= read -r file; do
		if [[ -L ${file} ]]; then
			hash=$(readlink "${file}" | git hash-object --stdin)
			records+=("mode 120000  ${file}"$'\n'"symlink ${hash}")
		elif [[ -f ${file} ]]; then
			if [[ -x ${file} ]]; then mode=100755; else mode=100644; fi
			records+=("mode ${mode}  ${file}")
			hash_files+=("${file}")
		else
			records+=("deleted  ${file}")
		fi
	done <"${files_path}"
	: >"${tmp}/file-hashes"
	if ((${#hash_files[@]})); then
		shasum -a 256 -- "${hash_files[@]}" >"${tmp}/file-hashes" || return
	fi
	{
		printf 'head=%s\n' "${head}"
		printf 'base_ref=%s\nresolved_base_sha=%s\nmerge_base_sha=%s\n' "${base_ref}" "${resolved_base_sha}" "${merge_base_sha}"
		for record in "${records[@]}"; do
			printf '%s\n' "${record}"
			case "${record}" in
			'mode 100644 '* | 'mode 100755 '*)
				IFS= read -r hash || return 1
				printf '%s\n' "${hash}"
				;;
			esac
		done
	} <"${tmp}/file-hashes" | shasum -a 256 | awk '{print $1}'
}

self_test() {
	local output
	(
		tmp=$(mktemp -d)
		trap 'rm -rf -- "${tmp}"' EXIT
		cd "${tmp}"
		git init -q
		git -c user.name=verify-test -c user.email=verify-test@example.invalid -c commit.gpgsign=false commit -qm fixture --allow-empty
		printf 'plain\n' >plain
		printf 'executable\n' >executable
		chmod +x executable
		ln -s plain link
		files_path=${tmp}/files
		printf '%s\n' plain executable link deleted >"${files_path}"
		base_ref=HEAD
		resolved_base_sha=$(git rev-parse HEAD)
		merge_base_sha=${resolved_base_sha}
		original=$(fingerprint_candidate)
		chmod +x plain
		[[ $(fingerprint_candidate) != "${original}" ]]
		chmod -x plain
		printf 'changed\n' >plain
		[[ $(fingerprint_candidate) != "${original}" ]]
		printf 'plain\n' >plain
		rm link
		ln -s executable link
		[[ $(fingerprint_candidate) != "${original}" ]]
		rm link
		ln -s plain link
		printf 'new\n' >deleted
		[[ $(fingerprint_candidate) != "${original}" ]]
		rm deleted
		[[ $(fingerprint_candidate) == "${original}" ]]
	)
	bash "${ROOT_DIR}/scripts/ci/git-changed-paths.sh" --self-test

	if CI='' ALLOW_FULL='' make test-all >"${TMPDIR:-/tmp}/verify-full-guard.$$" 2>&1; then
		echo "verify self-test accepted test-all without ALLOW_FULL=1" >&2
		return 1
	fi
	grep -q 'ALLOW_FULL=1' "${TMPDIR:-/tmp}/verify-full-guard.$$"
	rm -f "${TMPDIR:-/tmp}/verify-full-guard.$$"

	output=$(bash "$0" --plan --files README.md)
	grep -q 'documentation=true' <<<"${output}"
	grep -q 'make docs-contract-check' <<<"${output}"
	grep -q 'documentation: no repository-wide documentation validator' <<<"${output}"

	local scratch
	scratch=$(mktemp -d)
	output=$(TMPDIR="${scratch}" VERIFY_FORCE=1 bash "$0" --files README.md)
	# Both the unlocked planner and its locked child own temporary files.
	[[ -z $(find "${scratch}" -mindepth 1 -print -quit) ]] || {
		echo "verification leaked temporary files" >&2
		rm -rf -- "${scratch}"
		return 1
	}
	rmdir "${scratch}"
	grep -q 'docs contract check passed' <<<"${output}"
	grep -q 'documentation: no repository-wide documentation validator' <<<"${output}"
	if grep -q 'reusing exact passing receipt' <<<"${output}"; then return 1; fi

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

	output=$(bash "$0" --plan --files test/performance/http/single-flow.js)
	grep -q 'performance_harness=true' <<<"${output}"
	grep -q 'make performance-harness-check' <<<"${output}"
	if grep -q 'benchmark-http' <<<"${output}"; then return 1; fi

	output=$(bash "$0" --plan --files internal/failure/failure.go internal/problem/problem.go)
	test "$(grep -c "make fmt-files-check" <<<"${output}")" -eq 1
	test "$(grep -c "make lint-changed" <<<"${output}")" -eq 1
	grep -q 'make test-package PKG=' <<<"${output}"
	grep -q './internal/failure' <<<"${output}"
	grep -q './internal/problem' <<<"${output}"
	if grep -q 'make test-all' <<<"${output}"; then return 1; fi

	output=$(bash "$0" --plan --files internal/failure/failure_test.go)
	grep -q "make test-package PKG='./internal/failure'" <<<"${output}"
	if grep -q './internal/problem' <<<"${output}"; then return 1; fi
	if grep -q 'make test-all' <<<"${output}"; then return 1; fi

	output=$(bash "$0" --plan --files internal/failure/removed.go)
	grep -q "make lint-changed PKGS='./internal/failure'" <<<"${output}"
	grep -q 'make test-package PKG=' <<<"${output}"
	if grep -q 'removed.go' <<<"${output}" && grep -q 'fmt-files-check' <<<"${output}"; then
		if grep -q "FILES='internal/failure/removed.go'" <<<"${output}"; then return 1; fi
	fi
	if grep -q 'make test-all' <<<"${output}"; then return 1; fi

	output=$(bash "$0" --plan --files Makefile)
	grep -q 'make verify-check' <<<"${output}"
	grep -q 'make changed-surfaces-check' <<<"${output}"
	if grep -q 'proof-receipt' <<<"${output}"; then return 1; fi
	if grep -q 'make test-all' <<<"${output}"; then return 1; fi
	if grep -q 'test-integration' <<<"${output}"; then return 1; fi

	output=$(bash "$0" --plan --files make/template.mk)
	grep -q 'make verify-check' <<<"${output}"
	grep -q 'make changed-surfaces-check' <<<"${output}"

	output=$(bash "$0" --plan --files go.mod)
	grep -q 'make test-all' <<<"${output}"
	grep -q 'make root-mod-check' <<<"${output}"

	if CI='' ALLOW_HEAVY='' bash "$0" --files test/postgres_integration_test.go >/dev/null 2>"${TMPDIR:-/tmp}/verify-heavy.$$"; then
		echo "verify self-test accepted a heavy route without ALLOW_HEAVY=1" >&2
		return 1
	fi
	grep -q 'set ALLOW_HEAVY=1' "${TMPDIR:-/tmp}/verify-heavy.$$"
	rm -f "${TMPDIR:-/tmp}/verify-heavy.$$"

	if ALLOW_HEAVY=1 VERIFY_DOCKER_COMMAND=missing-docker-for-verify-test bash "$0" --files test/postgres_integration_test.go >/dev/null 2>"${TMPDIR:-/tmp}/verify-docker.$$"; then
		echo "verify self-test accepted an integration route without Docker" >&2
		return 1
	fi
	grep -q 'Docker is required' "${TMPDIR:-/tmp}/verify-docker.$$"
	rm -f "${TMPDIR:-/tmp}/verify-docker.$$"

	output=$(bash "$0" --plan --files api/openapi/service.yaml)
	grep -q 'make openapi-runtime-contract-check' <<<"${output}"
	if grep -q 'requires_network=true' <<<"${output}"; then return 1; fi
}

if [[ ${mode} == self-test ]]; then
	self_test
	exit
fi

tmp=$(mktemp -d)
trap 'rm -rf -- "${tmp}"' EXIT
files_path=${tmp}/files
base_ref=${BASE_REF:-origin/main}

if ((${#provided_files[@]})); then
	printf '%s\n' "${provided_files[@]}" >"${files_path}"
	if git rev-parse --verify "${base_ref}^{commit}" >/dev/null 2>&1; then
		resolved_base_sha=$(git rev-parse "${base_ref}^{commit}")
		merge_base_sha=$(git merge-base HEAD "${resolved_base_sha}")
	else
		resolved_base_sha=$(git rev-parse HEAD)
		merge_base_sha=${resolved_base_sha}
	fi
else
	git rev-parse --verify "${base_ref}^{commit}" >/dev/null 2>&1 || {
		echo "verification base is unavailable: ${base_ref}; set BASE_REF to a readable commit" >&2
		exit 2
	}
	resolved_base_sha=$(git rev-parse "${base_ref}^{commit}")
	merge_base_sha=$(git merge-base HEAD "${resolved_base_sha}")
	bash ./scripts/ci/git-changed-paths.sh --worktree "${base_ref}" >"${files_path}"
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
cost_classes=()
heavy_requirements=()
docker_requirements=()
network_requirements=()
keys=''
not_applicable=()

add_command() {
	local kind=$1 argument=$2 reason=$3 display=$4 cost_class=$5 requires_heavy=$6 requires_docker=$7 requires_network=$8 key
	key="${kind}|${argument}"
	case $'\n'"${keys}" in *$'\n'"${key}"$'\n'*) return ;; esac
	keys="${keys}${key}"$'\n'
	kinds[${#kinds[@]}]=${kind}
	arguments[${#arguments[@]}]=${argument}
	reasons[${#reasons[@]}]=${reason}
	displays[${#displays[@]}]=${display}
	cost_classes[${#cost_classes[@]}]=${cost_class}
	heavy_requirements[${#heavy_requirements[@]}]=${requires_heavy}
	docker_requirements[${#docker_requirements[@]}]=${requires_docker}
	network_requirements[${#network_requirements[@]}]=${requires_network}
}

add_na() {
	not_applicable[${#not_applicable[@]}]="$1: $2"
}

root_dependencies=false
if is_true go_root_dependencies; then
	root_dependencies=true
	add_command make root-mod-check "root Go dependencies changed" "make root-mod-check" cpu false false false
	add_command make test-all "root dependency changes can affect every package" "make test-all" cpu false false false
fi
if is_true go_tool_dependencies; then
	add_command make tools-dependencies-check "tool dependencies changed" "make tools-dependencies-check" cpu false false false
fi

affected_format=''
affected_lint=''
affected_tests=''
affected_fallback=false
affected_reason=''
if grep -qE '\.go$|/testdata/' "${files_path}" || is_true go_root_dependencies || is_true go_testdata; then
	affected_path=${tmp}/affected
	bash ./scripts/ci/affected-go-packages.sh <"${files_path}" >"${affected_path}"
	while IFS='=' read -r name value; do
		case "${name}" in
		format_files) affected_format=${value} ;;
		lint_packages) affected_lint=${value} ;;
		affected_test_packages) affected_tests=${value} ;;
		fallback) affected_fallback=${value} ;;
		fallback_reason) affected_reason=${value} ;;
		esac
	done <"${affected_path}"
	if [[ -n ${affected_format} ]]; then
		add_command fmt "${affected_format}" "handwritten Go files changed" "make fmt-files-check FILES='${affected_format}'" cpu false false false
	fi
	if [[ -n ${affected_lint} ]] && ! is_true go_lint_config; then
		add_command lint "${affected_lint}" "Go package owners changed" "make lint-changed PKGS='${affected_lint}'" cpu false false false
	fi
	if [[ ${root_dependencies} != true ]]; then
		if [[ ${affected_fallback} == true || ${affected_tests} == ./... ]]; then
			reason="Go changes require the module test oracle"
			[[ -n ${affected_reason} ]] && reason="${reason} (${affected_reason})"
			add_command make test-all "${reason}" "make test-all" cpu false false false
		elif [[ -n ${affected_tests} ]]; then
			add_command test "${affected_tests}" "affected Go packages changed" "make test-package PKG='${affected_tests}'" cpu false false false
		fi
	fi
fi

tests_cover_openapi_runtime() {
	if [[ ${root_dependencies} == true || ${affected_fallback} == true || ${affected_tests} == ./... ]]; then
		return 0
	fi
	case " ${affected_tests} " in
	*" ./internal/openapi "*)
		case " ${affected_tests} " in
		*" ./internal/infra/http "*) return 0 ;;
		esac
		;;
	esac
	return 1
}

if is_true go_lint_config; then add_command make lint-all "the complete lint configuration changed" "make lint-all" cpu false false false; fi
if is_true openapi; then
	add_command make openapi-drift-check "OpenAPI source or generated output changed" "make openapi-drift-check" cpu false false false
	add_command make openapi-lint "OpenAPI source or generated output changed" "make openapi-lint" cpu false false false
	add_command make openapi-validate "OpenAPI source or generated output changed" "make openapi-validate" cpu false false false
	if tests_cover_openapi_runtime; then
		add_na openapi "Go tests already cover the OpenAPI runtime packages"
	else
		add_command make openapi-runtime-contract-check "OpenAPI runtime packages are not in the selected Go tests" "make openapi-runtime-contract-check" cpu false false false
	fi
fi
if is_true protobuf; then
	if find api/proto -type f -name '*.proto' -print -quit 2>/dev/null | grep -q . || [[ -f examples/grpc-reference-service/buf.yaml ]]; then
		add_command make check-proto "Protobuf source or generated output changed" "make check-proto" cpu false false false
	else add_na protobuf "no protobuf sources remain"; fi
fi
if is_true sqlc; then
	if find internal/infra/postgres/queries internal/infra/postgres/sqlcgen -type f -print -quit 2>/dev/null | grep -q .; then
		add_command make check-sqlc "SQLC source or generated output changed" "make check-sqlc" cpu false false false
	else add_na sqlc "no query sources or generated output remain"; fi
fi
if is_true module_initializer; then add_command make template-init-check "module initializer contract changed" "make template-init-check" cpu true false false; fi
if is_true integration_initializer; then add_command integration-init row_e1_http "integration initializer changed; use one representative row locally" "make integration-init-check INTEGRATION_INIT_ROWS=row_e1_http" cpu false false false; fi
if is_true agent_instructions; then add_command make check-instructions "agent instructions or their carriers changed" "make check-instructions" cpu false false false; fi
if is_true validation_system; then
	add_command make verify-check "validation routing changed" "make verify-check" cpu false false false
	add_command make changed-surfaces-check "validation routing changed" "make changed-surfaces-check" cpu false false false
	add_command make validation-lock-self-test "validation routing changed" "make validation-lock-self-test" cpu false false false
	add_command make affected-go-packages-check "validation routing changed" "make affected-go-packages-check" cpu false false false
fi
if is_true shell; then
	shell_files=$(awk '/\.sh$/ { print }' "${files_path}" | while IFS= read -r file; do [[ -f ${file} ]] && printf '%s ' "${file}"; done)
	shell_files=${shell_files% }
	if [[ -n ${shell_files} ]]; then add_command shell "${shell_files}" "shell sources changed" "make shellcheck SHELL_FILES='${shell_files}'" cpu false false false; else add_na shell "no changed shell source remains"; fi
fi
if is_true github_workflows; then add_command make actionlint "GitHub workflow or action source changed" "make actionlint" cpu false false false; fi
if is_true dependency_automation; then add_na dependency_automation "GitHub owns Dependabot schema validation; CI dependency review remains applicable"; fi
if is_true performance_harness; then add_command make performance-harness-check "performance harness changed; inspect without load" "make performance-harness-check" docker false true false; fi
if is_true docs_contract; then add_command make docs-contract-check "high-risk runtime or documentation contract changed" "make docs-contract-check" cheap false false false; fi
if is_true documentation; then add_na documentation "no repository-wide documentation validator is configured"; fi
if is_true compose_environment; then add_command make compose-environment-check "compose environment changed" "make compose-environment-check" docker false true false; fi
if is_true publication_metadata; then add_command make publish-image-metadata-check "publication metadata changed" "make publish-image-metadata-check" cheap false false false; fi
if is_true secret_scanning; then add_command make secret-scan "secret scanning policy changed" "make secret-scan" cpu false false false; fi

if is_true db_integration; then add_command integration test-integration-db "database integration owners changed" "REQUIRE_DOCKER=1 make test-integration-db" docker true true false; fi
if is_true messaging_integration; then add_command integration test-integration-messaging "messaging integration owners changed" "REQUIRE_DOCKER=1 make test-integration-messaging" docker true true false; fi
if is_true process_integration; then add_command integration test-integration-process "process integration owners changed" "REQUIRE_DOCKER=1 make test-integration-process" docker true true false; fi
if is_true integration_race_messaging; then add_command integration test-messaging-race "messaging race owners changed" "REQUIRE_DOCKER=1 make test-messaging-race" docker true true false; fi
if is_true integration_race_outbox; then add_command integration test-outbox-race "outbox race owners changed" "REQUIRE_DOCKER=1 make test-outbox-race" docker true true false; fi
if is_true integration_race_webhook; then add_command integration test-webhook-race "webhook race owners changed" "REQUIRE_DOCKER=1 make test-webhook-race" docker true true false; fi
if is_true integration_race && ! is_true integration_race_messaging && ! is_true integration_race_outbox && ! is_true integration_race_webhook; then
	add_command integration test-integration-race "concurrency-sensitive integration owners changed" "REQUIRE_DOCKER=1 make test-integration-race" docker true true false
fi

needs_image=false
if is_true runtime_image || is_true image_security || is_true migrations; then needs_image=true; fi
if is_true runtime_image; then add_command make dockerfile-check "runtime image inputs changed" "make dockerfile-check" docker false true false; fi
if is_true migrations; then add_command make migration-check "migration history or runner changed" "make migration-check" cpu false false false; fi
if [[ ${needs_image} == true ]]; then add_command image-build "${VERIFY_RUNTIME_IMAGE:-service:verify}" "one image is shared by selected runtime gates" "make runtime-image-build RUNTIME_IMAGE=${VERIFY_RUNTIME_IMAGE:-service:verify}" docker true true false; fi
if is_true runtime_image && ! is_true migrations; then add_command image-check "${VERIFY_RUNTIME_IMAGE:-service:verify}" "runtime image lifecycle changed" "make runtime-image-check RUNTIME_IMAGE=${VERIFY_RUNTIME_IMAGE:-service:verify}" docker true true false; fi
if is_true migrations; then add_command migration-validate "${VERIFY_RUNTIME_IMAGE:-service:verify}" "migrations require exact-image rehearsal" "make migration-validate RUNTIME_IMAGE=${VERIFY_RUNTIME_IMAGE:-service:verify}" docker true true false; fi
if is_true image_security; then add_command image-security "${VERIFY_RUNTIME_IMAGE:-service:verify}" "runtime image inputs changed" "make container-security CONTAINER_IMAGE=${VERIFY_RUNTIME_IMAGE:-service:verify}" docker true true false; fi

print_plan() {
	echo "files:"
	sed 's/^/  /' "${files_path}"
	echo "surfaces:"
	sed 's/^/  /' "${surfaces_path}"
	echo "commands:"
	if ((${#kinds[@]} == 0)); then echo "  none"; else
		for i in "${!kinds[@]}"; do
			printf '  %s\n    because %s\n    cost_class=%s requires_heavy=%s requires_docker=%s requires_network=%s\n' \
				"${displays[$i]}" "${reasons[$i]}" "${cost_classes[$i]}" "${heavy_requirements[$i]}" "${docker_requirements[$i]}" "${network_requirements[$i]}"
		done
	fi
	if ((${#not_applicable[@]})); then echo "not applicable:"; printf '  %s\n' "${not_applicable[@]}"; fi
}

if [[ ${mode} == plan ]]; then
	print_plan
	exit
fi

if ((${#kinds[@]} == 0)); then
	print_plan
	echo "verification not applicable: no executable checks for changed surfaces"
	exit 0
fi

requires_heavy=false
requires_docker=false
requires_network=false
for i in "${!kinds[@]}"; do
	[[ ${heavy_requirements[$i]} == true ]] && requires_heavy=true
	[[ ${docker_requirements[$i]} == true ]] && requires_docker=true
	[[ ${network_requirements[$i]} == true ]] && requires_network=true
done

blocked() {
	printf 'claim: surface-aware verification\nresult: blocked\nstatus: blocked\ngap_or_next_owner: %s\n' "$1" >&2
	exit 2
}

if [[ ${requires_heavy} == true && ${ALLOW_HEAVY:-} != 1 && ${CI:-} != true ]]; then blocked "set ALLOW_HEAVY=1 before verification"; fi
for binary in git make shasum; do command -v "${binary}" >/dev/null 2>&1 || blocked "required binary is unavailable: ${binary}"; done
if grep -q '\.go$' "${files_path}" || is_true go_root_dependencies || is_true go_tool_dependencies || is_true go_lint_config || is_true validation_system; then
	command -v go >/dev/null 2>&1 || blocked "required binary is unavailable: go"
fi
if is_true openapi; then command -v npx >/dev/null 2>&1 || blocked "required binary is unavailable: npx"; fi
docker_command=${VERIFY_DOCKER_COMMAND:-docker}
if [[ ${requires_docker} == true ]]; then
	command -v "${docker_command}" >/dev/null 2>&1 || blocked "Docker is required"
	"${docker_command}" info >/dev/null 2>&1 || blocked "Docker is required and the daemon is unavailable"
fi


candidate=$(fingerprint_candidate)
command_summary=none
if ((${#displays[@]})); then command_summary=$(IFS='; '; echo "${displays[*]}"); fi
plan_input=${tmp}/plan
for i in "${!kinds[@]}"; do printf '%s|%s|%s|%s|%s\n' "${displays[$i]}" "${cost_classes[$i]}" "${heavy_requirements[$i]}" "${docker_requirements[$i]}" "${network_requirements[$i]}"; done >"${plan_input}"
plan=$(shasum -a 256 "${plan_input}" | awk '{print $1}')
build_tags=none
if is_true db_integration || is_true messaging_integration || is_true process_integration || is_true integration_race; then build_tags=integration; fi
if is_true integration_race; then build_tags=integration,race; fi
tool_versions=$(awk '$1 ~ /golangci-lint\/v2$|gosec\/v2$|x\/vuln$/ { printf "%s@%s,", $1, $2 }' tools/go.mod)
go_environment=$(go version 2>/dev/null || echo unavailable)
docker_environment=not-used
if [[ ${requires_docker} == true ]]; then docker_environment=$("${docker_command}" version --format '{{.Client.Version}}/{{.Server.Version}}' 2>/dev/null || echo unavailable); fi
environment_detail="$(uname -srm); ${go_environment}; docker=${docker_environment}; GOFLAGS=${GOFLAGS:-}; GOTOOLCHAIN=${GOTOOLCHAIN:-}; LINT_BASE_REF=${LINT_BASE_REF:-}; REQUIRE_DOCKER=$([[ ${requires_docker} == true ]] && echo 1 || echo 0); ALLOW_HEAVY=${ALLOW_HEAVY:-}; GOMAXPROCS=${GOMAXPROCS:-}; VALIDATION_JOBS=${VALIDATION_JOBS:-2}; VALIDATION_PARALLEL_TESTS=${VALIDATION_PARALLEL_TESTS:-2}; build_tags=${build_tags}; tools=${tool_versions}"
environment=$(printf '%s\n' "${environment_detail}" | shasum -a 256 | awk '{print $1}')
common_dir=$(git rev-parse --git-common-dir)
[[ ${common_dir} == /* ]] || common_dir=${ROOT_DIR}/${common_dir}
receipt_dir=${common_dir}/codex/verify
receipt=${receipt_dir}/${candidate}-${plan}-${environment}.receipt

if [[ -f ${receipt} && ${VERIFY_FORCE:-} != 1 ]]; then
	echo "reusing exact passing receipt: ${receipt}"
	cat "${receipt}"
	exit
fi

if [[ ${locked} != true ]]; then
	args=(bash "$0" --locked)
	if ((${#provided_files[@]})); then args+=(--files "${provided_files[@]}"); fi
	rm -rf -- "${tmp}"
	exec bash ./scripts/ci/validation-lock.sh -- "${args[@]}"
fi

print_plan

execute() {
	local kind=$1 argument=$2
	case "${kind}" in
		make)
			case "${argument}" in
			test-all | lint-all | check-go)
				ALLOW_FULL=1 make "${argument}"
				;;
			*)
				make "${argument}"
				;;
			esac
			;;
		fmt) make fmt-files-check FILES="${argument}" ;;
		lint) make lint-changed PKGS="${argument}" ;;
		test) make test-package PKG="${argument}" ;;
		shell) make shellcheck SHELL_FILES="${argument}" ;;
		integration-init) make integration-init-check INTEGRATION_INIT_ROWS="${argument}" ;;
		integration) REQUIRE_DOCKER=1 make "${argument}" ;;
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
candidate_after=$(fingerprint_candidate)
if [[ ${candidate_after} != "${candidate}" ]]; then
	printf 'claim: surface-aware verification\nresult: invalidated\ncandidate: %s\nstatus: not_verified\ngap_or_next_owner: candidate changed during verification\n' "${candidate}" >&2
	exit 1
fi

mkdir -p "${receipt_dir}"
receipt_tmp=${receipt}.tmp.$$
{
	printf 'claim: surface-aware verification\n'
	printf 'result: pass\n'
	printf 'candidate: %s\n' "${candidate}"
	printf 'base_ref: %s\nresolved_base_sha: %s\nmerge_base_sha: %s\n' "${base_ref}" "${resolved_base_sha}" "${merge_base_sha}"
	printf 'scope: %s\n' "$(tr '\n' ',' <"${files_path}")"
	printf 'command: make verify [%s]\n' "${command_summary}"
	printf 'inputs: %s\n' "$(tr '\n' ',' <"${files_path}")"
	printf 'environment: %s\n' "${environment_detail}"
	printf 'duration: %ss\n' "${duration}"
	printf 'status: verified\n'
	if ((${#not_applicable[@]})); then printf 'not_applicable: %s\n' "$(IFS='; '; echo "${not_applicable[*]}")"; fi
	printf 'gap_or_next_owner: none\n'
} >"${receipt_tmp}"
mv "${receipt_tmp}" "${receipt}"
cat "${receipt}"
