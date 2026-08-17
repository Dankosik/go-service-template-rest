package httpidempotency

import (
	"net/http"
	"testing"
	"time"
)

func testContract() Contract {
	return Contract{
		OperationID:         "CreateWidget",
		APIVersion:          "v1",
		KeyMaxBytes:         64,
		FingerprintVersions: []string{"v1"},
		ResultCodecs:        []string{"create-widget/v1"},
		ReplayStatuses:      []int{http.StatusCreated},
		StableHeaders:       []string{"location"},
		ResultMaxBytes:      117,
		ReplayTTL:           time.Hour,
		DuplicateRisk:       DuplicateRiskPolicy{Duration: 2 * time.Hour},
		InProgressWait:      time.Second,
		RetryAfter:          time.Second,
		ExternalEffect:      ExternalEffectNone,
	}
}

func TestContractDuplicateRiskPolicy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		set  func(*Contract)
		want bool
	}{
		{name: "finite", want: true},
		{
			name: "permanent",
			set: func(contract *Contract) {
				contract.DuplicateRisk = DuplicateRiskPolicy{Permanent: true}
			},
			want: true,
		},
		{
			name: "missing finite duration",
			set: func(contract *Contract) {
				contract.DuplicateRisk = DuplicateRiskPolicy{}
			},
		},
		{
			name: "finite duration before replay",
			set: func(contract *Contract) {
				contract.DuplicateRisk = DuplicateRiskPolicy{Duration: contract.ReplayTTL - time.Nanosecond}
			},
		},
		{
			name: "permanent with duration",
			set: func(contract *Contract) {
				contract.DuplicateRisk = DuplicateRiskPolicy{Duration: time.Hour, Permanent: true}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			contract := testContract()
			if test.set != nil {
				test.set(&contract)
			}
			if (contract.Validate() == nil) != test.want {
				t.Fatalf("Validate() = %v, want valid=%t", contract.Validate(), test.want)
			}
		})
	}
}

func TestContractCloneDoesNotAliasDeclaration(t *testing.T) {
	t.Parallel()

	contract := testContract()
	clone := contract.Clone()
	contract.FingerprintVersions[0] = "changed"
	contract.StableHeaders[0] = "changed"
	if clone.FingerprintVersions[0] != "v1" || clone.StableHeaders[0] != "location" {
		t.Fatalf("clone = %+v, aliases declaration", clone)
	}
}

func TestContractValidationRejectsIncompleteOrUnsafeDeclarations(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name   string
		update func(*Contract)
	}{
		{name: "missing operation", update: func(contract *Contract) { contract.OperationID = " " }},
		{name: "missing API version", update: func(contract *Contract) { contract.APIVersion = " " }},
		{name: "invalid key bound", update: func(contract *Contract) { contract.KeyMaxBytes = 0 }},
		{name: "missing fingerprint version", update: func(contract *Contract) { contract.FingerprintVersions = nil }},
		{name: "duplicate fingerprint version", update: func(contract *Contract) { contract.FingerprintVersions = []string{"v1", "v1"} }},
		{name: "missing codec", update: func(contract *Contract) { contract.ResultCodecs = nil }},
		{name: "blank codec", update: func(contract *Contract) { contract.ResultCodecs = []string{" "} }},
		{name: "missing replay status", update: func(contract *Contract) { contract.ReplayStatuses = nil }},
		{name: "failed replay status", update: func(contract *Contract) { contract.ReplayStatuses = []int{http.StatusBadRequest} }},
		{name: "duplicate replay status", update: func(contract *Contract) { contract.ReplayStatuses = []int{http.StatusCreated, http.StatusCreated} }},
		{name: "uppercase stable header", update: func(contract *Contract) { contract.StableHeaders = []string{"Location"} }},
		{name: "forbidden stable header", update: func(contract *Contract) { contract.StableHeaders = []string{"set-cookie"} }},
		{name: "duplicate stable header", update: func(contract *Contract) { contract.StableHeaders = []string{"location", "location"} }},
		{name: "invalid result bound", update: func(contract *Contract) { contract.ResultMaxBytes = 0 }},
		{name: "missing replay TTL", update: func(contract *Contract) { contract.ReplayTTL = 0 }},
		{name: "fractional retry after", update: func(contract *Contract) { contract.RetryAfter += time.Nanosecond }},
		{name: "invalid external effect", update: func(contract *Contract) { contract.ExternalEffect = "other" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			contract := testContract()
			test.update(&contract)
			if err := contract.Validate(); err == nil {
				t.Fatal("Contract.Validate() error = nil")
			}
		})
	}
	for _, effect := range []ExternalEffectDisposition{
		ExternalEffectNone,
		ExternalEffectTransactionalOutbox,
		ExternalEffectDownstreamKey,
		ExternalEffectReconciliation,
		ExternalEffectCompensation,
	} {
		contract := testContract()
		contract.ExternalEffect = effect
		if err := contract.Validate(); err != nil {
			t.Fatalf("Contract.Validate(%q) error = %v", effect, err)
		}
	}
}
