package httpx

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/Dankosik/billing-service/internal/api"
	"github.com/Dankosik/billing-service/internal/app/health"
	appmicrolease "github.com/Dankosik/billing-service/internal/app/microlease"
	"github.com/Dankosik/billing-service/internal/app/ping"
	"github.com/Dankosik/billing-service/internal/infra/telemetry"
)

type Handlers struct {
	Health               *health.Service
	Ping                 *ping.Service
	Microleases          MicroleaseService
	Authority            BillingAuthorityService
	ServiceAuthenticator ServiceAuthenticator
	ReadinessGate        func(context.Context) error
}

type MicroleaseService interface {
	IssueMicrolease(context.Context, api.MicroleaseIssueRequest) (api.MicroleaseIssueResponse, error)
	ReadMicrolease(context.Context, api.MicroleaseReadbackRequest) (api.MicroleaseReadbackResponse, error)
	CloseMicrolease(context.Context, api.MicroleaseCloseRequest) (api.MicroleaseCloseResponse, error)
	ReadBillingOperation(context.Context, api.BillingOperationReadbackRequest) (api.BillingOperationReadbackResponse, error)
}

type BillingAuthorityService interface {
	ResolveBillingAccount(context.Context, api.AccountResolveRequest) (api.AccountResolveResponse, error)
	ReadBillingBalance(context.Context, string, api.ReadBillingBalanceParams) (api.BalanceReadResponse, error)
	ReserveUsage(context.Context, api.UsageReserveRequest) (api.UsageOperationResponse, error)
	FinalizeUsage(context.Context, api.UsageTerminalRequest) (api.UsageOperationResponse, error)
	WriteOffUsage(context.Context, api.UsageTerminalRequest) (api.UsageOperationResponse, error)
	ReverseUsage(context.Context, api.UsageReversalRequest) (api.UsageOperationResponse, error)
	ReadUsageOperation(context.Context, api.UsageReadbackRequest) (api.UsageOperationReadbackResponse, error)
	ListReconciliationCases(context.Context, api.ListReconciliationCasesParams) (api.ReconciliationCasesResponse, error)
	ReadAdminLedger(context.Context, string, api.ReadAdminLedgerParams) (api.AdminLedgerResponse, error)
	ReadAdminExposure(context.Context, string) (api.AdminExposureResponse, error)
}

type strictHandlers struct {
	health               *health.Service
	ping                 *ping.Service
	microleases          MicroleaseService
	authority            BillingAuthorityService
	serviceAuthenticator ServiceAuthenticator
	metrics              *telemetry.Metrics
	readinessGate        func(context.Context) error
	readinessTimeout     time.Duration
}

var _ api.StrictServerInterface = (*strictHandlers)(nil)

func newStrictHandlers(h Handlers, metrics *telemetry.Metrics, readinessTimeout time.Duration) (strictHandlers, error) {
	if h.Health == nil {
		return strictHandlers{}, fmt.Errorf("http router: health service is required")
	}
	if h.Ping == nil {
		return strictHandlers{}, fmt.Errorf("http router: ping service is required")
	}
	if h.ReadinessGate == nil {
		return strictHandlers{}, fmt.Errorf("http router: readiness gate is required")
	}
	if metrics == nil {
		return strictHandlers{}, fmt.Errorf("http router: metrics is required")
	}
	if readinessTimeout <= 0 {
		return strictHandlers{}, fmt.Errorf("http router: readiness timeout must be > 0")
	}

	return strictHandlers{
		health:               h.Health,
		ping:                 h.Ping,
		microleases:          h.Microleases,
		authority:            h.Authority,
		serviceAuthenticator: h.ServiceAuthenticator,
		metrics:              metrics,
		readinessGate:        h.ReadinessGate,
		readinessTimeout:     readinessTimeout,
	}, nil
}

func (h strictHandlers) Ping(_ context.Context, _ api.PingRequestObject) (api.PingResponseObject, error) {
	return api.Ping200TextResponse(h.ping.Pong()), nil
}

func (h strictHandlers) HealthLive(_ context.Context, _ api.HealthLiveRequestObject) (api.HealthLiveResponseObject, error) {
	return api.HealthLive200TextResponse("ok"), nil
}

func (h strictHandlers) HealthReady(ctx context.Context, _ api.HealthReadyRequestObject) (api.HealthReadyResponseObject, error) {
	readyCtx, cancel := context.WithTimeout(ctx, h.readinessTimeout)
	defer cancel()

	if err := h.readinessGate(readyCtx); err != nil {
		return api.HealthReady503TextResponse("not ready"), nil
	}
	if err := h.health.Ready(readyCtx); err != nil {
		return api.HealthReady503TextResponse("not ready"), nil
	}

	return api.HealthReady200TextResponse("ok"), nil
}

func (h strictHandlers) Metrics(_ context.Context, _ api.MetricsRequestObject) (api.MetricsResponseObject, error) {
	// /metrics is served by the documented root-router exception, not the generated strict path.
	return nil, fmt.Errorf("metrics strict handler is not runtime-owned")
}

