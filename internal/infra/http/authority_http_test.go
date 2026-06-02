package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Dankosik/billing-service/internal/api"
	"github.com/Dankosik/billing-service/internal/app/billingauthority"
)

type fakeBillingAuthorityService struct {
	resolveRequest       api.AccountResolveRequest
	balanceAccountScope  string
	balanceParams        api.ReadBillingBalanceParams
	reserveRequest       api.UsageReserveRequest
	terminalRequest      api.UsageTerminalRequest
	reversalRequest      api.UsageReversalRequest
	readbackRequest      api.UsageReadbackRequest
	reconciliationParams api.ListReconciliationCasesParams
	adminLedgerScope     string
	adminLedgerParams    api.ReadAdminLedgerParams
	adminExposureScope   string
	resolveCalls         int
	balanceCalls         int
	reserveCalls         int
	finalizeCalls        int
	writeOffCalls        int
	reverseCalls         int
	readbackCalls        int
	reconciliationCalls  int
	adminLedgerCalls     int
	adminExposureCalls   int
	resolveErr           error
	balanceErr           error
	reserveErr           error
	finalizeErr          error
	writeOffErr          error
	reverseErr           error
	readbackErr          error
	reconciliationErr    error
	adminLedgerErr       error
	adminExposureErr     error
}

func (s *fakeBillingAuthorityService) ResolveBillingAccount(_ context.Context, request api.AccountResolveRequest) (api.AccountResolveResponse, error) {
	s.resolveCalls++
	s.resolveRequest = request
	if s.resolveErr != nil {
		return api.AccountResolveResponse{}, s.resolveErr
	}
	return api.AccountResolveResponse{
		AccountScopeKey:     "acct_test",
		AccountState:        api.AccountStateActive,
		BalanceReadEligible: true,
		ImportState:         api.ImportStateAccepted,
		MigrationState:      api.Migrated,
		ResultCode:          api.AccountResolveResultCodeResolved,
	}, nil
}

func (s *fakeBillingAuthorityService) ReadBillingBalance(_ context.Context, accountScopeKey string, params api.ReadBillingBalanceParams) (api.BalanceReadResponse, error) {
	s.balanceCalls++
	s.balanceAccountScope = accountScopeKey
	s.balanceParams = params
	if s.balanceErr != nil {
		return api.BalanceReadResponse{}, s.balanceErr
	}
	return api.BalanceReadResponse{
		AccountScopeKey:  "acct_test",
		BalanceVersion:   1,
		ImportState:      api.ImportStateAccepted,
		ResultCode:       api.BalanceReadResultCodeFound,
		RuntimeGateState: api.RuntimeGateStateReady,
	}, nil
}

func (s *fakeBillingAuthorityService) ReserveUsage(_ context.Context, request api.UsageReserveRequest) (api.UsageOperationResponse, error) {
	s.reserveCalls++
	s.reserveRequest = request
	if s.reserveErr != nil {
		return api.UsageOperationResponse{}, s.reserveErr
	}
	return usageAcceptedResponse(request.UsageOperationId, request.IdempotencyKey, request.RequestFingerprint, api.UsageOperationStateReserved), nil
}

func (s *fakeBillingAuthorityService) FinalizeUsage(_ context.Context, request api.UsageTerminalRequest) (api.UsageOperationResponse, error) {
	s.finalizeCalls++
	s.terminalRequest = request
	if s.finalizeErr != nil {
		return api.UsageOperationResponse{}, s.finalizeErr
	}
	return usageAcceptedResponse(request.UsageOperationId, request.IdempotencyKey, request.RequestFingerprint, api.UsageOperationStateFinalized), nil
}

func (s *fakeBillingAuthorityService) WriteOffUsage(_ context.Context, request api.UsageTerminalRequest) (api.UsageOperationResponse, error) {
	s.writeOffCalls++
	s.terminalRequest = request
	if s.writeOffErr != nil {
		return api.UsageOperationResponse{}, s.writeOffErr
	}
	return usageAcceptedResponse(request.UsageOperationId, request.IdempotencyKey, request.RequestFingerprint, api.UsageOperationStateWrittenOff), nil
}

func (s *fakeBillingAuthorityService) ReverseUsage(_ context.Context, request api.UsageReversalRequest) (api.UsageOperationResponse, error) {
	s.reverseCalls++
	s.reversalRequest = request
	if s.reverseErr != nil {
		return api.UsageOperationResponse{}, s.reverseErr
	}
	return usageAcceptedResponse(request.UsageOperationId, request.IdempotencyKey, request.RequestFingerprint, api.UsageOperationStateReversed), nil
}

