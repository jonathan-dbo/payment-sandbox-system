package merchant

// MerchantStatus represents the lifecycle state of a merchant account.
type MerchantStatus string

const (
	StatusPending   MerchantStatus = "PENDING"
	StatusActive    MerchantStatus = "ACTIVE"
	StatusSuspended MerchantStatus = "SUSPENDED"
	StatusInactive  MerchantStatus = "INACTIVE"
)

// IsValid returns true when the status is a known merchant state.
func (ms MerchantStatus) IsValid() bool {
	switch ms {
	case StatusPending, StatusActive, StatusSuspended, StatusInactive:
		return true
	default:
		return false
	}
}

// CanTransitionTo returns true when the next status is allowed by the domain rules.
func (ms MerchantStatus) CanTransitionTo(next MerchantStatus) bool {
	switch ms {
	case StatusPending:
		return next == StatusActive || next == StatusSuspended || next == StatusInactive
	case StatusActive:
		return next == StatusSuspended || next == StatusInactive
	case StatusSuspended:
		return next == StatusActive || next == StatusInactive
	case StatusInactive:
		return false
	default:
		return false
	}
}