func (h strictHandlers) IssueMicrolease(ctx context.Context, request api.IssueMicroleaseRequestObject) (api.IssueMicroleaseResponseObject, error) {
	if h.microleases == nil {
		return api.IssueMicrolease503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(notReadyProblem(ctx)),
		}, nil
	}
	if err := validateIssueMicroleaseRequest(request.Body); err != nil {
		return issueMicroleaseProblemResponse(ctx, err), nil
	}
	response, err := h.microleases.IssueMicrolease(ctx, *request.Body)
	if err != nil {
		return issueMicroleaseProblemResponse(ctx, err), nil
	}
	return api.IssueMicrolease200JSONResponse(response), nil
}

func (h strictHandlers) ReadMicrolease(ctx context.Context, request api.ReadMicroleaseRequestObject) (api.ReadMicroleaseResponseObject, error) {
	if h.microleases == nil {
		return api.ReadMicrolease503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(notReadyProblem(ctx)),
		}, nil
	}
	if err := validateReadMicroleaseRequest(request.Body); err != nil {
		return readMicroleaseProblemResponse(ctx, err), nil
	}
	response, err := h.microleases.ReadMicrolease(ctx, *request.Body)
	if err != nil {
		return readMicroleaseProblemResponse(ctx, err), nil
	}
	return api.ReadMicrolease200JSONResponse(response), nil
}

func (h strictHandlers) CloseMicrolease(ctx context.Context, request api.CloseMicroleaseRequestObject) (api.CloseMicroleaseResponseObject, error) {
	if h.microleases == nil {
		return api.CloseMicrolease503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(notReadyProblem(ctx)),
		}, nil
	}
	if err := validateCloseMicroleaseRequest(request.Body); err != nil {
		return closeMicroleaseProblemResponse(ctx, err), nil
	}
	response, err := h.microleases.CloseMicrolease(ctx, *request.Body)
	if err != nil {
		return closeMicroleaseProblemResponse(ctx, err), nil
	}
	return api.CloseMicrolease200JSONResponse(response), nil
}

func (h strictHandlers) ReadBillingOperation(ctx context.Context, request api.ReadBillingOperationRequestObject) (api.ReadBillingOperationResponseObject, error) {
	if h.microleases == nil {
		return api.ReadBillingOperation503ApplicationProblemPlusJSONResponse{
			ServiceUnavailableApplicationProblemPlusJSONResponse: api.ServiceUnavailableApplicationProblemPlusJSONResponse(notReadyProblem(ctx)),
		}, nil
	}
	if err := validateBillingOperationReadbackRequest(request.Body); err != nil {
		return readBillingOperationProblemResponse(ctx, err), nil
	}
	response, err := h.microleases.ReadBillingOperation(ctx, *request.Body)
	if err != nil {
		return readBillingOperationProblemResponse(ctx, err), nil
	}
	return api.ReadBillingOperation200JSONResponse(response), nil
}

func notReadyProblem(ctx context.Context) api.Problem {
	problem := api.Problem{
		Detail:    optionalProblemString("microlease runtime is not ready"),
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

func validateIssueMicroleaseRequest(body *api.MicroleaseIssueRequest) error {
	if body == nil {
		return NewProblemError(http.StatusBadRequest, "bad request", "request body is required")
	}
	if !body.RouteContractId.Valid() || body.RouteContractId != api.BillingMicroleaseIssue {
		return NewProblemError(http.StatusBadRequest, "bad request", "schema_contract_mismatch")
	}
	if !body.ContractVersion.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_contract_version")
	}
	if err := validateIssueMicroleaseEnums(body); err != nil {
		return err
	}
	if err := validateIssueMicroleasePricingSnapshot(body); err != nil {
		return err
	}
	if err := validateIssueMicroleaseRequiredFields(body); err != nil {
		return err
	}
	return validateCallerContext(body.CallerContext)
}

func validateIssueMicroleaseEnums(body *api.MicroleaseIssueRequest) error {
	if !body.RequestedCutoffClass.Valid() || !body.RiskClass.Valid() || !body.UseClass.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_request_enum")
	}
	if body.StrictReason != nil && !body.StrictReason.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_request_enum")
	}
	if body.TerminalLagClass != nil && !body.TerminalLagClass.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_request_enum")
	}
	if body.LocalBacklogClass != nil && !body.LocalBacklogClass.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_request_enum")
	}
	return nil
}

func validateIssueMicroleasePricingSnapshot(body *api.MicroleaseIssueRequest) error {
	if !body.PricingSnapshot.PriceUnit.Valid() || !body.PricingSnapshot.QuoteAsset.Valid() || !body.PricingSnapshot.UseClass.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_pricing_enum")
	}
	if body.PricingSnapshot.FreshnessClass != nil && !body.PricingSnapshot.FreshnessClass.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_pricing_enum")
	}
	if body.PricingSnapshot.PricingSnapshotId == "" ||
		body.PricingSnapshot.SnapshotFingerprint == "" ||
		body.PricingSnapshot.PolicyVersion == "" ||
		body.PricingSnapshot.SelectorKey == "" ||
		body.PricingSnapshot.ContractVersion == "" ||
		body.PricingSnapshot.DecisionAt.IsZero() {
		return NewProblemError(http.StatusBadRequest, "bad request", "pricing_snapshot_evidence_required")
	}
	return nil
}

