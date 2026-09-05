# Nil, Zero Values, And Method Sets

## Load When
Load when a Go review touches an interface return on a disabled, empty, or error path, a zero value used before its constructor, a nil map or channel write, nil-versus-empty in a serialized field, or a value stored where only its pointer implements an interface.

## Decide
- A nil concrete pointer stored in an interface is not a nil interface, and no configured linter catches it. On a disabled or absent path, return a real nil interface, a no-op implementation, or an explicit presence result.
- Exported types invite `var t T`. Where the zero value cannot serve, either make the first method work on it or make construction part of the documented contract; a write to a nil map panics on the first call.
- nil versus empty is a finding only where it is observable: a serialized field, an explicit nil check, an equality, a documented contract. Turning `[]string{}` into `nil` turns a JSON `[]` into `null` for every existing client, and `len` and `range` behaving the same is not evidence of safety.
- `recvcheck` gates mixed receivers on a type; what it does not gate is reachability. A value stored in an interface-typed field satisfies only the method set of `T`, so a cleanup or lifecycle method declared on `*T` becomes uncallable through that contract.
- Reflection-based nil probes preserve the confusing contract they inspect. Change the signature that made absence ambiguous instead.

## Inspect
`var e *EmailNotifier; return e` from a constructor returning `Notifier` — the caller's `notifier != nil` is true and the disabled path panics on first use.

## Reject
- "Nil and empty slices are the same" — true inside Go, false at every encoding and equality boundary.
- "Call the constructor" — an exported type whose zero value panics has a contract problem the caller cannot see from the signature.

## Reopen
- Name where absence becomes observable — a wire format, a nil check, an interface comparison — before reporting nil-versus-empty.
- Prove which type stops satisfying which interface at which storage site before reporting a method-set finding.
- Leave payload-shape policy and optional-dependency architecture to their own lanes.

## Prove
- Compare the returned interface to nil directly on the absent path.
- Exercise the zero value of an exported type through its first mutating method.
- Golden-test the serialized form of a nil-versus-empty field.
