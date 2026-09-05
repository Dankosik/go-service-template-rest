#!/usr/bin/env bash
# Plan the smallest Go test/lint/format scope for a changed-file list.
#
# Reads paths on stdin. Prints:
#   format_files=...
#   changed_packages=...
#   lint_packages=...
#   affected_test_packages=...
#   fallback=true|false
#   fallback_reason=...
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
cd "${ROOT_DIR}"

is_generated_go() {
	case "$1" in
	internal/openapi/*.gen.go | examples/reference-service/internal/openapi/*.gen.go | internal/infra/*/internal/openapi/*.gen.go | internal/infra/postgres/sqlcgen/*.go | internal/gen/proto/*.go | examples/grpc-reference-service/internal/gen/proto/*.go)
		return 0
		;;
	*)
		return 1
		;;
	esac
}

package_dir_of() {
	local file=$1 dir
	case "${file}" in
	*/testdata/*)
		dir=${file%%/testdata/*}
		;;
	*)
		dir=${file}
		if [[ ${dir} == */* ]]; then
			dir=${dir%/*}
		else
			dir=.
		fi
		;;
	esac
	printf '%s\n' "${dir}"
}

is_test_path() {
	local file=$1
	case "${file}" in
	*_test.go | */testdata/*) return 0 ;;
	*) return 1 ;;
	esac
}

join_sorted() {
	if [[ ! -s $1 ]]; then
		return
	fi
	LC_ALL=C sort -u "$1" | tr '\n' ' ' | sed 's/[[:space:]]*$//'
}

emit() {
	printf 'format_files=%s\n' "${format_files}"
	printf 'changed_packages=%s\n' "${changed_packages}"
	printf 'lint_packages=%s\n' "${lint_packages}"
	printf 'affected_test_packages=%s\n' "${affected_test_packages}"
	printf 'fallback=%s\n' "${fallback}"
	printf 'fallback_reason=%s\n' "${fallback_reason}"
}

self_test() (
	local output fixture script
	fixture=$(mktemp -d)
	trap 'rm -rf -- "${fixture}"' EXIT
	mkdir -p "${fixture}/scripts/ci" "${fixture}/internal/failure" \
		"${fixture}/internal/problem" "${fixture}/internal/testconsumer" "${fixture}/internal/openapi"
	cp "${ROOT_DIR}/scripts/ci/affected-go-packages.sh" "${fixture}/scripts/ci/"
	printf 'module example.invalid/affected\n\n' >"${fixture}/go.mod"
	awk '/^go / { print; exit }' "${ROOT_DIR}/go.mod" >>"${fixture}/go.mod"
	printf 'package failure\n' >"${fixture}/internal/failure/failure.go"
	printf 'package failure\n' >"${fixture}/internal/failure/failure_test.go"
	printf 'package problem\n\nimport _ "example.invalid/affected/internal/failure"\n' >"${fixture}/internal/problem/problem.go"
	printf 'package testconsumer\n' >"${fixture}/internal/testconsumer/consumer.go"
	printf 'package testconsumer\n\nimport _ "example.invalid/affected/internal/failure"\n' >"${fixture}/internal/testconsumer/consumer_test.go"
	printf 'package openapi\n' >"${fixture}/internal/openapi/openapi.gen.go"
	script=${fixture}/scripts/ci/affected-go-packages.sh

	output=$(printf '%s\n' README.md | bash "${script}")
	grep -qx 'format_files=' <<<"${output}"
	grep -qx 'changed_packages=' <<<"${output}"
	grep -qx 'affected_test_packages=' <<<"${output}"
	grep -qx 'fallback=false' <<<"${output}"

	output=$(printf '%s\n' go.mod | bash "${script}")
	grep -qx 'fallback=true' <<<"${output}"
	grep -qx 'fallback_reason=root_module' <<<"${output}"
	grep -qx 'affected_test_packages=./...' <<<"${output}"

	output=$(printf '%s\n' internal/failure/failure_test.go | bash "${script}")
	grep -qx 'fallback=false' <<<"${output}"
	grep -q 'changed_packages=./internal/failure' <<<"${output}"
	grep -qx 'affected_test_packages=./internal/failure' <<<"${output}"
	if grep -qE './internal/(problem|testconsumer)' <<<"${output}"; then
		echo "test-only change expanded to reverse importers" >&2
		return 1
	fi

	output=$(printf '%s\n' internal/failure/failure.go | bash "${script}")
	grep -qx 'fallback=false' <<<"${output}"
	grep -q './internal/failure' <<<"${output}"
	grep -q './internal/problem' <<<"${output}"
	grep -q './internal/testconsumer' <<<"${output}"
	if grep -qx 'affected_test_packages=./...' <<<"${output}"; then
		echo "leaf production change fell back to the full module" >&2
		return 1
	fi
	if grep -qx 'affected_test_packages=./internal/failure' <<<"${output}"; then
		echo "production change did not include reverse importers" >&2
		return 1
	fi

	output=$(printf '%s\n' internal/openapi/openapi.gen.go | bash "${script}")
	grep -qx 'format_files=' <<<"${output}"
	grep -qx 'lint_packages=' <<<"${output}"
	grep -q 'changed_packages=./internal/openapi' <<<"${output}"

	# Both sides of a production move must remain seeds.
	output=$(printf '%s\n' internal/failure/failure.go internal/problem/problem.go | bash "${script}")
	grep -q './internal/failure' <<<"${output}"
	grep -q './internal/problem' <<<"${output}"

	output=$(printf '%s\n' internal/packagetest/packagetest.go | bash "${script}")
	grep -qx 'affected_test_packages=' <<<"${output}"

	output=$(printf '%s\n' test/inbound_webhook_process_integration_test.go | bash "${script}")
	grep -qx 'lint_packages=' <<<"${output}"
	grep -qx 'affected_test_packages=' <<<"${output}"

	printf 'package mismatch\n' >"${fixture}/internal/problem/mismatch.go"
	output=$(printf '%s\n' internal/failure/failure.go | bash "${script}")
	grep -qx 'fallback=true' <<<"${output}"
	grep -qx 'fallback_reason=go_list_error' <<<"${output}"
	grep -qx 'affected_test_packages=./...' <<<"${output}"
)

if [[ ${1:-} == --self-test ]]; then
	self_test
	exit
fi

tmp=$(mktemp -d)
trap 'rm -rf -- "${tmp}"' EXIT
files_path=${tmp}/files
cat >"${files_path}"
LC_ALL=C sort -u -o "${files_path}" "${files_path}"

format_files=''
changed_packages=''
lint_packages=''
affected_test_packages=''
fallback=false
fallback_reason=

if [[ ! -s ${files_path} ]]; then
	emit
	exit
fi

go_files=${tmp}/go-files
: >"${go_files}"
awk '/\.go$/ || /\/testdata\//' "${files_path}" >"${go_files}" || true

if grep -qx 'go.mod' "${files_path}" || grep -qx 'go.sum' "${files_path}"; then
	fallback=true
	fallback_reason=root_module
fi

format_list=${tmp}/format
changed_list=${tmp}/changed
lint_list=${tmp}/lint
test_only_list=${tmp}/test-only
production_list=${tmp}/production
: >"${format_list}"
: >"${changed_list}"
: >"${lint_list}"
: >"${test_only_list}"
: >"${production_list}"

declare_package() {
	local file=$1 dir pkg
	dir=$(package_dir_of "${file}")
	if [[ ${dir} == . ]]; then
		pkg=.
	else
		pkg=./${dir}
	fi
	printf '%s\n' "${pkg}" >>"${changed_list}"
	if is_test_path "${file}"; then
		printf '%s\n' "${pkg}" >>"${test_only_list}"
	else
		printf '%s\n' "${pkg}" >>"${production_list}"
	fi
	if [[ -f ${file} ]] && [[ ${file} == *.go ]] && ! is_generated_go "${file}"; then
		printf '%s\n' "${file}" >>"${format_list}"
	fi
	if ! is_generated_go "${file}"; then
		printf '%s\n' "${pkg}" >>"${lint_list}"
	fi
}

while IFS= read -r file; do
	[[ -n ${file} ]] || continue
	case "${file}" in
	*.go | */testdata/*) declare_package "${file}" ;;
	esac
done <"${go_files}"

format_files=$(join_sorted "${format_list}")
changed_packages=$(join_sorted "${changed_list}")
lint_packages=''

if [[ ${fallback} == true ]]; then
	affected_test_packages=./...
	emit
	exit
fi

if [[ ! -s ${changed_list} ]]; then
	emit
	exit
fi

if ! command -v go >/dev/null 2>&1; then
	fallback=true
	fallback_reason=go_unavailable
	affected_test_packages=./...
	emit
	exit
fi

if [[ ! -s ${production_list} ]]; then
	test_packages=()
	while IFS= read -r pkg; do
		[[ -n ${pkg} ]] && test_packages+=("${pkg}")
	done < <(LC_ALL=C sort -u "${test_only_list}")
	test_list=${tmp}/test-list
	if ! go list -e -find -f '{{if not .Error}}OK{{"\t"}}{{.Dir}}{{else if or .GoFiles .CgoFiles .TestGoFiles .XTestGoFiles .InvalidGoFiles}}ERR{{"\t"}}{{.Error.Err}}{{end}}' \
		"${test_packages[@]}" >"${test_list}"; then
		fallback=true
		fallback_reason=go_list_error
		affected_test_packages=./...
		emit
		exit
	fi
	if grep -q $'^ERR\t' "${test_list}"; then
		fallback=true
		fallback_reason=go_list_error
		affected_test_packages=./...
		emit
		exit
	fi
	valid_test_list=${tmp}/valid-test
	lint_keep=${tmp}/lint-keep
	: >"${valid_test_list}"
	: >"${lint_keep}"
	while IFS=$'\t' read -r state dir; do
		[[ ${state} == OK ]] || continue
		if [[ ${dir} == "${ROOT_DIR}" ]]; then
			pkg=.
		elif [[ ${dir} == "${ROOT_DIR}"/* ]]; then
			pkg=./${dir#"${ROOT_DIR}"/}
		else
			continue
		fi
		printf '%s\n' "${pkg}" >>"${valid_test_list}"
		if grep -Fxq "${pkg}" "${lint_list}"; then
			printf '%s\n' "${pkg}" >>"${lint_keep}"
		fi
	done <"${test_list}"
	lint_packages=$(join_sorted "${lint_keep}")
	affected_test_packages=$(join_sorted "${valid_test_list}")
	emit
	exit
fi

module=$(go list -m) || {
	fallback=true
	fallback_reason=go_list_error
	affected_test_packages=./...
	emit
	exit
}

graph=${tmp}/graph
dirs=${tmp}/dirs
list_err=${tmp}/list.err
: >"${graph}"
: >"${dirs}"
if ! go list -e -test -f '{{if not .ForTest}}{{.ImportPath}}	{{.Dir}}	{{if .Error}}ERR{{else}}OK{{end}}	{{join .Imports " "}}	{{join .TestImports " "}}	{{join .XTestImports " "}}{{end}}' ./... >"${tmp}/list" 2>"${list_err}"; then
	fallback=true
	fallback_reason=go_list_error
	affected_test_packages=./...
	emit
	exit
fi
if grep -q $'\tERR\t' "${tmp}/list"; then
	fallback=true
	fallback_reason=go_list_error
	affected_test_packages=./...
	emit
	exit
fi

while IFS=$'\t' read -r import_path dir _ imports test_imports xtest_imports; do
	[[ -n ${import_path} ]] || continue
	printf '%s\t%s\n' "${import_path}" "${dir}" >>"${dirs}"
	local_imports=''
	# shellcheck disable=SC2086
	for imp in ${imports} ${test_imports} ${xtest_imports}; do
		case "${imp}" in
		"${module}" | "${module}"/*) local_imports+=" ${imp}" ;;
		esac
	done
	printf '%s%s\n' "${import_path}" "${local_imports}" >>"${graph}"
done <"${tmp}/list"

pkg_to_import() {
	local pkg=$1
	if [[ ${pkg} == . ]]; then
		printf '%s\n' "${module}"
	else
		printf '%s/%s\n' "${module}" "${pkg#./}"
	fi
}

import_to_pkg() {
	local import_path=$1 dir
	dir=$(awk -F '\t' -v p="${import_path}" '$1 == p { print $2; exit }' "${dirs}")
	[[ -n ${dir} ]] || return 0
	if [[ ${dir} == "${ROOT_DIR}" ]]; then
		printf '.\n'
		return
	fi
	if [[ ${dir} == "${ROOT_DIR}"/* ]]; then
		printf './%s\n' "${dir#"${ROOT_DIR}"/}"
	fi
}

lint_keep=${tmp}/lint-keep
: >"${lint_keep}"
if [[ -s ${lint_list} ]]; then
	while IFS= read -r pkg; do
		[[ -n ${pkg} ]] || continue
		import_path=$(pkg_to_import "${pkg}")
		if grep -Fq "${import_path}"$'\t' "${dirs}"; then
			printf '%s\n' "${pkg}" >>"${lint_keep}"
		fi
	done < <(LC_ALL=C sort -u "${lint_list}")
fi
lint_packages=$(join_sorted "${lint_keep}")

seeds=${tmp}/seeds
affected_imports=${tmp}/affected-imports
: >"${seeds}"
if [[ -s ${production_list} ]]; then
	while IFS= read -r pkg; do
		[[ -n ${pkg} ]] || continue
		# Skip packages that also appear only as test-only when production_list has them.
		pkg_to_import "${pkg}" >>"${seeds}"
	done < <(LC_ALL=C sort -u "${production_list}")
fi

if [[ -s ${seeds} ]]; then
	awk '
		FNR == NR {
			importer = $1
			for (i = 2; i <= NF; i++) {
				if ($i == "") continue
				reverse[$i] = reverse[$i] SUBSEP importer
			}
			next
		}
		{
			if (!seen[$1]++) {
				queue[n++] = $1
			}
		}
		END {
			for (i = 0; i < n; i++) {
				p = queue[i]
				split(reverse[p], imps, SUBSEP)
				for (j = 1; j <= length(imps); j++) {
					q = imps[j]
					if (q == "") continue
					if (!seen[q]++) {
						queue[n++] = q
					}
				}
			}
			for (i = 0; i < n; i++) {
				print queue[i]
			}
		}
	' "${graph}" "${seeds}" >"${affected_imports}"
else
	: >"${affected_imports}"
fi

affected_pkgs=${tmp}/affected-pkgs
: >"${affected_pkgs}"
while IFS= read -r import_path; do
	[[ -n ${import_path} ]] || continue
	import_to_pkg "${import_path}" >>"${affected_pkgs}"
done <"${affected_imports}"

if [[ -s ${test_only_list} ]]; then
	while IFS= read -r pkg; do
		[[ -n ${pkg} ]] || continue
		if grep -Fxq "${pkg}" "${production_list}" 2>/dev/null; then
			continue
		fi
		import_path=$(pkg_to_import "${pkg}")
		if ! grep -Fq "${import_path}"$'\t' "${dirs}"; then
			continue
		fi
		printf '%s\n' "${pkg}" >>"${affected_pkgs}"
	done < <(LC_ALL=C sort -u "${test_only_list}")
fi

total=$(cut -f1 "${dirs}" | LC_ALL=C sort -u | grep -c . || true)
count=$(LC_ALL=C sort -u "${affected_pkgs}" | grep -c . || true)
if ((count == 0)); then
	affected_test_packages=''
elif ((count >= 50)) || ((count * 10 >= total * 8)); then
	fallback=true
	fallback_reason=wide_reverse_closure
	affected_test_packages=./...
else
	affected_test_packages=$(join_sorted "${affected_pkgs}")
fi

emit
