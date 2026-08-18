package postgreswebhook

import (
	"cmp"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"mime"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	MaxPayloadBytes          = 256 << 10
	MaxDestinationBytes      = 8 << 10
	MaxFanoutMembers         = 1000
	MaxPolicyListItems       = 64
	MaxDNSAddresses          = 64
	MaxSecretManifestBytes   = 1 << 20
	MaxSecretManifestEntries = 4096
	MaxAcceptanceBytes       = MaxPayloadBytes + MaxFanoutMembers*MaxDestinationBytes + 64<<10
	MaxAttempts              = 100
	MaxConcurrency           = 256
	MaxClaimScanPage         = 256
	MaxStoreOperationTime    = 30 * time.Second
	MaxResponseBytes         = 1 << 20
	MaxAttemptTime           = 10 * time.Minute
	MaxDrainTime             = 30 * time.Minute
	MaxDeliveryTime          = 365 * 24 * time.Hour
	MaxRetentionTime         = 10 * 365 * 24 * time.Hour
)

//nolint:recvcheck // JSON marshaling needs a value receiver; unmarshaling must mutate the value.
type RetentionHorizons struct {
	Payload               time.Duration
	Active                time.Duration
	TerminalSummary       time.Duration
	Attempt               time.Duration
	Action                time.Duration
	DestinationGeneration time.Duration
	RedriveEligibility    time.Duration
	ReceiverDedup         time.Duration
}

type retentionHorizonValues [8]time.Duration

func (h RetentionHorizons) values() retentionHorizonValues {
	return retentionHorizonValues{h.Payload, h.Active, h.TerminalSummary, h.Attempt, h.Action, h.DestinationGeneration, h.RedriveEligibility, h.ReceiverDedup}
}

func (h RetentionHorizons) MarshalJSON() ([]byte, error) {
	encoded, err := json.Marshal(h.values())
	if err != nil {
		return nil, fmt.Errorf("encode retention horizons: %w", err)
	}
	return encoded, nil
}

func (h *RetentionHorizons) UnmarshalJSON(data []byte) error {
	var values []time.Duration
	if err := json.Unmarshal(data, &values); err != nil {
		return fmt.Errorf("decode retention horizons: %w", err)
	}
	if len(values) != 8 {
		return fmt.Errorf("retention horizons: got %d values, want 8", len(values))
	}
	*h = RetentionHorizons{Payload: values[0], Active: values[1], TerminalSummary: values[2], Attempt: values[3], Action: values[4], DestinationGeneration: values[5], RedriveEligibility: values[6], ReceiverDedup: values[7]}
	return nil
}

type DeliveryPolicy struct {
	MaximumPayloadBytes     int               `json:"maximum_payload_bytes"`
	AcceptedContentTypes    []string          `json:"accepted_content_types"`
	AcceptedBusinessSchemas []string          `json:"accepted_business_schemas"`
	MaximumAttempts         int               `json:"maximum_attempts"`
	MaximumDeliveryAge      time.Duration     `json:"maximum_delivery_age"`
	BackoffBase             time.Duration     `json:"backoff_base"`
	BackoffCap              time.Duration     `json:"backoff_cap"`
	RetryAfterCap           time.Duration     `json:"retry_after_cap"`
	AttemptTimeout          time.Duration     `json:"attempt_timeout"`
	ResponseHeaderTimeout   time.Duration     `json:"response_header_timeout"`
	ResponseHeaderBytes     int               `json:"response_header_bytes"`
	ResponseBodyBytes       int               `json:"response_body_bytes"`
	MinimumTLSVersion       string            `json:"minimum_tls_version,omitempty"`
	DestinationConcurrency  int               `json:"destination_concurrency"`
	GlobalConcurrency       int               `json:"global_concurrency"`
	DrainTimeout            time.Duration     `json:"drain_timeout"`
	RedriveAttempts         int               `json:"redrive_attempts"`
	RedriveAge              time.Duration     `json:"redrive_age"`
	Horizons                RetentionHorizons `json:"horizons"`
	AutomaticPause          bool              `json:"automatic_pause"`
	AutomaticPauseClasses   []string          `json:"automatic_pause_classes"`
	PauseWindow             time.Duration     `json:"pause_window"`
	PauseDuration           time.Duration     `json:"pause_duration"`
	PauseThreshold          int               `json:"pause_threshold"`
	PauseMinimumTraffic     int               `json:"pause_minimum_traffic"`
	PauseManualOnly         bool              `json:"pause_manual_only"`
	PauseRetentionEffect    string            `json:"pause_retention_effect"`
	PauseAlertPolicy        string            `json:"pause_alert_policy"`
}

