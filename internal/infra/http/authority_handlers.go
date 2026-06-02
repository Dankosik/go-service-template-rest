package httpx

import (
	"context"
	"net/http"

	"github.com/Dankosik/billing-service/internal/api"
	appmicrolease "github.com/Dankosik/billing-service/internal/app/microlease"
)

func (h strictHandlers) ResolveBillingAccount(ctx context.Context, request api.ResolveBillingAccountRequestObject) (api.ResolveBillingAccountResponseObject, error) {
	if h.authority == nil {
		return api.ResolveBillingAccount503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(authorityNotReadyProblem(ctx)),
		}, nil
	}
	if err := validateAccountResolveRequest(request.Body); err != nil {
		return resolveBillingAccountProblemResponse(ctx, err), nil
	}
	response, err := h.authority.ResolveBillingAccount(ctx, *request.Body)
	if err != nil {
		return resolveBillingAccountProblemResponse(ctx, err), nil
	}
	return api.ResolveBillingAccount200JSONResponse(response), nil
}

func (h strictHandlers) ReadBillingBalance(ctx context.Context, request api.ReadBillingBalanceRequestObject) (api.ReadBillingBalanceResponseObject, error) {
	if h.authority == nil {
		return api.ReadBillingBalance503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(authorityNotReadyProblem(ctx)),
		}, nil
	}
	if err := validateBillingBalanceReadRequest(request.AccountScopeKey, request.Params); err != nil {
		return readBillingBalanceProblemResponse(ctx, err), nil
	}
	response, err := h.authority.ReadBillingBalance(ctx, request.AccountScopeKey, request.Params)
	if err != nil {
		return readBillingBalanceProblemResponse(ctx, err), nil
	}
	return api.ReadBillingBalance200JSONResponse(response), nil
}

func (h strictHandlers) ReserveUsage(ctx context.Context, request api.ReserveUsageRequestObject) (api.ReserveUsageResponseObject, error) {
	if h.authority == nil {
		return api.ReserveUsage503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(authorityNotReadyProblem(ctx)),
		}, nil
	}
	if err := validateUsageReserveRequest(request.Body); err != nil {
		return reserveUsageProblemResponse(ctx, err), nil
	}
	response, err := h.authority.ReserveUsage(ctx, *request.Body)
	if err != nil {
		return reserveUsageProblemResponse(ctx, err), nil
	}
	return api.ReserveUsage200JSONResponse(response), nil
}

func (h strictHandlers) FinalizeUsage(ctx context.Context, request api.FinalizeUsageRequestObject) (api.FinalizeUsageResponseObject, error) {
	if h.authority == nil {
		return api.FinalizeUsage503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(authorityNotReadyProblem(ctx)),
		}, nil
	}
	if err := validateUsageTerminalRequest(request.Body, api.BillingUsageFinalize); err != nil {
		return finalizeUsageProblemResponse(ctx, err), nil
	}
	response, err := h.authority.FinalizeUsage(ctx, *request.Body)
	if err != nil {
		return finalizeUsageProblemResponse(ctx, err), nil
	}
	return api.FinalizeUsage200JSONResponse(response), nil
}

func (h strictHandlers) WriteOffUsage(ctx context.Context, request api.WriteOffUsageRequestObject) (api.WriteOffUsageResponseObject, error) {
	if h.authority == nil {
		return api.WriteOffUsage503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(authorityNotReadyProblem(ctx)),
		}, nil
	}
	if err := validateUsageTerminalRequest(request.Body, api.BillingUsageWriteOff); err != nil {
		return writeOffUsageProblemResponse(ctx, err), nil
	}
	response, err := h.authority.WriteOffUsage(ctx, *request.Body)
	if err != nil {
		return writeOffUsageProblemResponse(ctx, err), nil
	}
	return api.WriteOffUsage200JSONResponse(response), nil
}

func (h strictHandlers) ReverseUsage(ctx context.Context, request api.ReverseUsageRequestObject) (api.ReverseUsageResponseObject, error) {
	if h.authority == nil {
		return api.ReverseUsage503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(authorityNotReadyProblem(ctx)),
		}, nil
	}
	if err := validateUsageReversalRequest(request.Body); err != nil {
		return reverseUsageProblemResponse(ctx, err), nil
	}
	response, err := h.authority.ReverseUsage(ctx, *request.Body)
	if err != nil {
		return reverseUsageProblemResponse(ctx, err), nil
	}
	return api.ReverseUsage200JSONResponse(response), nil
}

