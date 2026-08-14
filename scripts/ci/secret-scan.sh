#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
MODE="${1:-}"
BASE_REF="${2:-origin/main}"
REPOSITORY="${3:-${ROOT_DIR}}"

if [[ -z "${MODE}" ]]; then
	echo "usage: $0 <change|history|self-test> [base-ref] [repository]" >&2
	exit 2
fi

REPOSITORY="$(cd "${REPOSITORY}" && pwd)"
GITLEAKS=(bash "${ROOT_DIR}/scripts/run-go-tool.sh" gitleaks)
GITLEAKS_ARGS=(--no-banner --redact --exit-code 1)
if [[ "${SECRET_SCAN_VERBOSE:-}" == "1" ]]; then
	GITLEAKS_ARGS+=(--verbose)
fi
CLEANUP_DIR=""

cleanup() {
	if [[ -n "${CLEANUP_DIR}" ]]; then
		rm -rf "${CLEANUP_DIR}"
	fi
}
trap cleanup EXIT

if [[ -f "${REPOSITORY}/.gitleaks.toml" ]]; then
	GITLEAKS_ARGS+=(--config "${REPOSITORY}/.gitleaks.toml")
fi
if [[ -f "${REPOSITORY}/.gitleaks.baseline.json" ]]; then
	GITLEAKS_ARGS+=(--baseline-path "${REPOSITORY}/.gitleaks.baseline.json")
fi

copy_current_path() {
	local snapshot="$1"
	local path="$2"
	local parent

	if [[ ! -e "${REPOSITORY}/${path}" && ! -L "${REPOSITORY}/${path}" ]]; then
		rm -rf "${snapshot:?}/${path}"
		return
	fi

	parent="${path%/*}"
	if [[ "${parent}" != "${path}" ]]; then
		mkdir -p "${snapshot}/${parent}"
	fi
	rm -rf "${snapshot:?}/${path}"
	cp -pP "${REPOSITORY}/${path}" "${snapshot}/${path}"
}

scan_worktree() {
	local snapshot
	local status
	local path

	snapshot="$(mktemp -d -t secret-scan-worktree.XXXXXX)"
	CLEANUP_DIR="${snapshot}"
	if ! git -C "${REPOSITORY}" checkout-index --all --prefix="${snapshot}/"; then
		rm -rf "${snapshot}"
		CLEANUP_DIR=""
		return 1
	fi

	while IFS= read -r -d '' path; do
		copy_current_path "${snapshot}" "${path}"
	done < <(git -C "${REPOSITORY}" ls-files -z --modified --deleted)
	while IFS= read -r -d '' path; do
		copy_current_path "${snapshot}" "${path}"
	done < <(git -C "${REPOSITORY}" ls-files -z --others --exclude-standard)

	status=0
	(
		cd "${snapshot}"
		"${GITLEAKS[@]}" dir "${GITLEAKS_ARGS[@]}" .
	) || status=$?
	rm -rf "${snapshot}"
	CLEANUP_DIR=""
	return "${status}"
}

scan_history() {
	(
		cd "${REPOSITORY}"
		"${GITLEAKS[@]}" git "${GITLEAKS_ARGS[@]}" .
	)
}

scan_change() {
	local base_commit
	local head_commit

	scan_worktree

	if ! base_commit="$(git -C "${REPOSITORY}" rev-parse --verify "${BASE_REF}^{commit}" 2>/dev/null)"; then
		echo "secret scan: base ref ${BASE_REF} is unavailable; scanning full history" >&2
		scan_history
		return
	fi
	base_commit="$(git -C "${REPOSITORY}" merge-base "${base_commit}" HEAD)"
	head_commit="$(git -C "${REPOSITORY}" rev-parse HEAD)"
	if [[ "${base_commit}" == "${head_commit}" ]]; then
		echo "secret scan: worktree checked; no commits after merge base ${base_commit}"
		return
	fi

	(
		cd "${REPOSITORY}"
		"${GITLEAKS[@]}" git "${GITLEAKS_ARGS[@]}" \
			--log-opts="${base_commit}..HEAD" .
	)
}

expect_rule() {
	local description="$1"
	local expected_rule="$2"
	local output
	local status
	shift 2
	set +e
	output="$(SECRET_SCAN_VERBOSE=1 "$@" 2>&1)"
	status=$?
	set -e
	if [[ "${status}" -eq 0 ]]; then
		echo "secret scan self-test: ${description} was not detected" >&2
		exit 1
	fi
	if [[ "${status}" -ne 1 ]]; then
		echo "secret scan self-test: ${description} returned unexpected status ${status}" >&2
		exit 1
	fi
	if [[ ! "${output}" =~ RuleID:[[:space:]]+${expected_rule} ]]; then
		echo "secret scan self-test: ${description} did not report RuleID ${expected_rule}" >&2
		exit 1
	fi
}

