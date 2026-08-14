package oauth2clientcredentials

import (
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/example/go-service-template-rest/internal/infra/httpclient"
)

const (
	clientAuthenticationBasic = "client_secret_basic"
	earlyExpiryMargin         = 10 * time.Second
	maxClientIDBytes          = 512
	maxClientSecretBytes      = 4096
	maxEndpointBytes          = 2048
	maxScopes                 = 32
	maxScopeBytes             = 256
	maxJoinedScopeBytes       = 4096
	maxResourceBytes          = 2048
	maxAudienceBytes          = 2048
	minAcquisitionTimeout     = 100 * time.Millisecond
	maxAcquisitionTimeout     = 30 * time.Second
)

var dependencyNamePattern = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)

// Config is the immutable binding for one OAuth client and one resource
// authority.
type Config struct {
	DependencyName         string
	ClientID               string
	ClientSecret           string
	ClientAuthentication   string
	TokenEndpoint          string
	TokenTargetClass       httpclient.TargetClass
	TokenPrivateHostSuffix string
	Scopes                 []string
	Resource               string
	Audience               string
	ResourceAuthority      string
	AcquisitionTimeout     time.Duration
}

func validateConfig(cfg Config) (Config, error) {
	cfg.DependencyName = strings.TrimSpace(cfg.DependencyName)
	cfg.TokenEndpoint = strings.TrimSpace(cfg.TokenEndpoint)
	cfg.TokenPrivateHostSuffix = canonicalPrivateSuffix(cfg.TokenPrivateHostSuffix)
	cfg.ResourceAuthority = strings.TrimSpace(cfg.ResourceAuthority)

	if !validDependencyName(cfg.DependencyName) ||
		!validExactValue(cfg.ClientID, maxClientIDBytes, false) ||
		!validExactValue(cfg.ClientSecret, maxClientSecretBytes, true) ||
		cfg.ClientAuthentication != clientAuthenticationBasic ||
		cfg.AcquisitionTimeout < minAcquisitionTimeout ||
		cfg.AcquisitionTimeout > maxAcquisitionTimeout {
		return Config{}, failure(FailureInvalidConfiguration)
	}

	tokenEndpoint, err := canonicalHTTPSURL(cfg.TokenEndpoint, true)
	if err != nil {
		return Config{}, failure(FailureInvalidConfiguration)
	}
	cfg.TokenEndpoint = tokenEndpoint

	if !validTokenTarget(cfg.TokenTargetClass, cfg.TokenPrivateHostSuffix) {
		return Config{}, failure(FailureInvalidConfiguration)
	}

	scopes, err := canonicalScopes(cfg.Scopes)
	if err != nil {
		return Config{}, failure(FailureInvalidConfiguration)
	}
	cfg.Scopes = scopes

	if cfg.Resource != "" && cfg.Audience != "" {
		return Config{}, failure(FailureInvalidConfiguration)
	}
	if cfg.Resource != "" && !validAbsoluteURI(cfg.Resource) {
		return Config{}, failure(FailureInvalidConfiguration)
	}
	if cfg.Audience != "" && !validExactValue(cfg.Audience, maxAudienceBytes, false) {
		return Config{}, failure(FailureInvalidConfiguration)
	}
	resourceAuthority, err := canonicalHTTPSURL(cfg.ResourceAuthority, false)
	if err != nil {
		return Config{}, failure(FailureInvalidConfiguration)
	}
	cfg.ResourceAuthority = resourceAuthority
	return cfg, nil
}

func validDependencyName(value string) bool {
	return len(value) <= maxDependencyNameBytes && dependencyNamePattern.MatchString(value)
}

func validExactValue(value string, maximum int, nonBlank bool) bool {
	if value == "" || len(value) > maximum || !utf8.ValidString(value) || nonBlank && strings.TrimSpace(value) == "" {
		return false
	}
	return !strings.ContainsFunc(value, unicode.IsControl)
}

func canonicalHTTPSURL(raw string, allowPath bool) (string, error) {
	if raw == "" || len(raw) > maxEndpointBytes {
		return "", failure(FailureInvalidConfiguration)
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.Opaque != "" || parsed.Host == "" || parsed.Hostname() == "" ||
		!strings.EqualFold(parsed.Scheme, "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", failure(FailureInvalidConfiguration)
	}
	if !allowPath && parsed.Path != "" && parsed.Path != "/" {
		return "", failure(FailureInvalidConfiguration)
	}
	parsed.Scheme = "https"
	parsed.Host = strings.ToLower(parsed.Host)
	if !allowPath {
		parsed.Path = ""
		parsed.RawPath = ""
	}
	return parsed.String(), nil
}

func canonicalPrivateSuffix(raw string) string {
	suffix := strings.ToLower(strings.TrimSpace(raw))
	suffix = strings.TrimSuffix(suffix, ".")
	return strings.TrimPrefix(suffix, ".")
}

func validPrivateSuffix(suffix string) bool {
	if suffix == "" || len(suffix) > 253 {
		return false
	}
	for label := range strings.SplitSeq(suffix, ".") {
		if label == "" || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
		for _, char := range label {
			if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func canonicalScopes(values []string) ([]string, error) {
	if len(values) > maxScopes {
		return nil, failure(FailureInvalidConfiguration)
	}
	canonical := slices.Clone(values)
	seen := make(map[string]struct{}, len(canonical))
	for _, scope := range canonical {
		if scope == "" || len(scope) > maxScopeBytes || !validScopeToken(scope) {
			return nil, failure(FailureInvalidConfiguration)
		}
		if _, exists := seen[scope]; exists {
			return nil, failure(FailureInvalidConfiguration)
		}
		seen[scope] = struct{}{}
	}
	slices.Sort(canonical)
	if len(strings.Join(canonical, " ")) > maxJoinedScopeBytes {
		return nil, failure(FailureInvalidConfiguration)
	}
	return canonical, nil
}

func validScopeToken(scope string) bool {
	for _, char := range []byte(scope) {
		if char == 0x21 || char >= 0x23 && char <= 0x5b || char >= 0x5d && char <= 0x7e {
			continue
		}
		return false
	}
	return true
}

func validAbsoluteURI(raw string) bool {
	if raw == "" || len(raw) > maxResourceBytes || !utf8.ValidString(raw) {
		return false
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed != nil && parsed.IsAbs() && parsed.User == nil && parsed.Fragment == ""
}

func validTokenTarget(class httpclient.TargetClass, privateSuffix string) bool {
	switch class {
	case httpclient.ExternalHTTPS:
		return privateSuffix == ""
	case httpclient.PrivateHTTPS:
		return validPrivateSuffix(privateSuffix)
	case httpclient.PrivateHTTP:
		return false
	}
	return false
}
