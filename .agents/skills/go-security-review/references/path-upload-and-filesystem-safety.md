# Path, Upload, And Filesystem Safety Review

## When To Load
Load this when changed Go code handles user-controlled file paths, upload filenames, multipart data, archive extraction, static file serving, file downloads, config file loading, temporary files, or filesystem writes derived from external input.

## Attacker Preconditions
- The attacker can influence a path, filename, archive member name, upload content, content type, extension, symlink, or directory component.
- The service reads, writes, extracts, serves, scans, transforms, or deletes a file based on that input.
- The file operation has access to data, directories, interpreters, public serving paths, or storage capacity the attacker should not control.

## Review Signals
- `filepath.Join(base, userPath)` is used as the only confinement control.
- `filepath.Clean` or `filepath.Abs` is treated as proof of safety without `filepath.IsLocal`, `filepath.Localize`, `os.Root`, `os.OpenInRoot`, or equivalent root confinement.
- Symlink handling is split into check-then-open steps that can race.
- Uploaded filenames are reused as storage keys or public paths.
- Upload code trusts `Content-Type` or extension alone, lacks size limits, or stores directly under webroot with execute or public read behavior.
- Archive extraction writes member names without local-path and symlink policy.
- Error messages disclose internal absolute paths.

## Bad Finding Examples
- "This could be path traversal."
- "Sanitize uploaded filenames."
- "Do not trust Content-Type."

These are incomplete unless they identify the file operation, attacker-controlled component, and escape or storage impact.

## Good Finding Examples
- "[high] [go-security-review] internal/app/files.go:51 Axis: Path And Filesystem Safety; `Download` joins `baseDir` with the request `name` and opens the result without proving the final path remains under `baseDir`. A caller can request a path outside the download root on platforms where lexical cleanup is insufficient. Use `os.OpenInRoot` or an `os.Root` opened on `baseDir`, and reject non-local names before the open."
- "[high] [go-security-review] internal/app/upload.go:88 Axis: Upload Safety; the upload handler stores `header.Filename` directly under the public assets directory after checking only the extension. A user can choose colliding or dangerous names and publish active content under a served path. Generate a server-side storage key, enforce body and per-file size limits, validate decoded extension plus detected content type, and keep uploads outside direct public serving until scanned or approved."
- "[medium] [go-security-review] internal/app/archive.go:119 Axis: Archive Extraction Safety; archive entries are extracted with raw member paths. A crafted archive can write outside the extraction root or plant a symlink for a later member. Use `os.Root`/`OpenInRoot` style extraction, reject non-local member names, and define symlink policy before extraction."

## Smallest Safe Correction
- Prefer `os.Root` or `os.OpenInRoot` for untrusted filenames when Go version and platform allow it.
- Use `filepath.IsLocal` or `filepath.Localize` for lexical path validation when attacker filesystem access is out of threat model; do not claim it handles symlink races.
- Generate server-side filenames or opaque IDs for storage keys.
- Enforce request body and per-file size limits before parsing and processing.
- Validate extension allowlist after decoding, then validate content type or magic bytes according to business need.
- Store uploads outside webroot or serve through an authorization-checking handler.
- Keep path and upload errors generic to clients while logging only sanitized, non-sensitive context.

## Validation Ideas
- Add tests for absolute paths, `..`, empty paths, Windows reserved names when supported, long names, leading dots, duplicate separators, and symlink attempts when the threat model includes local filesystem manipulation.
- Add upload tests for oversized bodies, unsupported extensions, spoofed content type, duplicate filenames, and public retrieval policy.
- Add archive extraction tests that prove members cannot escape the root and symlinks are handled according to policy.
- Run targeted package tests plus `make go-security` when filesystem APIs or upload handlers changed.

## Repo-Local Anchors
- `internal/config/load_koanf.go` is a local example of bounded config file reads, allowed root checks, symlink rejection outside local environments, and permission checks.
- `internal/infra/http/middleware.go` uses `http.MaxBytesReader` for request body limits.

## Exa Source Links
- OWASP File Upload Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html
- OWASP Input Validation Cheat Sheet: https://cheatsheetseries.owasp.org/cheatsheets/Input_Validation_Cheat_Sheet.html
- Go `path/filepath` package docs: https://pkg.go.dev/path/filepath
- Go "Traversal-resistant file APIs": https://go.dev/blog/osroot
- Go `net/http` package docs: https://pkg.go.dev/net/http
