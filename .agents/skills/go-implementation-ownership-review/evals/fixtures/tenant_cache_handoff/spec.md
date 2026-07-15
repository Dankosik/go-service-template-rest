# Account reader ownership

- `AccountReader` owns tenant-scoped lookup construction and the cache/repository read sequence.
- Every cache key and repository lookup must include the authenticated tenant ID and account ID.
- A timeout must remain an error unless an accepted reliability decision explicitly defines a fallback and its freshness bound.
- Security, cache, and reliability reviewers own their deep mechanics; this artifact fixes the implementation seam they must preserve.
