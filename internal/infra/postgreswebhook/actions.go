package postgreswebhook

import (
	"fmt"
	"strconv"
	"time"
)

type ActionKind string

const (
	ActionDestinationState ActionKind = "destination_state"
	ActionKeyRotation      ActionKind = "key_rotation"
	ActionRedrive          ActionKind = "redrive"
	ActionCloseUnknown     ActionKind = "close_unknown"
	ActionRetentionHold    ActionKind = "retention_hold"
	ActionPrivacyDelete    ActionKind = "privacy_delete"
	ActionNamespaceRetire  ActionKind = "namespace_retire"

	targetKindDestination = "destination"
	targetKindDelivery    = "delivery"
	targetKindEvent       = "event"
	targetKindNamespace   = "namespace"
)

type ActionPayload interface {
	actionKind() ActionKind
	canonicalValues() (string, [][]byte, bool, error)
}

type DestinationStateAction struct {
	Disposition      string `json:"disposition"`
	AuthorityReceipt string `json:"authority_receipt,omitempty"`
}

func (*DestinationStateAction) actionKind() ActionKind { return ActionDestinationState }
func (payload *DestinationStateAction) canonicalValues() (string, [][]byte, bool, error) {
	if payload == nil {
		return "", nil, false, ErrConfig
	}
	return "webhook-action-destination-state-v1", textFields(payload.Disposition, payload.AuthorityReceipt), false, nil
}

type KeyRotationAction struct {
	SecretRevision        int64     `json:"secret_revision"`
	KeyRevision           int64     `json:"key_revision"`
	ActiveKeyReference    string    `json:"active_key_reference"`
	PredecessorReference  string    `json:"predecessor_key_reference"`
	OverlapStartsAt       time.Time `json:"overlap_starts_at"`
	PredecessorValidUntil time.Time `json:"predecessor_valid_until"`
	AuthorityReceipt      string    `json:"authority_receipt"`
}

func (*KeyRotationAction) actionKind() ActionKind { return ActionKeyRotation }
func (payload *KeyRotationAction) canonicalValues() (string, [][]byte, bool, error) {
	if payload == nil {
		return "", nil, false, ErrConfig
	}
	return "webhook-action-key-rotation-v1", textFields(
		strconv.FormatInt(payload.SecretRevision, 10), strconv.FormatInt(payload.KeyRevision, 10),
		payload.ActiveKeyReference, payload.PredecessorReference,
		strconv.FormatInt(payload.OverlapStartsAt.Unix(), 10), strconv.FormatInt(payload.PredecessorValidUntil.Unix(), 10),
		payload.AuthorityReceipt,
	), false, nil
}

type RedriveAction struct {
	MaximumAttempts          int           `json:"maximum_attempts"`
	MaximumAge               time.Duration `json:"maximum_age"`
	AcknowledgeDuplicateRisk bool          `json:"acknowledge_duplicate_risk"`
}

func (*RedriveAction) actionKind() ActionKind { return ActionRedrive }
func (payload *RedriveAction) canonicalValues() (string, [][]byte, bool, error) {
	if payload == nil {
		return "", nil, false, ErrConfig
	}
	return "webhook-action-redrive-v1", textFields(strconv.Itoa(payload.MaximumAttempts), strconv.FormatInt(int64(payload.MaximumAge), 10)), payload.AcknowledgeDuplicateRisk, nil
}

type CloseUnknownAction struct {
	Disposition              string `json:"disposition"`
	AcknowledgeDuplicateRisk bool   `json:"acknowledge_duplicate_risk"`
}

func (*CloseUnknownAction) actionKind() ActionKind { return ActionCloseUnknown }
func (payload *CloseUnknownAction) canonicalValues() (string, [][]byte, bool, error) {
	if payload == nil {
		return "", nil, false, ErrConfig
	}
	return "webhook-action-close-unknown-v1", textFields(payload.Disposition), payload.AcknowledgeDuplicateRisk, nil
}

type RetentionHoldAction struct {
	Enabled bool `json:"enabled"`
}

func (*RetentionHoldAction) actionKind() ActionKind { return ActionRetentionHold }
func (payload *RetentionHoldAction) canonicalValues() (string, [][]byte, bool, error) {
	if payload == nil {
		return "", nil, false, ErrConfig
	}
	value := "off"
	if payload.Enabled {
		value = "on"
	}
	return "webhook-action-retention-hold-v1", textFields(value), false, nil
}

