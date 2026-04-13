#!/usr/bin/env bash
set -euo pipefail

usage() {
	echo "usage: $0 [--check] [branch]"
	echo "examples:"
	echo "  $0 main"
	echo "  $0 --check main"
}

mode="apply"
branch="main"
branch_set=0

while [[ $# -gt 0 ]]; do
	case "$1" in
		--check)
			mode="check"
			shift
			;;
		-h | --help)
			usage
			exit 0
			;;
		-*)
			echo "unknown option: $1"
			usage
			exit 1
			;;
		*)
			if [[ "${branch_set}" -eq 1 ]]; then
				echo "unexpected argument: $1"
				usage
				exit 1
			fi
			branch="$1"
			branch_set=1
			shift
			;;
	esac
done

required_contexts=(
	"repo-integrity"
	"lint"
	"openapi-contract"
	"openapi-breaking"
	"test"
	"test-race"
	"test-coverage"
	"test-integration"
	"migration-validate"
	"go-security"
	"secret-scan"
	"container-security"
)

codeowner_placeholder="@your-org/your-team"

if [[ "${mode}" == "apply" && -f ".github/CODEOWNERS" ]] && grep -Fq "${codeowner_placeholder}" ".github/CODEOWNERS"; then
	echo "CODEOWNERS still contains placeholder '${codeowner_placeholder}'."
	echo "Update .github/CODEOWNERS (or run make init-module ... CODEOWNER=...) before enabling code owner review protection."
	exit 1
fi

if ! command -v gh >/dev/null 2>&1; then
	echo "gh CLI is required. Install from https://cli.github.com/."
	exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
	echo "gh is not authenticated. Run: gh auth login"
	exit 1
fi

repo="${GITHUB_REPOSITORY:-}"
if [[ -z "${repo}" ]]; then
	repo="$(gh repo view --json nameWithOwner -q '.nameWithOwner' 2>/dev/null || true)"
fi

if [[ -z "${repo}" ]]; then
	echo "cannot detect repository. Set GITHUB_REPOSITORY=<owner/repo> or run inside a cloned GitHub repo."
	exit 1
fi

required_checks_json() {
	local first=1

	for context in "${required_contexts[@]}"; do
		if [[ "${first}" -eq 0 ]]; then
			printf ',\n'
		fi
		printf '      {"context": "%s"}' "${context}"
		first=0
	done
}

check_required_contexts() {
	local expected_file
	local actual_file
	local error_file
	local live_contexts
	local missing
	local unexpected

	expected_file="$(mktemp)"
	actual_file="$(mktemp)"
	error_file="$(mktemp)"

	printf '%s\n' "${required_contexts[@]}" | LC_ALL=C sort -u >"${expected_file}"

	echo "Checking branch protection required status contexts for ${repo}:${branch}..."
	if ! live_contexts="$(
		gh api \
			-H "Accept: application/vnd.github+json" \
			"/repos/${repo}/branches/${branch}/protection/required_status_checks" \
			--jq '[.checks[]?.context, .contexts[]?] | unique | .[]' 2>"${error_file}"
	)"; then
		echo "cannot read required status checks for ${repo}:${branch}."
		echo "Confirm the branch exists, branch protection is enabled, and your gh token can read repository administration settings."
		cat "${error_file}"
		rm -f "${expected_file}" "${actual_file}" "${error_file}"
		exit 1
	fi

	printf '%s\n' "${live_contexts}" | sed '/^$/d' | LC_ALL=C sort -u >"${actual_file}"
	missing="$(LC_ALL=C comm -23 "${expected_file}" "${actual_file}")"
	unexpected="$(LC_ALL=C comm -13 "${expected_file}" "${actual_file}")"

	if [[ -z "${missing}" && -z "${unexpected}" ]]; then
		echo "Required status contexts match for ${branch}."
		rm -f "${expected_file}" "${actual_file}" "${error_file}"
		return 0
	fi

	echo "Branch protection required status contexts differ for ${repo}:${branch}."
	if [[ -n "${missing}" ]]; then
		echo "Missing required contexts:"
		printf '%s\n' "${missing}" | sed 's/^/- /'
	fi
	if [[ -n "${unexpected}" ]]; then
		echo "Unexpected required contexts:"
		printf '%s\n' "${unexpected}" | sed 's/^/- /'
	fi

	rm -f "${expected_file}" "${actual_file}" "${error_file}"
	exit 1
}

if [[ "${mode}" == "check" ]]; then
	check_required_contexts
	exit 0
fi

checks_json="$(required_checks_json)"

read -r -d '' payload <<JSON || true
{
  "required_status_checks": {
    "strict": true,
    "checks": [
${checks_json}
    ]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": {
    "dismiss_stale_reviews": true,
    "require_code_owner_reviews": true,
    "required_approving_review_count": 1
  },
  "restrictions": null,
  "required_conversation_resolution": true,
  "allow_force_pushes": false,
  "allow_deletions": false,
  "block_creations": false,
  "required_linear_history": false,
  "lock_branch": false,
  "allow_fork_syncing": true
}
JSON

echo "Configuring branch protection for ${repo}:${branch}..."
gh api \
	--method PUT \
	-H "Accept: application/vnd.github+json" \
	"/repos/${repo}/branches/${branch}/protection" \
	--input - <<<"${payload}" >/dev/null

echo "Branch protection configured for ${branch}."