func (s *fakeBillingAuthorityService) ReadUsageOperation(_ context.Context, request api.UsageReadbackRequest) (api.UsageOperationReadbackResponse, error) {
	s.readbackCalls++
	s.readbackRequest = request
	if s.readbackErr != nil {
		return api.UsageOperationReadbackResponse{}, s.readbackErr
	}
	return api.UsageOperationReadbackResponse{
		ResultCode:       api.Readback,
		State:            api.UsageOperationStateReserved,
		UsageOperationId: request.UsageOperationId,
	}, nil
}

func (s *fakeBillingAuthorityService) ListReconciliationCases(_ context.Context, params api.ListReconciliationCasesParams) (api.ReconciliationCasesResponse, error) {
	s.reconciliationCalls++
	s.reconciliationParams = params
	if s.reconciliationErr != nil {
		return api.ReconciliationCasesResponse{}, s.reconciliationErr
	}
	return api.ReconciliationCasesResponse{
		Cases: []api.ReconciliationCaseSummary{{
			AccountScopeKey:      "acct_test",
			ReconciliationCaseId: "case-1",
			Reason:               api.StaleMicrolease,
			Severity:             api.ReconciliationCaseSummarySeverityCritical,
			State:                api.ReconciliationCaseSummaryStateOpen,
		}},
		ResultCode: api.ReconciliationCasesResponseResultCodeListed,
	}, nil
}

func (s *fakeBillingAuthorityService) ReadAdminLedger(_ context.Context, accountScopeKey string, params api.ReadAdminLedgerParams) (api.AdminLedgerResponse, error) {
	s.adminLedgerCalls++
	s.adminLedgerScope = accountScopeKey
	s.adminLedgerParams = params
	if s.adminLedgerErr != nil {
		return api.AdminLedgerResponse{}, s.adminLedgerErr
	}
	return api.AdminLedgerResponse{
		AccountScopeKey: accountScopeKey,
		Entries:         []api.LedgerEntrySummary{{AmountUsdAtoms: -10, BalanceVersionAfter: 8, EffectType: "usage_charge", LedgerEntryId: "ledger-1"}},
		ResultCode:      api.AdminLedgerResponseResultCodeFound,
	}, nil
}

func (s *fakeBillingAuthorityService) ReadAdminExposure(_ context.Context, accountScopeKey string) (api.AdminExposureResponse, error) {
	s.adminExposureCalls++
	s.adminExposureScope = accountScopeKey
	if s.adminExposureErr != nil {
		return api.AdminExposureResponse{}, s.adminExposureErr
	}
	return api.AdminExposureResponse{AccountScopeKey: accountScopeKey, ActiveMicroleaseUsdAtoms: 90, ResultCode: api.AdminExposureResponseResultCodeFound, RuntimeGateState: api.RuntimeGateStateReady}, nil
}

