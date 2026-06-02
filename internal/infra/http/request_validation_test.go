package httpx

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/Dankosik/billing-service/internal/api"
)

func TestStrictHandlersFailClosedWhenAuthorityRuntimesAreMissing(t *testing.T) {
	t.Parallel()

	ctx := context.WithValue(context.Background(), requestIDContextKey{}, "req-not-ready")
	h := strictHandlers{}
	ledgerLimit := 1
	accountScope := "acct_test"

	cases := []struct {
		name string
		call func() (any, error)
	}{
		{
			name: "account resolve authority",
			call: func() (any, error) {
				return h.ResolveBillingAccount(ctx, api.ResolveBillingAccountRequestObject{Body: new(validAccountResolveRequest())})
			},
		},
		{
			name: "balance read authority",
			call: func() (any, error) {
				return h.ReadBillingBalance(ctx, api.ReadBillingBalanceRequestObject{
					AccountScopeKey: accountScope,
					Params: api.ReadBillingBalanceParams{
						DeadlineAtEpochMs:    1780315210000,
						RepresentedSubjectId: "user_1",
						TraceRequestId:       "trace-balance",
					},
				})
			},
		},
		{
			name: "reserve authority",
			call: func() (any, error) {
				return h.ReserveUsage(ctx, api.ReserveUsageRequestObject{Body: new(validUsageReserveRequest())})
			},
		},
		{
			name: "finalize authority",
			call: func() (any, error) {
				return h.FinalizeUsage(ctx, api.FinalizeUsageRequestObject{Body: new(validUsageTerminalRequest(api.BillingUsageFinalize))})
			},
		},
		{
			name: "write off authority",
			call: func() (any, error) {
				request := validUsageTerminalRequest(api.BillingUsageWriteOff)
				request.ChargedUsdAtoms = 0
				request.ReleasedUsdAtoms = 0
				request.WriteOffUsdAtoms = 100
				return h.WriteOffUsage(ctx, api.WriteOffUsageRequestObject{Body: &request})
			},
		},
		{
			name: "reversal authority",
			call: func() (any, error) {
				return h.ReverseUsage(ctx, api.ReverseUsageRequestObject{Body: new(validUsageReversalRequest())})
			},
		},
		{
			name: "usage readback authority",
			call: func() (any, error) {
				return h.ReadUsageOperation(ctx, api.ReadUsageOperationRequestObject{Body: new(validUsageReadbackRequest())})
			},
		},
		{
			name: "reconciliation authority",
			call: func() (any, error) {
				return h.ListReconciliationCases(ctx, api.ListReconciliationCasesRequestObject{})
			},
		},
		{
			name: "admin ledger authority",
			call: func() (any, error) {
				return h.ReadAdminLedger(ctx, api.ReadAdminLedgerRequestObject{
					AccountScopeKey: accountScope,
					Params:          api.ReadAdminLedgerParams{Limit: &ledgerLimit},
				})
			},
		},
		{
			name: "admin exposure authority",
			call: func() (any, error) {
				return h.ReadAdminExposure(ctx, api.ReadAdminExposureRequestObject{AccountScopeKey: accountScope})
			},
		},
		{
			name: "microlease issue runtime",
			call: func() (any, error) {
				return h.IssueMicrolease(ctx, api.IssueMicroleaseRequestObject{Body: new(validIssueRequest())})
			},
		},
		{
			name: "microlease read runtime",
			call: func() (any, error) {
				return h.ReadMicrolease(ctx, api.ReadMicroleaseRequestObject{Body: new(validMicroleaseReadbackRequest())})
			},
		},
		{
			name: "microlease close runtime",
			call: func() (any, error) {
				return h.CloseMicrolease(ctx, api.CloseMicroleaseRequestObject{Body: new(validCloseRequest())})
			},
		},
		{
			name: "operation readback runtime",
			call: func() (any, error) {
				return h.ReadBillingOperation(ctx, api.ReadBillingOperationRequestObject{Body: new(validOperationReadbackRequest())})
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			response, err := tt.call()
			if err != nil {
				t.Fatalf("%s returned error = %v", tt.name, err)
			}
			problem, ok := problemFromNotReadyResponse(response)
			if !ok {
				t.Fatalf("%s response = %T, want generated 503 Problem response", tt.name, response)
			}
			if problem.Status != http.StatusServiceUnavailable || problem.RequestId == nil || *problem.RequestId != "req-not-ready" {
				t.Fatalf("%s problem = %+v, want 503 with request id", tt.name, problem)
			}
			if problem.Detail == nil || !strings.Contains(*problem.Detail, "runtime is not ready") {
				t.Fatalf("%s problem detail = %v, want runtime not ready", tt.name, problem.Detail)
			}
		})
	}
}

