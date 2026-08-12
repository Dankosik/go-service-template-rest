package jobs

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"time"
)

var ErrInvalidAcceptance = errors.New("invalid jobs acceptance")

type (
	LogicalJobID    string
	ProducerScope   string
	ProducerKey     string
	OccurrenceScope string
	OccurrenceID    string
	EffectScope     string
	EffectKey       string
)

type AcceptanceIdentity struct {
	LogicalJobID    LogicalJobID
	ProducerScope   ProducerScope
	ProducerKey     ProducerKey
	OccurrenceScope OccurrenceScope
	OccurrenceID    OccurrenceID
	EffectScope     EffectScope
	EffectKey       EffectKey
}

func (i AcceptanceIdentity) Validate() error {
	for _, field := range []struct {
		name  string
		value string
	}{
		{name: "logical_job_id", value: string(i.LogicalJobID)},
		{name: "producer_scope", value: string(i.ProducerScope)},
		{name: "producer_key", value: string(i.ProducerKey)},
		{name: "occurrence_scope", value: string(i.OccurrenceScope)},
		{name: "occurrence_id", value: string(i.OccurrenceID)},
		{name: "effect_scope", value: string(i.EffectScope)},
		{name: "effect_key", value: string(i.EffectKey)},
	} {
		if err := validateToken(field.name, field.value); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidAcceptance, err)
		}
	}
	return nil
}

type Prepared struct {
	revision    Revision
	identity    AcceptanceIdentity
	availableAt time.Time
	payload     []byte
	fingerprint [sha256.Size]byte
}

func newPrepared(revision Revision, identity AcceptanceIdentity, availableAt time.Time, payload []byte, fingerprint [sha256.Size]byte) Prepared {
	return Prepared{
		revision: revision, identity: identity, availableAt: availableAt,
		payload: slices.Clone(payload), fingerprint: fingerprint,
	}
}

func (p Prepared) Revision() Revision             { return p.revision }
func (p Prepared) Identity() AcceptanceIdentity   { return p.identity }
func (p Prepared) AvailableAt() time.Time         { return p.availableAt }
func (p Prepared) Payload() []byte                { return slices.Clone(p.payload) }
func (p Prepared) Fingerprint() [sha256.Size]byte { return p.fingerprint }

type ReadbackExpectation struct {
	identity    AcceptanceIdentity
	fingerprint [sha256.Size]byte
}

func (p Prepared) ReadbackExpectation() ReadbackExpectation {
	return ReadbackExpectation{identity: p.identity, fingerprint: p.fingerprint}
}

func (e ReadbackExpectation) Identity() AcceptanceIdentity   { return e.identity }
func (e ReadbackExpectation) Fingerprint() [sha256.Size]byte { return e.fingerprint }

func (e ReadbackExpectation) Validate() error {
	return e.identity.Validate()
}

func (p Prepared) Validate() error {
	if err := p.revision.Validate(); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidAcceptance, err)
	}
	if err := p.identity.Validate(); err != nil {
		return err
	}
	if p.availableAt.IsZero() {
		return fmt.Errorf("%w: available_at is required", ErrInvalidAcceptance)
	}
	if len(p.payload) < 1 || len(p.payload) > MaxPayloadBytes {
		return fmt.Errorf("%w: payload is %d bytes, limit is %d", ErrInvalidAcceptance, len(p.payload), MaxPayloadBytes)
	}
	if got := fingerprintIntent(p.revision, p.identity, p.availableAt, p.payload); got != p.fingerprint {
		return fmt.Errorf("%w: fingerprint does not match immutable intent", ErrInvalidAcceptance)
	}
	return nil
}

type StageOutcome string

const (
	StageNew      StageOutcome = "new"
	StageExisting StageOutcome = "existing"
	StageConflict StageOutcome = "conflict"
	StageRejected StageOutcome = "rejected"
)

type StageResult struct {
	Outcome      StageOutcome
	LogicalJobID LogicalJobID
}

func (r StageResult) Validate() error {
	switch r.Outcome {
	case StageNew, StageExisting, StageConflict:
		if err := validateToken("logical_job_id", string(r.LogicalJobID)); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidAcceptance, err)
		}
	case StageRejected:
		if r.LogicalJobID != "" {
			return fmt.Errorf("%w: rejected result must not carry logical_job_id", ErrInvalidAcceptance)
		}
	default:
		return fmt.Errorf("%w: unknown stage outcome", ErrInvalidAcceptance)
	}
	return nil
}

type ReadbackOutcome string

const (
	ReadbackAccepted    ReadbackOutcome = "accepted"
	ReadbackNotAccepted ReadbackOutcome = "not_accepted"
	ReadbackConflict    ReadbackOutcome = "conflict"
	ReadbackUnknown     ReadbackOutcome = "unknown"
)

type ReadbackResult struct {
	Outcome      ReadbackOutcome
	LogicalJobID LogicalJobID
}

func (r ReadbackResult) Validate() error {
	switch r.Outcome {
	case ReadbackAccepted, ReadbackConflict:
		if err := validateToken("logical_job_id", string(r.LogicalJobID)); err != nil {
			return fmt.Errorf("%w: %w", ErrInvalidAcceptance, err)
		}
	case ReadbackNotAccepted, ReadbackUnknown:
		if r.LogicalJobID != "" {
			return fmt.Errorf("%w: %s result must not carry logical_job_id", ErrInvalidAcceptance, r.Outcome)
		}
	default:
		return fmt.Errorf("%w: unknown readback outcome", ErrInvalidAcceptance)
	}
	return nil
}
