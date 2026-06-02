package billingauthority

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidRequest = errors.New("billing authority request invalid")
	ErrNotReady       = errors.New("billing authority not ready")
	ErrNotFound       = errors.New("billing authority record not found")
	ErrConflict       = errors.New("billing authority payload conflict")
	ErrRejected       = errors.New("billing authority request rejected")
)

const (
	AuthorityModeMicroleaseChildDebit  = "billing_microlease_with_proxy_child_debit"
	AuthorityModeDirectReserveFallback = "direct_reserve_fallback"
)

type Store interface {
	ResolveAccount(context.Context, AccountResolveRequest) (AccountSnapshot, error)
	ReadBalance(context.Context, BalanceReadRequest) (BalanceSnapshot, error)
	ReserveUsage(context.Context, UsageReserveCommand) (UsageOperationSnapshot, error)
	CompleteUsage(context.Context, UsageTerminalCommand) (UsageOperationSnapshot, error)
	ReverseUsage(context.Context, UsageReversalCommand) (UsageOperationSnapshot, error)
	ReadUsageOperation(context.Context, UsageReadbackRequest) (UsageOperationSnapshot, error)
	ListReconciliationCases(context.Context, ReconciliationCasesRequest) ([]ReconciliationCase, error)
	ListLedgerEntries(context.Context, AdminLedgerRequest) ([]LedgerEntry, error)
	ReadExposure(context.Context, AdminExposureRequest) (ExposureSnapshot, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

type AccountResolveRequest struct {
	RepresentedSubjectID string
	RepresentedTenantID  string
	RepresentedAccountID string
	CallerPrincipal      string
	CallerScopeKey       string
	TraceRequestID       string
	DeadlineAt           time.Time
}

type AccountSnapshot struct {
	AccountID           string
	AccountScopeKey     string
	AccountState        string
	ImportState         string
	MigrationState      string
	BalanceReadEligible bool
	BalanceVersion      int64
	FailureClass        string
	Retryable           bool
}

type BalanceReadRequest struct {
	AccountScopeKey      string
	RepresentedSubjectID string
	TraceRequestID       string
	DeadlineAt           time.Time
}

type BalanceSnapshot struct {
	AccountID                    string
	AccountScopeKey              string
	AccountState                 string
	SettledUSDAtoms              int64
	ReservedUSDAtoms             int64
	AvailableUSDAtoms            int64
	PendingUSDAtoms              int64
	BalanceVersion               int64
	ImportState                  string
	RuntimeGateState             string
	ActiveMicroleaseUSDAtoms     int64
	ActiveUsageHoldUSDAtoms      int64
	UnresolvedChildDebitUSDAtoms int64
	ManualReview                 bool
	ReconciliationRequired       bool
	ReasonCode                   string
}

type PricingSnapshot struct {
	ID              string
	Fingerprint     string
	PolicyVersion   string
	DecisionAt      time.Time
	SelectorKey     string
	UseClass        string
	ContractVersion string
}

type UsageReserveCommand struct {
	AccountScopeKey        string
	UsageOperationID       string
	AuthorityMode          string
	IdempotencyKey         string
	RequestFingerprint     string
	RequestID              string
	MicroleaseID           string
	MicroleaseChildDebitID string
	DebitAuthorizationID   string
	ProxyAllocatorOwnerID  string
	MicroleaseGeneration   int64
	LeaseFence             string
	ChildSequence          int64
	ChildCapUSDAtoms       int64
	RepresentedSubjectID   string
	Pricing                PricingSnapshot
	CallerPrincipal        string
	CallerScopeKey         string
	TraceRequestID         string
	DeadlineAt             time.Time
	Metadata               map[string]string
}

type UsageTerminalCommand struct {
	AccountScopeKey              string
	UsageOperationID             string
	TerminalKind                 string
	IdempotencyKey               string
	RequestFingerprint           string
	MicroleaseID                 string
	MicroleaseChildDebitID       string
	DebitAuthorizationID         string
	TerminalOutcomeID            string
	TerminalFingerprint          string
	ChargedUSDAtoms              int64
	ReleasedUSDAtoms             int64
	WriteOffUSDAtoms             int64
	QualifiedInferenceEvidenceID string
	Pricing                      PricingSnapshot
	CallerPrincipal              string
	CallerScopeKey               string
	TraceRequestID               string
	DeadlineAt                   time.Time
	Metadata                     map[string]string
}

type UsageReversalCommand struct {
	AccountScopeKey       string
	UsageOperationID      string
	IdempotencyKey        string
	RequestFingerprint    string
	OriginalLedgerEntryID string
	ReversalUSDAtoms      int64
	ReasonCode            string
	CallerPrincipal       string
	CallerScopeKey        string
	TraceRequestID        string
	DeadlineAt            time.Time
	Metadata              map[string]string
}

type UsageReadbackRequest struct {
	AccountScopeKey  string
	UsageOperationID string
	TraceRequestID   string
	DeadlineAt       time.Time
}

type UsageOperationSnapshot struct {
	UsageOperationID     string
	BillingOperationID   string
	State                string
	ResultCode           string
	ReasonCode           string
	ReconciliationCaseID string
	IdempotencyKey       string
	RequestFingerprint   string
	StoredOutcomeID      string
}

type ReconciliationCasesRequest struct {
	AccountScopeKey string
	State           string
	Severity        string
	Limit           int32
}

type ReconciliationCase struct {
	ReconciliationCaseID string
	AccountScopeKey      string
	Reason               string
	State                string
	Severity             string
	SafeLineageID        string
}

type AdminLedgerRequest struct {
	AccountScopeKey string
	Limit           int32
}

type LedgerEntry struct {
	LedgerEntryID       string
	EffectType          string
	AmountUSDAtoms      int64
	BalanceVersionAfter int64
}

type AdminExposureRequest struct {
	AccountScopeKey string
}

type ExposureSnapshot struct {
	AccountScopeKey              string
	RuntimeGateState             string
	ActiveMicroleaseUSDAtoms     int64
	ActiveUsageHoldUSDAtoms      int64
	UnresolvedChildDebitUSDAtoms int64
}
