// Package domain is reserved for small, stable contracts shared across app
// packages when a consumer-owned interface is no longer enough.
//
// Keep this package empty until a real shared domain contract exists. Prefer an
// interface beside internal/app/<feature> first, and let concrete adapters live
// under internal/infra.
package domain
