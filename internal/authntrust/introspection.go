package authntrust

const (
	TargetClassExternalHTTPS = "external-https"
	TargetClassPrivateHTTPS  = "private-https"
)

// ValidIntrospectionEndpoint reports whether raw is a configured RFC 7662
// endpoint this service may call. The path may be present; user information,
// query, forced query, and fragment are forbidden so the outbound request has
// one exact destination.
func ValidIntrospectionEndpoint(raw string) bool {
	parsed, ok := validHTTPSURL(raw)
	return ok && parsed.RawQuery == "" && !parsed.ForceQuery
}

// ValidIntrospectionTargetClass reports whether raw is one of the two exact
// fixed-authority classes. There is no inferred default.
func ValidIntrospectionTargetClass(raw string) bool {
	return raw == TargetClassExternalHTTPS || raw == TargetClassPrivateHTTPS
}
