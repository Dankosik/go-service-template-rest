# Go LLM instruction pack

This pack splits Go guidance into one always-on core file and several optional files.

## Loading strategy

Always load:
- `01-go-core-always-load.md`

Load optional files only when the task clearly needs them:
- `10-go-errors-and-context.md`
- `20-go-concurrency.md`
- `30-go-project-layout-and-modules.md`
- `40-go-testing-and-quality.md`
- `50-go-public-api-and-docs.md`
- `60-go-performance-and-profiling.md`
- `70-go-review-checklist.md`

## Resolution rules

1. The core file is the base layer for every Go task.
2. Optional files add detail for a narrower topic.
3. If guidance overlaps, the more specific file should be treated as the decisive rule for that topic.
4. Load the smallest set of files that fully covers the task. Do not load everything by default.

## Suggested file combinations

### Simple function or small refactor
- `01-go-core-always-load.md`

### HTTP handler, database call, CLI command, file I/O
- `01-go-core-always-load.md`
- `10-go-errors-and-context.md`
- `40-go-testing-and-quality.md`

### Worker pool, pipeline, fan-out/fan-in, background processing
- `01-go-core-always-load.md`
- `10-go-errors-and-context.md`
- `20-go-concurrency.md`
- `40-go-testing-and-quality.md`

### New service or repository scaffolding
- `01-go-core-always-load.md`
- `30-go-project-layout-and-modules.md`
- `40-go-testing-and-quality.md`

### Reusable library or exported package
- `01-go-core-always-load.md`
- `10-go-errors-and-context.md`
- `50-go-public-api-and-docs.md`
- `40-go-testing-and-quality.md`

### Performance optimization or memory/latency investigation
- `01-go-core-always-load.md`
- `40-go-testing-and-quality.md`
- `60-go-performance-and-profiling.md`
- `20-go-concurrency.md` if goroutines, channels, or locking are involved

### Code review or "make this more idiomatic"
- `01-go-core-always-load.md`
- `70-go-review-checklist.md`
- add any topic-specific file that matches the code under review

## Notes

- These files are written as direct instructions to an LLM.
- The core file is intentionally compact enough for permanent context.
- The optional files are detailed and should be loaded only when relevant.
