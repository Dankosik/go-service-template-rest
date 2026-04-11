# Path, Upload, And Filesystem Safety Review

## Behavior Change Thesis
When loaded for symptom "caller-influenced data selects a file operation," this file makes the model choose root-constrained file access, upload isolation, and archive member policy instead of likely mistake trusting `filepath.Clean`, `filepath.Join`, or uploaded filenames.

## When To Load
Load this when changed Go code handles user-controlled file paths, upload filenames, multipart data, archive extraction, static file serving, file downloads, config file loading, temporary files, or filesystem writes derived from external input.

If the primary issue is only request body size, load the abuse reference. If the primary issue is a config file trust boundary, load trust-boundary first and use this only for path/root/symlink behavior.

## Decision Rubric
- Identify the operation first: read, write, extract, serve, scan, transform, delete, or temporary-file creation.
- Treat `filepath.Join(base, userPath)`, `filepath.Clean`, and `filepath.Abs` as lexical helpers, not confinement proof.
- Prefer `os.Root` or `os.OpenInRoot` for untrusted filenames when available and appropriate.
- Use `filepath.IsLocal` or `filepath.Localize` only for lexical validation; do not claim they solve symlink races.
- Generate server-side storage keys; do not reuse uploaded filenames as durable keys or public paths.
- Enforce request body and per-file limits before parsing or processing uploads.
- Validate decoded extension plus detected content type or magic bytes according to the business need.
- Keep uploads outside direct public serving unless an authorization, scan, or publish gate explicitly approves exposure.
- Keep path and upload errors generic to clients while logging only sanitized, non-sensitive context.

## Imitate
```text
[high] [go-security-review] internal/app/files.go:51
Issue: Axis: Path And Filesystem Safety; `Download` joins `baseDir` with the request `name` and opens the result without proving the final path remains under `baseDir`.
Impact: A caller can request a path outside the download root where lexical cleanup is insufficient or symlinks are in scope.
Suggested fix: Use `os.OpenInRoot` or an `os.Root` opened on `baseDir`, and reject non-local names before the open.
Reference: download root confinement.
```

Copy this shape when the file operation can escape a root.

```text
[high] [go-security-review] internal/app/upload.go:88
Issue: Axis: Upload Safety; the upload handler stores `header.Filename` directly under the public assets directory after checking only the extension.
Impact: A user can choose colliding or dangerous names and publish active content under a served path.
Suggested fix: Generate a server-side storage key, enforce body and per-file size limits, validate decoded extension plus detected content type, and keep uploads outside direct public serving until scanned or approved.
Reference: upload storage boundary.
```

Copy this shape when filename trust and public serving combine.

```text
[medium] [go-security-review] internal/app/archive.go:119
Issue: Axis: Archive Extraction Safety; archive entries are extracted with raw member paths.
Impact: A crafted archive can write outside the extraction root or plant a symlink for a later member.
Suggested fix: Use root-constrained extraction, reject non-local member names, and define symlink policy before extraction.
Reference: archive extraction boundary.
```

Copy this shape when archive member paths create a second filesystem boundary.

## Reject
```text
Issue: This could be path traversal.
```

Reject because it does not name the file operation, attacker-controlled path segment, or escape impact.

```text
Suggested fix: Run `filepath.Clean` before joining the filename.
```

Reject because lexical cleanup alone does not prove root confinement or symlink behavior.

## Agent Traps
- Do not trust `header.Filename`, `Content-Type`, or extension alone.
- Do not forget Windows path forms and reserved names when the code claims cross-platform support.
- Do not assume temp directories are safe if attacker-controlled names or permissions are introduced.
- Do not expose internal absolute paths in client-visible errors.
- Do not require malware scanning unless the product contract or upload/publish path makes it relevant; if relevant but absent, name it as a policy gap.

## Validation Shape
- Add tests for absolute paths, `..`, empty paths, long names, leading dots, duplicate separators, and platform-specific path forms when supported.
- Add symlink tests when local filesystem manipulation is in the threat model.
- Add upload tests for oversized bodies, unsupported extensions, spoofed content type, duplicate filenames, and public retrieval policy.
- Add archive extraction tests proving members cannot escape the root and symlinks follow the chosen policy.

## Repo-Local Anchors
- `internal/config/load_koanf.go` shows bounded config reads, allowed-root checks, symlink rejection outside local environments, and permission checks.
- `internal/infra/http/middleware.go` uses `http.MaxBytesReader` for request body limits.
