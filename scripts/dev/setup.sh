#!/usr/bin/env bash
set -euo pipefail

if [[ $# -gt 1 ]]; then
	echo "usage: $0 [golangci-lint-version]"
	echo "example: $0 v2.10.1"
	exit 1
fi

lint_version="${1:-v2.10.1}"
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if ! command -v go >/dev/null 2>&1; then
	echo "go is required for setup. install Go from https://go.dev/dl/"
	exit 1
fi

echo "Installing golangci-lint ${lint_version}..."
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@"${lint_version}"

if [[ ! -f ".env" ]]; then
	cp env/.env.example .env
	echo "Created .env from env/.env.example"
fi

echo "Downloading Go modules..."
go mod download

echo "Running environment doctor..."
"${ROOT_DIR}/scripts/dev/doctor.sh"

echo "Setup complete."
echo "Next steps:"
echo "  1) make init-module MODULE=github.com/<your-org>/<your-service>"
echo "  2) make test"