func TestBillingAuthorityRequestValidationProtectsCutoverBoundaries(t *testing.T) {
	t.Parallel()

	unsafeMetadata := api.SafeOperationMetadata{"raw_prompt": "raw-prompt-secret"}
	badReconciliationState := api.ListReconciliationCasesParamsState("archived")
	badReconciliationSeverity := api.ListReconciliationCasesParamsSeverity("panic")

	cases := []struct {
		name       string
		validate   func() error
		wantStatus int
		wantDetail string
	}{
		{
			name: "nil account resolve body",
			validate: func() error {
				return validateAccountResolveRequest(nil)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "request body is required",
		},
		{
			name: "account resolve requires represented subject",
			validate: func() error {
				request := validAccountResolveRequest()
				request.RepresentedUserContext.SubjectId = nil
				return validateAccountResolveRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "represented_subject_required",
		},
		{
			name: "balance read requires path account scope",
			validate: func() error {
				return validateBillingBalanceReadRequest("", api.ReadBillingBalanceParams{
					DeadlineAtEpochMs:    1780315210000,
					RepresentedSubjectId: "user_1",
					TraceRequestId:       "trace-balance",
				})
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "account_scope_key_required",
		},
		{
			name: "reserve requires microlease child lineage",
			validate: func() error {
				request := validUsageReserveRequest()
				request.MicroleaseChildDebitId = ""
				return validateUsageReserveRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "microlease_child_lineage_required",
		},
		{
			name: "reserve rejects unsafe metadata",
			validate: func() error {
				request := validUsageReserveRequest()
				request.SafeOperationMetadata = &unsafeMetadata
				return validateUsageReserveRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "unsafe_operation_metadata",
		},
		{
			name: "terminal rejects missing lineage",
			validate: func() error {
				request := validUsageTerminalRequest(api.BillingUsageFinalize)
				request.TerminalOutcomeId = ""
				return validateUsageTerminalRequest(&request, api.BillingUsageFinalize)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "terminal_lineage_required",
		},
		{
			name: "terminal rejects missing pricing evidence",
			validate: func() error {
				request := validUsageTerminalRequest(api.BillingUsageFinalize)
				request.PricingSnapshot.SnapshotFingerprint = ""
				return validateUsageTerminalRequest(&request, api.BillingUsageFinalize)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "pricing_snapshot_evidence_required",
		},
		{
			name: "write off rejects charged usage",
			validate: func() error {
				request := validUsageTerminalRequest(api.BillingUsageWriteOff)
				request.WriteOffUsdAtoms = 10
				return validateUsageTerminalRequest(&request, api.BillingUsageWriteOff)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "invalid_terminal_amount",
		},
		{
			name: "reversal requires original ledger lineage",
			validate: func() error {
				request := validUsageReversalRequest()
				request.OriginalLedgerEntryId = ""
				return validateUsageReversalRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "reversal_lineage_required",
		},
		{
			name: "usage readback requires operation identity",
			validate: func() error {
				request := validUsageReadbackRequest()
				request.UsageOperationId = ""
				return validateUsageReadbackRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "required_identity_missing",
		},
		{
			name: "reconciliation rejects unsupported state",
			validate: func() error {
				return validateReconciliationCasesRequest(api.ListReconciliationCasesParams{State: &badReconciliationState})
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "unsupported_state",
		},
		{
			name: "reconciliation rejects unsupported severity",
			validate: func() error {
				return validateReconciliationCasesRequest(api.ListReconciliationCasesParams{Severity: &badReconciliationSeverity})
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "unsupported_severity",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.validate()
			if err == nil {
				t.Fatalf("%s validation error = nil, want ProblemError", tt.name)
			}
			var problem ProblemError
			if !errors.As(err, &problem) {
				t.Fatalf("%s validation error = %v, want ProblemError", tt.name, err)
			}
			if problem.Status != tt.wantStatus || problem.Detail != tt.wantDetail {
				t.Fatalf("%s problem = %+v, want status=%d detail=%q", tt.name, problem, tt.wantStatus, tt.wantDetail)
			}
			assertBodyExcludesSensitiveValues(t, problem.Error())
		})
	}
}

func TestMicroleaseRequestValidationProtectsRuntimeBoundaries(t *testing.T) {
	t.Parallel()

	unsafeMetadata := api.SafeOperationMetadata{"provider_payload": "raw-prompt-secret"}
	idempotencyLookup := api.IdempotencyKey

	cases := []struct {
		name       string
		validate   func() error
		wantStatus int
		wantDetail string
	}{
		{
			name: "issue rejects unsupported enum",
			validate: func() error {
				request := validIssueRequest()
				request.RiskClass = api.MicroleaseIssueRequestRiskClass("unknown")
				return validateIssueMicroleaseRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "unsupported_request_enum",
		},
		{
			name: "issue requires immutable pricing evidence",
			validate: func() error {
				request := validIssueRequest()
				request.PricingSnapshot.PricingSnapshotId = ""
				return validateIssueMicroleaseRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "pricing_snapshot_evidence_required",
		},
		{
			name: "issue rejects unsafe metadata",
			validate: func() error {
				request := validIssueRequest()
				request.SafeOperationMetadata = &unsafeMetadata
				return validateIssueMicroleaseRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "unsafe_operation_metadata",
		},
		{
			name: "readback requires idempotency key when lookup kind asks for it",
			validate: func() error {
				request := validMicroleaseReadbackRequest()
				request.LookupKind = idempotencyLookup
				request.MicroleaseId = nil
				request.IdempotencyKey = nil
				return validateReadMicroleaseRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "idempotency_key_required",
		},
		{
			name: "readback requires microlease id when lookup kind asks for it",
			validate: func() error {
				request := validMicroleaseReadbackRequest()
				request.MicroleaseId = nil
				return validateReadMicroleaseRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "microlease_id_required",
		},
		{
			name: "close requires lineage",
			validate: func() error {
				request := validCloseRequest()
				request.MicroleaseGeneration = 0
				return validateCloseMicroleaseRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "invalid_close_request",
		},
		{
			name: "operation readback requires identity",
			validate: func() error {
				request := validOperationReadbackRequest()
				request.OperationIdentity = ""
				return validateBillingOperationReadbackRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "required_identity_missing",
		},
		{
			name: "caller context requires supported principal",
			validate: func() error {
				request := validOperationReadbackRequest()
				request.CallerContext.CallerPrincipal = api.CallerContextCallerPrincipal("browser")
				return validateBillingOperationReadbackRequest(&request)
			},
			wantStatus: http.StatusBadRequest,
			wantDetail: "unsupported_caller_principal",
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tt.validate()
			if err == nil {
				t.Fatalf("%s validation error = nil, want ProblemError", tt.name)
			}
			var problem ProblemError
			if !errors.As(err, &problem) {
				t.Fatalf("%s validation error = %v, want ProblemError", tt.name, err)
			}
			if problem.Status != tt.wantStatus || problem.Detail != tt.wantDetail {
				t.Fatalf("%s problem = %+v, want status=%d detail=%q", tt.name, problem, tt.wantStatus, tt.wantDetail)
			}
			assertBodyExcludesSensitiveValues(t, problem.Error())
		})
	}
}

func problemFromNotReadyResponse(response any) (api.Problem, bool) {
	switch resp := response.(type) {
	case api.ResolveBillingAccount503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.ReadBillingBalance503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.ReserveUsage503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.FinalizeUsage503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.WriteOffUsage503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.ReverseUsage503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.ReadUsageOperation503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.ListReconciliationCases503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.ReadAdminLedger503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.ReadAdminExposure503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.IssueMicrolease503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.ReadMicrolease503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.CloseMicrolease503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	case api.ReadBillingOperation503ApplicationProblemPlusJSONResponse:
		return api.Problem(resp.ServiceUnavailableApplicationProblemPlusJSONResponse), true
	default:
		return api.Problem{}, false
	}
}