type DestinationSnapshot struct {
	DestinationID                string
	Generation                   int64
	OwnershipVerificationReceipt string
	URL                          string
	SelectionRevision            string
	PayloadVersionPreference     string
	SignatureProfile             string
	SigningAuthorityBinding      string
	Policy                       DeliveryPolicy
}

type Acceptance struct {
	OwnerScope               string
	AcceptanceID             string
	BusinessEventID          string
	FanoutSnapshotID         string
	EventType                string
	BusinessSchemaVersion    string
	ContentType              string
	Body                     []byte
	DeliveryEnvelopeVersion  string
	SubscriberPolicyRevision string
	Destinations             []DestinationSnapshot
}

type PreparedDestination struct {
	DestinationSnapshot

	DeliveryID string
}

type PreparedAcceptance struct {
	Acceptance     Acceptance
	Destinations   []PreparedDestination
	CanonicalBytes []byte
	Fingerprint    [32]byte
}

func PrepareAcceptance(input Acceptance) (PreparedAcceptance, error) {
	if err := validateAcceptance(input); err != nil {
		return PreparedAcceptance{}, err
	}
	encodedDestinations := make([][]byte, 0, len(input.Destinations))
	destinations := make([]PreparedDestination, 0, len(input.Destinations))
	for _, destination := range input.Destinations {
		destination = cloneDestination(destination)
		encoded, err := encodeDestinationIntent(destination)
		if err != nil {
			return PreparedAcceptance{}, err
		}
		if len(encoded) > MaxDestinationBytes {
			return PreparedAcceptance{}, fmt.Errorf("%w: destination intent exceeds %d bytes", ErrConfig, MaxDestinationBytes)
		}
		encodedDestinations = append(encodedDestinations, encoded)
		destinations = append(destinations, PreparedDestination{DestinationSnapshot: destination})
	}
	list, err := canonicalList(encodedDestinations)
	if err != nil {
		return PreparedAcceptance{}, fmt.Errorf("%w: destination set: %w", ErrConfig, err)
	}
	canonical, err := canonicalRecord("webhook-acceptance-intent-v2",
		[]byte(input.OwnerScope), []byte(input.AcceptanceID), []byte(input.BusinessEventID), []byte(input.FanoutSnapshotID),
		[]byte(input.EventType), []byte(input.BusinessSchemaVersion), []byte(input.ContentType), input.Body,
		[]byte(input.DeliveryEnvelopeVersion), []byte(input.SubscriberPolicyRevision), list,
	)
	if err != nil {
		return PreparedAcceptance{}, err
	}
	if len(canonical) > MaxAcceptanceBytes {
		return PreparedAcceptance{}, fmt.Errorf("%w: prepared acceptance exceeds %d bytes", ErrConfig, MaxAcceptanceBytes)
	}
	fingerprint := canonicalDigest(canonical)
	slices.SortFunc(destinations, func(a, b PreparedDestination) int {
		if order := strings.Compare(a.DestinationID, b.DestinationID); order != 0 {
			return order
		}
		return cmp.Compare(a.Generation, b.Generation)
	})
	for i := range destinations {
		deliveryID, err := deriveDeliveryID(fingerprint, destinations[i].DestinationSnapshot)
		if err != nil {
			return PreparedAcceptance{}, err
		}
		destinations[i].DeliveryID = deliveryID
	}
	cloned := input
	cloned.Body = slices.Clone(input.Body)
	cloned.Destinations = nil
	return PreparedAcceptance{Acceptance: cloned, Destinations: destinations, CanonicalBytes: canonical, Fingerprint: fingerprint}, nil
}

