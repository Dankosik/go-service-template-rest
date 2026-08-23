// Package oauthintrospection is the RFC 7662 trust engine for the shared
// bearer runtime.
//
// It owns one immutable endpoint, one client_secret_basic POST per accepted
// credential, strict response admission, and idle-connection close. It does not
// cache, retry, follow redirects, honor a proxy, discover metadata, or authorize
// roles, scopes, or tenants. Inbound bearer grammar, transport adapters, and
// the shared verification counter live in internal/infra/bearerauthn.
package oauthintrospection
