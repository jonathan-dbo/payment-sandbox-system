package payment

// PaymentStatus represents the state of a payment in the domain.
type PaymentStatus string

const (
	// StatusPending: Payment created but not yet authorized.
	StatusPending PaymentStatus = "PENDING"

	// StatusAuthorized: Payment authorized by merchant, ready for capture.
	StatusAuthorized PaymentStatus = "AUTHORIZED"

	// StatusCaptured: Payment funds have been captured.
	StatusCaptured PaymentStatus = "CAPTURED"

	// StatusRefunded: Payment has been refunded (partially or fully).
	StatusRefunded PaymentStatus = "REFUNDED"

	// StatusFailed: Payment authorization or capture failed.
	StatusFailed PaymentStatus = "FAILED"
)

// IsValid checks if the status is a valid PaymentStatus.
func (ps PaymentStatus) IsValid() bool {
	switch ps {
	case StatusPending, StatusAuthorized, StatusCaptured, StatusRefunded, StatusFailed:
		return true
	default:
		return false
	}
}

// String implements fmt.Stringer for PaymentStatus.
func (ps PaymentStatus) String() string {
	return string(ps)
}

// CanTransitionTo checks if a transition from current status to next status is valid.
func (ps PaymentStatus) CanTransitionTo(next PaymentStatus) bool {
	transitions := map[PaymentStatus]map[PaymentStatus]bool{
		StatusPending: {
			StatusAuthorized: true,
			StatusFailed:     true,
		},
		StatusAuthorized: {
			StatusCaptured: true,
			StatusRefunded: true,
			StatusFailed:   true,
		},
		StatusCaptured: {
			StatusRefunded: true,
		},
		StatusRefunded: {},
		StatusFailed:   {},
	}

	if allowed, exists := transitions[ps]; exists {
		return allowed[next]
	}
	return false
}

// IsTerminalState returns true if the payment status is terminal (no further transitions).
func (ps PaymentStatus) IsTerminalState() bool {
	return ps == StatusRefunded || ps == StatusFailed
}