func deriveDeliveryID(intent [32]byte, destination DestinationSnapshot) (string, error) {
	canonical, err := canonicalRecord("webhook-delivery-id-v2", intent[:], []byte(destination.DestinationID), []byte(strconv.FormatInt(destination.Generation, 10)))
	if err != nil {
		return "", err
	}
	digest := canonicalDigest(canonical)
	return "whd_" + hex.EncodeToString(digest[:]), nil
}

func validateAcceptance(input Acceptance) error {
	for name, value := range map[string]string{
		"owner_scope": input.OwnerScope, "acceptance_id": input.AcceptanceID, "business_event_id": input.BusinessEventID,
		"fanout_snapshot_id": input.FanoutSnapshotID, "event_type": input.EventType, "business_schema_version": input.BusinessSchemaVersion,
		"content_type": input.ContentType, "delivery_envelope_version": input.DeliveryEnvelopeVersion,
		"subscriber_policy_revision": input.SubscriberPolicyRevision,
	} {
		if err := validateToken(name, value); err != nil {
			return err
		}
	}
	if len(input.Body) == 0 || len(input.Body) > MaxPayloadBytes {
		return fmt.Errorf("%w: body must be 1..%d bytes", ErrConfig, MaxPayloadBytes)
	}
	if len(input.Destinations) == 0 || len(input.Destinations) > MaxFanoutMembers {
		return fmt.Errorf("%w: destinations must be 1..%d", ErrConfig, MaxFanoutMembers)
	}
	seen := make(map[string]struct{}, len(input.Destinations))
	for _, destination := range input.Destinations {
		key := destination.DestinationID + "\x00" + strconv.FormatInt(destination.Generation, 10)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%w: duplicate destination generation", ErrConfig)
		}
		seen[key] = struct{}{}
		if err := validateDestination(destination); err != nil {
			return err
		}
		if len(input.Body) > destination.Policy.MaximumPayloadBytes || !acceptsContentType(destination.Policy.AcceptedContentTypes, input.ContentType) || !slices.Contains(destination.Policy.AcceptedBusinessSchemas, input.BusinessSchemaVersion) {
			return fmt.Errorf("%w: event is outside destination admission policy", ErrConfig)
		}
	}
	return nil
}

func validateDestination(destination DestinationSnapshot) error {
	for name, value := range map[string]string{"destination_id": destination.DestinationID, "ownership_receipt": destination.OwnershipVerificationReceipt, "selection_revision": destination.SelectionRevision, "payload_version": destination.PayloadVersionPreference, "signature_profile": destination.SignatureProfile, "signing_authority": destination.SigningAuthorityBinding} {
		if err := validateToken(name, value); err != nil {
			return err
		}
	}
	if destination.Generation <= 0 || destination.SignatureProfile != "v1" {
		return fmt.Errorf("%w: destination generation and v1 signature are required", ErrConfig)
	}
	parsed, err := url.Parse(destination.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Port() != "" && parsed.Port() != "443") || len(destination.URL) > 2048 {
		return fmt.Errorf("%w: destination URL must be absolute public HTTPS on port 443", ErrDestinationDenied)
	}
	return destination.Policy.validate()
}

