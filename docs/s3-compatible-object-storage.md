# S3-compatible object storage

<!-- profile:object-storage:start -->
`OBJECT_STORAGE=s3` retains one static S3-compatible adapter. Supply its
complete provider, endpoint, region, bucket, static credentials, and finite
bounds through `APP__OBJECT_STORAGE__*`; the generated examples are deliberately
empty and unusable.

The adapter does not provision a bucket, make startup or readiness requests, or
create a deployed-provider claim. `make test-s3-conformance-amazon` and
`make test-s3-conformance-r2` are reserved fail-closed entrypoints until their
separate authorized provider certifications are implemented.
<!-- profile:object-storage:end -->
