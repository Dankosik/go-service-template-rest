package postgreswebhook

import (
	"encoding/hex"
	"testing"
)

func TestWebhookActionCanonicalVectors(t *testing.T) {
	tests := []struct {
		request  ActionRequest
		expected string
	}{
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-1", Kind: ActionDestinationState, TargetKind: "destination", TargetID: "dest-01", TargetGeneration: 3, Expected: "11", Reason: "admin_disable", Values: []string{"disabled", ""}}, "acdf9c1ab2e21fbc66c0069002d1b4c9cf01fa742433f2a498120519cfacfd8b"},
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-2", Kind: ActionKeyRotation, TargetKind: "destination", TargetID: "dest-01", TargetGeneration: 3, Expected: "11", Reason: "rotate", Values: []string{"12", "5", "key-new", "key-old", "1700000000", "1700086400", "stage-receipt-12"}}, "28c10f3a83ebed44bf22010f2ff054c5447fd971bd428595deb41f10f3b335ff"},
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-3", Kind: ActionRedrive, TargetKind: "delivery", TargetID: "delivery-01", Expected: "4", Reason: "remediated", DuplicateRisk: true, Values: []string{"3", "3600000000000"}}, "908fad852adaba75e0fedb5cc3b9b5bed75b088fd039c90a1d45edf5c49df27d"},
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-4", Kind: ActionCloseUnknown, TargetKind: "delivery", TargetID: "delivery-01", Expected: "4", Reason: "stop_recovery", DuplicateRisk: true, Values: []string{"closed_unknown"}}, "20d5e8bdc876f18f8f14297b6014b8676e2c66839510e83ae4718da18cd6d51e"},
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-5", Kind: ActionPrivacyDelete, TargetKind: "event", TargetID: "evt-01", Expected: "2", Reason: "privacy_request", Values: []string{"event", "evt-01", "minimal_tombstone", "privacy-ticket-44"}}, "70066ff4fc7faea45b319b0e67b9004be1084a7803295168d21bc0619dda13e2"},
		{ActionRequest{OwnerScope: "owner-a", Actor: "actor-7", ActionID: "action-6", Kind: ActionNamespaceRetire, TargetKind: "namespace", TargetID: "owner-a", Reason: "privacy_request", Values: []string{"full_erasure", "privacy-ticket-44"}}, "002038c16d40dc98ccdb637c9b8067617a7dc13a6bde80b67d419590a831195c"},
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