func TestProtectedBillingAuthorityRoutesReadAndTerminalSurfaces(t *testing.T) {
	t.Parallel()

	service := &fakeBillingAuthorityService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Authority: service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{
			scopeAccountsResolve,
			scopeBalancesRead,
			scopeUsageWrite,
			scopeUsageRead,
			scopeReconciliationRead,
			scopeAdminRead,
		}},
	}, nil, RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	expectOKJSONStatus(t, serveJSON(t, router, http.MethodPost, "/internal/billing/v1/accounts/resolve", validAccountResolveRequest()))
	if service.resolveCalls != 1 || service.resolveRequest.RouteContractId != api.BillingAccountsResolve {
		t.Fatalf("resolve calls=%d request=%+v, want route contract", service.resolveCalls, service.resolveRequest)
	}

	balancePath := "/internal/billing/v1/accounts/acct_test/balance?representedSubjectId=user_1&traceRequestId=trace-balance&deadlineAtEpochMs=1780315210000"
	expectOKJSONStatus(t, serveJSON(t, router, http.MethodGet, balancePath, nil))
	if service.balanceCalls != 1 || service.balanceAccountScope != "acct_test" || service.balanceParams.RepresentedSubjectId != "user_1" {
		t.Fatalf("balance calls=%d scope=%q params=%+v, want acct_test user_1", service.balanceCalls, service.balanceAccountScope, service.balanceParams)
	}

	finalize := validUsageTerminalRequest(api.BillingUsageFinalize)
	expectOKJSONStatus(t, serveJSON(t, router, http.MethodPost, "/internal/billing/v1/usage/finalizations", finalize))
	if service.finalizeCalls != 1 || service.terminalRequest.RouteContractId != api.BillingUsageFinalize {
		t.Fatalf("finalize calls=%d request=%+v, want finalize route", service.finalizeCalls, service.terminalRequest)
	}

	writeOff := validUsageTerminalRequest(api.BillingUsageWriteOff)
	writeOff.ChargedUsdAtoms = 0
	writeOff.ReleasedUsdAtoms = 0
	writeOff.WriteOffUsdAtoms = 100
	expectOKJSONStatus(t, serveJSON(t, router, http.MethodPost, "/internal/billing/v1/usage/write-offs", writeOff))
	if service.writeOffCalls != 1 || service.terminalRequest.RouteContractId != api.BillingUsageWriteOff {
		t.Fatalf("write-off calls=%d request=%+v, want write-off route", service.writeOffCalls, service.terminalRequest)
	}

	expectOKJSONStatus(t, serveJSON(t, router, http.MethodPost, "/internal/billing/v1/usage/reversals", validUsageReversalRequest()))
	if service.reverseCalls != 1 || service.reversalRequest.OriginalLedgerEntryId != "ledger-1" {
		t.Fatalf("reverse calls=%d request=%+v, want ledger-1", service.reverseCalls, service.reversalRequest)
	}

	expectOKJSONStatus(t, serveJSON(t, router, http.MethodPost, "/internal/billing/v1/usage/readback", validUsageReadbackRequest()))
	if service.readbackCalls != 1 || service.readbackRequest.UsageOperationId != "usage-op-1" {
		t.Fatalf("readback calls=%d request=%+v, want usage-op-1", service.readbackCalls, service.readbackRequest)
	}

	reconciliationPath := "/internal/billing/v1/reconciliation/cases?accountScopeKey=acct_test&state=open&severity=critical"
	expectOKJSONStatus(t, serveJSON(t, router, http.MethodGet, reconciliationPath, nil))
	if service.reconciliationCalls != 1 || service.reconciliationParams.AccountScopeKey == nil || *service.reconciliationParams.AccountScopeKey != "acct_test" {
		t.Fatalf("reconciliation calls=%d params=%+v, want account scope", service.reconciliationCalls, service.reconciliationParams)
	}

	expectOKJSONStatus(t, serveJSON(t, router, http.MethodGet, "/internal/billing/v1/admin/accounts/acct_test/ledger?limit=25", nil))
	if service.adminLedgerCalls != 1 || service.adminLedgerScope != "acct_test" || service.adminLedgerParams.Limit == nil || *service.adminLedgerParams.Limit != 25 {
		t.Fatalf("ledger calls=%d scope=%q params=%+v, want limit 25", service.adminLedgerCalls, service.adminLedgerScope, service.adminLedgerParams)
	}

	expectOKJSONStatus(t, serveJSON(t, router, http.MethodGet, "/internal/billing/v1/admin/accounts/acct_test/exposure", nil))
	if service.adminExposureCalls != 1 || service.adminExposureScope != "acct_test" {
		t.Fatalf("exposure calls=%d scope=%q, want acct_test", service.adminExposureCalls, service.adminExposureScope)
	}
}

func TestProtectedBillingAuthorityTerminalValidationSuppressesServiceCalls(t *testing.T) {
	t.Parallel()

	service := &fakeBillingAuthorityService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Authority:            service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeUsageWrite}},
	}, nil, RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	request := validUsageTerminalRequest(api.BillingUsageFinalize)
	request.RouteContractId = api.BillingUsageWriteOff
	resp := serveJSON(t, router, http.MethodPost, "/internal/billing/v1/usage/finalizations", request)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if service.finalizeCalls != 0 {
		t.Fatalf("finalize calls = %d, want 0", service.finalizeCalls)
	}
	assertProblemContentType(t, resp.Header())

	request = validUsageTerminalRequest(api.BillingUsageWriteOff)
	request.ChargedUsdAtoms = 1
	request.WriteOffUsdAtoms = 100
	resp = serveJSON(t, router, http.MethodPost, "/internal/billing/v1/usage/write-offs", request)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if service.writeOffCalls != 0 {
		t.Fatalf("write-off calls = %d, want 0", service.writeOffCalls)
	}
	assertBodyExcludesSensitiveValues(t, resp.Body.String())
}