//nolint:cyclop // The immutable policy relations are clearest as one fail-closed validator.
func (p DeliveryPolicy) validate() error {
	if p.MaximumPayloadBytes < 1 || p.MaximumPayloadBytes > MaxPayloadBytes || p.MaximumAttempts < 1 || p.MaximumAttempts > MaxAttempts || p.DestinationConcurrency < 1 || p.GlobalConcurrency < 1 || p.DestinationConcurrency > p.GlobalConcurrency || p.GlobalConcurrency > MaxConcurrency || p.RedriveAttempts < 1 || p.RedriveAttempts > MaxAttempts {
		return fmt.Errorf("%w: delivery policy count bound is invalid", ErrConfig)
	}
	for _, value := range []time.Duration{p.MaximumDeliveryAge, p.BackoffBase, p.BackoffCap, p.RetryAfterCap, p.AttemptTimeout, p.ResponseHeaderTimeout, p.DrainTimeout, p.RedriveAge} {
		if value <= 0 {
			return fmt.Errorf("%w: delivery policy duration must be positive", ErrConfig)
		}
	}
	if p.MaximumDeliveryAge > MaxDeliveryTime || p.BackoffCap > p.MaximumDeliveryAge || p.RetryAfterCap > p.MaximumDeliveryAge || p.RedriveAge > MaxDeliveryTime || p.AttemptTimeout > MaxAttemptTime || p.DrainTimeout > MaxDrainTime || p.BackoffBase > p.BackoffCap || p.AttemptTimeout >= p.DrainTimeout || p.ResponseHeaderTimeout > p.AttemptTimeout || p.ResponseHeaderBytes <= 0 || p.ResponseHeaderBytes > MaxResponseBytes || p.ResponseBodyBytes <= 0 || p.ResponseBodyBytes > MaxResponseBytes || len(p.AcceptedContentTypes) == 0 || len(p.AcceptedContentTypes) > MaxPolicyListItems || len(p.AcceptedBusinessSchemas) == 0 || len(p.AcceptedBusinessSchemas) > MaxPolicyListItems || len(p.AutomaticPauseClasses) > MaxPolicyListItems || p.MinimumTLSVersion != "" && p.MinimumTLSVersion != "1.2" && p.MinimumTLSVersion != "1.3" {
		return fmt.Errorf("%w: delivery policy relation is invalid", ErrConfig)
	}
	seenContentTypes := make(map[string]struct{}, len(p.AcceptedContentTypes))
	for _, value := range p.AcceptedContentTypes {
		canonical, err := canonicalContentType(value)
		if err != nil {
			return err
		}
		if _, duplicate := seenContentTypes[canonical]; duplicate {
			return fmt.Errorf("%w: duplicate accepted content type", ErrConfig)
		}
		seenContentTypes[canonical] = struct{}{}
	}
	seenSchemas := make(map[string]struct{}, len(p.AcceptedBusinessSchemas))
	for _, value := range p.AcceptedBusinessSchemas {
		if err := validateToken("accepted_business_schema", value); err != nil {
			return err
		}
		if _, duplicate := seenSchemas[value]; duplicate {
			return fmt.Errorf("%w: duplicate accepted business schema", ErrConfig)
		}
		seenSchemas[value] = struct{}{}
	}
	for _, horizon := range p.Horizons.values() {
		if horizon <= 0 || horizon > MaxRetentionTime {
			return fmt.Errorf("%w: retention horizon must be positive", ErrConfig)
		}
	}
	if p.Horizons.Active < max(p.MaximumDeliveryAge, p.Horizons.RedriveEligibility) ||
		p.Horizons.TerminalSummary < max(p.MaximumDeliveryAge, p.Horizons.RedriveEligibility) ||
		p.Horizons.Payload < max(p.MaximumDeliveryAge, p.Horizons.RedriveEligibility) ||
		p.Horizons.Attempt < p.MaximumDeliveryAge ||
		p.Horizons.Action < p.Horizons.RedriveEligibility ||
		p.Horizons.DestinationGeneration < max(p.MaximumDeliveryAge, p.Horizons.RedriveEligibility) ||
		p.Horizons.ReceiverDedup < max(p.MaximumDeliveryAge, p.Horizons.RedriveEligibility) {
		return fmt.Errorf("%w: retention horizons do not cover delivery and redrive authority", ErrConfig)
	}
	if p.AutomaticPause {
		return fmt.Errorf("%w: automatic pause is not implemented", ErrConfig)
	}
	if len(p.AutomaticPauseClasses) != 0 || p.PauseWindow != 0 || p.PauseDuration != 0 || p.PauseThreshold != 0 || p.PauseMinimumTraffic != 0 || p.PauseManualOnly || p.PauseRetentionEffect != "" || p.PauseAlertPolicy != "" {
		return fmt.Errorf("%w: disabled automatic pause has configuration", ErrConfig)
	}
	return nil
}

