#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: $0 <module-path>"
	echo "example: $0 github.com/acme/my-service"
	exit 1
fi

new_module="$1"

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

if [[ "$new_module" == "$current_module" ]]; then
	echo "module path is already set to $new_module"
	exit 0
fi

go mod edit -module "$new_module"

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
		CURRENT_MODULE="$current_module" NEW_MODULE="$new_module" \
			perl -i -pe 's/\Q$ENV{CURRENT_MODULE}\E/$ENV{NEW_MODULE}/g' "$file"
	fi
done

go mod tidy

echo "module path updated:"
echo "  old: $current_module"
echo "  new: $new_module"
echo "next step: run 'go test ./...'"
