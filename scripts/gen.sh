#!/usr/bin/env bash
set -euo pipefail

echo "Generating OpenAPI Go bindings..."
go generate ./internal/api