func TestProtectedBillingAuthorityRejectsDirectReserveFallbackBeforeServiceCall(t *testing.T) {
	t.Parallel()

	service := &fakeBillingAuthorityService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Authority:            service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeUsageWrite}},
	}, nil, RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	request := validUsageReserveRequest()
	request.AuthorityMode = api.DirectReserveFallback
	resp := postReserveJSON(t, router, request)

	if resp.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusUnprocessableEntity, resp.Body.String())
	}
	if service.reserveCalls != 0 {
		t.Fatalf("reserve calls = %d, want 0", service.reserveCalls)
	}
	assertProblemContentType(t, resp.Header())
	assertBodyExcludesSensitiveValues(t, resp.Body.String())
	if !strings.Contains(resp.Body.String(), "direct_reserve_fallback_rejected") {
		t.Fatalf("body = %s, want direct reserve fallback rejection", resp.Body.String())
	}
}

func TestProtectedBillingAuthorityAcceptsMicroleaseBackedReserve(t *testing.T) {
	t.Parallel()

	service := &fakeBillingAuthorityService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Authority:            service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeUsageWrite}},
	}, nil, RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	resp := postReserveJSON(t, router, validUsageReserveRequest())

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if service.reserveCalls != 1 {
		t.Fatalf("reserve calls = %d, want 1", service.reserveCalls)
	}
	if service.reserveRequest.AuthorityMode != api.BillingMicroleaseWithProxyChildDebit {
		t.Fatalf("authority mode = %q, want microlease child debit", service.reserveRequest.AuthorityMode)
	}
	var body api.UsageOperationResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ResultCode != api.Accepted || body.State != api.UsageOperationStateReserved || body.UsageOperationId != "usage-op-1" {
		t.Fatalf("response = %+v, want accepted reserved usage-op-1", body)
	}
}

func TestProtectedBillingAuthorityHandlersMapServiceErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		method     string
		path       string
		body       any
		mutate     func(*fakeBillingAuthorityService)
		wantStatus int
		wantCalls  func(*fakeBillingAuthorityService) int
	}{
		{
			name:   "account resolve dependency unavailable",
			method: http.MethodPost,
			path:   "/internal/billing/v1/accounts/resolve",
			body:   validAccountResolveRequest(),
			mutate: func(service *fakeBillingAuthorityService) {
				service.resolveErr = NewProblemError(http.StatusServiceUnavailable, "not ready", "dependency_not_ready")
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCalls:  func(service *fakeBillingAuthorityService) int { return service.resolveCalls },
		},
		{
			name:   "balance read locked by migration gate",
			method: http.MethodGet,
			path:   "/internal/billing/v1/accounts/acct_test/balance?representedSubjectId=user_1&traceRequestId=trace-balance&deadlineAtEpochMs=1780315210000",
			mutate: func(service *fakeBillingAuthorityService) {
				service.balanceErr = NewProblemError(http.StatusLocked, "locked", "account_not_migrated")
			},
			wantStatus: http.StatusLocked,
			wantCalls:  func(service *fakeBillingAuthorityService) int { return service.balanceCalls },
		},
		{
			name:   "reversal conflict",
			method: http.MethodPost,
			path:   "/internal/billing/v1/usage/reversals",
			body:   validUsageReversalRequest(),
			mutate: func(service *fakeBillingAuthorityService) {
				service.reverseErr = NewProblemError(http.StatusConflict, "conflict", "reversal_replay_conflict")
			},
			wantStatus: http.StatusConflict,
			wantCalls:  func(service *fakeBillingAuthorityService) int { return service.reverseCalls },
		},
		{
			name:   "usage readback unexpected error",
			method: http.MethodPost,
			path:   "/internal/billing/v1/usage/readback",
			body:   validUsageReadbackRequest(),
			mutate: func(service *fakeBillingAuthorityService) {
				service.readbackErr = errors.New("readback store includes raw-prompt-secret but response must not leak")
			},
			wantStatus: http.StatusInternalServerError,
			wantCalls:  func(service *fakeBillingAuthorityService) int { return service.readbackCalls },
		},
		{
			name:   "admin exposure dependency unavailable",
			method: http.MethodGet,
			path:   "/internal/billing/v1/admin/accounts/acct_test/exposure",
			mutate: func(service *fakeBillingAuthorityService) {
				service.adminExposureErr = NewProblemError(http.StatusServiceUnavailable, "not ready", "dependency_not_ready")
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCalls:  func(service *fakeBillingAuthorityService) int { return service.adminExposureCalls },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeBillingAuthorityService{}
			tc.mutate(service)
			router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
				Authority: service,
				ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{
					scopeAccountsResolve,
					scopeBalancesRead,
					scopeUsageRead,
					scopeUsageWrite,
					scopeAdminRead,
				}},
			}, nil, RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

			resp := serveJSON(t, router, tc.method, tc.path, tc.body)
			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", resp.Code, tc.wantStatus, resp.Body.String())
			}
			if got := tc.wantCalls(service); got != 1 {
				t.Fatalf("service calls = %d, want 1", got)
			}
			assertProblemContentType(t, resp.Header())
			assertBodyExcludesSensitiveValues(t, resp.Body.String())
		})
	}
}

