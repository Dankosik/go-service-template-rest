#!/usr/bin/env bash
set -euo pipefail

TEMPLATE_MODULE="github.com/example/go-service-template-rest"
TEMPLATE_SOURCE="github.com/Dankosik/go-service-template-rest"
TEMPLATE_OWNER="@Dankosik"

usage() {
	echo "usage: CODEOWNER=@user-or-org/team $0 [module-path]"
	echo "module-path is derived from git remote origin when omitted"
}

detect_module_from_origin() {
	local remote_url host path remainder

	remote_url="$(git config --get remote.origin.url 2>/dev/null || true)"
	[[ -n "${remote_url}" ]] || return 1

	case "${remote_url}" in
	git@*:*)
		host="${remote_url#git@}"
		host="${host%%:*}"
		path="${remote_url#*:}"
		;;
	ssh://git@*/*)
		remainder="${remote_url#ssh://git@}"
		host="${remainder%%/*}"
		path="${remainder#*/}"
		;;
	http://* | https://*)
		remainder="${remote_url#*://}"
		host="${remainder%%/*}"
		path="${remainder#*/}"
		;;
	*)
		return 1
		;;
	esac

	host="${host%%:*}"
	path="${path#/}"
	path="${path%/}"
	path="${path%.git}"
	[[ -n "${host}" && -n "${path}" ]] || return 1
	printf '%s/%s\n' "${host}" "${path}"
}

replace_literal() {
	local file="$1"
	local old="$2"
	local new="$3"
	local temporary

	temporary="$(mktemp)"
	awk -v old="${old}" -v new="${new}" '{
		line = $0
		while ((index_at = index(line, old)) != 0) {
			line = substr(line, 1, index_at - 1) new substr(line, index_at + length(old))
		}
		print line
	}' "${file}" >"${temporary}"
	cat "${temporary}" >"${file}"
	rm -f "${temporary}"
}

replace_codeowner_rules() {
	local owner="$1"
	local temporary

	temporary="$(mktemp)"
	awk -v old="${TEMPLATE_OWNER}" -v new="${owner}" '
		/^[[:space:]]*#/ { print; next }
		{ gsub(old, new); print }
	' .github/CODEOWNERS >"${temporary}"
	cat "${temporary}" >.github/CODEOWNERS
	rm -f "${temporary}"
}

if [[ $# -gt 1 ]]; then
	usage
	exit 1
fi

for required_file in go.mod env/.env.example .github/CODEOWNERS .golangci.yml; do
	[[ -f "${required_file}" ]] || {
		echo "required template file not found: ${required_file}"
		exit 1
	}
done

current_module="$(awk '/^module / { print $2; exit }' go.mod)"
[[ -n "${current_module}" ]] || {
	echo "failed to read module path from go.mod"
	exit 1
}

detected_module="$(detect_module_from_origin || true)"
new_module="${1:-}"
source_checkout=false

if [[ -z "${new_module}" ]]; then
	[[ -n "${detected_module}" ]] || {
		echo "module path is required when git remote origin cannot be derived"
		usage
		exit 1
	}
	if [[ "${detected_module}" == "${TEMPLATE_SOURCE}" && "${current_module}" == "${TEMPLATE_MODULE}" ]]; then
		new_module="${current_module}"
		source_checkout=true
	else
		new_module="${detected_module}"
	fi
fi

validation_mod="$(mktemp)"
cp go.mod "${validation_mod}"
trap 'rm -f "${validation_mod}"' EXIT
if ! go mod edit -module="${new_module}" "${validation_mod}" >/dev/null 2>&1; then
	echo "invalid Go module path: ${new_module}"
	exit 1
fi

codeowner="${CODEOWNER:-}"
if [[ "${source_checkout}" != true ]]; then
	[[ -n "${codeowner}" ]] || {
		echo "CODEOWNER is required when initializing a repository derived from the template"
		exit 1
	}
	if [[ ! "${codeowner}" =~ ^@[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?(/[A-Za-z0-9]([A-Za-z0-9_-]*[A-Za-z0-9])?)?$ ]]; then
		echo "CODEOWNER must be one @username or @org/team-name token"
		exit 1
	fi
fi

if [[ "${new_module}" != "${current_module}" ]] && ! grep -Fq "${current_module}" .golangci.yml; then
	echo ".golangci.yml does not contain the current module path; refusing to disable depguard during initialization"
	exit 1
fi

if [[ ! -f .env ]]; then
	cp env/.env.example .env
	echo "created .env from env/.env.example"
fi

if [[ "${new_module}" != "${current_module}" ]]; then
	go mod edit -module="${new_module}"

	while IFS= read -r file; do
		[[ -f "${file}" ]] || continue
		if grep -Fq "${current_module}" "${file}"; then
			replace_literal "${file}" "${current_module}" "${new_module}"
		fi
	done < <(git ls-files --cached --others --exclude-standard -- '*.go' '*.proto')

	replace_literal .golangci.yml "${current_module}" "${new_module}"
fi

if [[ "${source_checkout}" != true ]]; then
	replace_codeowner_rules "${codeowner}"
fi

go mod tidy

echo "template initialization complete"
echo "  module: ${new_module}"
if [[ -n "${codeowner}" ]]; then
	echo "  codeowner: ${codeowner}"
fi
