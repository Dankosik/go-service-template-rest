package config

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	outboundAuthClientAuthentication = "client_secret_basic"
	outboundAuthExternalHTTPS        = "external_https"
	outboundAuthPrivateHTTPS         = "private_https"
)

var outboundDependencyPattern = regexp.MustCompile(`^[a-z][a-z0-9._-]*$`)

// OutboundAuthConfig is the immutable source representation for one outbound
// OAuth client and resource authority.
type OutboundAuthConfig struct {
	Dependency             string        `koanf:"dependency"`
	ClientID               string        `koanf:"client_id"`
	ClientSecret           string        `koanf:"client_secret"`
	ClientAuthentication   string        `koanf:"client_authentication"`
	TokenEndpoint          string        `koanf:"token_endpoint"`
	TokenTargetClass       string        `koanf:"token_target_class"`
	TokenPrivateHostSuffix string        `koanf:"token_private_host_suffix"`
	Scopes                 string        `koanf:"scopes"`
	Resource               string        `koanf:"resource"`
	Audience               string        `koanf:"audience"`
	ResourceAuthority      string        `koanf:"resource_authority"`
	AcquisitionTimeout     time.Duration `koanf:"acquisition_timeout"`
}

func outboundAuthDefaults() map[string]any {
	return map[string]any{
		"outbound_auth.dependency":                "",
		"outbound_auth.client_id":                 "",
		"outbound_auth.client_secret":             "",
		"outbound_auth.client_authentication":     "",
		"outbound_auth.token_endpoint":            "",
		"outbound_auth.token_target_class":        "",
		"outbound_auth.token_private_host_suffix": "",
		"outbound_auth.scopes":                    "",
		"outbound_auth.resource":                  "",
		"outbound_auth.audience":                  "",
		"outbound_auth.resource_authority":        "",
		"outbound_auth.acquisition_timeout":       "0s",
	}
}

func validateOutboundAuthConfig(cfg *OutboundAuthConfig) error {
	cfg.Dependency = strings.TrimSpace(cfg.Dependency)
	cfg.TokenEndpoint = strings.TrimSpace(cfg.TokenEndpoint)
	cfg.TokenTargetClass = strings.TrimSpace(cfg.TokenTargetClass)
	cfg.TokenPrivateHostSuffix = outboundAuthPrivateSuffix(cfg.TokenPrivateHostSuffix)
	cfg.Scopes = strings.TrimSpace(cfg.Scopes)
	cfg.ResourceAuthority = strings.TrimSpace(cfg.ResourceAuthority)

	if !validOutboundDependency(cfg.Dependency) {
		return fmt.Errorf("%w: outbound_auth.dependency is invalid", ErrValidate)
	}
	if !validOutboundExactValue(cfg.ClientID, 512, false) {
		return fmt.Errorf("%w: outbound_auth.client_id is invalid", ErrValidate)
	}
	if !validOutboundExactValue(cfg.ClientSecret, 4096, true) {
		return fmt.Errorf("%w: outbound_auth.client_secret is invalid", ErrValidate)
	}
	if cfg.ClientAuthentication != outboundAuthClientAuthentication {
		return fmt.Errorf("%w: outbound_auth.client_authentication must be client_secret_basic", ErrValidate)
	}
	endpoint, err := canonicalOutboundHTTPSURL(cfg.TokenEndpoint, true)
	if err != nil {
		return fmt.Errorf("%w: outbound_auth.token_endpoint is invalid", ErrValidate)
	}
	cfg.TokenEndpoint = endpoint

	if err := validateOutboundTokenTarget(cfg.TokenTargetClass, cfg.TokenPrivateHostSuffix); err != nil {
		return err
	}

	scopes, err := canonicalOutboundScopes(cfg.Scopes)
	if err != nil {
		return fmt.Errorf("%w: outbound_auth.scopes is invalid", ErrValidate)
	}
	cfg.Scopes = strings.Join(scopes, " ")
	if cfg.Resource != "" && cfg.Audience != "" {
		return fmt.Errorf("%w: outbound_auth.resource and outbound_auth.audience are mutually exclusive", ErrValidate)
	}
	if cfg.Resource != "" && !validOutboundAbsoluteURI(cfg.Resource) {
		return fmt.Errorf("%w: outbound_auth.resource is invalid", ErrValidate)
	}
	if cfg.Audience != "" && !validOutboundExactValue(cfg.Audience, 2048, false) {
		return fmt.Errorf("%w: outbound_auth.audience is invalid", ErrValidate)
	}
	authority, err := canonicalOutboundHTTPSURL(cfg.ResourceAuthority, false)
	if err != nil {
		return fmt.Errorf("%w: outbound_auth.resource_authority is invalid", ErrValidate)
	}
	cfg.ResourceAuthority = authority
	if cfg.AcquisitionTimeout < 100*time.Millisecond || cfg.AcquisitionTimeout > 30*time.Second {
		return fmt.Errorf("%w: outbound_auth.acquisition_timeout must be in range [100ms,30s]", ErrValidate)
	}
	return nil
}

