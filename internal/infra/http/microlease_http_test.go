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
	"github.com/Dankosik/billing-service/internal/infra/telemetry"
)

type headerScopeAuthenticator struct {
	scopes []string
}

func (a headerScopeAuthenticator) AuthenticateService(_ context.Context, r *http.Request) (ServicePrincipal, error) {
	if r.Header.Get("Authorization") != "Bearer valid-service-token" {
		return ServicePrincipal{}, ErrServiceAuthMissing
	}
	return ServicePrincipal{Subject: "svc:gonka-proxy", Scopes: a.scopes}, nil
}

type fakeMicroleaseService struct {
	issueRequest     api.MicroleaseIssueRequest
	readRequest      api.MicroleaseReadbackRequest
	closeRequest     api.MicroleaseCloseRequest
	operationRequest api.BillingOperationReadbackRequest
	issueErr         error
	readErr          error
	closeErr         error
	operationErr     error
	issueCalls       int
	readCalls        int
	closeCalls       int
	operationCalls   int
}

func (s *fakeMicroleaseService) IssueMicrolease(_ context.Context, request api.MicroleaseIssueRequest) (api.MicroleaseIssueResponse, error) {
	s.issueCalls++
	s.issueRequest = request
	if s.issueErr != nil {
		return api.MicroleaseIssueResponse{}, s.issueErr
	}
	return api.MicroleaseIssueResponse{
		Microlease: &api.MicroleaseGrant{
			AccountScopeKey:            request.AccountScopeKey,
			DebitCutoffAt:              time.UnixMilli(request.DeadlineAtEpochMs).UTC().Add(10 * time.Second),
			ExpiresAt:                  time.UnixMilli(request.DeadlineAtEpochMs).UTC().Add(15 * time.Second),
			IssuedCapUsdAtoms:          request.MaxRequestedUsdAtoms,
			LeaseFence:                 1,
			MicroleaseGeneration:       1,
			MicroleaseId:               "11111111-1111-1111-1111-111111111111",
			MicroleasePolicyVersion:    "microlease-policy-v1",
			PricingSnapshotFingerprint: request.PricingSnapshot.SnapshotFingerprint,
			PricingSnapshotId:          request.PricingSnapshot.PricingSnapshotId,
			ProxyAllocatorOwnerId:      request.ProxyAllocatorOwnerId,
			RemainingUsdAtoms:          request.MaxRequestedUsdAtoms,
			State:                      api.MicroleaseGrantStateActive,
		},
		ReplayIdentity: api.ReplayIdentity{
			IdempotencyKey:     request.IdempotencyKey,
			RequestFingerprint: request.RequestFingerprint,
			StoredOutcomeId:    "22222222-2222-2222-2222-222222222222",
		},
		ResultCode: api.MicroleaseIssueResultCodeIssued,
	}, nil
}

func (s *fakeMicroleaseService) ReadMicrolease(_ context.Context, request api.MicroleaseReadbackRequest) (api.MicroleaseReadbackResponse, error) {
	s.readCalls++
	s.readRequest = request
	if s.readErr != nil {
		return api.MicroleaseReadbackResponse{}, s.readErr
	}
	return api.MicroleaseReadbackResponse{ResultCode: api.MicroleaseReadbackResultCodeFound}, nil
}

func (s *fakeMicroleaseService) CloseMicrolease(_ context.Context, request api.MicroleaseCloseRequest) (api.MicroleaseCloseResponse, error) {
	s.closeCalls++
	s.closeRequest = request
	if s.closeErr != nil {
		return api.MicroleaseCloseResponse{}, s.closeErr
	}
	return api.MicroleaseCloseResponse{
		MicroleaseId:               &request.MicroleaseId,
		ReleasedUsdAtoms:           request.LocalRemainingUsdAtoms,
		ReplayIdentity:             api.ReplayIdentity{IdempotencyKey: request.IdempotencyKey, RequestFingerprint: request.RequestFingerprint, StoredOutcomeId: "33333333-3333-3333-3333-333333333333"},
		ResultCode:                 api.MicroleaseCloseResultCodeClosedReleased,
		UnresolvedReservedUsdAtoms: 0,
	}, nil
}

func (s *fakeMicroleaseService) ReadBillingOperation(_ context.Context, request api.BillingOperationReadbackRequest) (api.BillingOperationReadbackResponse, error) {
	s.operationCalls++
	s.operationRequest = request
	if s.operationErr != nil {
		return api.BillingOperationReadbackResponse{}, s.operationErr
	}
	return api.BillingOperationReadbackResponse{
		OperationIdentity: &request.OperationIdentity,
		ResultCode:        api.BillingOperationReadbackResultCodeFound,
	}, nil
}