self_test() {
	local fixture
	local script_receipt
	local design_receipt
	local generic_api_key
	local fake_secret

	fixture="$(mktemp -d -t secret-scan-check.XXXXXX)"
	CLEANUP_DIR="${fixture}"
	script_receipt=$'\t'
	script_receipt+="'github.com/aws/aws-sdk-go-v2/"
	script_receipt+='credentials v1.19.5 h1:'
	script_receipt+='xMo63RlqP3ZZydpJDMBsH9uJ10hgHYfQFIk1cHDXrR4='
	script_receipt+="' \\"
	design_receipt='github.com/aws/aws-sdk-go-v2/'
	design_receipt+='credentials v1.19.5 h1:'
	design_receipt+='xMo63RlqP3ZZydpJDMBsH9uJ10hgHYfQFIk1cHDXrR4='
	generic_api_key="$(printf '%s%s' 'A1b2C3d4E5f6G7h8I9j0' 'K1l2M3n4O5p6Q7r8')"
	fake_secret='ghp_'
	fake_secret+='A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8'

	git init -q "${fixture}"
	git -C "${fixture}" checkout -q -b main
	printf 'title = "secret-scan self-test"\n\n[extend]\nuseDefault = true\n' >"${fixture}/.gitleaks.toml"
	printf '[]\n' >"${fixture}/.gitleaks.baseline.json"
	printf 'safe\n' >"${fixture}/README.md"
	git -C "${fixture}" add .
	git -C "${fixture}" -c user.name=secret-scan-check \
		-c user.email=secret-scan-check@example.com commit -qm initial

	mkdir -p "${fixture}/scripts/ci" "${fixture}/specs/s3-compatible-object-storage/design"
	printf '%s\n' "${script_receipt}" >"${fixture}/scripts/ci/s3-source-receipt.sh"
	expect_rule "S3 source receipt before its exception" "generic-api-key" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	rm -f "${fixture}/scripts/ci/s3-source-receipt.sh"
	printf '%s\n' "${design_receipt}" >"${fixture}/specs/s3-compatible-object-storage/design/overview.md"
	expect_rule "S3 design receipt before its exception" "generic-api-key" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	rm -f "${fixture}/specs/s3-compatible-object-storage/design/overview.md"

	cp "${ROOT_DIR}/.gitleaks.toml" "${fixture}/.gitleaks.toml"
	printf '%s\n' "${script_receipt}" >"${fixture}/scripts/ci/s3-source-receipt.sh"
	bash "${BASH_SOURCE[0]}" change main "${fixture}" >/dev/null
	printf '%s\n' "${design_receipt}" >"${fixture}/specs/s3-compatible-object-storage/design/overview.md"
	bash "${BASH_SOURCE[0]}" change main "${fixture}" >/dev/null

	printf '%s\n' "${script_receipt}" >"${fixture}/specs/s3-compatible-object-storage/design/overview.md"
	expect_rule "S3 source receipt at the design path" "generic-api-key" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	printf '%s\n' "${design_receipt}" >"${fixture}/specs/s3-compatible-object-storage/design/overview.md"
	printf '%s\n' "${design_receipt}" >"${fixture}/scripts/ci/s3-source-receipt.sh"
	expect_rule "S3 design receipt at the source path" "generic-api-key" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	printf '%s\n' "${script_receipt}" >"${fixture}/scripts/ci/s3-source-receipt.sh"

	printf 'api_key = %s\n' "${generic_api_key}" >>"${fixture}/scripts/ci/s3-source-receipt.sh"
	expect_rule "generic API key on a separate S3 source receipt line" "generic-api-key" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	printf '%s\n' "${script_receipt}" >"${fixture}/scripts/ci/s3-source-receipt.sh"
	printf 'api_key = %s\n' "${generic_api_key}" >>"${fixture}/specs/s3-compatible-object-storage/design/overview.md"
	expect_rule "generic API key on a separate S3 design receipt line" "generic-api-key" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	printf '%s\n' "${design_receipt}" >"${fixture}/specs/s3-compatible-object-storage/design/overview.md"

	printf '%s api_key = %s\n' "${script_receipt}" "${generic_api_key}" >"${fixture}/scripts/ci/s3-source-receipt.sh"
	expect_rule "generic API key appended to the S3 source receipt line" "generic-api-key" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	printf '%s\n' "${script_receipt}" >"${fixture}/scripts/ci/s3-source-receipt.sh"
	printf '%s api_key = %s\n' "${design_receipt}" "${generic_api_key}" >"${fixture}/specs/s3-compatible-object-storage/design/overview.md"
	expect_rule "generic API key appended to the S3 design receipt line" "generic-api-key" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	printf '%s\n' "${design_receipt}" >"${fixture}/specs/s3-compatible-object-storage/design/overview.md"

	printf '%s\n' "${script_receipt}" >"${fixture}/unapproved.txt"
	expect_rule "S3 source receipt in an unapproved path" "generic-api-key" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	printf '%s\n' "${design_receipt}" >"${fixture}/unapproved.txt"
	expect_rule "S3 design receipt in an unapproved path" "generic-api-key" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	rm -f "${fixture}/unapproved.txt"

	printf 'token=%s\n' "${fake_secret}" >"${fixture}/leak.txt"
	expect_rule "untracked worktree secret" "github-pat" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	rm -f "${fixture}/leak.txt"

	git -C "${fixture}" checkout -q -b feature
	printf 'token=%s\n' "${fake_secret}" >"${fixture}/leak.txt"
	git -C "${fixture}" add leak.txt
	git -C "${fixture}" -c user.name=secret-scan-check \
		-c user.email=secret-scan-check@example.com commit -qm add-secret
	git -C "${fixture}" rm -q leak.txt
	git -C "${fixture}" -c user.name=secret-scan-check \
		-c user.email=secret-scan-check@example.com commit -qm remove-secret

	expect_rule "secret deleted inside the change range" "github-pat" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	expect_rule "full-history secret" "github-pat" \
		bash "${BASH_SOURCE[0]}" history main "${fixture}"
	expect_rule "missing-base fail-safe history scan" "github-pat" \
		bash "${BASH_SOURCE[0]}" change refs/heads/missing "${fixture}"

	rm -rf "${fixture}"
	CLEANUP_DIR=""
	echo "secret scan routing is fail-closed"
}

case "${MODE}" in
change)
	scan_change
	;;
history)
	scan_history
	;;
self-test)
	self_test
	;;
*)
	echo "unknown secret scan mode: ${MODE}" >&2
	exit 2
	;;
esac
