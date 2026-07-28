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

expect_secret() {
	local description="$1"
	local status
	shift
	set +e
	"$@" >/dev/null 2>&1
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
}

self_test() {
	local fixture
	local fake_secret

	fixture="$(mktemp -d -t secret-scan-check.XXXXXX)"
	CLEANUP_DIR="${fixture}"
	fake_secret='ghp_'
	fake_secret+='A1b2C3d4E5f6G7h8I9j0K1l2M3n4O5p6Q7r8'

	git init -q "${fixture}"
	git -C "${fixture}" checkout -q -b main
	cp "${ROOT_DIR}/.gitleaks.toml" "${fixture}/.gitleaks.toml"
	printf '[]\n' >"${fixture}/.gitleaks.baseline.json"
	printf 'safe\n' >"${fixture}/README.md"
	git -C "${fixture}" add .
	git -C "${fixture}" -c user.name=secret-scan-check \
		-c user.email=secret-scan-check@example.com commit -qm initial

	bash "${BASH_SOURCE[0]}" change main "${fixture}" >/dev/null

	printf 'token=%s\n' "${fake_secret}" >"${fixture}/leak.txt"
	expect_secret "untracked worktree secret" \
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

	expect_secret "secret deleted inside the change range" \
		bash "${BASH_SOURCE[0]}" change main "${fixture}"
	expect_secret "full-history secret" \
		bash "${BASH_SOURCE[0]}" history main "${fixture}"
	expect_secret "missing-base fail-safe history scan" \
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
