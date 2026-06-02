package microlease

import (
	"errors"
	"time"
)

var (
	ErrInvalidConfig   = errors.New("microlease config invalid")
	ErrInvalidRequest  = errors.New("microlease request invalid")
	ErrPayloadConflict = errors.New("microlease idempotency payload conflict")
	ErrUnsafeMetadata  = errors.New("microlease metadata is not support-safe")
)

type Config struct {
	MaxMicroleaseUSDAtoms            int64
	AccountMicroleaseExposureCapAtom int64
	MinSafetyFloorUSDAtoms           int64
	TTL                              time.Duration
	DebitCutoffBeforeExpiry          time.Duration
	TerminalDeadline                 time.Duration
	StaleDebitWarningAge             time.Duration
	StaleDebitCriticalAge            time.Duration
	ReconciliationSLA                time.Duration
}

type PricingSnapshot struct {
	ID              string
	Fingerprint     string
	PolicyVersion   string
	DecisionAt      time.Time
	SelectorKey     string
	ContractVersion string
}

func (p PricingSnapshot) Valid(now time.Time) bool {
	return p.ID != "" &&
		p.Fingerprint != "" &&
		p.PolicyVersion != "" &&
		p.SelectorKey != "" &&
		p.ContractVersion != "" &&
		!p.DecisionAt.IsZero() &&
		!p.DecisionAt.After(now)
}

type AccountExposure struct {
	SettledUSDAtoms            int64
	ReservedUSDAtoms           int64
	ActiveMicroleaseUSDAtoms   int64
	ManualReview               bool
	Suspended                  bool
	AdmissionControl           AdmissionControl
	TerminalLagCritical        bool
	ReconciliationBacklogBlock bool
}

func (e AccountExposure) AvailableUSDAtoms() int64 {
	return e.SettledUSDAtoms - e.ReservedUSDAtoms
}

type AdmissionControl struct {
	State     AdmissionState
	Reason    string
	ExpiresAt time.Time
}

type AdmissionState string

const (
	AdmissionStateOpen       AdmissionState = "open"
	AdmissionStateThrottle   AdmissionState = "throttle"
	AdmissionStateStrict     AdmissionState = "strict"
	AdmissionStateFailClosed AdmissionState = "fail_closed"
)

type IssueRequest struct {
	AccountScopeKey     string
	ProxyAllocatorOwner string
	Generation          int64
	LeaseFence          string
	IdempotencyKey      string
	RequestFingerprint  string
	RequestedUSDAtoms   int64
	MaxChildUSDAtoms    int64
	UseClass            string
	RiskClass           string
	Pricing             PricingSnapshot
	Metadata            map[string]string
	Now                 time.Time
}

type IssueDecision struct {
	Result                 IssueResult
	StrictReason           string
	IssuedUSDAtoms         int64
	AvailableChildUSDAtoms int64
	DebitCutoffAt          time.Time
	ExpiresAt              time.Time
}

type IssueResult string

const (
	IssueResultIssued             IssueResult = "issued"
	IssueResultReducedIssued      IssueResult = "reduced_issued"
	IssueResultStrictRequired     IssueResult = "strict_required"
	IssueResultRejected           IssueResult = "rejected"
	IssueResultPayloadConflict    IssueResult = "payload_conflict"
	IssueResultManualReview       IssueResult = "manual_review"
	IssueResultReconcileRequired  IssueResult = "reconcile_required"
	IssueResultInsufficientFunds  IssueResult = "insufficient_funds"
	IssueResultStalePricing       IssueResult = "stale_pricing_snapshot"
	IssueResultAdmissionFailClose IssueResult = "admission_fail_closed"
)

type StoredOutcome[T any] struct {
	IdempotencyKey     string
	RequestFingerprint string
	Outcome            T
}

type MicroleaseState string

const (
	MicroleaseStateActive            MicroleaseState = "active"
	MicroleaseStateCutoff            MicroleaseState = "cutoff"
	MicroleaseStateClosing           MicroleaseState = "closing"
	MicroleaseStateClosed            MicroleaseState = "closed"
	MicroleaseStateExpired           MicroleaseState = "expired"
	MicroleaseStateReconcileRequired MicroleaseState = "reconcile_required"
	MicroleaseStateManualReview      MicroleaseState = "manual_review"
	MicroleaseStateCanceled          MicroleaseState = "canceled"
)

type GrantSnapshot struct {
	MicroleaseID                 string
	State                        MicroleaseState
	IssuedCapUSDAtoms            int64
	AvailableChildCapUSDAtoms    int64
	AllocatedChildCapUSDAtoms    int64
	TerminalChargedUSDAtoms      int64
	TerminalReleasedUSDAtoms     int64
	WriteOffUSDAtoms             int64
	LastCheckpointSequence       int64
	LastCheckpointFingerprint    string
	ExpiresAt                    time.Time
	ClosedAt                     time.Time
	ProxyAllocatorOwner          string
	Generation                   int64
	LeaseFence                   string
	ReconciliationRequiredReason string
}

type ChildDebit struct {
	AuthorizationID          string
	Sequence                 int64
	ChildCapUSDAtoms         int64
	RequestBasisFingerprint  string
	TerminalBasisFingerprint string
}

type TerminalReport struct {
	Child       ChildDebit
	Kind        TerminalKind
	Charged     int64
	Released    int64
	WriteOff    int64
	ObservedAt  time.Time
	EventID     string
	Fingerprint string
}

type TerminalKind string

const (
	TerminalKindFinalize     TerminalKind = "finalize"
	TerminalKindWriteOff     TerminalKind = "write_off"
	TerminalKindAbortRelease TerminalKind = "abort_release"
	TerminalKindReversal     TerminalKind = "reversal"
	TerminalKindCompensation TerminalKind = "compensation"
)

type TerminalDecision struct {
	ChargedUSDAtoms        int64
	ReleasedUSDAtoms       int64
	WriteOffUSDAtoms       int64
	ReconcileRequired      bool
	ReconciliationReason   string
	ParentRemainingUSDAtom int64
}

type CloseProof struct {
	Sequence                      int64
	Kind                          string
	AllocatedChildCapUSDAtoms     int64
	TerminalAcceptedCount         int64
	UnresolvedChildCount          int64
	UnresolvedChildCapUSDAtoms    int64
	LocalRemainingUSDAtoms        int64
	CheckpointFingerprint         string
	ProxyAllocatorOwner           string
	Generation                    int64
	LeaseFence                    string
	ProofObservedAt               time.Time
	ReleaseRequiresTerminalProofs bool
}

type CloseDecision struct {
	State                     MicroleaseState
	ReleasedUSDAtoms          int64
	UnresolvedReservedUSDAtom int64
	ReconcileRequired         bool
	ReconciliationReason      string
}
