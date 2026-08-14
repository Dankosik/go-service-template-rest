// Package s3 implements the bounded Amazon S3 and Cloudflare R2 adapter. One
// adapter owns one immutable, strictly bounded public-root snapshot loaded from
// the runtime image; it has no provider registry, trust reload, or system-root
// fallback.
package s3