func TestProtectedMicroleaseHTTPAuthAndMapping(t *testing.T) {
	t.Parallel()

	service := &fakeMicroleaseService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Microleases:          service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeMicroleaseWrite, scopeMicroleaseRead}},
	}, telemetry.New(), RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	req := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/microleases/issue", strings.NewReader(mustJSON(t, validIssueRequest())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-service-token")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if service.issueCalls != 1 {
		t.Fatalf("issue calls = %d, want 1", service.issueCalls)
	}
	if service.issueRequest.IdempotencyKey != "idem-issue-1" {
		t.Fatalf("idempotency key = %q, want body idempotency key", service.issueRequest.IdempotencyKey)
	}

	var response api.MicroleaseIssueResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.ResultCode != api.MicroleaseIssueResultCodeIssued {
		t.Fatalf("result code = %q, want issued", response.ResultCode)
	}
}

func TestProtectedMicroleaseHTTPRejectsMissingAndWrongScopes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		authHeader  string
		scopes      []string
		wantStatus  int
		wantNoCall  bool
		wantProblem string
	}{
		{
			name:        "missing auth",
			wantStatus:  http.StatusUnauthorized,
			wantNoCall:  true,
			wantProblem: "service authentication required",
		},
		{
			name:        "wrong scope",
			authHeader:  "Bearer valid-service-token",
			scopes:      []string{scopeMicroleaseRead},
			wantStatus:  http.StatusForbidden,
			wantNoCall:  true,
			wantProblem: "required service scope is missing",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeMicroleaseService{}
			router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
				Microleases:          service,
				ServiceAuthenticator: headerScopeAuthenticator{scopes: tc.scopes},
			}, telemetry.New(), RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

			req := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/microleases/issue", strings.NewReader(mustJSON(t, validIssueRequest())))
			req.Header.Set("Content-Type", "application/json")
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", resp.Code, tc.wantStatus, resp.Body.String())
			}
			if tc.wantNoCall && service.issueCalls != 0 {
				t.Fatalf("issue calls = %d, want 0", service.issueCalls)
			}
			assertProblemContentType(t, resp.Header())
			if !strings.Contains(resp.Body.String(), tc.wantProblem) {
				t.Fatalf("problem body = %s, want detail %q", resp.Body.String(), tc.wantProblem)
			}
		})
	}
}

func TestProtectedMicroleaseHTTPValidatesContractBeforeCallingService(t *testing.T) {
	t.Parallel()

	service := &fakeMicroleaseService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Microleases:          service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeMicroleaseWrite}},
	}, telemetry.New(), RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	request := validIssueRequest()
	request.RouteContractId = "billing.microlease.close"
	req := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/microleases/issue", strings.NewReader(mustJSON(t, request)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-service-token")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if service.issueCalls != 0 {
		t.Fatalf("issue calls = %d, want 0", service.issueCalls)
	}
	assertProblemContentType(t, resp.Header())
	assertBodyExcludesSensitiveValues(t, resp.Body.String())
}

func TestProtectedMicroleaseIssueRequiresImmutablePricingEvidence(t *testing.T) {
	t.Parallel()

	service := &fakeMicroleaseService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Microleases:          service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeMicroleaseWrite}},
	}, telemetry.New(), RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	request := validIssueRequest()
	request.PricingSnapshot.SnapshotFingerprint = ""
	req := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/microleases/issue", strings.NewReader(mustJSON(t, request)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-service-token")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if service.issueCalls != 0 {
		t.Fatalf("issue calls = %d, want 0", service.issueCalls)
	}
	if !strings.Contains(resp.Body.String(), "pricing_snapshot_evidence_required") {
		t.Fatalf("body = %s, want pricing evidence problem", resp.Body.String())
	}
	assertBodyExcludesSensitiveValues(t, resp.Body.String())
}

