package httpidempotency

// ReservationRecovery records why the writer allowed a reservation to proceed.
type ReservationRecovery uint8

const (
	ReservationRecoveryNone ReservationRecovery = iota + 1
	ReservationRecoveryDue
	ReservationRecoveryReconciled
)

// Reservation carries the writer-issued generation fence between reservation
// classification and the endpoint-owned transaction.
type Reservation struct {
	Attempt    Attempt
	Generation int64
	Recovery   ReservationRecovery
}
