# Collection Transformations

For pure generic slice or map transformations, use a clear one-call standard
library operation first, then the already-installed `github.com/samber/lo` when
it removes real loop noise. Do not add local generic helpers or wrappers around
`lo`.

Keep domain policy, errors, lifecycle, concurrency, security, and transactions
in explicit local Go; those are not collection transformations.
