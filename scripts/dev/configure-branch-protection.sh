#!/usr/bin/env bash
set -euo pipefail

if [[ $# -gt 1 ]]; then
	echo "usage: $0 [branch]"
	echo "example: $0 main"
	exit 1
fi

branch="${1:-main}"
codeowner_placeholder="@your-org/your-team"

if [[ -f ".github/CODEOWNERS" ]] && grep -Fq "${codeowner_placeholder}" ".github/CODEOWNERS"; then
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

read -r -d '' payload <<JSON || true
{
  "required_status_checks": {
    "strict": true,
    "checks": [
      {"context": "repo-integrity"},
      {"context": "lint"},
      {"context": "openapi-contract"},
      {"context": "openapi-breaking"},
      {"context": "test"},
      {"context": "test-race"},
      {"context": "test-coverage"},
      {"context": "test-integration"},
      {"context": "migration-validate"},
      {"context": "go-security"},
      {"context": "secret-scan"},
      {"context": "container-security"}
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
