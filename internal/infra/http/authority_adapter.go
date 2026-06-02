package httpx

import (
	"context"
	"errors"
	"math"
	"net/http"
	"time"

	"github.com/Dankosik/billing-service/internal/api"
	"github.com/Dankosik/billing-service/internal/app/billingauthority"
)

type BillingAuthorityHTTPAdapter struct {
	service *billingauthority.Service
}

func NewBillingAuthorityHTTPAdapter(service *billingauthority.Service) (*BillingAuthorityHTTPAdapter, error) {
	if service == nil {
		return nil, NewProblemError(http.StatusServiceUnavailable, "not ready", "billing_authority_not_ready")
	}
	return &BillingAuthorityHTTPAdapter{service: service}, nil
}

func (a *BillingAuthorityHTTPAdapter) ResolveBillingAccount(ctx context.Context, request api.AccountResolveRequest) (api.AccountResolveResponse, error) {
	snapshot, err := a.service.ResolveAccount(ctx, billingauthority.AccountResolveRequest{
		RepresentedSubjectID: stringValue(request.RepresentedUserContext.SubjectId),
		RepresentedTenantID:  stringValue(request.RepresentedUserContext.TenantId),
		RepresentedAccountID: stringValue(request.RepresentedUserContext.AccountId),
		CallerPrincipal:      string(request.CallerContext.CallerPrincipal),
		CallerScopeKey:       request.CallerContext.CallerScopeKey,
		TraceRequestID:       request.TraceRequestId,
		DeadlineAt:           time.UnixMilli(request.DeadlineAtEpochMs).UTC(),
	})
	if err != nil {
		return api.AccountResolveResponse{}, authorityProblemError(err)
	}
	return accountResolveResponse(snapshot), nil
}

func (a *BillingAuthorityHTTPAdapter) ReadBillingBalance(ctx context.Context, accountScopeKey string, params api.ReadBillingBalanceParams) (api.BalanceReadResponse, error) {
	snapshot, err := a.service.ReadBalance(ctx, billingauthority.BalanceReadRequest{
		AccountScopeKey:      accountScopeKey,
		RepresentedSubjectID: params.RepresentedSubjectId,
		TraceRequestID:       params.TraceRequestId,
		DeadlineAt:           time.UnixMilli(params.DeadlineAtEpochMs).UTC(),
	})
	if err != nil {
		return api.BalanceReadResponse{}, authorityProblemError(err)
	}
	return balanceReadResponse(snapshot), nil
}

func (a *BillingAuthorityHTTPAdapter) ReserveUsage(ctx context.Context, request api.UsageReserveRequest) (api.UsageOperationResponse, error) {
	snapshot, err := a.service.ReserveUsage(ctx, billingauthority.UsageReserveCommand{
		AccountScopeKey:        request.AccountScopeKey,
		UsageOperationID:       request.UsageOperationId,
		AuthorityMode:          string(request.AuthorityMode),
		IdempotencyKey:         request.IdempotencyKey,
		RequestFingerprint:     request.RequestFingerprint,
		RequestID:              request.CallerContext.RequestId,
		MicroleaseID:           request.MicroleaseId,
		MicroleaseChildDebitID: request.MicroleaseChildDebitId,
		DebitAuthorizationID:   request.DebitAuthorizationId,
		ProxyAllocatorOwnerID:  request.ProxyAllocatorOwnerId,
		MicroleaseGeneration:   request.MicroleaseGeneration,
		LeaseFence:             request.LeaseFence,
		ChildSequence:          request.ChildSequence,
		ChildCapUSDAtoms:       request.ChildCapUsdAtoms,
		RepresentedSubjectID:   stringValue(request.RepresentedUserContext.SubjectId),
		Pricing:                pricingSnapshot(request.PricingSnapshot),
		CallerPrincipal:        string(request.CallerContext.CallerPrincipal),
		CallerScopeKey:         request.CallerContext.CallerScopeKey,
		TraceRequestID:         request.TraceRequestId,
		DeadlineAt:             time.UnixMilli(request.DeadlineAtEpochMs).UTC(),
		Metadata:               safeOperationMetadata(request.SafeOperationMetadata),
	})
	if err != nil {
		return api.UsageOperationResponse{}, authorityProblemError(err)
	}
	return usageOperationResponse(snapshot), nil
}

func (a *BillingAuthorityHTTPAdapter) FinalizeUsage(ctx context.Context, request api.UsageTerminalRequest) (api.UsageOperationResponse, error) {
	snapshot, err := a.service.FinalizeUsage(ctx, terminalCommand(request))
	if err != nil {
		return api.UsageOperationResponse{}, authorityProblemError(err)
	}
	return usageOperationResponse(snapshot), nil
}

