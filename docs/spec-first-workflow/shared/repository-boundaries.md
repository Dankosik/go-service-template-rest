# Repository Boundaries

## Read When

- An accepted change, build, or fix outcome first requires inspection or edits
  in another checkout.

## Method

An accepted outcome may cross into an available neighboring repository without
a second approval solely because ownership crosses a checkout. Before editing,
load that repository's instructions, inspect its dirty state, preserve unrelated
changes, and validate every changed repository.

Treat the neighbor as an external blocker only when it is unavailable,
read-only, outside the accepted outcome, or the required action needs authority
the request did not grant. When several deployables or managed dependencies are
affected, System Design's release-closure contract owns the deployment graph.

## Stop Rule

Every changed repository has an explicit owner and matching proof; otherwise
narrow the completion claim to the proved checkout and name the blocker.