func (h strictHandlers) ReadUsageOperation(ctx context.Context, request api.ReadUsageOperationRequestObject) (api.ReadUsageOperationResponseObject, error) {
	if h.authority == nil {
		return api.ReadUsageOperation503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(authorityNotReadyProblem(ctx)),
		}, nil
	}
	if err := validateUsageReadbackRequest(request.Body); err != nil {
		return readUsageOperationProblemResponse(ctx, err), nil
	}
	response, err := h.authority.ReadUsageOperation(ctx, *request.Body)
	if err != nil {
		return readUsageOperationProblemResponse(ctx, err), nil
	}
	return api.ReadUsageOperation200JSONResponse(response), nil
}

func (h strictHandlers) ListReconciliationCases(ctx context.Context, request api.ListReconciliationCasesRequestObject) (api.ListReconciliationCasesResponseObject, error) {
	if h.authority == nil {
		return api.ListReconciliationCases503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(authorityNotReadyProblem(ctx)),
		}, nil
	}
	if err := validateReconciliationCasesRequest(request.Params); err != nil {
		return listReconciliationCasesProblemResponse(ctx, err), nil
	}
	response, err := h.authority.ListReconciliationCases(ctx, request.Params)
	if err != nil {
		return listReconciliationCasesProblemResponse(ctx, err), nil
	}
	return api.ListReconciliationCases200JSONResponse(response), nil
}

func (h strictHandlers) ReadAdminLedger(ctx context.Context, request api.ReadAdminLedgerRequestObject) (api.ReadAdminLedgerResponseObject, error) {
	if h.authority == nil {
		return api.ReadAdminLedger503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(authorityNotReadyProblem(ctx)),
		}, nil
	}
	if request.AccountScopeKey == "" {
		return readAdminLedgerProblemResponse(ctx, NewProblemError(http.StatusBadRequest, "bad request", "account_scope_key_required")), nil
	}
	response, err := h.authority.ReadAdminLedger(ctx, request.AccountScopeKey, request.Params)
	if err != nil {
		return readAdminLedgerProblemResponse(ctx, err), nil
	}
	return api.ReadAdminLedger200JSONResponse(response), nil
}

func (h strictHandlers) ReadAdminExposure(ctx context.Context, request api.ReadAdminExposureRequestObject) (api.ReadAdminExposureResponseObject, error) {
	if h.authority == nil {
		return api.ReadAdminExposure503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(authorityNotReadyProblem(ctx)),
		}, nil
	}
	if request.AccountScopeKey == "" {
		return readAdminExposureProblemResponse(ctx, NewProblemError(http.StatusBadRequest, "bad request", "account_scope_key_required")), nil
	}
	response, err := h.authority.ReadAdminExposure(ctx, request.AccountScopeKey)
	if err != nil {
		return readAdminExposureProblemResponse(ctx, err), nil
	}
	return api.ReadAdminExposure200JSONResponse(response), nil
}

func validateAccountResolveRequest(body *api.AccountResolveRequest) error {
	if body == nil {
		return NewProblemError(http.StatusBadRequest, "bad request", "request body is required")
	}
	if !body.RouteContractId.Valid() || body.RouteContractId != api.BillingAccountsResolve {
		return NewProblemError(http.StatusBadRequest, "bad request", "schema_contract_mismatch")
	}
	if !body.ContractVersion.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_contract_version")
	}
	if body.RepresentedUserContext.SubjectId == nil || *body.RepresentedUserContext.SubjectId == "" {
		return NewProblemError(http.StatusBadRequest, "bad request", "represented_subject_required")
	}
	if body.DeadlineAtEpochMs <= 0 || body.TraceRequestId == "" {
		return NewProblemError(http.StatusBadRequest, "bad request", "required_identity_missing")
	}
	return validateCallerContext(body.CallerContext)
}

func validateBillingBalanceReadRequest(accountScopeKey string, params api.ReadBillingBalanceParams) error {
	if accountScopeKey == "" {
		return NewProblemError(http.StatusBadRequest, "bad request", "account_scope_key_required")
	}
	if params.RepresentedSubjectId == "" || params.TraceRequestId == "" || params.DeadlineAtEpochMs <= 0 {
		return NewProblemError(http.StatusBadRequest, "bad request", "required_identity_missing")
	}
	return nil
}

