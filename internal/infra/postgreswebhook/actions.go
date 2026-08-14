package postgreswebhook

import (
	"fmt"
	"strconv"
)

type ActionKind string

const (
	ActionDestinationState ActionKind = "destination_state"
	ActionKeyRotation      ActionKind = "key_rotation"
	ActionRedrive          ActionKind = "redrive"
	ActionCloseUnknown     ActionKind = "close_unknown"
	ActionPrivacyDelete    ActionKind = "privacy_delete"
	ActionNamespaceRetire  ActionKind = "namespace_retire"
)

type ActionRequest struct {
	OwnerScope, Actor, ActionID string
	Kind                        ActionKind
	TargetKind, TargetID        string
	TargetGeneration            int64
	Expected                    string
	Reason, Note                string
	DuplicateRisk               bool
	Values                      []string
}

func (request ActionRequest) Fingerprint() ([32]byte, error) {
	for name, value := range map[string]string{"owner_scope": request.OwnerScope, "actor": request.Actor, "action_id": request.ActionID, "target_kind": request.TargetKind, "target_id": request.TargetID, "reason": request.Reason} {
		if err := validateToken(name, value); err != nil {
			return [32]byte{}, err
		}
	}
	tag, expected := actionPayloadContract(request.Kind)
	if tag == "" || len(request.Values) != expected {
		return [32]byte{}, fmt.Errorf("%w: invalid action payload", ErrConfig)
	}
	values := make([][]byte, len(request.Values))
	for i := range request.Values {
		values[i] = []byte(request.Values[i])
	}
	payload, err := canonicalRecord(tag, values...)
	if err != nil {
		return [32]byte{}, err
	}
	canonical, err := canonicalRecord("webhook-operator-action-v1", []byte(request.OwnerScope), []byte(request.Actor), []byte(request.Kind), []byte(request.TargetKind), []byte(request.TargetID), []byte(strconv.FormatInt(request.TargetGeneration, 10)), []byte(request.Expected), []byte(request.Reason), []byte(request.Note), boolText(request.DuplicateRisk), payload)
	if err != nil {
		return [32]byte{}, err
	}
	return canonicalDigest(canonical), nil
}

func actionPayloadContract(kind ActionKind) (string, int) {
	switch kind {
	case ActionDestinationState:
		return "webhook-action-destination-state-v1", 2
	case ActionKeyRotation:
		return "webhook-action-key-rotation-v1", 7
	case ActionRedrive:
		return "webhook-action-redrive-v1", 2
	case ActionCloseUnknown:
		return "webhook-action-close-unknown-v1", 1
	case ActionPrivacyDelete:
		return "webhook-action-privacy-delete-v1", 4
	case ActionNamespaceRetire:
		return "webhook-action-namespace-retire-v1", 2
	default:
		return "", 0
	}
}