func validOutboundDependency(value string) bool {
	return len(value) <= 64 && outboundDependencyPattern.MatchString(value)
}

func validOutboundExactValue(value string, maximum int, nonBlank bool) bool {
	if value == "" || len(value) > maximum || !utf8.ValidString(value) || nonBlank && strings.TrimSpace(value) == "" {
		return false
	}
	return !strings.ContainsFunc(value, unicode.IsControl)
}

func canonicalOutboundHTTPSURL(raw string, allowPath bool) (string, error) {
	if raw == "" || len(raw) > 2048 {
		return "", ErrValidate
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed == nil || !parsed.IsAbs() || parsed.Opaque != "" || parsed.Host == "" || parsed.Hostname() == "" ||
		!strings.EqualFold(parsed.Scheme, "https") || parsed.User != nil || parsed.RawQuery != "" || parsed.ForceQuery || parsed.Fragment != "" {
		return "", ErrValidate
	}
	if !allowPath && parsed.Path != "" && parsed.Path != "/" {
		return "", ErrValidate
	}
	parsed.Scheme = "https"
	parsed.Host = strings.ToLower(parsed.Host)
	if !allowPath {
		parsed.Path = ""
		parsed.RawPath = ""
	}
	return parsed.String(), nil
}

func outboundAuthPrivateSuffix(raw string) string {
	suffix := strings.ToLower(strings.TrimSpace(raw))
	suffix = strings.TrimSuffix(suffix, ".")
	return strings.TrimPrefix(suffix, ".")
}

func validOutboundPrivateSuffix(suffix string) bool {
	if suffix == "" || len(suffix) > 253 {
		return false
	}
	for label := range strings.SplitSeq(suffix, ".") {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
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

func canonicalOutboundScopes(raw string) ([]string, error) {
	for _, char := range raw {
		if unicode.IsSpace(char) && char != ' ' {
			return nil, ErrValidate
		}
	}
	values := strings.Fields(raw)
	if len(values) > 32 {
		return nil, ErrValidate
	}
	seen := make(map[string]struct{}, len(values))
	for _, scope := range values {
		if scope == "" || len(scope) > 256 || !validOutboundScopeToken(scope) {
			return nil, ErrValidate
		}
		if _, duplicate := seen[scope]; duplicate {
			return nil, ErrValidate
		}
		seen[scope] = struct{}{}
	}
	slices.Sort(values)
	if len(strings.Join(values, " ")) > 4096 {
		return nil, ErrValidate
	}
	return values, nil
}

func validOutboundScopeToken(scope string) bool {
	for _, char := range []byte(scope) {
		if char == 0x21 || char >= 0x23 && char <= 0x5b || char >= 0x5d && char <= 0x7e {
			continue
		}
		return false
	}
	return true
}

func validOutboundAbsoluteURI(raw string) bool {
	if raw == "" || len(raw) > 2048 || !utf8.ValidString(raw) {
		return false
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed != nil && parsed.IsAbs() && parsed.User == nil && parsed.Fragment == ""
}

func validateOutboundTokenTarget(class, privateSuffix string) error {
	switch class {
	case outboundAuthExternalHTTPS:
		if privateSuffix != "" {
			return fmt.Errorf("%w: outbound_auth.token_private_host_suffix must be empty for external_https", ErrValidate)
		}
	case outboundAuthPrivateHTTPS:
		if !validOutboundPrivateSuffix(privateSuffix) {
			return fmt.Errorf("%w: outbound_auth.token_private_host_suffix is invalid", ErrValidate)
		}
	default:
		return fmt.Errorf("%w: outbound_auth.token_target_class is invalid", ErrValidate)
	}
	return nil
}