type PrivacyDeletionAction struct {
	TargetKind        string `json:"target_kind"`
	TargetID          string `json:"target_id"`
	Mode              string `json:"mode"`
	DeletionAuthority string `json:"deletion_authority"`
}

func (*PrivacyDeletionAction) actionKind() ActionKind { return ActionPrivacyDelete }
func (payload *PrivacyDeletionAction) canonicalValues() (string, [][]byte, bool, error) {
	if payload == nil {
		return "", nil, false, ErrConfig
	}
	return "webhook-action-privacy-delete-v1", textFields(payload.TargetKind, payload.TargetID, payload.Mode, payload.DeletionAuthority), false, nil
}

type NamespaceRetirementAction struct {
	Mode              string `json:"mode"`
	DeletionAuthority string `json:"deletion_authority"`
}

func (*NamespaceRetirementAction) actionKind() ActionKind { return ActionNamespaceRetire }
func (payload *NamespaceRetirementAction) canonicalValues() (string, [][]byte, bool, error) {
	if payload == nil {
		return "", nil, false, ErrConfig
	}
	return "webhook-action-namespace-retire-v1", textFields(payload.Mode, payload.DeletionAuthority), false, nil
}

type ActionRequest struct {
	OwnerScope, Actor, ActionID string
	Kind                        ActionKind
	TargetKind, TargetID        string
	TargetGeneration            int64
	ExpectedRevision            int64
	Reason                      string
	Payload                     ActionPayload
}

func (request ActionRequest) Fingerprint() ([32]byte, error) {
	if request.TargetGeneration < 0 || request.ExpectedRevision < 0 || request.Payload == nil || request.Payload.actionKind() != request.Kind {
		return [32]byte{}, fmt.Errorf("%w: action identity or payload is invalid", ErrConfig)
	}
	if err := validateActionTarget(request); err != nil {
		return [32]byte{}, err
	}
	for name, value := range map[string]string{"owner_scope": request.OwnerScope, "actor": request.Actor, "action_id": request.ActionID, "target_kind": request.TargetKind, "target_id": request.TargetID, "reason": request.Reason} {
		if err := validateToken(name, value); err != nil {
			return [32]byte{}, err
		}
	}
	tag, values, duplicateRisk, err := request.Payload.canonicalValues()
	if err != nil {
		return [32]byte{}, fmt.Errorf("%w: invalid action payload", ErrConfig)
	}
	for _, value := range values {
		if len(value) != 0 {
			if err := validateToken("action_value", string(value)); err != nil {
				return [32]byte{}, err
			}
		}
	}
	payload, err := canonicalRecord(tag, values...)
	if err != nil {
		return [32]byte{}, err
	}
	expected := []byte(strconv.FormatInt(request.ExpectedRevision, 10))
	if request.Kind == ActionPrivacyDelete || request.Kind == ActionNamespaceRetire {
		expected = nil
	}
	canonical, err := canonicalRecord("webhook-operator-action-v1", []byte(request.OwnerScope), []byte(request.Actor), []byte(request.Kind), []byte(request.TargetKind), []byte(request.TargetID), []byte(strconv.FormatInt(request.TargetGeneration, 10)), expected, []byte(request.Reason), nil, boolText(duplicateRisk), payload)
	if err != nil {
		return [32]byte{}, err
	}
	return canonicalDigest(canonical), nil
}

func validateActionTarget(request ActionRequest) error {
	valid := false
	switch request.Kind {
	case ActionDestinationState, ActionKeyRotation:
		valid = request.TargetKind == targetKindDestination && request.TargetGeneration > 0 && request.ExpectedRevision > 0
	case ActionRedrive, ActionCloseUnknown, ActionRetentionHold:
		valid = request.TargetKind == targetKindDelivery && request.TargetGeneration == 0
	case ActionPrivacyDelete:
		valid = request.TargetKind == targetKindEvent && request.TargetGeneration == 0 && request.ExpectedRevision == 0
	case ActionNamespaceRetire:
		valid = request.TargetKind == targetKindNamespace && request.TargetID == request.OwnerScope && request.TargetGeneration == 0 && request.ExpectedRevision == 0
	}
	if !valid {
		return fmt.Errorf("%w: action target is invalid", ErrConfig)
	}
	return nil
}

func textFields(values ...string) [][]byte {
	fields := make([][]byte, len(values))
	for i := range values {
		fields[i] = []byte(values[i])
	}
	return fields
}