func TestProtectedMicroleaseHTTPProblemStatusMapping(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "payload conflict", err: NewProblemError(http.StatusConflict, "conflict", "payload_conflict"), wantStatus: http.StatusConflict},
		{name: "insufficient funds", err: NewProblemError(http.StatusUnprocessableEntity, "unprocessable entity", "insufficient_funds"), wantStatus: http.StatusUnprocessableEntity},
		{name: "throttle", err: NewProblemError(http.StatusTooManyRequests, "too many requests", "throttle"), wantStatus: http.StatusTooManyRequests},
		{name: "dependency not ready", err: NewProblemError(http.StatusServiceUnavailable, "not ready", "dependency_not_ready"), wantStatus: http.StatusServiceUnavailable},
		{name: "unexpected error", err: errors.New("database includes raw-prompt-secret but must not leak"), wantStatus: http.StatusInternalServerError},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			service := &fakeMicroleaseService{issueErr: tc.err}
			router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
				Microleases:          service,
				ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeMicroleaseWrite}},
			}, telemetry.New(), RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

			req := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/microleases/issue", strings.NewReader(mustJSON(t, validIssueRequest())))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer valid-service-token")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", resp.Code, tc.wantStatus, resp.Body.String())
			}
			assertProblemContentType(t, resp.Header())
			assertBodyExcludesSensitiveValues(t, resp.Body.String())
		})
	}
}

func TestProtectedMicroleaseReadCloseAndOperationMapServiceErrors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		scopes     []string
		path       string
		body       any
		service    *fakeMicroleaseService
		wantStatus int
		wantCalls  func(*fakeMicroleaseService) int
	}{
		{
			name:   "readback dependency unavailable",
			scopes: []string{scopeMicroleaseRead},
			path:   "/internal/billing/v1/microleases/readback",
			body:   validMicroleaseReadbackRequest(),
			service: &fakeMicroleaseService{
				readErr: NewProblemError(http.StatusServiceUnavailable, "not ready", "dependency_not_ready"),
			},
			wantStatus: http.StatusServiceUnavailable,
			wantCalls:  func(service *fakeMicroleaseService) int { return service.readCalls },
		},
		{
			name:   "close conflict",
			scopes: []string{scopeMicroleaseWrite},
			path:   "/internal/billing/v1/microleases/close",
			body:   validCloseRequest(),
			service: &fakeMicroleaseService{
				closeErr: NewProblemError(http.StatusConflict, "conflict", "close_replay_conflict"),
			},
			wantStatus: http.StatusConflict,
			wantCalls:  func(service *fakeMicroleaseService) int { return service.closeCalls },
		},
		{
			name:   "operation readback unexpected error",
			scopes: []string{scopeOperationsRead},
			path:   "/internal/billing/v1/operations/readback",
			body:   validOperationReadbackRequest(),
			service: &fakeMicroleaseService{
				operationErr: errors.New("store unavailable with raw-prompt-secret hidden"),
			},
			wantStatus: http.StatusInternalServerError,
			wantCalls:  func(service *fakeMicroleaseService) int { return service.operationCalls },
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
				Microleases:          tc.service,
				ServiceAuthenticator: headerScopeAuthenticator{scopes: tc.scopes},
			}, telemetry.New(), RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

			req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(mustJSON(t, tc.body)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer valid-service-token")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", resp.Code, tc.wantStatus, resp.Body.String())
			}
			if got := tc.wantCalls(tc.service); got != 1 {
				t.Fatalf("service calls = %d, want 1", got)
			}
			assertProblemContentType(t, resp.Header())
			assertBodyExcludesSensitiveValues(t, resp.Body.String())
		})
	}
}

func TestProtectedMicroleaseReadbacksKeepIdentifiersInBodies(t *testing.T) {
	t.Parallel()

	service := &fakeMicroleaseService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Microleases:          service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeOperationsRead}},
	}, telemetry.New(), RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	req := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/operations/readback", strings.NewReader(mustJSON(t, validOperationReadbackRequest())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-service-token")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	if service.operationCalls != 1 {
		t.Fatalf("operation readback calls = %d, want 1", service.operationCalls)
	}
	if service.operationRequest.OperationIdentity != "operation-ambiguous-timeout-1" {
		t.Fatalf("operation identity = %q, want body value", service.operationRequest.OperationIdentity)
	}
}