func (a *BillingAuthorityHTTPAdapter) WriteOffUsage(ctx context.Context, request api.UsageTerminalRequest) (api.UsageOperationResponse, error) {
	snapshot, err := a.service.WriteOffUsage(ctx, terminalCommand(request))
	if err != nil {
		return api.UsageOperationResponse{}, authorityProblemError(err)
	}
	return usageOperationResponse(snapshot), nil
}

func (a *BillingAuthorityHTTPAdapter) ReverseUsage(ctx context.Context, request api.UsageReversalRequest) (api.UsageOperationResponse, error) {
	snapshot, err := a.service.ReverseUsage(ctx, billingauthority.UsageReversalCommand{
		AccountScopeKey:       request.AccountScopeKey,
		UsageOperationID:      request.UsageOperationId,
		IdempotencyKey:        request.IdempotencyKey,
		RequestFingerprint:    request.RequestFingerprint,
		OriginalLedgerEntryID: request.OriginalLedgerEntryId,
		ReversalUSDAtoms:      request.ReversalUsdAtoms,
		ReasonCode:            request.ReasonCode,
		CallerPrincipal:       string(request.CallerContext.CallerPrincipal),
		CallerScopeKey:        request.CallerContext.CallerScopeKey,
		TraceRequestID:        request.TraceRequestId,
		DeadlineAt:            time.UnixMilli(request.DeadlineAtEpochMs).UTC(),
		Metadata:              safeOperationMetadata(request.SafeOperationMetadata),
	})
	if err != nil {
		return api.UsageOperationResponse{}, authorityProblemError(err)
	}
	return usageOperationResponse(snapshot), nil
}

func (a *BillingAuthorityHTTPAdapter) ReadUsageOperation(ctx context.Context, request api.UsageReadbackRequest) (api.UsageOperationReadbackResponse, error) {
	snapshot, err := a.service.ReadUsageOperation(ctx, billingauthority.UsageReadbackRequest{
		AccountScopeKey:  request.AccountScopeKey,
		UsageOperationID: request.UsageOperationId,
		TraceRequestID:   request.TraceRequestId,
		DeadlineAt:       time.UnixMilli(request.DeadlineAtEpochMs).UTC(),
	})
	if err != nil {
		return api.UsageOperationReadbackResponse{}, authorityProblemError(err)
	}
	response := api.UsageOperationReadbackResponse{
		ResultCode:       api.UsageOperationResultCode(snapshot.ResultCode),
		State:            api.UsageOperationState(snapshot.State),
		UsageOperationId: snapshot.UsageOperationID,
	}
	if snapshot.IdempotencyKey != "" || snapshot.StoredOutcomeID != "" {
		response.ReplayIdentity = &api.ReplayIdentity{
			IdempotencyKey:     snapshot.IdempotencyKey,
			RequestFingerprint: snapshot.RequestFingerprint,
			StoredOutcomeId:    snapshot.StoredOutcomeID,
		}
	}
	response.ReasonCode = optionalAPIString(snapshot.ReasonCode)
	response.ReconciliationCaseId = optionalAPIString(snapshot.ReconciliationCaseID)
	return response, nil
}

func (a *BillingAuthorityHTTPAdapter) ListReconciliationCases(ctx context.Context, params api.ListReconciliationCasesParams) (api.ReconciliationCasesResponse, error) {
	state := ""
	if params.State != nil {
		state = string(*params.State)
	}
	severity := ""
	if params.Severity != nil {
		severity = string(*params.Severity)
	}
	accountScopeKey := ""
	if params.AccountScopeKey != nil {
		accountScopeKey = *params.AccountScopeKey
	}
	cases, err := a.service.ListReconciliationCases(ctx, billingauthority.ReconciliationCasesRequest{
		AccountScopeKey: accountScopeKey,
		State:           state,
		Severity:        severity,
		Limit:           100,
	})
	if err != nil {
		return api.ReconciliationCasesResponse{}, authorityProblemError(err)
	}
	items := make([]api.ReconciliationCaseSummary, 0, len(cases))
	for _, item := range cases {
		items = append(items, api.ReconciliationCaseSummary{
			AccountScopeKey:      item.AccountScopeKey,
			Reason:               api.ReconciliationCaseSummaryReason(item.Reason),
			ReconciliationCaseId: item.ReconciliationCaseID,
			SafeLineageId:        optionalAPIString(item.SafeLineageID),
			Severity:             api.ReconciliationCaseSummarySeverity(item.Severity),
			State:                api.ReconciliationCaseSummaryState(item.State),
		})
	}
	return api.ReconciliationCasesResponse{Cases: items, ResultCode: api.ReconciliationCasesResponseResultCodeListed}, nil
}