func acceptsContentType(accepted []string, value string) bool {
	canonical, err := canonicalContentType(value)
	if err != nil {
		return false
	}
	for _, candidate := range accepted {
		if normalized, candidateErr := canonicalContentType(candidate); candidateErr == nil && normalized == canonical {
			return true
		}
	}
	return false
}

func canonicalContentType(value string) (string, error) {
	mediaType, parameters, err := mime.ParseMediaType(value)
	if err != nil || len(value) > 256 {
		return "", fmt.Errorf("%w: content type is invalid", ErrConfig)
	}
	return mime.FormatMediaType(mediaType, parameters), nil
}

func encodeDestinationIntent(destination DestinationSnapshot) ([]byte, error) {
	policy, err := encodeDeliveryPolicy(destination.Policy)
	if err != nil {
		return nil, err
	}
	return canonicalRecord("webhook-destination-intent-v2", []byte(destination.DestinationID), []byte(strconv.FormatInt(destination.Generation, 10)), []byte(destination.OwnershipVerificationReceipt), []byte(destination.URL), []byte(destination.SelectionRevision), []byte(destination.PayloadVersionPreference), []byte(destination.SignatureProfile), []byte(destination.SigningAuthorityBinding), policy)
}

func encodeDeliveryPolicy(p DeliveryPolicy) ([]byte, error) {
	contents, err := stringList(p.AcceptedContentTypes)
	if err != nil {
		return nil, err
	}
	schemas, err := stringList(p.AcceptedBusinessSchemas)
	if err != nil {
		return nil, err
	}
	classes, err := stringList(p.AutomaticPauseClasses)
	if err != nil {
		return nil, err
	}
	fields := [][]byte{uintText(p.MaximumPayloadBytes), contents, schemas, uintText(p.MaximumAttempts), durationText(p.MaximumDeliveryAge), durationText(p.BackoffBase), durationText(p.BackoffCap), durationText(p.RetryAfterCap), durationText(p.AttemptTimeout), durationText(p.ResponseHeaderTimeout), uintText(p.ResponseHeaderBytes), uintText(p.ResponseBodyBytes), []byte(normalizedTLSVersion(p.MinimumTLSVersion)), uintText(p.DestinationConcurrency), uintText(p.GlobalConcurrency), durationText(p.DrainTimeout), uintText(p.RedriveAttempts), durationText(p.RedriveAge)}
	for _, horizon := range p.Horizons.values() {
		fields = append(fields, durationText(horizon))
	}
	fields = append(fields, boolText(p.AutomaticPause), classes, durationTextOptional(p.PauseWindow), uintTextOptional(p.PauseThreshold), uintTextOptional(p.PauseMinimumTraffic), durationTextOptional(p.PauseDuration), boolTextOptional(p.PauseManualOnly), []byte(p.PauseRetentionEffect), []byte(p.PauseAlertPolicy))
	return canonicalRecord("webhook-delivery-policy-v2", fields...)
}

