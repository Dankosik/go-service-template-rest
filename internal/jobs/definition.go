package jobs

import (
	"bytes"
	"cmp"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	MaxIdentityBytes = 256
	MaxPayloadBytes  = 256 << 10
)

var (
	ErrInvalidDefinition = errors.New("invalid jobs definition")
	ErrInvalidPayload    = errors.New("invalid jobs payload")
)

type Revision struct {
	Kind          string
	ArgsVersion   string
	PolicyVersion string
}

// Compare returns -1, 0, or 1 when r sorts before, equal to, or after other.
// Revisions are ordered by kind, arguments version, then policy version.
func (r Revision) Compare(other Revision) int {
	return cmp.Or(
		cmp.Compare(r.Kind, other.Kind),
		cmp.Compare(r.ArgsVersion, other.ArgsVersion),
		cmp.Compare(r.PolicyVersion, other.PolicyVersion),
	)
}

func (r Revision) Validate() error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "kind", value: r.Kind},
		{name: "args_version", value: r.ArgsVersion},
		{name: "policy_version", value: r.PolicyVersion},
	} {
		if err := validateToken(field.name, field.value); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidDefinition, err)
		}
	}
	return nil
}

type AmbiguousEffectAction string

const (
	AmbiguousEffectRetry          AmbiguousEffectAction = "retry"
	AmbiguousEffectOutcomeUnknown AmbiguousEffectAction = "outcome_unknown"
)

type RetryHintPolicy string

const (
	RetryHintIgnore       RetryHintPolicy = "ignore"
	RetryHintPrefer       RetryHintPolicy = "prefer"
	RetryHintBackoffFloor RetryHintPolicy = "backoff_floor"
)

type JitterMode string

const (
	JitterNone   JitterMode = "none"
	JitterSHA256 JitterMode = "sha256"
)

type RecoveryMode string

const (
	RecoveryUnavailable RecoveryMode = "unavailable"
	RecoveryAllowed     RecoveryMode = "allowed"
)

type BudgetResetMode string

const (
	BudgetPreserved BudgetResetMode = "preserved"
	BudgetReset     BudgetResetMode = "reset"
)

type EffectPolicy struct {
	AmbiguousAction AmbiguousEffectAction
}

type RetryPolicy struct {
	MaxAttempts     uint32
	MaxElapsed      time.Duration
	InitialBackoff  time.Duration
	MaxBackoff      time.Duration
	HintPolicy      RetryHintPolicy
	Jitter          JitterMode
	JitterPermille  uint16
	MaxRecoveryWave uint32
}

type RecoveryPolicy struct {
	Mode     RecoveryMode
	Eligible []State
	Attempts BudgetResetMode
	Elapsed  BudgetResetMode
}

type Policy struct {
	Effect              EffectPolicy
	Retry               RetryPolicy
	Recovery            RecoveryPolicy
	MaxAttemptDuration  time.Duration
	TerminationEnvelope time.Duration
}

type DefinitionInput[A any] struct {
	Revision        Revision
	MaxPayloadBytes int
	Validate        func(A) error
	Policy          Policy
}

type Definition[A any] struct {
	revision        Revision
	maxPayloadBytes int
	validate        func(A) error
	policy          Policy
}

func NewDefinition[A any](input DefinitionInput[A]) (Definition[A], error) {
	if err := input.Revision.Validate(); err != nil {
		return Definition[A]{}, err
	}
	if input.MaxPayloadBytes < 1 || input.MaxPayloadBytes > MaxPayloadBytes {
		return Definition[A]{}, fmt.Errorf("%w: max_payload_bytes must be in [1,%d]", ErrInvalidDefinition, MaxPayloadBytes)
	}
	if input.Validate == nil {
		return Definition[A]{}, fmt.Errorf("%w: validate is required", ErrInvalidDefinition)
	}
	if err := input.Policy.validate(); err != nil {
		return Definition[A]{}, err
	}
	input.Policy.Recovery.Eligible = slices.Clone(input.Policy.Recovery.Eligible)
	return Definition[A]{
		revision:        input.Revision,
		maxPayloadBytes: input.MaxPayloadBytes,
		validate:        input.Validate,
		policy:          input.Policy,
	}, nil
}