func (a *BillingAuthorityHTTPAdapter) ReadAdminLedger(ctx context.Context, accountScopeKey string, params api.ReadAdminLedgerParams) (api.AdminLedgerResponse, error) {
	limit := int32(50)
	if params.Limit != nil {
		if *params.Limit < 1 || *params.Limit > math.MaxInt32 {
			return api.AdminLedgerResponse{}, NewProblemError(http.StatusBadRequest, "bad request", "invalid_limit")
		}
		limit = int32(*params.Limit)
	}
	entries, err := a.service.ReadAdminLedger(ctx, billingauthority.AdminLedgerRequest{
		AccountScopeKey: accountScopeKey,
		Limit:           limit,
	})
	if err != nil {
		return api.AdminLedgerResponse{}, authorityProblemError(err)
	}
	items := make([]api.LedgerEntrySummary, 0, len(entries))
	for _, entry := range entries {
		items = append(items, api.LedgerEntrySummary{
			AmountUsdAtoms:      entry.AmountUSDAtoms,
			BalanceVersionAfter: entry.BalanceVersionAfter,
			EffectType:          entry.EffectType,
			LedgerEntryId:       entry.LedgerEntryID,
		})
	}
	return api.AdminLedgerResponse{AccountScopeKey: accountScopeKey, Entries: items, ResultCode: api.AdminLedgerResponseResultCodeFound}, nil
}

func (a *BillingAuthorityHTTPAdapter) ReadAdminExposure(ctx context.Context, accountScopeKey string) (api.AdminExposureResponse, error) {
	snapshot, err := a.service.ReadAdminExposure(ctx, billingauthority.AdminExposureRequest{AccountScopeKey: accountScopeKey})
	if err != nil {
		return api.AdminExposureResponse{}, authorityProblemError(err)
	}
	return api.AdminExposureResponse{
		AccountScopeKey:              snapshot.AccountScopeKey,
		ActiveMicroleaseUsdAtoms:     snapshot.ActiveMicroleaseUSDAtoms,
		ActiveUsageHoldUsdAtoms:      snapshot.ActiveUsageHoldUSDAtoms,
		ResultCode:                   api.AdminExposureResponseResultCodeFound,
		RuntimeGateState:             api.RuntimeGateState(snapshot.RuntimeGateState),
		UnresolvedChildDebitUsdAtoms: snapshot.UnresolvedChildDebitUSDAtoms,
	}, nil
}

func terminalCommand(request api.UsageTerminalRequest) billingauthority.UsageTerminalCommand {
	return billingauthority.UsageTerminalCommand{
		AccountScopeKey:              request.AccountScopeKey,
		UsageOperationID:             request.UsageOperationId,
		IdempotencyKey:               request.IdempotencyKey,
		RequestFingerprint:           request.RequestFingerprint,
		MicroleaseID:                 request.MicroleaseId,
		MicroleaseChildDebitID:       request.MicroleaseChildDebitId,
		DebitAuthorizationID:         request.DebitAuthorizationId,
		TerminalOutcomeID:            request.TerminalOutcomeId,
		TerminalFingerprint:          request.TerminalFingerprint,
		ChargedUSDAtoms:              request.ChargedUsdAtoms,
		ReleasedUSDAtoms:             request.ReleasedUsdAtoms,
		WriteOffUSDAtoms:             request.WriteOffUsdAtoms,
		QualifiedInferenceEvidenceID: stringValue(request.QualifiedInferenceEvidenceId),
		Pricing:                      pricingSnapshot(request.PricingSnapshot),
		CallerPrincipal:              string(request.CallerContext.CallerPrincipal),
		CallerScopeKey:               request.CallerContext.CallerScopeKey,
		TraceRequestID:               request.TraceRequestId,
		DeadlineAt:                   time.UnixMilli(request.DeadlineAtEpochMs).UTC(),
		Metadata:                     safeOperationMetadata(request.SafeOperationMetadata),
	}
}

func accountResolveResponse(snapshot billingauthority.AccountSnapshot) api.AccountResolveResponse {
	resultCode := api.AccountResolveResultCodeResolved
	switch {
	case snapshot.AccountState == "not_found":
		resultCode = api.AccountResolveResultCodeNotFound
	case snapshot.AccountState == "suspended":
		resultCode = api.AccountResolveResultCodeSuspended
	case snapshot.AccountState == "manual_review":
		resultCode = api.AccountResolveResultCodeManualReview
	case snapshot.ImportState == "missing" || snapshot.ImportState == "pending":
		resultCode = api.AccountResolveResultCodeImportRequired
	case snapshot.ImportState == "mismatch":
		resultCode = api.AccountResolveResultCodeReconcileRequired
	}
	return api.AccountResolveResponse{
		AccountScopeKey:     snapshot.AccountScopeKey,
		AccountState:        api.AccountState(snapshot.AccountState),
		BalanceReadEligible: snapshot.BalanceReadEligible,
		BalanceVersion:      optionalAPIInt64(snapshot.BalanceVersion),
		BillingAccountId:    optionalAPIString(snapshot.AccountID),
		FailureClass:        optionalAPIString(snapshot.FailureClass),
		ImportState:         api.ImportState(snapshot.ImportState),
		MigrationState:      api.AccountResolveResponseMigrationState(snapshot.MigrationState),
		ResultCode:          resultCode,
		Retryable:           snapshot.Retryable,
	}
}