func validateIssueMicroleaseRequiredFields(body *api.MicroleaseIssueRequest) error {
	if body.AccountScopeKey == "" || body.IdempotencyKey == "" || body.RequestFingerprint == "" || body.TraceRequestId == "" {
		return NewProblemError(http.StatusBadRequest, "bad request", "required_identity_missing")
	}
	if body.ProxyAllocatorOwnerId == "" || body.MaxRequestedUsdAtoms <= 0 || body.MaxChildCapUsdAtoms <= 0 || body.RequestedTtlSeconds <= 0 {
		return NewProblemError(http.StatusBadRequest, "bad request", "invalid_microlease_request")
	}
	if body.SafeOperationMetadata != nil {
		if err := appmicrolease.ValidateSafeMetadata(map[string]string(*body.SafeOperationMetadata)); err != nil {
			return NewProblemError(http.StatusBadRequest, "bad request", "unsafe_operation_metadata")
		}
	}
	return nil
}

func validateReadMicroleaseRequest(body *api.MicroleaseReadbackRequest) error {
	if body == nil {
		return NewProblemError(http.StatusBadRequest, "bad request", "request body is required")
	}
	if !body.RouteContractId.Valid() || body.RouteContractId != api.BillingMicroleaseReadback {
		return NewProblemError(http.StatusBadRequest, "bad request", "schema_contract_mismatch")
	}
	if !body.ContractVersion.Valid() || !body.LookupKind.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_request_enum")
	}
	if body.LookupKind == api.IdempotencyKey && (body.IdempotencyKey == nil || *body.IdempotencyKey == "") {
		return NewProblemError(http.StatusBadRequest, "bad request", "idempotency_key_required")
	}
	if body.LookupKind == api.MicroleaseId && (body.MicroleaseId == nil || *body.MicroleaseId == "") {
		return NewProblemError(http.StatusBadRequest, "bad request", "microlease_id_required")
	}
	if body.AccountScopeKey == "" || body.TraceRequestId == "" {
		return NewProblemError(http.StatusBadRequest, "bad request", "required_identity_missing")
	}
	return validateCallerContext(body.CallerContext)
}

func validateCloseMicroleaseRequest(body *api.MicroleaseCloseRequest) error {
	if body == nil {
		return NewProblemError(http.StatusBadRequest, "bad request", "request body is required")
	}
	if !body.RouteContractId.Valid() || body.RouteContractId != api.BillingMicroleaseClose {
		return NewProblemError(http.StatusBadRequest, "bad request", "schema_contract_mismatch")
	}
	if !body.ContractVersion.Valid() || !body.CloseKind.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_request_enum")
	}
	if body.AccountScopeKey == "" || body.IdempotencyKey == "" || body.RequestFingerprint == "" || body.TraceRequestId == "" {
		return NewProblemError(http.StatusBadRequest, "bad request", "required_identity_missing")
	}
	if body.MicroleaseId == "" || body.ProxyAllocatorOwnerId == "" || body.MicroleaseGeneration <= 0 || body.CloseSequence <= 0 {
		return NewProblemError(http.StatusBadRequest, "bad request", "invalid_close_request")
	}
	if body.SafeOperationMetadata != nil {
		if err := appmicrolease.ValidateSafeMetadata(map[string]string(*body.SafeOperationMetadata)); err != nil {
			return NewProblemError(http.StatusBadRequest, "bad request", "unsafe_operation_metadata")
		}
	}
	return validateCallerContext(body.CallerContext)
}

func validateBillingOperationReadbackRequest(body *api.BillingOperationReadbackRequest) error {
	if body == nil {
		return NewProblemError(http.StatusBadRequest, "bad request", "request body is required")
	}
	if !body.RouteContractId.Valid() || body.RouteContractId != api.BillingOperationReadback {
		return NewProblemError(http.StatusBadRequest, "bad request", "schema_contract_mismatch")
	}
	if !body.ContractVersion.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_contract_version")
	}
	if body.AccountScopeKey == "" || body.OperationIdentity == "" || body.TraceRequestId == "" {
		return NewProblemError(http.StatusBadRequest, "bad request", "required_identity_missing")
	}
	return validateCallerContext(body.CallerContext)
}

func validateCallerContext(c api.CallerContext) error {
	if !c.CallerPrincipal.Valid() {
		return NewProblemError(http.StatusBadRequest, "bad request", "unsupported_caller_principal")
	}
	if c.CallerScopeKey == "" || c.RequestId == "" || c.DeadlineAtEpochMs <= 0 {
		return NewProblemError(http.StatusBadRequest, "bad request", "caller_context_incomplete")
	}
	return nil
}
