#!/usr/bin/env bash
# Git pre-push hook: runs the checks `make verify` selects for the changed
# surfaces and leaves heavy steps to CI. It checks the working tree, so
# uncommitted changes count. Skip once with `git push --no-verify`.
set -euo pipefail

exec make verify
