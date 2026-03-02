#!/usr/bin/env bash
set -euo pipefail

required_files=(
  ".editorconfig"
  ".gitattributes"
  ".github/CODEOWNERS"
  ".github/pull_request_template.md"
  "CONTRIBUTING.md"
  "SECURITY.md"
  "LICENSE"
)

missing=()
for file in "${required_files[@]}"; do
  if [[ ! -f "${file}" ]]; then
    missing+=("${file}")
  fi
done

if [[ ${#missing[@]} -gt 0 ]]; then
  echo "required repository guardrails are missing:"
  for file in "${missing[@]}"; do
    echo "- ${file}"
  done
  exit 1
fi

echo "required repository guardrails check passed"
