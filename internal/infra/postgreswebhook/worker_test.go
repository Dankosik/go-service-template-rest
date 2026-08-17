package postgreswebhook

import (
	"errors"
	"testing"
)

func TestWebhookWorkerConstructor(t *testing.T) {
	t.Parallel()
	if _, err := NewWorker(nil, nil, WorkerConfig{}); !errors.Is(err, ErrConfig) {
		t.Fatalf("NewWorker(nil) error = %v", err)
	}
}

func TestWebhookOwnerScope(t *testing.T) {
	t.Parallel()
	base := ActionRequest{OwnerScope: "owner-a", Actor: "actor-a", ActionID: "action-a", Kind: ActionDestinationState, TargetKind: "destination", TargetID: "dest-a", TargetGeneration: 1, ExpectedRevision: 1, Reason: "admin_disable", Payload: &DestinationStateAction{Disposition: "disabled"}}
	first, err := base.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	base.OwnerScope = "owner-b"
	second, err := base.Fingerprint()
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("owner scope did not bind action fingerprint")
	}
}
