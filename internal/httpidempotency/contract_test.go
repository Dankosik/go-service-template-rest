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