func TestProtectedMicroleaseCloseAndReadbackPreserveCloseSemantics(t *testing.T) {
	t.Parallel()

	service := &fakeMicroleaseService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Microleases:          service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeMicroleaseRead, scopeMicroleaseWrite}},
	}, telemetry.New(), RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	closeReq := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/microleases/close", strings.NewReader(mustJSON(t, validCloseRequest())))
	closeReq.Header.Set("Content-Type", "application/json")
	closeReq.Header.Set("Authorization", "Bearer valid-service-token")
	closeResp := httptest.NewRecorder()
	router.ServeHTTP(closeResp, closeReq)

	if closeResp.Code != http.StatusOK {
		t.Fatalf("close status = %d, want %d; body=%s", closeResp.Code, http.StatusOK, closeResp.Body.String())
	}
	if service.closeRequest.MicroleaseId != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("close microlease id = %q, want body value", service.closeRequest.MicroleaseId)
	}
	var closeResponse api.MicroleaseCloseResponse
	if err := json.Unmarshal(closeResp.Body.Bytes(), &closeResponse); err != nil {
		t.Fatalf("decode close response: %v", err)
	}
	if closeResponse.ReleasedUsdAtoms != 6_000_000 || closeResponse.UnresolvedReservedUsdAtoms != 0 {
		t.Fatalf("close response = %+v, want release from proven local remaining and no unresolved reserve", closeResponse)
	}

	readReq := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/microleases/readback", strings.NewReader(mustJSON(t, validMicroleaseReadbackRequest())))
	readReq.Header.Set("Content-Type", "application/json")
	readReq.Header.Set("Authorization", "Bearer valid-service-token")
	readResp := httptest.NewRecorder()
	router.ServeHTTP(readResp, readReq)
	if readResp.Code != http.StatusOK {
		t.Fatalf("read status = %d, want %d; body=%s", readResp.Code, http.StatusOK, readResp.Body.String())
	}
	if service.readCalls != 1 {
		t.Fatalf("read calls = %d, want 1", service.readCalls)
	}
	if service.readRequest.MicroleaseId == nil || *service.readRequest.MicroleaseId != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("read microlease id = %v, want body microlease id", service.readRequest.MicroleaseId)
	}
}

func TestProtectedMicroleaseCloseRejectsUnsafeOperatorMetadata(t *testing.T) {
	t.Parallel()

	service := &fakeMicroleaseService{}
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Microleases:          service,
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeMicroleaseWrite}},
	}, telemetry.New(), RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	request := validCloseRequest()
	request.SafeOperationMetadata = &api.SafeOperationMetadata{"operator_note": "raw_prompt=hello"}
	req := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/microleases/close", strings.NewReader(mustJSON(t, request)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-service-token")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", resp.Code, http.StatusBadRequest, resp.Body.String())
	}
	if service.closeCalls != 0 {
		t.Fatalf("close calls = %d, want 0", service.closeCalls)
	}
	assertBodyExcludesSensitiveValues(t, resp.Body.String())
}

func TestProtectedMicroleaseAuthFailuresKeepLowCardinalityRouteMetrics(t *testing.T) {
	t.Parallel()

	metrics := telemetry.New()
	router := mustNewRouter(t, slog.New(slog.DiscardHandler), Handlers{
		Microleases:          &fakeMicroleaseService{},
		ServiceAuthenticator: headerScopeAuthenticator{scopes: []string{scopeMicroleaseRead}},
	}, metrics, RouterConfig{ServiceAuth: ServiceAuthConfig{Enabled: true}})

	req := httptest.NewRequest(http.MethodPost, "/internal/billing/v1/microleases/issue", strings.NewReader(mustJSON(t, validIssueRequest())))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid-service-token")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	if resp.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusForbidden)
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsResp := httptest.NewRecorder()
	router.ServeHTTP(metricsResp, metricsReq)

	body := metricsResp.Body.String()
	if !strings.Contains(body, `method="POST",route="POST /internal/billing/v1/microleases/issue",status_code="403"`) {
		t.Fatalf("metrics output does not contain bounded route labels for protected microlease auth failure")
	}
	if strings.Contains(body, "acct_test") || strings.Contains(body, "idem-issue-1") || strings.Contains(body, "trace-issue-1") {
		t.Fatalf("metrics output leaked request identifiers: %s", body)
	}
}

