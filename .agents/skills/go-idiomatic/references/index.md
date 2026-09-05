# Reference Selector

`make lint` already decides the mechanical half of every row below — `contextcheck`, `containedctx`, `lostcancel`, `bodyclose`, `rowserrcheck`, `errcheck`, `errorlint`, `nilnil`, `nilerr`, `copylocks`, `recvcheck`, `ireturn`, and `modernize` among them. Load a reference for the contract those checks cannot see.

| Pressure | Load | Required effect |
| --- | --- | --- |
| An error is created, wrapped, joined, logged, classified, used as a cancellation cause, or recovered from panic. | [error-contracts.md](error-contracts.md) | Assign one handling owner and decide which identity or type remains caller-observable, rather than reading a satisfied mechanical check as a sound contract. |
| Detached or background work, resources opened in a loop, completion probes, partial reads, or `defer` scope. | [lifetime-and-release.md](lifetime-and-release.md) | Separate release from completion and name the scope that owns each, rather than treating a present `Close` as proof. |
| Mutable slices, maps, headers, or buffers crossing a boundary; clone depth; observable map order; copied values. | [aliasing-and-ownership.md](aliasing-and-ownership.md) | Trace who can still write through the backing store, rather than reading a clone call as isolation. |
| Typed nil, zero-value usability, nil-versus-empty at an observer, or interface satisfaction through a stored value. | [nil-zero-and-method-sets.md](nil-zero-and-method-sets.md) | Check what the caller observes at the boundary, rather than what the code says it returns. |

When pressures overlap, choose by the violated contract: error identity versus release scope, or mutation authority versus observable absence.