func validateUsageReserveRequest(body *api.UsageReserveRequest) error {
	if body == nil {
		return NewProblemError(http.StatusBadRequest, "bad request", "request body is required")
	}
	if !body.RouteContractId.Valid() || body.RouteContractId != api.BillingUsageReserve {
		return NewProblemError(http.StatusBadRequest, "bad request", "schema_contract_mismatch")
	}
	if !body.ContractVersion.Valid() || !body.AuthorityMode.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_request_enum")
	}
	if body.AuthorityMode != api.BillingMicroleaseWithProxyChildDebit {
		return NewProblemError(http.StatusUnprocessableEntity, "unprocessable entity", "direct_reserve_fallback_rejected")
	}
	if err := validateUsageCommandIdentity(body.AccountScopeKey, body.IdempotencyKey, body.RequestFingerprint, body.UsageOperationId, body.DeadlineAtEpochMs, body.TraceRequestId); err != nil {
		return err
	}
	if body.MicroleaseId == "" || body.MicroleaseChildDebitId == "" || body.DebitAuthorizationId == "" ||
		body.ProxyAllocatorOwnerId == "" || body.MicroleaseGeneration <= 0 || body.LeaseFence == "" ||
		body.ChildSequence <= 0 || body.ChildCapUsdAtoms <= 0 {
		return NewProblemError(http.StatusBadRequest, "bad request", "microlease_child_lineage_required")
	}
	if body.RepresentedUserContext.SubjectId == nil || *body.RepresentedUserContext.SubjectId == "" {
		return NewProblemError(http.StatusBadRequest, "bad request", "represented_subject_required")
	}
	if err := validatePricingSnapshotEvidence(body.PricingSnapshot); err != nil {
		return err
	}
	return validateCallerAndMetadata(body.CallerContext, body.SafeOperationMetadata)
}

func validateUsageTerminalRequest(body *api.UsageTerminalRequest, routeID api.UsageTerminalRequestRouteContractId) error {
	if body == nil {
		return NewProblemError(http.StatusBadRequest, "bad request", "request body is required")
	}
	if !body.RouteContractId.Valid() || body.RouteContractId != routeID {
		return NewProblemError(http.StatusBadRequest, "bad request", "schema_contract_mismatch")
	}
	if !body.ContractVersion.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_contract_version")
	}
	if err := validateUsageCommandIdentity(body.AccountScopeKey, body.IdempotencyKey, body.RequestFingerprint, body.UsageOperationId, body.DeadlineAtEpochMs, body.TraceRequestId); err != nil {
		return err
	}
	if err := validateUsageTerminalLineage(body); err != nil {
		return err
	}
	if err := validateUsageTerminalAmounts(body, routeID); err != nil {
		return err
	}
	if err := validatePricingSnapshotEvidence(body.PricingSnapshot); err != nil {
		return err
	}
	return validateCallerAndMetadata(body.CallerContext, body.SafeOperationMetadata)
}

func validateUsageTerminalLineage(body *api.UsageTerminalRequest) error {
	if body.MicroleaseId == "" || body.MicroleaseChildDebitId == "" || body.DebitAuthorizationId == "" ||
		body.TerminalOutcomeId == "" || body.TerminalFingerprint == "" {
		return NewProblemError(http.StatusBadRequest, "bad request", "terminal_lineage_required")
	}
	return nil
}

func validateUsageTerminalAmounts(body *api.UsageTerminalRequest, routeID api.UsageTerminalRequestRouteContractId) error {
	if body.ChargedUsdAtoms < 0 || body.ReleasedUsdAtoms < 0 || body.WriteOffUsdAtoms < 0 {
		return NewProblemError(http.StatusBadRequest, "bad request", "invalid_terminal_amount")
	}
	if routeID == api.BillingUsageFinalize && body.WriteOffUsdAtoms != 0 {
		return NewProblemError(http.StatusBadRequest, "bad request", "invalid_terminal_amount")
	}
	if routeID == api.BillingUsageWriteOff && (body.ChargedUsdAtoms != 0 || body.ReleasedUsdAtoms != 0 || body.WriteOffUsdAtoms <= 0) {
		return NewProblemError(http.StatusBadRequest, "bad request", "invalid_terminal_amount")
	}
	return nil
}