func validIssueRequest() api.MicroleaseIssueRequest {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	return api.MicroleaseIssueRequest{
		AccountScopeKey:      "acct_test",
		ContractVersion:      api.V1,
		DeadlineAtEpochMs:    now.Add(10 * time.Second).UnixMilli(),
		IdempotencyKey:       "idem-issue-1",
		MaxChildCapUsdAtoms:  10_000_000,
		MaxRequestedUsdAtoms: 50_000_000,
		PricingSnapshot: api.PricingSnapshotEvidence{
			BaseAsset:           "GNK",
			ContractVersion:     "pricing.v1",
			DecisionAt:          now,
			NormalizedPrice:     "1.000000",
			PolicyVersion:       "pricing-policy-v1",
			PriceUnit:           api.QuotePerBaseUnit,
			PricingSnapshotId:   "pricing-snapshot-1",
			QuoteAsset:          api.USDT,
			SelectorKey:         "gnk_usdt:usage_reserve",
			SnapshotFingerprint: "pricing-fingerprint-1",
			UseClass:            api.PricingSnapshotEvidenceUseClassUsageReserve,
		},
		ProxyAllocatorOwnerId: "proxy-owner-1",
		RequestFingerprint:    "issue-fingerprint-1",
		RequestedCutoffClass:  api.MicroleaseIssueRequestRequestedCutoffClassStandard,
		RequestedTtlSeconds:   30,
		RiskClass:             api.MicroleaseIssueRequestRiskClassStandard,
		RouteContractId:       api.BillingMicroleaseIssue,
		TraceRequestId:        "trace-issue-1",
		UseClass:              api.MicroleaseIssueRequestUseClassUsageReserve,
		CallerContext: api.CallerContext{
			CallerPrincipal:   api.SvcGonkaProxy,
			CallerScopeKey:    "proxy-owner-1",
			DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
			RequestId:         "caller-request-1",
		},
	}
}

func validOperationReadbackRequest() api.BillingOperationReadbackRequest {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	return api.BillingOperationReadbackRequest{
		AccountScopeKey:   "acct_test",
		ContractVersion:   api.V1,
		DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
		OperationIdentity: "operation-ambiguous-timeout-1",
		RouteContractId:   api.BillingOperationReadback,
		TraceRequestId:    "trace-operation-1",
		CallerContext: api.CallerContext{
			CallerPrincipal:   api.SvcGonkaProxy,
			CallerScopeKey:    "proxy-owner-1",
			DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
			RequestId:         "caller-request-2",
		},
	}
}

func validMicroleaseReadbackRequest() api.MicroleaseReadbackRequest {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	microleaseID := "11111111-1111-1111-1111-111111111111"
	return api.MicroleaseReadbackRequest{
		AccountScopeKey:   "acct_test",
		ContractVersion:   api.V1,
		DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
		LookupKind:        api.MicroleaseId,
		MicroleaseId:      &microleaseID,
		RouteContractId:   api.BillingMicroleaseReadback,
		TraceRequestId:    "trace-read-1",
		CallerContext: api.CallerContext{
			CallerPrincipal:   api.SvcGonkaProxy,
			CallerScopeKey:    "proxy-owner-1",
			DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
			RequestId:         "caller-request-read-1",
		},
	}
}

func validCloseRequest() api.MicroleaseCloseRequest {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	return api.MicroleaseCloseRequest{
		AccountScopeKey:              "acct_test",
		AllocatedChildCapSumUsdAtoms: 4_000_000,
		AllocatedChildCount:          1,
		AllocatedChildHighWater:      1,
		CloseFingerprint:             "close-fingerprint-1",
		CloseKind:                    api.MicroleaseCloseRequestCloseKindClose,
		CloseSequence:                1,
		ContractVersion:              api.V1,
		DeadlineAtEpochMs:            now.Add(10 * time.Second).UnixMilli(),
		IdempotencyKey:               "idem-close-1",
		LocalRemainingUsdAtoms:       6_000_000,
		MicroleaseGeneration:         1,
		MicroleaseId:                 "11111111-1111-1111-1111-111111111111",
		ProxyAllocatorOwnerId:        "proxy-owner-1",
		RequestFingerprint:           "close-request-fingerprint-1",
		RouteContractId:              api.BillingMicroleaseClose,
		TerminalAcceptedCount:        1,
		TerminalPublishedCount:       1,
		TerminalSubmittedCount:       1,
		TraceRequestId:               "trace-close-1",
		CallerContext: api.CallerContext{
			CallerPrincipal:   api.SvcGonkaProxy,
			CallerScopeKey:    "proxy-owner-1",
			DeadlineAtEpochMs: now.Add(10 * time.Second).UnixMilli(),
			RequestId:         "caller-request-close-1",
		},
	}
}

func mustJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal json: %v", err)
	}
	return string(data)
}

func assertBodyExcludesSensitiveValues(t *testing.T, body string) {
	t.Helper()
	for _, forbidden := range []string{
		"raw-prompt-secret",
		"sk-",
		"Bearer ",
		"postgres://",
		"acct_test",
		"idem-issue-1",
		"trace-issue-1",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("body leaked forbidden value %q: %s", forbidden, body)
		}
	}
}
