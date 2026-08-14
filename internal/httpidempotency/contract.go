package httpidempotency

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"golang.org/x/net/http/httpguts"
)

// Header is the one request header an opted operation declares.
const Header = "Idempotency-Key"

// ExternalEffectDisposition declares how an operation handles any effect that
// cannot join its PostgreSQL transaction.
type ExternalEffectDisposition string

const (
	ExternalEffectNone                ExternalEffectDisposition = "none"
	ExternalEffectTransactionalOutbox ExternalEffectDisposition = "transactional_outbox"
	ExternalEffectDownstreamKey       ExternalEffectDisposition = "downstream_idempotency"
	ExternalEffectReconciliation      ExternalEffectDisposition = "reconciliation"
	ExternalEffectCompensation        ExternalEffectDisposition = "compensation"
)

// DuplicateRiskPolicy declares how long an operation must retain its duplicate
// guard after replay material may expire.
type DuplicateRiskPolicy struct {
	Duration  time.Duration
	Permanent bool
}

// Contract is the complete static declaration required before an operation can
// opt into idempotency.
type Contract struct {
	OperationID         string
	APIVersion          string
	KeyMaxBytes         int
	FingerprintVersions []string
	ResultCodecs        []string
	ReplayStatuses      []int
	StableHeaders       []string
	ResultMaxBytes      int
	ReplayTTL           time.Duration
	DuplicateRisk       DuplicateRiskPolicy
	InProgressWait      time.Duration
	RetryAfter          time.Duration
	ExternalEffect      ExternalEffectDisposition
}

// Clone returns an independent declaration so later caller mutation cannot
// change a registration already accepted by the router.
func (c Contract) Clone() Contract {
	c.FingerprintVersions = slices.Clone(c.FingerprintVersions)
	c.ResultCodecs = slices.Clone(c.ResultCodecs)
	c.ReplayStatuses = slices.Clone(c.ReplayStatuses)
	c.StableHeaders = slices.Clone(c.StableHeaders)
	return c
}

// Validate rejects an incomplete declaration instead of supplying template
// defaults for an operation it cannot understand.
func (c Contract) Validate() error {
	if strings.TrimSpace(c.OperationID) == "" {
		return errors.New("idempotency contract: operation ID is required")
	}
	if strings.TrimSpace(c.APIVersion) == "" {
		return errors.New("idempotency contract: API version is required")
	}
	if c.KeyMaxBytes <= 0 {
		return errors.New("idempotency contract: key max bytes must be positive")
	}
	if err := validateNames("fingerprint version", c.FingerprintVersions); err != nil {
		return err
	}
	if err := validateNames("result codec", c.ResultCodecs); err != nil {
		return err
	}
	if err := validateReplayStatuses(c.ReplayStatuses); err != nil {
		return err
	}
	if err := validateStableHeaders(c.StableHeaders); err != nil {
		return err
	}
	if c.ResultMaxBytes <= 0 {
		return errors.New("idempotency contract: result max bytes must be positive")
	}
	if err := validateDurations(c); err != nil {
		return err
	}
	switch c.ExternalEffect {
	case ExternalEffectNone, ExternalEffectTransactionalOutbox, ExternalEffectDownstreamKey, ExternalEffectReconciliation, ExternalEffectCompensation:
		return nil
	default:
		return fmt.Errorf("idempotency contract: external effect disposition %q is invalid", c.ExternalEffect)
	}
}

func validateReplayStatuses(statuses []int) error {
	if len(statuses) == 0 {
		return errors.New("idempotency contract: replay statuses are required")
	}
	seen := make(map[int]struct{}, len(statuses))
	for _, status := range statuses {
		if status < 200 || status >= 300 {
			return fmt.Errorf("idempotency contract: replay status %d is not a success", status)
		}
		if _, duplicate := seen[status]; duplicate {
			return fmt.Errorf("idempotency contract: replay status %d is duplicated", status)
		}
		seen[status] = struct{}{}
	}
	return nil
}

func validateDurations(contract Contract) error {
	if contract.ReplayTTL <= 0 || contract.InProgressWait <= 0 || contract.RetryAfter <= 0 {
		return errors.New("idempotency contract: replay, in-progress, and retry durations must be positive")
	}
	if contract.RetryAfter%time.Second != 0 {
		return errors.New("idempotency contract: retry after must be whole seconds")
	}
	if contract.DuplicateRisk.Permanent {
		if contract.DuplicateRisk.Duration != 0 {
			return errors.New("idempotency contract: permanent duplicate risk has a duration")
		}
		return nil
	}
	if contract.DuplicateRisk.Duration < contract.ReplayTTL {
		return errors.New("idempotency contract: finite duplicate risk must not precede replay TTL")
	}
	return nil
}

func validateNames(kind string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("idempotency contract: %s is required", kind)
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("idempotency contract: %s is blank", kind)
		}
		if _, duplicate := seen[value]; duplicate {
			return fmt.Errorf("idempotency contract: %s %q is duplicated", kind, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateStableHeaders(headers []string) error {
	seen := make(map[string]struct{}, len(headers))
	for _, header := range headers {
		if header != strings.ToLower(header) || !validToken(header) {
			return fmt.Errorf("idempotency contract: stable header %q must be a lowercase token", header)
		}
		if forbiddenResultHeader(header) {
			return fmt.Errorf("idempotency contract: stable header %q is forbidden", header)
		}
		if _, duplicate := seen[header]; duplicate {
			return fmt.Errorf("idempotency contract: stable header %q is duplicated", header)
		}
		seen[header] = struct{}{}
	}
	return nil
}

func validToken(value string) bool {
	return httpguts.ValidHeaderFieldName(value)
}

func forbiddenResultHeader(header string) bool {
	switch header {
	case "connection", "keep-alive", "proxy-authenticate", "proxy-authorization", "te", "trailer", "transfer-encoding", "upgrade", "authorization", "cookie", "set-cookie", "retry-after", "date", "x-request-id", "traceparent", "tracestate":
		return true
	default:
		return false
	}
}
