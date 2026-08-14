// Package oauth2clientcredentials owns one fixed OAuth 2.0 client-credentials
// transaction and its sanitized failure contract.
//
// The package deliberately exposes no token source, provider interface,
// registry, discovery, refresh, or retry seam. It exposes one process
// credential owner and its fixed HTTP and gRPC adapters.
package oauth2clientcredentials