func TestProtectedBillingAuthorityRequiresRouteScopeBeforeServiceCall(t *testing.T) {
	t.Parallel()

	service := &fakeBillingAuthorityService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Authority:            service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeUsageRead}},
	}, nil, RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	resp := postReserveJSON(t, router, validUsageReserveRequest())

	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusForbidden, resp.Body.String())
	}
	if service.reserveCalls != 0 {
		t.Fatalf("reserve calls = %d, want 0", service.reserveCalls)
	}
	assertProblemContentType(t, resp.Header())
}

func TestProtectedBillingAuthorityValidatesRequiredIdentitiesBeforeServiceCall(t *testing.T) {
	t.Parallel()

	service := &fakeBillingAuthorityService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Authority:            service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeUsageWrite}},
	}, nil, RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	request := validUsageReserveRequest()
	request.IdempotencyKey = ""
	resp := postReserveJSON(t, router, request)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if service.reserveCalls != 0 {
		t.Fatalf("reserve calls = %d, want 0", service.reserveCalls)
	}
	assertProblemContentType(t, resp.Header())
}

func TestBillingAuthorityHTTPAdapterMapsAppSnapshotsAndErrors(t *testing.T) {
	t.Parallel()

	store := &adapterAuthorityStore{}
	svc, err := billingauthority.New(store)
	if err != nil {
		t.Fatalf("billingauthority.New() error = %v", err)
	}
	adapter, err := NewBillingAuthorityHTTPAdapter(svc)
	if err != nil {
		t.Fatalf("NewBillingAuthorityHTTPAdapter() error = %v", err)
	}

	account, err := adapter.ResolveBillingAccount(context.Background(), validAccountResolveRequest())
	if err != nil {
		t.Fatalf("ResolveBillingAccount() error = %v", err)
	}
	if account.ResultCode != api.AccountResolveResultCodeReconcileRequired || account.BalanceVersion == nil || *account.BalanceVersion != 9 {
		t.Fatalf("ResolveBillingAccount() = %+v, want reconcile-required version 9", account)
	}

	balance, err := adapter.ReadBillingBalance(context.Background(), "acct_test", api.ReadBillingBalanceParams{
		RepresentedSubjectId: "user_1",
		TraceRequestId:       "trace-balance",
		DeadlineAtEpochMs:    time.Date(2026, 6, 1, 12, 0, 10, 0, time.UTC).UnixMilli(),
	})
	if err != nil {
		t.Fatalf("ReadBillingBalance() error = %v", err)
	}
	if balance.ResultCode != api.BalanceReadResultCodeManualReview || balance.ManualReview == nil || !*balance.ManualReview {
		t.Fatalf("ReadBillingBalance() = %+v, want manual review flag", balance)
	}

	reserveRequest := validUsageReserveRequest()
	metadata := api.SafeOperationMetadata{"surface": "chat"}
	reserveRequest.SafeOperationMetadata = &metadata
	reserved, err := adapter.ReserveUsage(context.Background(), reserveRequest)
	if err != nil {
		t.Fatalf("ReserveUsage() error = %v", err)
	}
	if reserved.ReplayIdentity.StoredOutcomeId != "outcome-1" || store.reserve.Metadata["surface"] != "chat" {
		t.Fatalf("ReserveUsage() = %+v command=%+v, want replay identity and safe metadata", reserved, store.reserve)
	}

	finalized, err := adapter.FinalizeUsage(context.Background(), validUsageTerminalRequest(api.BillingUsageFinalize))
	if err != nil {
		t.Fatalf("FinalizeUsage() error = %v", err)
	}
	if finalized.State != api.UsageOperationStateFinalized || store.terminal.TerminalKind != "finalize" {
		t.Fatalf("FinalizeUsage() = %+v terminal=%+v, want finalized", finalized, store.terminal)
	}

	reversed, err := adapter.ReverseUsage(context.Background(), validUsageReversalRequest())
	if err != nil {
		t.Fatalf("ReverseUsage() error = %v", err)
	}
	if reversed.State != api.UsageOperationStateReversed || store.reversal.OriginalLedgerEntryID != "ledger-1" {
		t.Fatalf("ReverseUsage() = %+v command=%+v, want reversed ledger-1", reversed, store.reversal)
	}

	readback, err := adapter.ReadUsageOperation(context.Background(), validUsageReadbackRequest())
	if err != nil {
		t.Fatalf("ReadUsageOperation() error = %v", err)
	}
	if readback.ReplayIdentity == nil || readback.ReplayIdentity.StoredOutcomeId != "outcome-1" {
		t.Fatalf("ReadUsageOperation() = %+v, want replay identity", readback)
	}

	cases, err := adapter.ListReconciliationCases(context.Background(), api.ListReconciliationCasesParams{})
	if err != nil {
		t.Fatalf("ListReconciliationCases() error = %v", err)
	}
	if len(cases.Cases) != 1 || cases.Cases[0].SafeLineageId == nil || *cases.Cases[0].SafeLineageId != "child-1" {
		t.Fatalf("ListReconciliationCases() = %+v, want safe lineage", cases)
	}

	ledgerLimit := 10
	ledger, err := adapter.ReadAdminLedger(context.Background(), "acct_test", api.ReadAdminLedgerParams{Limit: &ledgerLimit})
	if err != nil {
		t.Fatalf("ReadAdminLedger() error = %v", err)
	}
	if len(ledger.Entries) != 1 || ledger.Entries[0].LedgerEntryId != "ledger-1" {
		t.Fatalf("ReadAdminLedger() = %+v, want ledger-1", ledger)
	}

	exposure, err := adapter.ReadAdminExposure(context.Background(), "acct_test")
	if err != nil {
		t.Fatalf("ReadAdminExposure() error = %v", err)
	}
	if exposure.ActiveMicroleaseUsdAtoms != 70 || exposure.UnresolvedChildDebitUsdAtoms != 20 {
		t.Fatalf("ReadAdminExposure() = %+v, want active and unresolved exposure", exposure)
	}

	store.err = billingauthority.ErrConflict
	var problem ProblemError
	if _, err := adapter.ReserveUsage(context.Background(), validUsageReserveRequest()); !errors.As(err, &problem) || problem.Status != http.StatusConflict {
		t.Fatalf("ReserveUsage(conflict) error = %v, want ProblemError", err)
	}
}