func (d Definition[A]) Key() Revision { return d.revision }

func (d Definition[A]) Prepare(args A, identity AcceptanceIdentity, availableAt time.Time) (Prepared, error) {
	if err := d.valid(); err != nil {
		return Prepared{}, err
	}
	if err := identity.Validate(); err != nil {
		return Prepared{}, err
	}
	if availableAt.IsZero() {
		return Prepared{}, fmt.Errorf("%w: available_at is required", ErrInvalidAcceptance)
	}
	if err := d.validate(args); err != nil {
		return Prepared{}, fmt.Errorf("%w: arguments: %w", ErrInvalidPayload, err)
	}
	payload, err := json.Marshal(args)
	if err != nil {
		return Prepared{}, fmt.Errorf("%w: encode: %w", ErrInvalidPayload, err)
	}
	payload, err = canonicalJSON(payload)
	if err != nil {
		return Prepared{}, err
	}
	if len(payload) < 1 || len(payload) > d.maxPayloadBytes {
		return Prepared{}, fmt.Errorf("%w: payload is %d bytes, limit is %d", ErrInvalidPayload, len(payload), d.maxPayloadBytes)
	}
	availableAt = availableAt.UTC()
	fingerprint := fingerprintIntent(d.revision, identity, availableAt, payload)
	return newPrepared(d.revision, identity, availableAt, payload, fingerprint), nil
}

func canonicalJSON(payload []byte) ([]byte, error) {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, fmt.Errorf("%w: canonicalize: %w", ErrInvalidPayload, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("trailing value")
		}
		return nil, fmt.Errorf("%w: canonicalize: %w", ErrInvalidPayload, err)
	}
	canonical, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("%w: canonicalize: %w", ErrInvalidPayload, err)
	}
	return canonical, nil
}

