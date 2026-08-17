package postgreswebhook

import (
	"encoding/hex"
	"testing"
	"time"
)

func TestWebhookActionCanonicalVectors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		request  ActionRequest
		expected string
	}{
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-1", Kind: ActionDestinationState, TargetKind: "destination", TargetID: "dest-01", TargetGeneration: 3, ExpectedRevision: 11, Reason: "admin_disable", Payload: &DestinationStateAction{Disposition: "disabled"}}, "acdf9c1ab2e21fbc66c0069002d1b4c9cf01fa742433f2a498120519cfacfd8b"},
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-2", Kind: ActionKeyRotation, TargetKind: "destination", TargetID: "dest-01", TargetGeneration: 3, ExpectedRevision: 11, Reason: "rotate", Payload: &KeyRotationAction{SecretRevision: 12, KeyRevision: 5, ActiveKeyReference: "key-new", PredecessorReference: "key-old", OverlapStartsAt: time.Unix(1700000000, 0), PredecessorValidUntil: time.Unix(1700086400, 0), AuthorityReceipt: "stage-receipt-12"}}, "28c10f3a83ebed44bf22010f2ff054c5447fd971bd428595deb41f10f3b335ff"},
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-3", Kind: ActionRedrive, TargetKind: "delivery", TargetID: "delivery-01", ExpectedRevision: 4, Reason: "remediated", Payload: &RedriveAction{MaximumAttempts: 3, MaximumAge: time.Hour, AcknowledgeDuplicateRisk: true}}, "908fad852adaba75e0fedb5cc3b9b5bed75b088fd039c90a1d45edf5c49df27d"},
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-4", Kind: ActionCloseUnknown, TargetKind: "delivery", TargetID: "delivery-01", ExpectedRevision: 4, Reason: "stop_recovery", Payload: &CloseUnknownAction{Disposition: "closed_unknown", AcknowledgeDuplicateRisk: true}}, "20d5e8bdc876f18f8f14297b6014b8676e2c66839510e83ae4718da18cd6d51e"},
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-7", Kind: ActionRetentionHold, TargetKind: "delivery", TargetID: "delivery-01", ExpectedRevision: 4, Reason: "legal_hold", Payload: &RetentionHoldAction{Enabled: true}}, "f6d26ed5edf429d20b2318f9ab2a7977855b091c66acf59a8b3ce3a7601bf6de"},
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-5", Kind: ActionPrivacyDelete, TargetKind: "event", TargetID: "evt-01", Reason: "privacy_request", Payload: &PrivacyDeletionAction{TargetKind: "event", TargetID: "evt-01", Mode: "minimal_tombstone", DeletionAuthority: "privacy-ticket-44"}}, "9605ac89e1dbf168862a6d255750a258f56930559dc59b3525131bca4476adac"},
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-6", Kind: ActionNamespaceRetire, TargetKind: "namespace", TargetID: "owner-a", Reason: "privacy_request", Payload: &NamespaceRetirementAction{Mode: "full_erasure", DeletionAuthority: "privacy-ticket-44"}}, "002038c16d40dc98ccdb637c9b8067617a7dc13a6bde80b67d419590a831195c"},
	}
	for _, test := range tests {
		digest, err := test.request.Fingerprint()
		if err != nil {
			t.Fatal(err)
		}
		if got := hex.EncodeToString(digest[:]); got != test.expected {
			t.Errorf("%s digest = %s, want %s", test.request.Kind, got, test.expected)
		}
	}
}