func usageAcceptedResponse(usageOperationID, idempotencyKey, fingerprint string, state api.UsageOperationState) api.UsageOperationResponse {
	return api.UsageOperationResponse{
		ReplayIdentity: api.ReplayIdentity{
			IdempotencyKey:     idempotencyKey,
			RequestFingerprint: fingerprint,
			StoredOutcomeId:    "stored-outcome-1",
		},
		ResultCode:       api.Accepted,
		State:            state,
		UsageOperationId: usageOperationID,
	}
}

func validUsageReserveRequest() api.UsageReserveRequest {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	subjectID := "user_1"
	return api.UsageReserveRequest{
		AccountScopeKey:        "acct_test",
		AuthorityMode:          api.BillingMicroleaseWithProxyChildDebit,
		ChildCapUsdAtoms:       10_000_000,
		ChildSequence:          1,
		ContractVersion:        api.V1,
		DeadlineAtEpochMs:      now.Add(10 * time.Second).UnixMilli(),
		DebitAuthorizationId:   "debit-auth-1",
		IdempotencyKey:         "usage-reserve-idem-1",
		LeaseFence:             "fence-1",
		MicroleaseChildDebitId: "22222222-2222-2222-2222-222222222222",
		MicroleaseGeneration:   1,
		MicroleaseId:           "11111111-1111-1111-1111-111111111111",
		PricingSnapshot:        validIssueRequest().PricingSnapshot,
		ProxyAllocatorOwnerId:  "proxy-owner-1",
		RepresentedUserContext: api.RepresentedUserContext{
			SubjectId: &subjectID,
		},
		RequestFingerprint: "usage-reserve-fingerprint-1",
		RouteContractId:    api.BillingUsageReserve,
		TraceRequestId:     "trace-usage-reserve-1",
		UsageOperationId:   "usage-op-1",
		CallerContext: api.CallerContext{
			CallerPrincipal:   api.SvcGonkaProxy,
			CallerScopeKey:    "proxy-owner-1",
			DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
			RequestId:         "caller-request-usage-1",
		},
	}
}

