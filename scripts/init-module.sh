#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: $0 <module-path>"
	echo "example: CODEOWNER=@acme/backend-team $0 github.com/acme/my-service"
	exit 1
fi

new_module="$1"
codeowner="${CODEOWNER:-}"
codeowner_placeholder="@your-org/your-team"

if [[ "$new_module" =~ [[:space:]] ]]; then
	echo "module path must not contain spaces"
	exit 1
fi

if [[ "$new_module" != */* ]]; then
	echo "module path must look like an import path (for example github.com/acme/my-service)"
	exit 1
fi

if [[ ! -f "go.mod" ]]; then
	echo "go.mod not found in current directory"
	exit 1
fi

current_module="$(awk '/^module /{print $2; exit}' go.mod)"
if [[ -z "$current_module" ]]; then
	echo "failed to read current module path from go.mod"
	exit 1
fi

replace_all_in_file() {
	local file="$1"
	local old="$2"
	local new="$3"
	local tmp_file

	tmp_file="$(mktemp)"
	awk -v old="$old" -v new="$new" '{
		line = $0
		while ((idx = index(line, old)) != 0) {
			line = substr(line, 1, idx - 1) new substr(line, idx + length(old))
		}
		print line
	}' "$file" >"${tmp_file}"
	cat "${tmp_file}" >"${file}"
	rm -f "${tmp_file}"
}

if [[ "$new_module" == "$current_module" ]]; then
	echo "module path is already set to $new_module"
else
	go mod edit -module "$new_module"
fi

files=()
if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
	while IFS= read -r file; do
		files+=("$file")
	done < <(git ls-files --cached --others --exclude-standard -- '*.go' '*.proto')
else
	while IFS= read -r file; do
		files+=("$file")
	done < <(find . -type f \( -name '*.go' -o -name '*.proto' \) -not -path './vendor/*')
fi

for file in "${files[@]}"; do
	[[ -f "$file" ]] || continue
	if grep -Fq "$current_module" "$file"; then
		replace_all_in_file "$file" "$current_module" "$new_module"
	fi
done

if [[ -n "$codeowner" ]]; then
	if [[ "${codeowner}" != @* ]]; then
		echo "CODEOWNER must start with '@' (for example @acme/backend-team)"
		exit 1
	fi
	if [[ -f ".github/CODEOWNERS" ]] && grep -Fq "${codeowner_placeholder}" ".github/CODEOWNERS"; then
		replace_all_in_file ".github/CODEOWNERS" "${codeowner_placeholder}" "${codeowner}"
	fi
elif [[ -f ".github/CODEOWNERS" ]] && grep -Fq "${codeowner_placeholder}" ".github/CODEOWNERS"; then
	echo "warning: .github/CODEOWNERS still contains template placeholder ${codeowner_placeholder}"
	echo "         update CODEOWNERS before enabling required code owner reviews"
fi

go mod tidy

echo "module path updated:"
echo "  old: $current_module"
echo "  new: $new_module"
if [[ -n "$codeowner" ]]; then
	echo "  codeowner: ${codeowner}"
fi
echo "next step: run 'go test ./...'"