func balanceReadResponse(snapshot billingauthority.BalanceSnapshot) api.BalanceReadResponse {
	resultCode := api.BalanceReadResultCodeFound
	switch {
	case snapshot.AccountState == "not_found":
		resultCode = api.BalanceReadResultCodeNotFound
	case snapshot.AccountState == "manual_review":
		resultCode = api.BalanceReadResultCodeManualReview
	case snapshot.ImportState == "missing" || snapshot.ImportState == "pending":
		resultCode = api.BalanceReadResultCodeImportRequired
	case snapshot.ImportState == "mismatch":
		resultCode = api.BalanceReadResultCodeReconcileRequired
	}
	return api.BalanceReadResponse{
		AccountScopeKey:          snapshot.AccountScopeKey,
		ActiveMicroleaseUsdAtoms: snapshot.ActiveMicroleaseUSDAtoms,
		ActiveUsageHoldUsdAtoms:  snapshot.ActiveUsageHoldUSDAtoms,
		AvailableUsdAtoms:        snapshot.AvailableUSDAtoms,
		BalanceVersion:           snapshot.BalanceVersion,
		BillingAccountId:         optionalAPIString(snapshot.AccountID),
		ImportState:              api.ImportState(snapshot.ImportState),
		ManualReview:             optionalAPIBool(snapshot.ManualReview),
		PendingUsdAtoms:          snapshot.PendingUSDAtoms,
		ReasonCode:               optionalAPIString(snapshot.ReasonCode),
		ReconciliationRequired:   optionalAPIBool(snapshot.ReconciliationRequired),
		ReservedUsdAtoms:         snapshot.ReservedUSDAtoms,
		ResultCode:               resultCode,
		RuntimeGateState:         api.RuntimeGateState(snapshot.RuntimeGateState),
		SettledUsdAtoms:          snapshot.SettledUSDAtoms,
	}
}

func usageOperationResponse(snapshot billingauthority.UsageOperationSnapshot) api.UsageOperationResponse {
	return api.UsageOperationResponse{
		BillingOperationId: optionalAPIString(snapshot.BillingOperationID),
		ReasonCode:         optionalAPIString(snapshot.ReasonCode),
		ReplayIdentity: api.ReplayIdentity{
			IdempotencyKey:     snapshot.IdempotencyKey,
			RequestFingerprint: snapshot.RequestFingerprint,
			StoredOutcomeId:    snapshot.StoredOutcomeID,
		},
		ReconciliationCaseId: optionalAPIString(snapshot.ReconciliationCaseID),
		ResultCode:           api.UsageOperationResultCode(snapshot.ResultCode),
		State:                api.UsageOperationState(snapshot.State),
		UsageOperationId:     snapshot.UsageOperationID,
	}
}

func pricingSnapshot(snapshot api.PricingSnapshotEvidence) billingauthority.PricingSnapshot {
	return billingauthority.PricingSnapshot{
		ID:              snapshot.PricingSnapshotId,
		Fingerprint:     snapshot.SnapshotFingerprint,
		PolicyVersion:   snapshot.PolicyVersion,
		DecisionAt:      snapshot.DecisionAt,
		SelectorKey:     snapshot.SelectorKey,
		UseClass:        string(snapshot.UseClass),
		ContractVersion: snapshot.ContractVersion,
	}
}

func authorityProblemError(err error) error {
	switch {
	case errors.Is(err, billingauthority.ErrInvalidRequest):
		return NewProblemError(http.StatusBadRequest, "bad request", "invalid_request")
	case errors.Is(err, billingauthority.ErrRejected):
		return NewProblemError(http.StatusUnprocessableEntity, "unprocessable entity", "request_rejected")
	case errors.Is(err, billingauthority.ErrConflict):
		return NewProblemError(http.StatusConflict, "conflict", "payload_conflict")
	case errors.Is(err, billingauthority.ErrNotReady):
		return NewProblemError(http.StatusServiceUnavailable, "not ready", "dependency_not_ready")
	default:
		return err
	}
}

func safeOperationMetadata(metadata *api.SafeOperationMetadata) map[string]string {
	if metadata == nil {
		return nil
	}
	return map[string]string(*metadata)
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func optionalAPIString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func optionalAPIInt64(value int64) *int64 {
	if value == 0 {
		return nil
	}
	return &value
}

func optionalAPIBool(value bool) *bool {
	if !value {
		return nil
	}
	return &value
}