func validAccountResolveRequest() api.AccountResolveRequest {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	subjectID := "user_1"
	return api.AccountResolveRequest{
		ContractVersion:   api.V1,
		DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
		RepresentedUserContext: api.RepresentedUserContext{
			SubjectId: &subjectID,
		},
		RouteContractId: api.BillingAccountsResolve,
		TraceRequestId:  "trace-account-resolve",
		CallerContext: api.CallerContext{
			CallerPrincipal:   api.SvcGonkaProxy,
			CallerScopeKey:    "proxy-owner-1",
			DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
			RequestId:         "caller-request-account-1",
		},
	}
}

func validUsageTerminalRequest(routeID api.UsageTerminalRequestRouteContractId) api.UsageTerminalRequest {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	evidenceID := "qualified-evidence-1"
	return api.UsageTerminalRequest{
		AccountScopeKey:              "acct_test",
		ChargedUsdAtoms:              4_000_000,
		ContractVersion:              api.V1,
		DeadlineAtEpochMs:            now.Add(10 * time.Second).UnixMilli(),
		DebitAuthorizationId:         "debit-auth-1",
		IdempotencyKey:               "usage-terminal-idem-1",
		MicroleaseChildDebitId:       "22222222-2222-2222-2222-222222222222",
		MicroleaseId:                 "11111111-1111-1111-1111-111111111111",
		PricingSnapshot:              validIssueRequest().PricingSnapshot,
		QualifiedInferenceEvidenceId: &evidenceID,
		ReleasedUsdAtoms:             6_000_000,
		RequestFingerprint:           "usage-terminal-fingerprint-1",
		RouteContractId:              routeID,
		TerminalFingerprint:          "terminal-basis-1",
		TerminalOutcomeId:            "terminal-outcome-1",
		TraceRequestId:               "trace-usage-terminal-1",
		UsageOperationId:             "usage-op-1",
		CallerContext: api.CallerContext{
			CallerPrincipal:   api.SvcGonkaProxy,
			CallerScopeKey:    "proxy-owner-1",
			DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
			RequestId:         "caller-request-terminal-1",
		},
	}
}

func validUsageReversalRequest() api.UsageReversalRequest {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	return api.UsageReversalRequest{
		AccountScopeKey:       "acct_test",
		ContractVersion:       api.V1,
		DeadlineAtEpochMs:     now.Add(10 * time.Second).UnixMilli(),
		IdempotencyKey:        "usage-reversal-idem-1",
		OriginalLedgerEntryId: "ledger-1",
		ReasonCode:            "operator_reversal",
		RequestFingerprint:    "usage-reversal-fingerprint-1",
		ReversalUsdAtoms:      100,
		RouteContractId:       api.BillingUsageReversal,
		TraceRequestId:        "trace-usage-reversal-1",
		UsageOperationId:      "usage-op-1",
		CallerContext: api.CallerContext{
			CallerPrincipal:   api.SvcGonkaProxy,
			CallerScopeKey:    "proxy-owner-1",
			DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
			RequestId:         "caller-request-reversal-1",
		},
	}
}

func validUsageReadbackRequest() api.UsageReadbackRequest {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	return api.UsageReadbackRequest{
		AccountScopeKey:   "acct_test",
		ContractVersion:   api.V1,
		DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
		RouteContractId:   api.BillingUsageReadback,
		TraceRequestId:    "trace-usage-readback-1",
		UsageOperationId:  "usage-op-1",
		CallerContext: api.CallerContext{
			CallerPrincipal:   api.SvcGonkaProxy,
			CallerScopeKey:    "proxy-owner-1",
			DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
			RequestId:         "caller-request-readback-1",
		},
	}
}

