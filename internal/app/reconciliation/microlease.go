package reconciliation

import "time"

type MicroleaseSnapshot struct {
	MicroleaseID              string
	State                     string
	ExpiresAt                 time.Time
	LastTerminalAt            time.Time
	UnresolvedChildCapUSDAtom int64
	OldestUnresolvedChildAt   time.Time
}

type Policy struct {
	StaleChildCriticalAge time.Duration
	ReconciliationSLA     time.Duration
}

type Decision struct {
	OpenCase bool
	Reason   string
	DueBy    time.Time
	Severity string
}

func DecideMicroleaseRepair(policy Policy, snapshot MicroleaseSnapshot, now time.Time) Decision {
	if snapshot.MicroleaseID == "" || policy.ReconciliationSLA <= 0 {
		return Decision{}
	}
	if !snapshot.ExpiresAt.IsZero() && !now.Before(snapshot.ExpiresAt) && snapshot.UnresolvedChildCapUSDAtom > 0 {
		return Decision{
			OpenCase: true,
			Reason:   "stale_microlease",
			DueBy:    snapshot.ExpiresAt.Add(policy.ReconciliationSLA),
			Severity: severityByDeadline(snapshot.ExpiresAt.Add(policy.ReconciliationSLA), now),
		}
	}
	if policy.StaleChildCriticalAge > 0 && !snapshot.OldestUnresolvedChildAt.IsZero() {
		criticalAt := snapshot.OldestUnresolvedChildAt.Add(policy.StaleChildCriticalAge)
		if !now.Before(criticalAt) {
			return Decision{
				OpenCase: true,
				Reason:   "stale_child_debit",
				DueBy:    criticalAt.Add(policy.ReconciliationSLA),
				Severity: severityByDeadline(criticalAt, now),
			}
		}
	}
	return Decision{}
}

func severityByDeadline(deadline, now time.Time) string {
	if now.After(deadline) || now.Equal(deadline) {
		return "high"
	}
	return "medium"
}