func validateUsageReversalRequest(body *api.UsageReversalRequest) error {
	if body == nil {
		return NewProblemError(http.StatusBadRequest, "bad request", "request body is required")
	}
	if !body.RouteContractId.Valid() || body.RouteContractId != api.BillingUsageReversal {
		return NewProblemError(http.StatusBadRequest, "bad request", "schema_contract_mismatch")
	}
	if !body.ContractVersion.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_contract_version")
	}
	if err := validateUsageCommandIdentity(body.AccountScopeKey, body.IdempotencyKey, body.RequestFingerprint, body.UsageOperationId, body.DeadlineAtEpochMs, body.TraceRequestId); err != nil {
		return err
	}
	if body.OriginalLedgerEntryId == "" || body.ReversalUsdAtoms <= 0 || body.ReasonCode == "" {
		return NewProblemError(http.StatusBadRequest, "bad request", "reversal_lineage_required")
	}
	return validateCallerAndMetadata(body.CallerContext, body.SafeOperationMetadata)
}

func validateUsageReadbackRequest(body *api.UsageReadbackRequest) error {
	if body == nil {
		return NewProblemError(http.StatusBadRequest, "bad request", "request body is required")
	}
	if !body.RouteContractId.Valid() || body.RouteContractId != api.BillingUsageReadback {
		return NewProblemError(http.StatusBadRequest, "bad request", "schema_contract_mismatch")
	}
	if !body.ContractVersion.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_contract_version")
	}
	if body.AccountScopeKey == "" || body.UsageOperationId == "" || body.TraceRequestId == "" || body.DeadlineAtEpochMs <= 0 {
		return NewProblemError(http.StatusBadRequest, "bad request", "required_identity_missing")
	}
	return validateCallerContext(body.CallerContext)
}

func validateReconciliationCasesRequest(params api.ListReconciliationCasesParams) error {
	if params.State != nil && !params.State.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_state")
	}
	if params.Severity != nil && !params.Severity.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_severity")
	}
	return nil
}

func validateUsageCommandIdentity(accountScopeKey, idempotencyKey, fingerprint, usageOperationID string, deadlineAtEpochMs int64, traceRequestID string) error {
	if accountScopeKey == "" || idempotencyKey == "" || fingerprint == "" || usageOperationID == "" || traceRequestID == "" {
		return NewProblemError(http.StatusBadRequest, "bad request", "required_identity_missing")
	}
	if deadlineAtEpochMs <= 0 {
		return NewProblemError(http.StatusBadRequest, "bad request", "deadline_required")
	}
	return nil
}

func validatePricingSnapshotEvidence(snapshot api.PricingSnapshotEvidence) error {
	if !snapshot.PriceUnit.Valid() || !snapshot.QuoteAsset.Valid() || !snapshot.UseClass.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_pricing_enum")
	}
	if snapshot.PricingSnapshotId == "" ||
		snapshot.SnapshotFingerprint == "" ||
		snapshot.PolicyVersion == "" ||
		snapshot.SelectorKey == "" ||
		snapshot.ContractVersion == "" ||
		snapshot.DecisionAt.IsZero() {
		return NewProblemError(http.StatusBadRequest, "bad request", "pricing_snapshot_evidence_required")
	}
	return nil
}

func validateCallerAndMetadata(c api.CallerContext, metadata *api.SafeOperationMetadata) error {
	if err := validateCallerContext(c); err != nil {
		return err
	}
	if metadata != nil {
		if err := appmicrolease.ValidateSafeMetadata(map[string]string(*metadata)); err != nil {
			return NewProblemError(http.StatusBadRequest, "bad request", "unsafe_operation_metadata")
		}
	}
	return nil
}

func authorityNotReadyProblem(ctx context.Context) api.Problem {
	problem := api.Problem{
		Detail:    optionalProblemString("balance and usage authority runtime is not ready"),
		RequestId: nil,
		Status:    503,
		Title:     "not ready",
		Type:      "about:blank",
	}
	if ctx != nil {
		problem.RequestId = optionalProblemString(requestIDFromContext(ctx))
	}
	return problem
}