func legacyAcceptanceFingerprint(prepared PreparedAcceptance) ([32]byte, error) {
	encodedDestinations := make([][]byte, 0, len(prepared.Destinations))
	for _, destination := range prepared.Destinations {
		policy, err := encodeLegacyDeliveryPolicy(destination.Policy)
		if err != nil {
			return [32]byte{}, err
		}
		encoded, err := canonicalRecord("webhook-destination-intent-v1", []byte(destination.DestinationID), []byte(strconv.FormatInt(destination.Generation, 10)), []byte(destination.OwnershipVerificationReceipt), []byte(destination.URL), []byte(destination.SelectionRevision), []byte(destination.PayloadVersionPreference), []byte(destination.SignatureProfile), []byte(destination.SigningAuthorityBinding), policy)
		if err != nil {
			return [32]byte{}, err
		}
		encodedDestinations = append(encodedDestinations, encoded)
	}
	list, err := canonicalList(encodedDestinations)
	if err != nil {
		return [32]byte{}, err
	}
	input := prepared.Acceptance
	canonical, err := canonicalRecord("webhook-acceptance-intent-v1",
		[]byte(input.OwnerScope), []byte(input.AcceptanceID), []byte(input.BusinessEventID), []byte(input.FanoutSnapshotID),
		[]byte(input.EventType), []byte(input.BusinessSchemaVersion), []byte(input.ContentType), input.Body,
		[]byte(input.DeliveryEnvelopeVersion), []byte(input.SubscriberPolicyRevision), list,
	)
	if err != nil {
		return [32]byte{}, err
	}
	return canonicalDigest(canonical), nil
}

func encodeLegacyDeliveryPolicy(p DeliveryPolicy) ([]byte, error) {
	contents, err := stringList(p.AcceptedContentTypes)
	if err != nil {
		return nil, err
	}
	schemas, err := stringList(p.AcceptedBusinessSchemas)
	if err != nil {
		return nil, err
	}
	classes, err := stringList(p.AutomaticPauseClasses)
	if err != nil {
		return nil, err
	}
	fields := [][]byte{uintText(p.MaximumPayloadBytes), contents, schemas, uintText(p.MaximumAttempts), durationText(p.MaximumDeliveryAge), durationText(p.BackoffBase), durationText(p.BackoffCap), durationText(p.RetryAfterCap), durationText(p.AttemptTimeout), durationText(p.ResponseHeaderTimeout), uintText(p.ResponseHeaderBytes), uintText(p.ResponseBodyBytes), uintText(p.DestinationConcurrency), uintText(p.GlobalConcurrency), durationText(p.DrainTimeout), uintText(p.RedriveAttempts), durationText(p.RedriveAge)}
	for _, horizon := range p.Horizons.values() {
		fields = append(fields, durationText(horizon))
	}
	fields = append(fields, boolText(p.AutomaticPause), classes, durationTextOptional(p.PauseWindow), uintTextOptional(p.PauseThreshold), uintTextOptional(p.PauseMinimumTraffic), durationTextOptional(p.PauseDuration), boolTextOptional(p.PauseManualOnly), []byte(p.PauseRetentionEffect), []byte(p.PauseAlertPolicy))
	return canonicalRecord("webhook-delivery-policy-v1", fields...)
}

func stringList(values []string) ([]byte, error) {
	return canonicalList(textFields(values...))
}
func uintText[T ~int | ~int64](value T) []byte { return []byte(strconv.FormatInt(int64(value), 10)) }
func uintTextOptional(value int) []byte {
	if value == 0 {
		return nil
	}
	return uintText(value)
}

func durationText(value time.Duration) []byte {
	return []byte(strconv.FormatInt(value.Nanoseconds(), 10))
}

func durationTextOptional(value time.Duration) []byte {
	if value == 0 {
		return nil
	}
	return durationText(value)
}

func boolText(value bool) []byte {
	if value {
		return []byte("1")
	}
	return []byte("0")
}

func boolTextOptional(value bool) []byte {
	if !value {
		return nil
	}
	return []byte("1")
}

func validateToken(name, value string) error {
	if value == "" || len(value) > 256 || strings.IndexFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return fmt.Errorf("%w: %s is invalid", ErrConfig, name)
	}
	return nil
}

func cloneDestination(value DestinationSnapshot) DestinationSnapshot {
	if value.Policy.MinimumTLSVersion == "" {
		value.Policy.MinimumTLSVersion = "1.3"
	}
	value.Policy.AcceptedContentTypes = slices.Clone(value.Policy.AcceptedContentTypes)
	value.Policy.AcceptedBusinessSchemas = slices.Clone(value.Policy.AcceptedBusinessSchemas)
	value.Policy.AutomaticPauseClasses = slices.Clone(value.Policy.AutomaticPauseClasses)
	return value
}
