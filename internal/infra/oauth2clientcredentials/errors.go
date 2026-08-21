package oauth2clientcredentials

import "errors"

var (
	// ErrInvalidConfiguration reports a local construction or competing-auth defect.
	ErrInvalidConfiguration = errors.New("outbound authentication configuration is invalid")
	// ErrUnavailable is the only provider failure exposed outside this package.
	ErrUnavailable = errors.New("outbound authentication is unavailable")
)
