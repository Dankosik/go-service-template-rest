package authntrust

import "strings"

const (
	TargetClassExternalHTTPS = "external-https"
	TargetClassPrivateHTTPS  = "private-https"
)

// IntrospectionTargetIssue identifies which class and private-suffix rule a
// caller should report without coupling this leaf package to its error format.
type IntrospectionTargetIssue uint8

const (
	IntrospectionTargetValid IntrospectionTargetIssue = iota
	IntrospectionTargetClassInvalid
	IntrospectionPrivateSuffixRequired
	IntrospectionPrivateSuffixForbidden
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

// IntrospectionTargetPolicyIssue applies the shared class and suffix
// admissibility rules. It deliberately does not trim either value: the class
// is exact, and suffix whitespace is significant except that whitespace-only
// is empty for the private class.
func IntrospectionTargetPolicyIssue(targetClass, privateSuffix string) IntrospectionTargetIssue {
	if !ValidIntrospectionTargetClass(targetClass) {
		return IntrospectionTargetClassInvalid
	}
	if targetClass == TargetClassPrivateHTTPS {
		if strings.TrimSpace(privateSuffix) == "" {
			return IntrospectionPrivateSuffixRequired
		}
	} else if privateSuffix != "" {
		return IntrospectionPrivateSuffixForbidden
	}
	return IntrospectionTargetValid
}