func (d Definition[A]) Decode(payload []byte) (A, error) {
	var zero A
	if err := d.valid(); err != nil {
		return zero, err
	}
	if len(payload) < 1 || len(payload) > d.maxPayloadBytes {
		return zero, fmt.Errorf("%w: payload is %d bytes, limit is %d", ErrInvalidPayload, len(payload), d.maxPayloadBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	var args A
	if err := decoder.Decode(&args); err != nil {
		return zero, fmt.Errorf("%w: decode: %w", ErrInvalidPayload, err)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("trailing value")
		}
		return zero, fmt.Errorf("%w: decode: %w", ErrInvalidPayload, err)
	}
	if err := d.validate(args); err != nil {
		return zero, fmt.Errorf("%w: arguments: %w", ErrInvalidPayload, err)
	}
	return args, nil
}

func (d Definition[A]) valid() error {
	if err := d.revision.Validate(); err != nil {
		return err
	}
	if d.maxPayloadBytes < 1 || d.maxPayloadBytes > MaxPayloadBytes || d.validate == nil {
		return fmt.Errorf("%w: definition is not constructed", ErrInvalidDefinition)
	}
	return d.policy.validate()
}

func (p Policy) validate() error {
	if !p.Effect.AmbiguousAction.valid() {
		return fmt.Errorf("%w: effect.ambiguous_action is required", ErrInvalidDefinition)
	}
	if err := p.Retry.validate(); err != nil {
		return err
	}
	if err := p.Recovery.validate(); err != nil {
		return err
	}
	if p.MaxAttemptDuration <= 0 {
		return fmt.Errorf("%w: max_attempt_duration must be positive", ErrInvalidDefinition)
	}
	if p.TerminationEnvelope <= 0 || p.MaxAttemptDuration > p.TerminationEnvelope {
		return fmt.Errorf("%w: termination_envelope must cover max_attempt_duration", ErrInvalidDefinition)
	}
	return nil
}

func (p RetryPolicy) validate() error {
	if p.MaxAttempts == 0 {
		return fmt.Errorf("%w: retry.max_attempts must be positive", ErrInvalidDefinition)
	}
	if p.MaxElapsed <= 0 {
		return fmt.Errorf("%w: retry.max_elapsed must be positive", ErrInvalidDefinition)
	}
	if p.InitialBackoff <= 0 {
		return fmt.Errorf("%w: retry.initial_backoff must be positive", ErrInvalidDefinition)
	}
	if p.MaxBackoff < p.InitialBackoff {
		return fmt.Errorf("%w: retry.max_backoff must cover initial_backoff", ErrInvalidDefinition)
	}
	if !p.HintPolicy.valid() {
		return fmt.Errorf("%w: retry.hint_policy is required", ErrInvalidDefinition)
	}
	switch p.Jitter {
	case JitterNone:
		if p.JitterPermille != 0 {
			return fmt.Errorf("%w: retry.jitter_permille must be zero when jitter is none", ErrInvalidDefinition)
		}
	case JitterSHA256:
		if p.JitterPermille == 0 || p.JitterPermille > 1000 {
			return fmt.Errorf("%w: retry.jitter_permille must be in [1,1000] for sha256", ErrInvalidDefinition)
		}
	default:
		return fmt.Errorf("%w: retry.jitter is required", ErrInvalidDefinition)
	}
	if p.MaxRecoveryWave == 0 || p.MaxRecoveryWave > math.MaxInt32 {
		return fmt.Errorf("%w: retry.max_recovery_wave must be in [1,%d]", ErrInvalidDefinition, math.MaxInt32)
	}
	return nil
}

func (p RecoveryPolicy) validate() error {
	switch p.Mode {
	case RecoveryUnavailable:
		if len(p.Eligible) != 0 || p.Attempts != BudgetPreserved || p.Elapsed != BudgetPreserved {
			return fmt.Errorf("%w: unavailable recovery must preserve budgets and admit no terminal state", ErrInvalidDefinition)
		}
	case RecoveryAllowed:
		if len(p.Eligible) == 0 {
			return fmt.Errorf("%w: recovery.eligible is required", ErrInvalidDefinition)
		}
		if !p.Attempts.valid() || !p.Elapsed.valid() {
			return fmt.Errorf("%w: recovery budget policy is required", ErrInvalidDefinition)
		}
		seen := make(map[State]struct{}, len(p.Eligible))
		for _, state := range p.Eligible {
			if !state.manualRecoveryEligible() {
				return fmt.Errorf("%w: recovery.eligible contains a non-terminal state", ErrInvalidDefinition)
			}
			if _, ok := seen[state]; ok {
				return fmt.Errorf("%w: recovery.eligible contains a duplicate state", ErrInvalidDefinition)
			}
			seen[state] = struct{}{}
		}
	default:
		return fmt.Errorf("%w: recovery.mode is required", ErrInvalidDefinition)
	}
	return nil
}

func validateToken(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	if len(value) > MaxIdentityBytes {
		return fmt.Errorf("%s is %d bytes, limit is %d", name, len(value), MaxIdentityBytes)
	}
	if !utf8.ValidString(value) {
		return fmt.Errorf("%s must be valid UTF-8", name)
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return fmt.Errorf("%s contains a control character", name)
		}
	}
	return nil
}

func fingerprintIntent(revision Revision, identity AcceptanceIdentity, availableAt time.Time, payload []byte) [sha256.Size]byte {
	hash := sha256.New()
	for _, part := range [][]byte{
		[]byte(revision.Kind), []byte(revision.ArgsVersion), []byte(revision.PolicyVersion),
		[]byte(identity.LogicalJobID), []byte(identity.ProducerScope), []byte(identity.ProducerKey),
		[]byte(identity.OccurrenceScope), []byte(identity.OccurrenceID),
		[]byte(identity.EffectScope), []byte(identity.EffectKey),
		[]byte(availableAt.Format(time.RFC3339Nano)), payload,
	} {
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(len(part)))
		_, _ = hash.Write(size[:])
		_, _ = hash.Write(part)
	}
	var result [sha256.Size]byte
	copy(result[:], hash.Sum(nil))
	return result
}

func (v AmbiguousEffectAction) valid() bool {
	return v == AmbiguousEffectRetry || v == AmbiguousEffectOutcomeUnknown
}

func (v RetryHintPolicy) valid() bool {
	return v == RetryHintIgnore || v == RetryHintPrefer || v == RetryHintBackoffFloor
}

func (v BudgetResetMode) valid() bool { return v == BudgetPreserved || v == BudgetReset }
