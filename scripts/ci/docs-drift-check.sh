#!/usr/bin/env bash
set -euo pipefail

BASE_REF="${1:-}"
HEAD_REF="${2:-}"
ZERO_SHA="0000000000000000000000000000000000000000"

if [[ -z "${BASE_REF}" || -z "${HEAD_REF}" ]]; then
  echo "usage: $0 <base_ref> <head_ref>"
  exit 1
fi

if [[ "${BASE_REF}" == "${ZERO_SHA}" ]]; then
  echo "base ref is an empty SHA, skipping docs drift check"
  exit 0
fi

if ! git cat-file -e "${BASE_REF}^{commit}" 2>/dev/null; then
  echo "base ref commit not found: ${BASE_REF}"
  exit 1
fi

if ! git cat-file -e "${HEAD_REF}^{commit}" 2>/dev/null; then
  echo "head ref commit not found: ${HEAD_REF}"
  exit 1
fi

changed_files="$(git diff --name-only "${BASE_REF}" "${HEAD_REF}")"
if [[ -z "${changed_files}" ]]; then
  echo "no files changed, docs drift check passed"
  exit 0
fi

requires_docs_pattern='^(api/openapi/service\.yaml|env/migrations/|Makefile|\.github/workflows/|cmd/|internal/app/|internal/infra/http/)'
docs_pattern='^(docs/|README\.md$)'

if echo "${changed_files}" | grep -Eq "${requires_docs_pattern}"; then
  if ! echo "${changed_files}" | grep -Eq "${docs_pattern}"; then
    echo "docs drift: behavior/contract/ci-sensitive files changed without docs update"
    echo "changed files:"
    echo "${changed_files}"
    exit 1
  fi
fi

echo "docs drift check passed"