func postReserveJSON(t *testing.T, router http.Handler, body any) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/usage/reservations", strings.NewReader(mustJSON(t, body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-service-token")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func serveJSON(t *testing.T, router http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body == nil {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(mustJSON(t, body))
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer valid-service-token")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	return resp
}

func expectOKJSONStatus(t *testing.T, resp *httptest.ResponseRecorder) {
	t.Helper()
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if !strings.Contains(resp.Header().Get("Content-Type"), "application/json") {
		t.Fatalf("content-type = %q, want application/json", resp.Header().Get("Content-Type"))
	}
}

type adapterAuthorityStore struct {
	reserve  billingauthority.UsageReserveCommand
	terminal billingauthority.UsageTerminalCommand
	reversal billingauthority.UsageReversalCommand
	err      error
}

func (s *adapterAuthorityStore) ResolveAccount(context.Context, billingauthority.AccountResolveRequest) (billingauthority.AccountSnapshot, error) {
	return billingauthority.AccountSnapshot{
		AccountID:           "billing-account-1",
		AccountScopeKey:     "acct_test",
		AccountState:        "active",
		ImportState:         "mismatch",
		MigrationState:      "migrated",
		BalanceReadEligible: true,
		BalanceVersion:      9,
		FailureClass:        "import_mismatch",
		Retryable:           true,
	}, nil
}

func (s *adapterAuthorityStore) ReadBalance(context.Context, billingauthority.BalanceReadRequest) (billingauthority.BalanceSnapshot, error) {
	return billingauthority.BalanceSnapshot{
		AccountID:                    "billing-account-1",
		AccountScopeKey:              "acct_test",
		AccountState:                 "manual_review",
		SettledUSDAtoms:              100,
		ReservedUSDAtoms:             30,
		AvailableUSDAtoms:            70,
		BalanceVersion:               9,
		ImportState:                  "accepted",
		RuntimeGateState:             "ready",
		ActiveMicroleaseUSDAtoms:     20,
		UnresolvedChildDebitUSDAtoms: 10,
		ManualReview:                 true,
		ReconciliationRequired:       true,
		ReasonCode:                   "manual_review",
	}, nil
}

func (s *adapterAuthorityStore) ReserveUsage(_ context.Context, cmd billingauthority.UsageReserveCommand) (billingauthority.UsageOperationSnapshot, error) {
	s.reserve = cmd
	if s.err != nil {
		return billingauthority.UsageOperationSnapshot{}, s.err
	}
	return adapterUsageSnapshot(cmd.UsageOperationID, "reserved"), nil
}

func (s *adapterAuthorityStore) CompleteUsage(_ context.Context, cmd billingauthority.UsageTerminalCommand) (billingauthority.UsageOperationSnapshot, error) {
	s.terminal = cmd
	if s.err != nil {
		return billingauthority.UsageOperationSnapshot{}, s.err
	}
	return adapterUsageSnapshot(cmd.UsageOperationID, cmd.TerminalKind+"d"), nil
}

func (s *adapterAuthorityStore) ReverseUsage(_ context.Context, cmd billingauthority.UsageReversalCommand) (billingauthority.UsageOperationSnapshot, error) {
	s.reversal = cmd
	if s.err != nil {
		return billingauthority.UsageOperationSnapshot{}, s.err
	}
	return adapterUsageSnapshot(cmd.UsageOperationID, "reversed"), nil
}

func (s *adapterAuthorityStore) ReadUsageOperation(_ context.Context, req billingauthority.UsageReadbackRequest) (billingauthority.UsageOperationSnapshot, error) {
	return adapterUsageSnapshot(req.UsageOperationID, "reserved"), nil
}

func (s *adapterAuthorityStore) ListReconciliationCases(context.Context, billingauthority.ReconciliationCasesRequest) ([]billingauthority.ReconciliationCase, error) {
	return []billingauthority.ReconciliationCase{{
		ReconciliationCaseID: "case-1",
		AccountScopeKey:      "acct_test",
		Reason:               "stale_microlease",
		State:                "open",
		Severity:             "critical",
		SafeLineageID:        "child-1",
	}}, nil
}

func (s *adapterAuthorityStore) ListLedgerEntries(context.Context, billingauthority.AdminLedgerRequest) ([]billingauthority.LedgerEntry, error) {
	return []billingauthority.LedgerEntry{{LedgerEntryID: "ledger-1", EffectType: "usage_charge", AmountUSDAtoms: -10, BalanceVersionAfter: 8}}, nil
}

func (s *adapterAuthorityStore) ReadExposure(context.Context, billingauthority.AdminExposureRequest) (billingauthority.ExposureSnapshot, error) {
	return billingauthority.ExposureSnapshot{
		AccountScopeKey:              "acct_test",
		RuntimeGateState:             "ready",
		ActiveMicroleaseUSDAtoms:     70,
		ActiveUsageHoldUSDAtoms:      5,
		UnresolvedChildDebitUSDAtoms: 20,
	}, nil
}

func adapterUsageSnapshot(usageOperationID string, state string) billingauthority.UsageOperationSnapshot {
	return billingauthority.UsageOperationSnapshot{
		UsageOperationID:     usageOperationID,
		BillingOperationID:   "billing-op-1",
		State:                state,
		ResultCode:           string(api.Accepted),
		ReasonCode:           "accepted",
		ReconciliationCaseID: "case-1",
		IdempotencyKey:       "idem-1",
		RequestFingerprint:   "fingerprint-1",
		StoredOutcomeID:      "outcome-1",
	}
}
