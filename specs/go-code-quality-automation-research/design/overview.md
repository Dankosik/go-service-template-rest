# Design Overview

## Approach

This is a tooling and workflow improvement, not a runtime product feature. The design keeps the existing command hierarchy intact:

- `make check` remains the everyday quick path.
- `make check-full` remains the local CI-like entrypoint, preferring Docker and falling back to native partial evidence with clearer wording.
- `make docker-ci` remains the closest zero-setup parity path.
- Nightly remains the home for heavier flake/fuzz checks.

## Artifact Index

- `component-map.md`: affected files and responsibilities.
- `sequence.md`: command flow changes.
- `ownership-map.md`: source-of-truth ownership.

## Readiness

Implementation is ready. Research provided candidate linter evidence, and the remaining choices are now fixed in `spec.md`.
