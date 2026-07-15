# Reference Selector

| Pressure | Load |
| --- | --- |
| Error wrapping, identity, inspection, cancellation mapping, or hidden success. | [errors-and-contracts-review.md](errors-and-contracts-review.md) |
| Stored, nil, replaced, omitted, derived, or uncancelled context. | [context-and-lifetime-review.md](context-and-lifetime-review.md) |
| Receivers, method sets, interface satisfaction, value copies, sync fields, buffers, or pointer-to-container shapes. | [receivers-methodsets-and-copy-safety.md](receivers-methodsets-and-copy-safety.md) |
| Typed nil, nil containers/channels, constructors, zero values, or nil-versus-empty behavior. | [nil-zero-value-and-typed-nil.md](nil-zero-value-and-typed-nil.md) |
| Mutable containers, cloning, headers/URL values, aliasing, or map ordering. | [slices-maps-buffers-and-ownership.md](slices-maps-buffers-and-ownership.md) |
| Close/error probes, files, rows, scanner, body, cancel, timer/ticker, partial reads, or defer scope. | [resource-closure-and-iteration-probes.md](resource-closure-and-iteration-probes.md) |
| A local helper may duplicate current builtins or stdlib. | [stdlib-first-modern-go-review.md](stdlib-first-modern-go-review.md) |
| Exported names, docs, packages, constructors, interfaces, options, signatures, or compatibility. | [exported-api-and-interface-shape.md](exported-api-and-interface-shape.md) |

When pressures overlap, choose by the violated contract: context lifetime versus error inspectability, or mutable aliasing versus whether stdlib can replace the helper.
